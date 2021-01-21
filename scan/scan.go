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

// readPortsRange transforms a range of ports given in conf to an array of
// effective ports
func readPortsRange(ranges string) ([]int, error) {
	ports := []int{}

	parts := strings.Split(ranges, ",")

	for _, spec := range parts {
		if spec == "" {
			continue
		}
		switch spec {
		case "all":
			for port := 1; port <= 65535; port++ {
				ports = append(ports, port)
			}
		case "reserved":
			for port := 1; port < 1024; port++ {
				ports = append(ports, port)
			}
		default:
			var decomposedRange []string

			if !strings.Contains(spec, "-") {
				decomposedRange = []string{spec, spec}
			} else {
				decomposedRange = strings.Split(spec, "-")
			}

			min, err := strconv.Atoi(decomposedRange[0])
			if err != nil {
				return nil, err
			}
			max, err := strconv.Atoi(decomposedRange[len(decomposedRange)-1])
			if err != nil {
				return nil, err
			}

			if min > max {
				return nil, fmt.Errorf("lower port %d is higher than high port %d", min, max)
			}
			if max > 65535 {
				return nil, fmt.Errorf("port %d is higher than max port", max)
			}
			for i := min; i <= max; i++ {
				ports = append(ports, i)
			}
		}
	}

	return ports, nil
}

// getDuration transforms a protocol's period into a time.Duration value.
func getDuration(period string) (time.Duration, error) {
	// only hours, minutes and seconds are handled by ParseDuration
	if strings.ContainsAny(period, "hms") {
		t, err := time.ParseDuration(period)
		if err != nil {
			return 0, err
		}
		return t, nil
	}

	sep := strings.Split(period, "d")
	days, err := strconv.Atoi(sep[0])
	if err != nil {
		return 0, err
	}

	t := time.Duration(days) * time.Hour * 24
	return t, nil
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
