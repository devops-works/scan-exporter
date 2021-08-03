package scan

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/devops-works/scan-exporter/common"
	"github.com/devops-works/scan-exporter/config"
	"github.com/devops-works/scan-exporter/metrics"
	"github.com/devops-works/scan-exporter/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
)

type target struct {
	ip         string
	name       string
	ports      string
	expected   []string
	doTCP      bool
	doPing     bool
	tcpPeriod  string
	icmpPeriod string
	qps        int
}

// Scanner holds the targets list, global settings such as timeout and lock size,
// the logger and the metrics server.
type Scanner struct {
	Targets     []target
	Timeout     time.Duration
	Lock        *semaphore.Weighted
	Logger      zerolog.Logger
	MetricsServ metrics.Server
}

// Start configure targets and launches scans.
func (s *Scanner) Start(c *config.Conf) error {

	s.Logger.Info().Msgf("%d target(s) found in configuration file", len(c.Targets))
	s.MetricsServ.NumOfTargets.Set(float64(len(c.Targets)))

	// Check if shared values are set
	if c.Timeout == 0 {
		s.Logger.Fatal().Msgf("no timeout provided in configuration file")
	}
	if c.Limit == 0 {
		s.Logger.Fatal().Msgf("no limit provided in configuration file")
	}
	s.Lock = semaphore.NewWeighted(int64(c.Limit))
	s.Timeout = time.Second * time.Duration(c.Timeout)

	// If an ICMP period has been provided, it means that we want to ping the
	// target. But before, we need to check if we have enough privileges.
	if os.Geteuid() != 0 {
		s.Logger.Warn().Msgf("scan-exporter not launched as superuser, ICMP requests can fail")
	}

	// ping channel to send ICMP update to metrics
	pchan := make(chan metrics.PingInfo, len(c.Targets)*2)

	// Configure local target objects
	for _, t := range c.Targets {
		target := target{
			ip:         t.IP,
			name:       t.Name,
			tcpPeriod:  t.TCP.Period,
			icmpPeriod: t.ICMP.Period,
			ports:      t.TCP.Range,
			qps:        t.QueriesPerSecond,
		}

		// Set to global values if specific values are not set
		if target.qps == 0 {
			target.qps = c.QueriesPerSecond
		}
		if target.tcpPeriod == "" {
			target.tcpPeriod = c.TcpPeriod
		}
		if target.icmpPeriod == "" {
			target.icmpPeriod = c.IcmpPeriod
		}

		// Truth table for icmpPeriod value
		//
		// | global | target | doPing | period |
		// | ------ | ------ | ------ | :----: |
		// | ""     | ""     | false  |   -    |
		// | ""     | "0"    | false  |   -    |
		// | ""     | "y"    | true   |   y    |
		// | "0"    | ""     | false  |   -    |
		// | "0"    | "0"    | false  |   -    |
		// | "0"    | "y"    | true   |   y    |
		// | "x"    | ""     | true   |   x    |
		// | "x"    | "0"    | false  |   -    |
		// | "x"    | "y"    | true   |   y    |
		switch c.IcmpPeriod {
		case "", "0":
			if target.icmpPeriod != "" && target.icmpPeriod != "0" {
				target.doPing = true
			}
		default:
			if target.icmpPeriod != "0" {
				target.doPing = true
				if target.icmpPeriod == "" {
					target.icmpPeriod = c.IcmpPeriod
				}
			}
		}
		// Inform that ping is disabled
		if !target.doPing {
			s.Logger.Warn().Msgf("ping explicitly disabled for %s (%s) in configuration",
				target.name,
				target.ip)
		}

		// Read target's expected port range
		exp, err := readPortsRange(t.TCP.Expected)
		if err != nil {
			return err
		}

		// Append them to the target
		for _, port := range exp {
			target.expected = append(target.expected, strconv.Itoa(port))
		}

		// Inform that we can't parse the IP, and skip this target
		if ok := net.ParseIP(target.ip); ok == nil {
			s.Logger.Error().Msgf("cannot parse IP %s", target.ip)
			continue
		}

		// If TCP period or ports range has been provided, it means that we want
		// to do TCP scan on the target
		if target.tcpPeriod != "" || target.ports != "" || len(target.expected) != 0 {
			target.doTCP = true
		}

		// Launch target's ping goroutine. It embeds its own ticker
		if target.doPing {
			go target.ping(s.Logger, time.Duration(c.Timeout)*time.Second, pchan)
		}

		if target.doTCP {
			s.Targets = append(s.Targets, target)
		}
	}

	trigger := make(chan string, len(s.Targets)*2)

	// scanIsOver is used by s.run() to notify the receiver that all the ports
	// have been scanned
	scanIsOver := make(chan target, len(s.Targets))

	// singleResult is used by s.scanPort() to send an open port to the receiver.
	// The format is ip:port
	singleResult := make(chan string, c.Limit)

	s.Logger.Debug().Msgf("%d targets will be scanned using TCP", len(s.Targets))

	// Start scheduler for each target
	for _, t := range s.Targets {
		t := t
		s.Logger.Debug().Msgf("start scheduler for %s", t.name)
		go t.scheduler(s.Logger, trigger)
	}

	// Create channel for communication with metrics server
	mchan := make(chan metrics.NewMetrics, len(s.Targets)*2)

	// Channel that will hold the number of scans in the waiting line (len of
	// the trigger chan)
	pendingchan := make(chan int, len(s.Targets))

	// Goroutine that will send to metrics the number of pendings scan
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			pendingchan <- len(trigger)
		}
	}()

	// Start the metrics updater
	go s.MetricsServ.Updater(mchan, pchan, pendingchan)

	// Start the receiver
	go receiver(scanIsOver, singleResult, pchan, mchan)

	// Wait for triggers, build the scanner and run it
	for {
		select {
		case triggeredIP := <-trigger:
			s.Logger.Debug().Msgf("starting new scan for %s", triggeredIP)
			if err := s.run(triggeredIP, scanIsOver, singleResult); err != nil {
				s.Logger.Error().Err(err).Msg("error running scan")
			}
		}
	}
}

func (s *Scanner) run(ip string, scanIsOver chan target, singleResult chan string) error {
	for _, t := range s.Targets {
		// Find which target to scan
		if t.ip == ip {
			wg := sync.WaitGroup{}

			ports, err := readPortsRange(t.ports)
			if err != nil {
				return err
			}

			// Configure sleeping time for rate limiting
			var sleepingTime time.Duration
			if t.qps > 1000000 || t.qps <= 0 {
				// We want to wait less than a microsecond between each port scanning
				// so, we do not wait at all.
				// From time.Sleep documentation:
				// A negative or zero duration causes Sleep to return immediately
				sleepingTime = -1
				t.qps = 0
			} else {
				sleepingTime = time.Second / time.Duration(t.qps)
			}

			for _, p := range ports {
				wg.Add(1)
				s.Lock.Acquire(context.TODO(), 1)
				go func(port int) {
					defer s.Lock.Release(1)
					defer wg.Done()
					s.scanPort(ip, port, singleResult)
				}(p)
				time.Sleep(sleepingTime)
			}
			wg.Wait()

			// Inform the receiver that the scan for the target is over
			scanIsOver <- t
			return nil
		}
	}
	return fmt.Errorf("IP to scan not found: %s", ip)
}

// scanPort scans a single port and sends the result through singleResult.
// There is 2 formats: when a port is open, it sends `ip:port:OK`, and when it is
// closed, it sends `ip:port:NOP`
func (s *Scanner) scanPort(ip string, port int, singleResult chan string) {
	p := strconv.Itoa(port)
	target := ip + ":" + p
	conn, err := net.DialTimeout("tcp", target, s.Timeout)
	if err != nil {
		// If the error contains the message "too many open files", wait a little
		// and retry
		if strings.Contains(err.Error(), "too many open files") {
			time.Sleep(s.Timeout)
			s.scanPort(ip, port, singleResult)
		}
		// The result follows the format ip:port:NOP
		singleResult <- ip + ":" + p + ":NOP"
		return
	}
	conn.Close()

	// The result follows the format ip:port:OK
	singleResult <- ip + ":" + p + ":OK"
}

// scheduler create tickers for each protocol given and when they tick,
// it sends the protocol name in the trigger's channel in order to alert
// feeder that a scan must be started.
func (t *target) scheduler(logger zerolog.Logger, trigger chan string) {
	var ticker *time.Ticker
	tcpFreq, err := getDuration(t.tcpPeriod)
	if err != nil {
		logger.Error().Msgf("error getting TCP frequency for %s scheduler: %s", t.name, err)
	}
	ticker = time.NewTicker(tcpFreq)

	// starts its own ticker
	go func(trigger chan string, ticker *time.Ticker, ip string) {
		// Start scan at launch
		trigger <- t.ip
		for {
			select {
			case <-ticker.C:
				trigger <- t.ip
			}
		}
	}(trigger, ticker, t.ip)
}

func receiver(scanIsOver chan target, singleResult chan string, pchan chan metrics.PingInfo, mchan chan metrics.NewMetrics) {
	// openPorts holds the ports that are open for each target
	openPorts := make(map[string][]string)
	// closedPorts holds the ports that are closed
	closedPorts := make(map[string][]string)

	// Create the store for the values
	store := storage.Create()

	for {
		select {
		case t := <-scanIsOver:
			// Compare stored results with current results and get the delta
			delta := common.CompareStringSlices(store.Get(t.ip), openPorts[t.ip])

			// Update metrics
			updatedMetrics := metrics.NewMetrics{
				Name:     t.name,
				IP:       t.ip,
				Diff:     delta,
				Open:     openPorts[t.ip],
				Closed:   closedPorts[t.ip],
				Expected: t.expected,
			}

			// Send new metrics
			mchan <- updatedMetrics

			// Update the store
			store.Update(t.ip, openPorts[t.ip])

			// Clear slices
			openPorts[t.ip] = nil
			closedPorts[t.ip] = nil
		case res := <-singleResult:
			split := strings.Split(res, ":")
			// Useless allocations, but it's easier to read
			ip := string(split[0])
			port := string(split[1])
			status := string(split[2])

			if status == "OK" {
				openPorts[ip] = append(openPorts[ip], port)
			} else if status == "NOP" {
				closedPorts[ip] = append(closedPorts[ip], port)
			} else {
				log.Fatal().Msgf("port status not recognised: %s (%s)", status, ip)
			}
		}
	}
}
