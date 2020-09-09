package main

import (
	"fmt"
	"io/ioutil"
	"net"
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
	Range    string `yaml:"range"`
	Expected string `yaml:"expected"`
}

type app struct {
	infos targets
}

func main() {
	c := conf{}
	c.getConf("config.yaml")
	log.Infof("%d targets found in config file", len(c.Targets))

	for i := 0; i < len(c.Targets); i++ {
		a := app{
			infos: c.Targets[i],
		}
		fmt.Println(a)
	}
}

// getStatus returns true if the application respond to ping requests
func (a app) getStatus() bool {
	p := fastping.NewPinger()
	ra, err := net.ResolveIPAddr("ip4:icmp", a.infos.Name)
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
	return a.infos.Name + ":" + port
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
// func (a app) parsePortsRange() []string {
// 	var ports = []string{}

// 	switch a.infos. {
// 	// append all ports to the scan list
// 	case "all":
// 		for port := 1; port <= 65535; port++ {
// 			ports = append(ports, strconv.Itoa(port))
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
	yamlConf, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Errorf("Error while reading config.yaml: %v ", err)
	}

	if err = yaml.Unmarshal(yamlConf, &c); err != nil {
		log.Errorf("Error while unmarshalling yamlConf: %v", err)
	}

	return c
}
