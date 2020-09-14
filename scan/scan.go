package scan

import (
	"devops-works/scan-exporter/metrics"
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
	portsToScan map[string][]string
	portsOpen   map[string][]string
}

type protocol struct {
	Range    string `yaml:"range"`
	Expected string `yaml:"expected"`
}

type channelMsg struct {
	protocol string
	port     string
}

// reportChannel holds open reports returned by workers
var reportChannel = make(chan channelMsg, 1000)

// Scan starts a scan
func (t *Target) Scan() {
	var mainWg sync.WaitGroup
	mainWg.Add(2)
	go t.feeder(&mainWg)
	go t.reporter(&mainWg)
	mainWg.Wait()
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
	feeder() parses Target's port into maps. Once it is done, it sends the map content into a channel.
	Next to this, it starts workers that will gather ports from the channel.
*/
func (t *Target) feeder(mainWg *sync.WaitGroup) {
	defer mainWg.Done()
	t.portsToScan = make(map[string][]string)
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
	workerChannel := make(chan channelMsg, 100)
	for _, port := range t.portsToScan["tcp"] {
		// msg hold informations about port to scan
		var msg = channelMsg{protocol: "tcp", port: port}
		workerChannel <- msg
		wg.Add(1)
		go scanWorker(workerChannel, t.IP, &wg)
	}
	wg.Wait()
}

func (t *Target) reporter(mainWg *sync.WaitGroup) {
	defer mainWg.Done()
	currentTime := time.Now()
	logName := currentTime.Format("2006-01-02_15:04:05")
	// logName := time.Now().String()
	t.portsOpen = make(map[string][]string)
	for {
		select {
		case openPort := <-reportChannel:
			t.portsOpen[openPort.protocol] = append(t.portsOpen[openPort.protocol], openPort.port)
			fmt.Println(t.IP + ":" + openPort.port + "/" + openPort.protocol) // debug

			metrics.WriteLog(logName+"_"+t.Name, t.IP, openPort.port, openPort.protocol)

		case <-time.After(5 * time.Second):
			// when no new port fo 5sec, exit reporter
			return
		}
	}
}

func scanWorker(ch chan channelMsg, ip string, wg *sync.WaitGroup) {
	defer wg.Done()
	todo := <-ch
	// grâce aux map qui sont envoyées dans les chan, chaque worker recoit le protocol et le port
	conn, err := net.DialTimeout(todo.protocol, ip+":"+todo.port, 2*time.Second)
	if err != nil {
		// port is closed
		return
	}
	conn.Close()
	var toSend = channelMsg{protocol: todo.protocol, port: todo.port}
	reportChannel <- toSend
}
