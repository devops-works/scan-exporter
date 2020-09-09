package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
	"github.com/sparrc/go-ping"
)

type conf struct {
	Targets []target `yaml:"targets"`
}

type target struct {
	Name   string   `yaml:"name"`
	Period string   `yaml:"period"`
	IP     string   `yaml:"ip"`
	TCP    protocol `yaml:"tcp"`
	UDP    protocol `yaml:"udp"`
	// {tcp,udp}PortsToScan holds all the ports that will be scanned
	// those fields are fielded after having parsed the range given in
	// config file.
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

func main() {
	c := conf{}
	confPath := getConfPath(os.Args)
	c.getConf(confPath)
	log.Infof("%d targets found in %s", len(c.Targets), confPath)

	// targetList is an array that will contain each instance of up target found in conf file
	targetList := []target{}
	for _, target := range c.Targets {
		t := target
		if t.getStatus() {
			// if the target is up, we add it to targetList
			targetList = append(targetList, t)
		} else {
			// else, we log that the target is down
			// maybe we can send a mail or a notification to manually inspect this case ?
			log.Warnf("%s (%s) seems to be down", t.Name, t.IP)
		}
	}

	/*
		from now, we have a valid list of targets to scan in targetList.
		next step is to parse ports ranges for each protocol, and fill
		{tcp,udp}PortsToScan in each target instance in targetList
	*/

	for i := 0; i < len(targetList); i++ {
		t := targetList[i]
		t.parsePorts()
		log.Infof("Starting %s scan", t.Name)
		t.scanTarget()
	}
}

// getStatus returns true if the target respond to ping requests
func (t *target) getStatus() bool {
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
func (t *target) getAddress(port string) string {
	return t.IP + ":" + port
}

// parsePorts read app scanning range et fill {tcp,udp}PortsToScan
// with required ports.
// FOR NOW it doesn't support other parameters than 'all' and 'reserved'
func (t *target) parsePorts() {
	/*
		parse TCP ports
	*/
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

// getConf reads confFile and unmarshall it
func (c *conf) getConf(confFile string) {
	yamlConf, err := ioutil.ReadFile(confFile)
	if err != nil {
		log.Errorf("Error while reading %s: %v ", confFile, err)
	}

	if err = yaml.Unmarshal(yamlConf, &c); err != nil {
		log.Errorf("Error while unmarshalling yamlConf: %v", err)
	}
}

func getConfPath(args []string) string {
	if len(args) > 1 {
		return args[1]
	}
	// default config file
	return "config.yaml"
}

func (t *target) scanTarget() {
	var wg sync.WaitGroup
	for _, port := range t.tcpPortsToScan {
		wg.Add(1)
		go scanWorker(t.getAddress(port), &wg)
	}
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
