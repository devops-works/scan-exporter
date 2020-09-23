package scan

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/sparrc/go-ping"
)

// Target holds an IP and a range of ports to scan
type Target struct {
	name    string
	ip      string
	workers int
	protos  map[string]protocol

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

type jobMsg struct {
	ip       string
	protocol string
	ports    []string
}

// New checks that target specification is valid, and if target is responding
func New(name, ip string, workers int, o ...func(*Target) error) (*Target, error) {
	if i := net.ParseIP(ip); i == nil {
		return nil, fmt.Errorf("unable to parse IP address %s", ip)
	}

	t := &Target{
		name:        name,
		ip:          ip,
		workers:     workers,
		protos:      make(map[string]protocol),
		portsToScan: make(map[string][]string),
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

func (t *Target) setPorts(proto, period, rng, exp string) error {
	if !stringInSlice(proto, []string{"udp", "tcp", "icmp"}) {
		return fmt.Errorf("unsupported protocol %q for target %s", proto, t.name)
	}

	t.protos[proto] = protocol{
		period:   period,
		rng:      rng,
		expected: exp,
	}

	var err error
	t.portsToScan[proto], err = readPortsRange(rng)
	if err != nil {
		return err
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

// Name returns target name
func (t *Target) Name() string {
	return t.name
}

// Run should be called using `go` and will run forever running the scanning
// schedule
func (t *Target) Run() {
	// Create trigger channel for scheduler
	trigger := make(chan string, 100)
	workersCount := t.workers

	protoList := t.getWantedProto()

	// Start scheduler
	go t.scheduler(trigger, protoList)

	// Create channel to send jobMsg
	jobsChan := make(chan jobMsg, workersCount)

	// Start required number (n) of workers
	for w := 0; w < workersCount; w++ {
		go worker(jobsChan)
	}
	t.logger.Info().Msgf("%d workers started", workersCount)

	// Infinite loop that follow trigger
	for {
		select {
		case proto := <-trigger:
			// Create n jobs containing 1/n of total scan range
			jobs, err := t.createJobs(proto)
			if err != nil {
				t.logger.Error().Msgf("error creating jobs")
				return // TODO:  Handle error somehow
			}

			// Send jobs to channel
			for _, j := range jobs {
				jobsChan <- j
				t.logger.Debug().Msgf("appended job %s %s in channel", j.ip, j.protocol)
			}
		}
	}
}

// getWantedProto check if a protocol is set in config file and returns a slice of wnated protocols.
func (t *Target) getWantedProto() []string {
	var protoList = []string{}
	if p := t.protos["tcp"].period; p != "" {
		protoList = append(protoList, "tcp")
	}

	if p := t.protos["udp"].period; p != "" {
		protoList = append(protoList, "udp")
	}

	if p := t.protos["icmp"].period; p != "" {
		protoList = append(protoList, "icmp")
	}

	return protoList
}

func worker(jobsChan chan jobMsg) {
	for {
		select {
		case job := <-jobsChan:
			switch job.protocol {
			case "tcp":
				// Launch TCP scan
				for _, p := range job.ports {
					if tcpScan(job.ip, p) {
						fmt.Printf("%s:%s/tcp OPEN\n", job.ip, p)
					}
				}
			case "udp":
				// Launch UDP scan
				for _, p := range job.ports {
					if udpScan(job.ip, p) {
						fmt.Printf("%s:%s/udp OPEN\n", job.ip, p)
					}
				}
			case "icmp":
				if icmpScan(job.ip) {
					fmt.Printf("%s/icmp OPEN\n", job.ip)
				}
			}
		}
	}
}

func (t *Target) createJobs(proto string) ([]jobMsg, error) {
	jobs := []jobMsg{}
	if proto == "icmp" {
		return []jobMsg{
			jobMsg{ip: t.ip, protocol: proto},
		}, nil
	}
	if _, ok := t.portsToScan[proto]; !ok {
		return nil, fmt.Errorf("no such protocol %q in current protocol list", proto)
	}
	step := (len(t.portsToScan[proto]) + t.workers - 1) / t.workers

	for i := 0; i < len(t.portsToScan[proto]); i += step {
		right := i + step
		// Check right boundary for slice
		if right > len(t.portsToScan[proto]) {
			right = len(t.portsToScan[proto])
		}

		jobs = append(jobs, jobMsg{
			ip:       t.ip,
			protocol: proto,
			ports:    t.portsToScan[proto][i:right],
		})
		t.logger.Debug().Msgf("a job for %s has been appended", proto)
	}
	return jobs, nil
}

// readPortsRange transforms a range of ports given in conf to an array of
// effective ports
func readPortsRange(ranges string) ([]string, error) {
	ports := []string{}

	parts := strings.Split(ranges, ",")

	for _, spec := range parts {
		if spec == "" {
			continue
		}
		switch spec {
		case "all":
			for port := 1; port <= 65535; port++ {
				ports = append(ports, strconv.Itoa(port))
			}
		case "reserved":
			for port := 1; port < 1024; port++ {
				ports = append(ports, strconv.Itoa(port))
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
				ports = append(ports, strconv.Itoa(i))
			}
		}
	}

	return ports, nil
}

func stringInSlice(s string, sl []string) bool {
	for _, v := range sl {
		if v == s {
			return true
		}
	}
	return false
}

// scheduler create tickers for each protocol given and when they tick, it sends the protocol
// name in the trigger's channel in order to alert feeder that a scan must be started.
func (t *Target) scheduler(trigger chan string, protocols []string) {
	var tcpTicker, udpTicker, icmpTicker *time.Ticker
	for _, proto := range protocols {
		switch proto {
		case "tcp":
			tcpFreq, err := getDuration(t.protos[proto].period)
			if err != nil {
				t.logger.Error().Msgf("error getting %s frequency in scheduler: %s", proto, err)
			}
			tcpTicker = time.NewTicker(tcpFreq)
			// starts its own ticker
			go ticker(trigger, proto, tcpTicker)
		case "udp":
			udpFreq, err := getDuration(t.protos[proto].period)
			if err != nil {
				t.logger.Error().Msgf("error getting %s frequency in scheduler: %s", proto, err)
			}
			udpTicker = time.NewTicker(udpFreq)
			// starts its own ticker
			go ticker(trigger, proto, udpTicker)
		case "icmp":
			icmpFreq, err := getDuration(t.protos[proto].period)
			if err != nil {
				t.logger.Error().Msgf("error getting %s frequency in scheduler: %s", proto, err)
			}
			icmpTicker = time.NewTicker(icmpFreq)
			// starts its own ticker
			go ticker(trigger, proto, icmpTicker)
		}
	}
}

// ticker handles a protocol ticker, and send the protocol in a channel when the ticker ticks
func ticker(trigger chan string, proto string, protTicker *time.Ticker) {
	// First scan at the start
	trigger <- proto

	for {
		select {
		case <-protTicker.C:
			trigger <- proto
		}
	}
}

// tcpScan scans an ip and returns true if the port responds.
func tcpScan(ip, port string) bool {
	conn, err := net.DialTimeout("tcp", ip+":"+port, 5*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}

// udpScan scans an ip and returns true if the port responds.
func udpScan(ip, port string) bool {
	serverAddr, err := net.ResolveUDPAddr("udp", ip+":"+port)
	if err != nil {
		return false
	}
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return false
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
		return false
	}

	return true
}

// icmpScan pings a host
func icmpScan(ip string) bool {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		panic(err) // TODO: need a better error handling
	}
	pinger.Count = 3
	pinger.Run()
	stats := pinger.Statistics()
	if stats.PacketLoss == 100.0 {
		return false
	}

	return true
}

// getDuration transforms a protocol's period into a time.Duration value
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
