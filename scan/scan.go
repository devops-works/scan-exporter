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

	"github.com/rs/zerolog"
	"github.com/sparrc/go-ping"
)

// Target holds an IP and a range of ports to scan
type Target struct {
	name   string
	period string
	ip     string
	tcp    protocol
	udp    protocol

	logger zerolog.Logger

	// those maps hold the protocol and the ports
	portsToScan map[string][]string
	portsOpen   map[string][]string
}

type protocol struct {
	rng      string
	expected string
}

type channelMsg struct {
	protocol string
	port     string
}

// reportChannel holds open reports returned by workers
var reportChannel = make(chan channelMsg, 1000)

// Scan starts a scan
func (t *Target) Scan() {
	var wg sync.WaitGroup
	wg.Add(2)
	go t.feeder(&wg)
	go t.reporter(&wg)
	wg.Wait()
}

// New checks that target specification is valid, and if target is responding
func New(name, period, ip string, o ...func(*Target) error) (*Target, error) {
	if i := net.ParseIP(ip); i == nil {
		return nil, fmt.Errorf("unable to parse IP address %s", ip)
	}

	t := &Target{
		name:   name,
		period: period,
		ip:     ip,
	}

	for _, f := range o {
		if err := f(t); err != nil {
			return nil, err
		}
	}

	// if !t.getStatus() {
	// 	return fmt.Errorf("%s seems to be down", t.ip)
	// }

	return t, nil
}

// WithPorts adds TCP or UDP ports specifications to scan target
func WithPorts(proto, rng, expected string) func(*Target) error {
	return func(t *Target) error {
		return t.setPorts(proto, rng, expected)
	}
}

// Name returns the target's name
func (t *Target) Name() string {
	return t.name
}

func (t *Target) setPorts(proto, rng, exp string) error {
	// TODO: check ranges to see if they are valid
	switch proto {
	case "tcp":
		t.tcp = protocol{
			rng:      rng,
			expected: exp,
		}
	case "udp":
		t.udp = protocol{
			rng:      rng,
			expected: exp,
		}
	default:
		return fmt.Errorf("unsuppoted protocol %q for target %s", proto, t.name)
	}

	return nil
}

// WithLogger adds logger specifications to scan target
func WithLogger(l zerolog.Logger) func(*Target) error {
	return func(t *Target) error {
		return t.setLogger(l)
	}
}

// setLogger sets the logger on a target
func (t *Target) setLogger(l zerolog.Logger) error {
	t.logger = l
	return nil
}

// getStatus returns true if the target respond to ping requests
func (t *Target) getStatus() bool {
	pinger, err := ping.NewPinger(t.ip)
	if err != nil {
		log.Fatalf("error occured while creating the pinger %s: %s", t.ip, err)
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
	return t.ip + ":" + port
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
	if err := t.readPortsRange("tcp", t.tcp.rng); err != nil {
		log.Fatalf("an error occured while parsing tcp ports: %s", err)
	}
	// parse udp ports
	if err := t.readPortsRange("udp", t.udp.rng); err != nil {
		log.Fatalf("an error occured while parsing udp ports: %s", err)
	}

	var wg sync.WaitGroup
	workerChannel := make(chan channelMsg, 100)
	for _, port := range t.portsToScan["tcp"] {
		// msg hold informations about port to scan
		var msg = channelMsg{protocol: "tcp", port: port}
		workerChannel <- msg
		wg.Add(1)
		go scanWorker(workerChannel, t.ip, &wg)
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
			// fmt.Println(t.ip + ":" + openPort.port + "/" + openPort.protocol) // debug

			metrics.WriteLog(logName+"_"+t.Name(), t.ip, openPort.port, openPort.protocol)
			// do something like metrics.Expose()
			// check with team what and how
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
