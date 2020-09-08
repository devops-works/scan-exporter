package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tatsushid/go-fastping"
)

type app struct {
	hostname  string
	scanRange string
	// ports contains all the ports that will be scanned
	ports []string
}

func main() {
	/*
		lire config ici, et créer les app en fonction de ce qu'il y a dans
		le fichier, puis voir si elles sont up
	*/
	viper.SetConfigName("config")
	viper.AddConfigPath(".")    // optionally look for config in the working directory
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	app1 := app{hostname: "localhost", scanRange: "reserved"} // remplacement du fichier de conf
	if !app1.getStatus() {
		log.Warnf("%s seems to be down.", app1.hostname)
	}
	/*
		ici il faudra lire la conf pour comprendre quels sont les ports visés
		par le scan.
		Créer des go routines !
	*/
	app1.ports = app1.parsePortsRange()
	fmt.Println(len(app1.ports))
}

// getStatus returns true if the application respond to ping requests
func (a app) getStatus() bool {
	p := fastping.NewPinger()
	ra, err := net.ResolveIPAddr("ip4:icmp", a.hostname)
	if err != nil {
		return false
	}
	p.AddIPAddr(ra)
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		log.Infof("IP Addr: %s, RTT: %v\n", addr.String(), rtt)
	}
	err = p.Run()
	if err != nil {
		return false
	}

	return true
}

// getAddress returns hostname:port format
func (a app) getAddress(port string) string {
	return a.hostname + ":" + port
}

// scanPort dials a given address with a specified protocol
func scanPort(a app, protocol, port string) bool {
	conn, err := net.DialTimeout(protocol, a.getAddress(port), 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// parsePortsRange returns an array containing all the ports that
// will be scanned
func (a app) parsePortsRange() []string {
	var ports = []string{}

	switch a.scanRange {
	// append all ports to the scan list
	case "all":
		for port := 1; port <= 65535; port++ {
			ports = append(ports, strconv.Itoa(port))
		}
		return ports
	// append reserved ports to the scan list
	case "reserved":
		for port := 1; port <= 1024; port++ {
			ports = append(ports, strconv.Itoa(port))
		}
		return ports
	}

	if strings.Contains(a.scanRange, "-") {
		// get the list's bounds
		content := strings.Split(a.scanRange, "-")
		first, err := strconv.Atoi(content[0])
		last, err := strconv.Atoi(content[len(content)-1])
		if err != nil {
			log.Errorf("An error occured while getting ports to scan: %s", err)
		}

		for port := first; port <= last; port++ {
			ports = append(ports, strconv.Itoa(port))
		}
	}
	return ports
}
