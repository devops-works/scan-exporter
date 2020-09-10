package scan

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/sparrc/go-ping"
)

// Target holds an IP and a range of ports to scan
type Target struct {
	Name   string   `yaml:"name"`
	Period string   `yaml:"period"`
	IP     string   `yaml:"ip"`
	TCP    protocol `yaml:"tcp"`
	UDP    protocol `yaml:"udp"`
	// {tcp,udp}PortsToScan holds all the ports that will be scanned
	// those fields are fielded after having parsed the range given in
	// config file.
	// THOSE SLICES SHOULD BE MAPS :
	// map[protocol][port]
	// 2 elements instead of 4, but more computing time (read key...)
	tcpPortsToScan []string
	udpPortsToScan []string
	// those arrays will hold open ports
	tcpPortsOpen []string
	udpPortsOpen []string
}

type protocol struct {
	Range    string `yaml:"range"`
	Expected string `yaml:"expected"`
}

// Validate checks that target specification is valid
func (t *Target) Validate() error {
	if ip := net.ParseIP(t.IP); ip == nil {
		return fmt.Errorf("unable to parse IP address %s", t.IP)
	}
	return nil
}

// getStatus returns true if the target respond to ping requests
func (t *Target) getStatus() bool {
	pinger, err := ping.NewPinger(t.IP)
	pinger.Timeout = 2 * time.Second
	if err != nil {
		panic(err)
	}
	pinger.Count = 1
	pinger.Run()
	stats := pinger.Statistics()
	if stats.PacketLoss == 100.0 {
		return false
	}
	return true
}

// getAddress returns hostname:port format
func (t *Target) getAddress(port string) string {
	return t.IP + ":" + port
}

// ParsePorts read app scanning range et fill {tcp,udp}PortsToScan
// with required ports.
// FOR NOW it doesn't support other parameters than 'all' and 'reserved'
func (t *Target) ParsePorts() {
	// parse TCP ports
	cmd := t.TCP.Range
	switch cmd {
	case "all":
		for port := 1; port <= 65535; port++ {
			t.tcpPortsToScan = append(t.tcpPortsToScan, strconv.Itoa(port))
		}
		return
	case "reserved":
		for port := 1; port <= 1024; port++ {
			t.tcpPortsToScan = append(t.tcpPortsToScan, strconv.Itoa(port))
		}
		return
	}
	/*
		parse UDP ports
	*/
	cmd = t.UDP.Range
	switch cmd {
	case "all":
		for port := 1; port <= 65535; port++ {
			t.udpPortsToScan = append(t.udpPortsToScan, strconv.Itoa(port))
		}
		return
	case "reserved":
		for port := 1; port <= 1024; port++ {
			t.udpPortsToScan = append(t.udpPortsToScan, strconv.Itoa(port))
		}
		return
	}
}

// Scan starts a scan
func (t *Target) Scan() {
	var wg sync.WaitGroup
	for _, port := range t.tcpPortsToScan {
		wg.Add(1)
		go scanWorker(t.getAddress(port), &wg)
	}
	// comment lire le channel sans bloquer ?
	// regarder "close" pour terminer un channel
	wg.Wait()
}

func scanWorker(address string, wg *sync.WaitGroup) {
	defer wg.Done()
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		// port is closed
		return
	}
	conn.Close()
	fmt.Println(address) // debug
}
