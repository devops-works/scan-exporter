package main

import (
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
	"github.com/tatsushid/go-fastping"
)

type conf struct {
	Targets []targets `yaml:"targets"`
}

type targets struct {
	Name string   `yaml:"name"`
	IP   string   `yaml:"ip"`
	TCP  protocol `yaml:"tcp"`
	UDP  protocol `yaml:"udp"`
}

type protocol struct {
	Name     string
	Range    string `yaml:"range"`
	Expected string `yaml:"expected"`
}

type app struct {
	infos targets
	// {tcp,udp}PortsToScan holds all the ports that will be scanned
	// those fields are fielded after having parsed the range given in
	// config file.
	tcpPortsToScan []string
	udpPortsToScan []string
}

func main() {
	c := conf{}
	confPath := getConfPath(os.Args)
	c.getConf(confPath)

	log.Infof("%d targets found in %s", len(c.Targets), confPath)

	// appList is an array that will contain each instance of target foudn in conf file
	// it will be easier to work with apps.
	appList := []app{}
	for i := 0; i < len(c.Targets); i++ {
		a := app{
			infos: c.Targets[i],
		}
		if a.getStatus() {
			// if the target is up, we add it to appList
			appList = append(appList, a)
		} else {
			// else, we log that the target is down
			// maybe we can send a mail or a notification to manually inspect this case ?
			log.Warnf("%s (%s) seems to be down", a.infos.Name, a.infos.IP)
		}
	}

	/*
		from now, we have a valid list of apps to scan in appList.
		next step is to parse ports ranges for each protocol, and fill
		{tcp,udp}PortsToScan in each app instance in appList
	*/

	for i := 0; i < len(appList); i++ {
		a := appList[i]
		a.parsePorts()
	}
}

// getStatus returns true if the application respond to ping requests
func (a *app) getStatus() bool {
	p := fastping.NewPinger()
	ra, err := net.ResolveIPAddr("ip4:icmp", a.infos.IP)
	if err != nil {
		return false
	}
	p.AddIPAddr(ra)
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		log.Infof("%s RTT: %v\n", addr.String(), rtt)
	}
	if err = p.Run(); err != nil {
		// we can end up here if we do not run the program as sudo...
		return false
	}

	return true
}

// getAddress returns hostname:port format
func (a *app) getAddress(port string) string {
	return a.infos.Name + ":" + port
}

// parsePorts read app scanning range et fill {tcp,udp}PortsToScan
// with required ports.
// FOR NOW it doesn't support other parameters than 'all' and 'reserved'
func (a *app) parsePorts() {
	/*
		parse TCP ports
	*/
	cmd := a.infos.TCP.Range
	switch cmd {
	case "all":
		for port := 1; port <= 65535; port++ {
			a.tcpPortsToScan = append(a.tcpPortsToScan, strconv.Itoa(port))
		}
		return
	case "reserved":
		for port := 1; port <= 1024; port++ {
			a.tcpPortsToScan = append(a.tcpPortsToScan, strconv.Itoa(port))
		}
		return
	}
	/*
		parse UDP ports
	*/
	cmd = a.infos.UDP.Range
	switch cmd {
	case "all":
		for port := 1; port <= 65535; port++ {
			a.udpPortsToScan = append(a.udpPortsToScan, strconv.Itoa(port))
		}
		return
	case "reserved":
		for port := 1; port <= 1024; port++ {
			a.udpPortsToScan = append(a.udpPortsToScan, strconv.Itoa(port))
		}
		return
	}
}

// parsePortsRange returns an array containing all the ports that
// will be scanned
// func (a *app) parsePortsRange(protType string, prot protocol) []string {
// 	var ports = []string{}
// 	switch prot.Range {
// 	// append all ports to the scan list
// 	case "all":
// 		for port := 1; port <= 65535; port++ {

// 		}
// 		return ports
// 	// append reserved ports to the scan list
// 	case "reserved":
// 		for port := 1; port <= 1024; port++ {
// 			ports = append(ports, strconv.Itoa(port))
// 		}
// 		return ports
// 	}

// 	if strings.Contains(a.scanRange, "-") {
// 		// get the list's bounds
// 		content := strings.Split(a.scanRange, "-")
// 		first, err := strconv.Atoi(content[0])
// 		last, err := strconv.Atoi(content[len(content)-1])
// 		if err != nil {
// 			log.Errorf("An error occured while getting ports to scan: %s", err)
// 		}

// 		for port := first; port <= last; port++ {
// 			ports = append(ports, strconv.Itoa(port))
// 		}
// 	}
// 	return ports
// }

func (c *conf) getConf(confFile string) *conf {
	yamlConf, err := ioutil.ReadFile(confFile)
	if err != nil {
		log.Errorf("Error while reading %s: %v ", confFile, err)
	}

	if err = yaml.Unmarshal(yamlConf, &c); err != nil {
		log.Errorf("Error while unmarshalling yamlConf: %v", err)
	}

	return c
}

func getConfPath(args []string) string {
	if len(args) > 1 {
		return args[1]
	}
	// default config file
	return "config.yaml"
}
