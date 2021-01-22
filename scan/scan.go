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
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/semaphore"
)

type pinger struct {
	ip     string
	name   string
	period string
}

type target struct {
	ip         string
	name       string
	ports      string
	expected   []string
	doTCP      bool
	doPing     bool
	tcpPeriod  string
	icmpPeriod string
	shared     sharedConf
}

// sharedConf will not change during program execution
type sharedConf struct {
	timeout time.Duration
	lock    *semaphore.Weighted
}

// Start configure targets and launches scans.
func Start(c *config.Conf) error {
	var targetList []target

	// Configure shared values
	if c.Timeout == 0 {
		log.Fatal().Msgf("no timeout provided in configuration file")
	}
	if c.Limit == 0 {
		log.Fatal().Msgf("no limit provided in configuration file")
	}

	// If an ICMP period has been provided, it means that we want to ping the
	// target. But before, we need to check if we have enough privileges
	if os.Getenv("SUDO_USER") == "" {
		log.Warn().Msgf("scan-exporter not launched as superuser, ICMP requests can fail")
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
		}

		exp, err := readPortsRange(t.TCP.Expected)
		if err != nil {
			return err
		}

		for _, port := range exp {
			target.expected = append(target.expected, strconv.Itoa(port))
		}

		target.shared.timeout = time.Second * time.Duration(c.Timeout)
		target.shared.lock = semaphore.NewWeighted(int64(c.Limit))

		// Inform that we can't parse the IP, and skip this target
		if ok := net.ParseIP(target.ip); ok == nil {
			log.Error().Msgf("cannot parse IP %s", target.ip)
			continue
		}

		if target.icmpPeriod != "" {
			target.doPing = true
		}

		// If TCP period or ports range has been provided, it means that we want
		// to do TCP scan on the target
		if target.tcpPeriod != "" || target.ports != "" || len(target.expected) != 0 {
			target.doTCP = true
		}

		// Launch target's ping goroutine. It embeds its own ticker
		if target.doPing {
			go target.ping(time.Duration(c.Timeout)*time.Second, pchan)
		}

		if target.doTCP {
			targetList = append(targetList, target)
		}
	}

	trigger := make(chan string, len(targetList)*2)

	// scanIsOver is used by s.run() to notify the receiver that all the ports
	// fave been scanned
	scanIsOver := make(chan target, len(targetList))

	// singleResult is used by s.scanPort() to send an open port to the receiver.
	// The format is ip:port
	singleResult := make(chan string, c.Limit)

	// Start scheduler for each target
	for _, t := range targetList {
		t := t
		go t.scheduler(trigger)
	}

	// Create channel for communication with metrics server
	mchan := make(chan metrics.NewMetrics, len(targetList)*2)

	// Channel that will hold the number of scans in the waiting line (len of
	// the trigger chan)
	pendingchan := make(chan int, len(targetList))

	// Goroutine that will send to metrics the number of pendings scan
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			pendingchan <- len(trigger)
		}
	}()

	// Init and start the metrics server
	mserver := metrics.Init()
	go func(nt int) {
		if err := mserver.StartServ(nt); err != nil {
			log.Fatal().Err(err).Msg("metrics server failed critically")
		}
	}(len(targetList))
	go mserver.Updater(mchan, pchan, pendingchan)

	// Start the receiver
	go receiver(scanIsOver, singleResult, pchan, mchan)

	// Wait for triggers, build the scanner and run it
	for {
		select {
		case triggeredIP := <-trigger:
			for _, t := range targetList {
				if t.ip == triggeredIP {
					t.run(scanIsOver, singleResult)
				}
			}
		}
	}
}

// Run runs the portScanner.
func (t *target) run(scanIsOver chan target, singleResult chan string) error {
	wg := sync.WaitGroup{}

	ports, err := readPortsRange(t.ports)
	if err != nil {
		return err
	}

	for _, p := range ports {
		wg.Add(1)
		t.shared.lock.Acquire(context.TODO(), 1)
		go func(port int) {
			defer t.shared.lock.Release(1)
			defer wg.Done()
			t.scanPort(port, singleResult)
		}(p)
	}
	wg.Wait()
	// Inform the receiver that the scan for the target is over
	scanIsOver <- *t
	return nil
}

// scanPort scans a single port and sends the result through singleResult.
// There is 2 formats: when a port is open, it sends `ip:port:OK`, and when it is
// closed, it sends `ip:port:NOP`
func (t *target) scanPort(port int, singleResult chan string) {
	target := fmt.Sprintf("%s:%d", t.ip, port)
	conn, err := net.DialTimeout("tcp", target, t.shared.timeout)
	if err != nil {
		if strings.Contains(err.Error(), "too many open files") {
			time.Sleep(t.shared.timeout)
			t.scanPort(port, singleResult)
		}
		// The result follows the format ip:port:NOP
		singleResult <- t.ip + ":" + strconv.Itoa(port) + ":NOP"
		return
	}

	conn.Close()
	// The result follows the format ip:port:OK
	singleResult <- t.ip + ":" + strconv.Itoa(port) + ":OK"
}

// scheduler create tickers for each protocol given and when they tick,
// it sends the protocol name in the trigger's channel in order to alert
// feeder that a scan must be started.
func (t *target) scheduler(trigger chan string) {
	var ticker *time.Ticker
	tcpFreq, err := getDuration(t.tcpPeriod)
	if err != nil {
		log.Error().Msgf("error getting TCP frequency in scheduler: %s", err)
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
