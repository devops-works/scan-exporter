package scan

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/devops-works/scan-exporter/config"
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
	expected   string
	doTCP      bool
	doPing     bool
	tcpPeriod  string
	icmpPeriod string
}

type scanner struct {
	name   string
	ip     string
	ports  string
	shared sharedConf
}

// sharedConf will not change during program execution
type sharedConf struct {
	timeout time.Duration
	lock    *semaphore.Weighted
}

// Start configure targets and launches scans.
func Start(c *config.Conf) error {
	var s scanner
	var targetList []target

	// Configure shared values
	s.shared.timeout = time.Second * time.Duration(c.Timeout)
	s.shared.lock = semaphore.NewWeighted(int64(c.Limit))

	// Configure local target objects
	for _, t := range c.Targets {
		target := target{
			ip:         t.IP,
			name:       t.Name,
			tcpPeriod:  t.TCP.Period,
			icmpPeriod: t.ICMP.Period,
			ports:      t.TCP.Range,
			expected:   t.TCP.Expected,
		}

		// If an ICMP period has been provided, it means that we want to ping the
		// target
		if target.icmpPeriod != "" {
			target.doPing = true
		}

		// If TCP period or ports range has been provided, it means that we want
		// to do TCP scan on the target
		if target.tcpPeriod != "" || target.ports != "" || target.expected != "" {
			target.doTCP = true
		}

		// Launch target's ping goroutine. It embeds its own ticker
		if target.doPing {
			go target.ping(time.Duration(c.Timeout) * time.Second)
		}

		if target.doTCP {
			targetList = append(targetList, target)
		}
	}

	trigger := make(chan string, len(targetList))

	// scanIsOver is used by s.run() to notify the receiver that all the ports
	// fave been scanned
	scanIsOver := make(chan string, len(targetList))

	// singleResult is used by s.scanPort() to send an open port to the receiver.
	// The format is ip:port
	singleResult := make(chan string, c.Limit)

	// Start scheduler for each target
	for _, t := range targetList {
		t := t
		go t.scheduler(trigger)
	}

	// Start the receiver
	go receiver(scanIsOver, singleResult)

	// Wait for triggers, build the scanner and run it
	for {
		select {
		case triggeredIP := <-trigger:
			for _, t := range targetList {
				if t.ip == triggeredIP {
					s.ip = t.ip
					s.name = t.name
					s.ports = t.ports

					s.run(scanIsOver, singleResult)
				}
			}
		}
	}
}

// Run runs the portScanner.
func (ps *scanner) run(scanIsOver, singleResult chan string) error {
	wg := sync.WaitGroup{}

	ports, err := readPortsRange(ps.ports)
	if err != nil {
		return err
	}

	for _, p := range ports {
		wg.Add(1)
		ps.shared.lock.Acquire(context.TODO(), 1)
		go func(port int) {
			defer ps.shared.lock.Release(1)
			defer wg.Done()
			ps.scanPort(port, singleResult)
		}(p)
	}
	wg.Wait()
	// Inform the receiver that the scan for the target is over
	scanIsOver <- ps.ip
	return nil
}

// scanPort scans a single port and sends the result through singleResult.
// There is 2 formats: when a port is open, it sends ip:port:OK, and when it is
// closed, it sends ip:port:NOP
func (ps *scanner) scanPort(port int, singleResult chan string) {
	target := fmt.Sprintf("%s:%d", ps.ip, port)
	conn, err := net.DialTimeout("tcp", target, ps.shared.timeout)
	if err != nil {
		if strings.Contains(err.Error(), "too many open files") {
			time.Sleep(ps.shared.timeout)
			ps.scanPort(port, singleResult)
		}
		// The result follows the format ip:port:NOP
		singleResult <- ps.ip + ":" + strconv.Itoa(port) + ":NOP"
		return
	}

	conn.Close()
	// The result follows the format ip:port:OK
	singleResult <- ps.ip + ":" + strconv.Itoa(port) + ":OK"
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

func receiver(scanIsOver, singleResult chan string) {
	// openPorts holds the ports that are open for each target
	openPorts := make(map[string][]string)
	// closedPorts holds the ports that are closed
	closedPorts := make(map[string][]string)

	for {
		select {
		case ipEnded := <-scanIsOver:
			log.Info().Msgf("%s open ports: %s", ipEnded, openPorts[ipEnded])

			// TODO: send to datastore ?

			// Clear slices
			openPorts[ipEnded] = nil
			closedPorts[ipEnded] = nil
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
