package scan

import (
	"context"
	"fmt"
	"net"
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

	for _, t := range targetList {
		go t.scheduler(trigger)
	}

	// infinite for loop that waits for signals
	for {
		select {
		case triggeredIP := <-trigger:
			fmt.Println(triggeredIP)
			for _, t := range targetList {
				if t.ip == triggeredIP {
					s.ip = t.ip
					s.name = t.name
					s.ports = t.ports

					s.run()
				}
			}
		}
	}
}

// Run runs the portScanner.
func (ps *scanner) run() error {
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
			ps.scanPort(port)
		}(p)
	}
	wg.Wait()
	return nil
}

func (ps *scanner) scanPort(port int) {
	target := fmt.Sprintf("%s:%d", ps.ip, port)
	conn, err := net.DialTimeout("tcp", target, ps.shared.timeout)
	if err != nil {
		if strings.Contains(err.Error(), "too many open files") {
			time.Sleep(ps.shared.timeout)
			ps.scanPort(port)
		}
		return
	}

	conn.Close()
	fmt.Printf("%s:%d/tcp  \topen\n", ps.ip, port)
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
		// First scan at the start
		trigger <- t.ip

		for {
			select {
			case <-ticker.C:
				trigger <- t.ip
			}
		}
	}(trigger, ticker, t.ip)
}
