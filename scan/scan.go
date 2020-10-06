package scan

import (
	"fmt"
	"math/rand"
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
}

type protocol struct {
	period   string
	rng      string
	expected string
}

type jobMsg struct {
	id             string
	jobCount       int
	ip             string
	protocol       string
	ports          []string
	openPortsCount int
}

type resMsg struct {
	id             string
	ip             string
	protocol       string
	openPorts      []string
	openPortsCount int
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
	jobsChan := make(chan jobMsg, 3*workersCount)

	// Create channel to send scan results
	resChan := make(chan jobMsg, 3*workersCount)

	// postScan allow receiver to send scan results into the redis gouroutine
	postScan := make(chan resMsg, 3*workersCount)

	// Redis goroutine
	// go sendToRedis(postScan)

	// Create receiver that will receive done jobs.
	go t.receiver(resChan, postScan)

	// Start required number (n) of workers
	for w := 0; w < workersCount; w++ {
		go worker(jobsChan, resChan, t.logger)
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

			jobID := generateRandomString(10)

			// Send jobs to channel
			for _, j := range jobs {
				j.id = jobID
				j.jobCount = len(jobs)
				jobsChan <- j
				t.logger.Debug().Msgf("appended job %s %s in channel", j.ip, j.protocol)
			}
		}
	}
}

// receiver is created once
// It waits for incoming results (sent by workers when a port is open).
func (t *Target) receiver(resChan chan jobMsg, postScan chan resMsg) {
	// openPorts holds all openPorts for a jobID
	var openPorts = make(map[string][]string)
	var jobsStarted = make(map[string]int)

	for {
		select {
		case res := <-resChan:
			// Debug purposes... Unless ?
			if res.protocol == "icmp" {
				// l.Info().Msgf("%s scan started", res.protocol) // debug
				// fmt.Printf("[%s] STARTED at %s\n", res.id, time.Now().String()) // debug
			}

			jobsStarted[res.id]++

			// Append ports
			openPorts[res.id] = append(openPorts[res.id], res.ports...)

			if jobsStarted[res.id] == res.jobCount {
				// fmt.Printf("[%s] FINISHED at %s\n", res.id, time.Now().String()) // debug
				// l.Info().Msgf("[%s] - %s scan ended", res.id, res.protocol)
				// All jobs finished
				// fmt.Printf("[%s] open %s ports : %s\n", res.id, res.protocol, openPorts[res.id]) // debug

				// results holds all the informations about a finished scan
				results := resMsg{
					id:        res.id,
					ip:        res.ip,
					protocol:  res.protocol,
					openPorts: openPorts[res.id],
				}

				// fmt.Printf("%s : %d open ports\n", res.protocol, openPortsCounter) // debug

				// Check diff between expected and open
				if results.protocol != "icmp" {
					_, err := t.checkAccordance(results.protocol, results.openPorts)
					if err != nil {
						t.logger.Error().Msgf("error occured while checking port accordance: %s", err)
					}
				}

				postScan <- results
				// send results to redis channel
			}
		}
	}
}

// checkAccordance verifies if the open ports list matches the expected ports list given in config.
// It returns a list of unexpected ports. The list is empty if everything is ok.
func (t *Target) checkAccordance(proto string, open []string) ([]string, error) {
	var unexpectedPorts = []string{}

	expected, err := readPortsRange(t.protos[proto].expected)
	if err != nil {
		return unexpectedPorts, err
	}

	for _, port := range open {
		// If the open port is not in the expected
		if !stringInSlice(port, expected) {
			// Log it
			t.logger.Info().Msgf("%s/%s unexpected", port, proto)

			// Append it to unexpectedPorts
			unexpectedPorts = append(unexpectedPorts, port)
		}
	}

	return unexpectedPorts, nil
}

// generateRandomString generates a random string with a lenght of n.
// It is used to create a random jobID.
func generateRandomString(n int) string {
	rand.Seed(time.Now().UnixNano())

	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
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

// worker is a neverending goroutine which waits for incoming jobs.
// Depending of the job's protocol, it launches different kinds of scans.
// If a scan is successful, it sends a resMsg to the receiver.
func worker(jobsChan chan jobMsg, resChan chan jobMsg, l zerolog.Logger) {
	for {
		select {
		case job := <-jobsChan:
			// res holds the result of the scan and some more infos
			res := jobMsg{
				id:       job.id,
				ip:       job.ip,
				jobCount: job.jobCount,
				protocol: job.protocol,
			}
			switch res.protocol {
			case "tcp":
				// Launch TCP scan
				for _, p := range job.ports {
					// Fill res.ports with open ports
					if tcpScan(job.ip, p) {
						res.ports = append(res.ports, p)
						l.Info().Msgf("port %s/%s open", p, res.protocol)
					}

				}
				resChan <- res
			case "udp":
				// Launch UDP scan
				for _, p := range job.ports {
					if udpScan(job.ip, p) {
						res.ports = append(res.ports, p)
						l.Info().Msgf("port %s/%s open", p, res.protocol)
					}
				}
				resChan <- res
			case "icmp":
				if icmpScan(job.ip) {
					res.ports = append(res.ports, "1")
					l.Info().Msgf("%s responds", res.protocol)
				}
				resChan <- res
			}
		}
	}
}

// createJobs split portsToScan from a specified protocol into an even number of jobs that will be returned.
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

// stringInSlice checks if a string appears in a slice.
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
	conn, err := net.DialTimeout("tcp", ip+":"+port, 2*time.Second)
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
	// port is closed
	return errorCount <= 0
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

	return stats.PacketLoss != 100.0
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
