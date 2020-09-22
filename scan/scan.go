package scan

import (
	"devops-works/scan-exporter/metrics"
	"fmt"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/tatsushid/go-fastping"
)

// Target holds an IP and a range of ports to scan
type Target struct {
	name   string
	period string
	ip     string
	tcp    protocol
	udp    protocol
	icmp   protocol

	logger zerolog.Logger

	// those maps hold the protocol and the ports
	portsToScan map[string][]string
	portsOpen   map[string][]string
}

type protocol struct {
	period   string
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
func New(name, ip string, o ...func(*Target) error) (*Target, error) {
	if i := net.ParseIP(ip); i == nil {
		return nil, fmt.Errorf("unable to parse IP address %s", ip)
	}

	t := &Target{
		name: name,
		ip:   ip,
	}

	for _, f := range o {
		if err := f(t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

// WithPorts adds TCP or UDP ports specifications to scan target
func WithPorts(proto, period, rng, expected string) func(*Target) error {
	return func(t *Target) error {
		return t.setPorts(proto, period, rng, expected)
	}
}

// Name returns the target's name
func (t *Target) Name() string {
	return t.name
}

func (t *Target) setPorts(proto, period, rng, exp string) error {
	// check if protocol is supported
	switch proto {
	case "tcp":
		t.tcp = protocol{
			period:   period,
			rng:      rng,
			expected: exp,
		}
	case "udp":
		t.udp = protocol{
			period:   period,
			rng:      rng,
			expected: exp,
		}
	case "icmp":
		t.icmp = protocol{
			period: period,
		}
	default:
		return fmt.Errorf("unsupported protocol %q for target %s", proto, t.name)
	}

	// check if the period is in a correct format (1d, 60s, 45h ...)
	if period != "" {
		re := regexp.MustCompile(`^\d+[dhms]$`)
		if !re.Match([]byte(period)) {
			return fmt.Errorf("unsupported period format %q for protocol %q", period, proto)
		}
	}

	// test range and expected. ICMP does not need a port range.
	if proto != "icmp" {
		re := regexp.MustCompile(`(\d+)([-,]\d+)*|^all$|^reserved$`)
		if !re.Match([]byte(rng)) {
			return fmt.Errorf("unsupported range format %q for protocol %q", rng, proto)
		}
		if !re.Match([]byte(exp)) {
			return fmt.Errorf("unsupported expected range format %q for protocol %q", rng, proto)
		}
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

// getDuration transforms a protocol's period into a time.Duration value
func (p *protocol) getDuration() (time.Duration, error) {
	// only hours, minutes and seconds are handled by ParseDuration
	if strings.ContainsAny(p.period, "hms") {
		t, err := time.ParseDuration(p.period)
		if err != nil {
			return 0, err
		}
		return t, nil
	}

	sep := strings.Split(p.period, "d")
	days, err := strconv.Atoi(sep[0])
	if err != nil {
		return 0, err
	}

	t := time.Duration(days) * time.Hour * 24
	return t, nil
}

// feeder parses Target's port into maps. Once it is done, it sends the map content into a channel.
// Next to this, it starts workers that will gather ports from the channel.
func (t *Target) feeder(mainWg *sync.WaitGroup) {
	defer mainWg.Done()
	trigger := make(chan string, 10)

	go t.scheduler(trigger, "tcp", "udp", "icmp")

	t.portsToScan = make(map[string][]string)

	// parse tcp ports
	if err := t.readPortsRange("tcp", t.tcp.rng); err != nil {
		log.Fatalf("an error occured while parsing tcp ports: %s", err)
	}

	// parse udp ports
	if err := t.readPortsRange("udp", t.udp.rng); err != nil {
		log.Fatalf("an error occured while parsing udp ports: %s", err)
	}

	var wg sync.WaitGroup

	for {
		select {
		case proto := <-trigger:
			switch proto {
			case "icmp":
				t.logger.Info().Msgf("%s scan started", proto)
				// ping the target if asked in conf file
				if t.icmp.period != "" {
					icmpWorker(t.ip)
				}
			case "tcp":
				t.logger.Info().Msgf("%s scan started", proto)
				// TCP scan
				tcpChannel := make(chan channelMsg, 100)
				for _, port := range t.portsToScan["tcp"] {
					// msg hold informations about port to scan
					var msg = channelMsg{protocol: "tcp", port: port}
					tcpChannel <- msg
					wg.Add(1)
					go tcpWorker(tcpChannel, t.ip, &wg)
				}
				wg.Wait()
			case "udp":
				t.logger.Info().Msgf("%s scan started", proto)
				// UDP scan
				udpChannel := make(chan channelMsg, 100)
				for _, port := range t.portsToScan["udp"] {
					// msg hold informations about port to scan
					var msg = channelMsg{protocol: "udp", port: port}
					udpChannel <- msg
					wg.Add(1)
					go udpWorker(udpChannel, t.ip, &wg)
				}
				wg.Wait()
			}
		}
	}
}

// reporter get values from reportChannel and send them to the metrics package.
func (t *Target) reporter(wg *sync.WaitGroup) {
	defer wg.Done()

	t.portsOpen = make(map[string][]string)
	currentTime := time.Now()

	for {
		select {
		case openPort := <-reportChannel:
			t.portsOpen[openPort.protocol] = append(t.portsOpen[openPort.protocol], openPort.port)

			metrics.Exploit(currentTime, t.name, t.ip, openPort.port, openPort.protocol)
		}
	}
}

// scheduler create tickers for each protocol given and when they tick, it sends the protocol
// name in the trigger's channel in order to alert feeder that a scan must be started.
func (t *Target) scheduler(trigger chan string, protocols ...string) {
	var tcpTicker, udpTicker, icmpTicker *time.Ticker
	for _, proto := range protocols {
		switch proto {
		case "tcp":
			tcpFreq, err := t.tcp.getDuration()
			if err != nil {
				t.logger.Error().Msgf("error getting %s frequency in scheduler: %s", proto, err)
			}
			tcpTicker = time.NewTicker(tcpFreq)
		case "udp":
			udpFreq, err := t.udp.getDuration()
			if err != nil {
				t.logger.Error().Msgf("error getting %s frequency in scheduler: %s", proto, err)
			}
			udpTicker = time.NewTicker(udpFreq)
		case "icmp":
			icmpFreq, err := t.icmp.getDuration()
			if err != nil {
				t.logger.Error().Msgf("error getting %s frequency in scheduler: %s", proto, err)
			}
			icmpTicker = time.NewTicker(icmpFreq)
		}
	}
	for {
		select {
		case <-tcpTicker.C:
			trigger <- "tcp"
		case <-udpTicker.C:
			trigger <- "udp"
		case <-icmpTicker.C:
			trigger <- "icmp"
		}
	}
}

// tcpWorker does a TCP scan on a given ip and send the result to reportChannel
func tcpWorker(ch chan channelMsg, ip string, wg *sync.WaitGroup) {
	defer wg.Done()

	todo := <-ch

	conn, err := net.DialTimeout(todo.protocol, ip+":"+todo.port, 2*time.Second)
	if err != nil {
		// port is closed
		return
	}
	defer conn.Close()

	var toSend = channelMsg{protocol: todo.protocol, port: todo.port}
	reportChannel <- toSend
}

// udpWorker does an UDP scan on a given ip and send the result to reportChannel
func udpWorker(ch chan channelMsg, ip string, wg *sync.WaitGroup) {
	defer wg.Done()

	todo := <-ch

	serverAddr, err := net.ResolveUDPAddr(todo.protocol, ip+":"+todo.port)
	if err != nil {
		return
	}
	conn, err := net.DialUDP(todo.protocol, nil, serverAddr)
	if err != nil {
		return
	}
	defer conn.Close()

	// write 3 times to the udp socket and check
	// if there's any kind of error
	errorCount := 0
	for i := 0; i < 3; i++ {
		buf := []byte("0")
		_, err := conn.Write(buf)
		if err != nil {
			errorCount++
		}
	}
	if errorCount > 0 {
		// port is closed
		return
	}

	var toSend = channelMsg{protocol: todo.protocol, port: todo.port}
	reportChannel <- toSend
}

// icmpWorker does an ICMP request on a given ip and send the result to reportChannel
func icmpWorker(ip string) {
	var ra *net.IPAddr
	var err error

	p := fastping.NewPinger()

	// check if the ip is v4 or v6. We do not need to check IP validity as it is already
	// done in New().
	if strings.Contains(ip, ".") {
		ra, err = net.ResolveIPAddr("ip4:icmp", ip)
		if err != nil {
			return
		}
	} else if strings.Contains(ip, ":") {
		ra, err = net.ResolveIPAddr("ip6:icmp", ip)
		if err != nil {
			return
		}
	}

	p.AddIPAddr(ra)

	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		// icmpWorker does not send port. See metrics.WriteLog()
		var toSend = channelMsg{protocol: "icmp"}
		reportChannel <- toSend
	}

	p.OnIdle = func() {
		return
	}

	if err := p.Run(); err != nil {
		// it will end up here if the program is not launched as superuser
		return
	}

}
