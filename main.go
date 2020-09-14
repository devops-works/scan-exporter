package main

import (
	"flag"
	"io"
	"io/ioutil"
	"os"

	"devops-works/scan-exporter/scan"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type conf struct {
	Targets []scan.Target `yaml:"targets"`
}

var logPath = flag.String("logpath", "./", "Path to save log files")

func main() {
	var confFile string
	flag.StringVar(&confFile, "config", "config.yaml", "path to config file")
	flag.Parse()

	c := conf{}

	conf, err := os.Open(confFile)
	if err != nil {
		log.Fatalf("unable to open config %s: %v", confFile, err)
	}
	c.getConf(conf)
	log.Infof("%d targets found in %s", len(c.Targets), confFile)

	// targetList is an array that will contain each instance of up target found in conf file
	targetList := []scan.Target{}
	for _, target := range c.Targets {
		t := target
		if err := t.Validate(); err != nil {
			// Invalid target specification
			log.Fatalf("error with target %v: %v", target, err)
		}
		targetList = append(targetList, t)
	}

	for i := 0; i < len(targetList); i++ {
		t := targetList[i]
		log.Infof("Starting %s scan", t.Name)
		t.Scan()
	}
}

// getConf reads confFile and unmarshall it
func (c *conf) getConf(r io.Reader) {
	yamlConf, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("Error while reading: %v ", err)
	}

	if err = yaml.Unmarshal(yamlConf, &c); err != nil {
		log.Fatalf("Error while unmarshalling configuration: %v", err)
	}
}
