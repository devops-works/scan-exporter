package scan

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
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
	portsToScan    map[string][]interface{}
	tcpPortsToScan []string
	udpPortsToScan []string
	// those arrays will hold open ports
	portsOpen    map[string][]interface{}
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
		log.Fatalf("error occured when pinging the target %s: %s", t.IP, err)
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

// // ParsePorts read app scanning range et fill {tcp,udp}PortsToScan
// // with required ports.
// func (t *Target) ParsePorts() {
// 	// parse TCP ports
// 	cmd := t.TCP.Range
// 	switch cmd {
// 	case "all":
// 		for port := 1; port <= 65535; port++ {
// 			t.tcpPortsToScan = append(t.tcpPortsToScan, strconv.Itoa(port))
// 		}
// 		return
// 	case "reserved":
// 		for port := 1; port <= 1024; port++ {
// 			t.tcpPortsToScan = append(t.tcpPortsToScan, strconv.Itoa(port))
// 		}
// 		return
// 	default:
// 		ports, err := readNumericRange(t.TCP.Range)
// 		if err != nil {
// 			log.Fatalf("error reading udp ports to scan: %s", err)
// 		}
// 		t.tcpPortsToScan = ports
// 	}
// 	/*
// 		parse UDP ports
// 	*/
// 	cmd = t.UDP.Range
// 	switch cmd {
// 	case "all":
// 		for port := 1; port <= 65535; port++ {
// 			t.udpPortsToScan = append(t.udpPortsToScan, strconv.Itoa(port))
// 		}
// 		return
// 	case "reserved":
// 		for port := 1; port <= 1024; port++ {
// 			t.udpPortsToScan = append(t.udpPortsToScan, strconv.Itoa(port))
// 		}
// 		return
// 	default:
// 		ports, err := readNumericRange(t.UDP.Range)
// 		if err != nil {
// 			log.Fatalf("error reading udp ports to scan: %s", err)
// 		}
// 		t.udpPortsToScan = ports
// 	}
// }

// Scan starts a scan
func (t *Target) Scan() {
	t.feeder()
}

func scanWorker(protocol, address string, wg *sync.WaitGroup) {
	defer wg.Done()
	// grâce aux map qui sont envoyées dans les chan, chaque worker recoit le protocol et le port
	conn, err := net.DialTimeout(protocol, address, 2*time.Second)
	if err != nil {
		// port is closed
		return
	}
	conn.Close()
	fmt.Println(address) // debug
}

// readPortsRange transforms a range of ports given in conf to an array of
// effective ports
func (t *Target) readPortsRange(protocol, portsRange string) error {
	switch portsRange {
	case "all":
		for port := 1; port <= 65535; port++ {
			t.portsToScan[protocol] = append(t.portsToScan[protocol], strconv.Itoa(port))
		}
	case "reserved":
		for port := 1; port <= 1024; port++ {
			t.portsToScan[protocol] = append(t.portsToScan[protocol], strconv.Itoa(port))
		}
	default:
		comaSplit := strings.Split(portsRange, ",")
		for _, char := range comaSplit {
			if strings.Contains(char, "-") {
				decomposedRange := strings.Split(char, "-")
				min, err := strconv.Atoi(decomposedRange[0])
				if err != nil {
					return err
				}
				max, err := strconv.Atoi(decomposedRange[len(decomposedRange)-1])
				if err != nil {
					return err
				}

				for j := min; j <= max; j++ {
					t.portsToScan[protocol] = append(t.portsToScan[protocol], strconv.Itoa(j))
				}
			} else {
				t.portsToScan[protocol] = append(t.portsToScan[protocol], char)
			}
		}
	}
	return nil
}

/*
	feeder receive a target
	parse it ports into a map
	send the map content into a worker channel
	it also starts workers
*/
func (t *Target) feeder() {
	t.portsToScan = make(map[string][]interface{})
	/*
		make it concurrent ?
	*/
	// parse tcp ports
	if err := t.readPortsRange("tcp", t.TCP.Range); err != nil {
		log.Fatalf("an error occured while parsing tcp ports: %s")
	}
	// parse udp ports
	if err := t.readPortsRange("udp", t.UDP.Range); err != nil {
		log.Fatalf("an error occured while parsing udp ports: %s")
	}

	var wg sync.WaitGroup
	for _, port := range t.portsToScan["tcp"] {
		wg.Add(1)
		go scanWorker("tcp", t.getAddress(fmt.Sprintf("%v", port)), &wg)
	}
	// comment lire le channel sans bloquer ?
	// regarder "close" pour terminer un channel
	wg.Wait()
}
