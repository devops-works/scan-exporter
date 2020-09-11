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

	// those maps hold the protocol and the ports
	portsToScan map[string][]interface{}
	portsOpen   map[string][]interface{}
}

type protocol struct {
	Range    string `yaml:"range"`
	Expected string `yaml:"expected"`
}

// Validate checks that target specification is valid, and if target is responding
func (t *Target) Validate() error {
	if ip := net.ParseIP(t.IP); ip == nil {
		return fmt.Errorf("unable to parse IP address %s", t.IP)
	}

	if !t.getStatus() {
		return fmt.Errorf("%s seems to be down", t.IP)
	}

	return nil
}

// getStatus returns true if the target respond to ping requests
func (t *Target) getStatus() bool {
	pinger, err := ping.NewPinger(t.IP)
	if err != nil {
		log.Fatalf("error occured while creating the pinger %s: %s", t.IP, err)
	}
	pinger.Timeout = 2 * time.Second
	pinger.Count = 3
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
		log.Fatalf("an error occured while parsing tcp ports: %s", err)
	}
	// parse udp ports
	if err := t.readPortsRange("udp", t.UDP.Range); err != nil {
		log.Fatalf("an error occured while parsing udp ports: %s", err)
	}

	var wg sync.WaitGroup
	for _, port := range t.portsToScan["tcp"] {
		wg.Add(2)
		go scanWorker("tcp", t.getAddress(fmt.Sprintf("%v", port)), &wg)
		go scanWorker("udp", t.getAddress(fmt.Sprintf("%v", port)), &wg)
	}
	// comment lire le channel sans bloquer ?
	// regarder "close" pour terminer un channel
	wg.Wait()
}
