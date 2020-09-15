package main

import (
	"flag"
	"io"
	"io/ioutil"
	"os"

	"devops-works/scan-exporter/scan"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v2"
)

// target holds an IP and a range of ports to scan
type target struct {
	Name   string   `yaml:"name"`
	Period string   `yaml:"period"`
	IP     string   `yaml:"ip"`
	TCP    protocol `yaml:"tcp"`
	UDP    protocol `yaml:"udp"`
}

type protocol struct {
	Range    string `yaml:"range"`
	Expected string `yaml:"expected"`
}

type conf struct {
	Targets []target `yaml:"targets"`
}

var logPath = flag.String("logpath", "./", "Path to save log files")

func main() {
	var confFile, logLevel string
	flag.StringVar(&confFile, "config", "config.yaml", "path to config file")
	flag.StringVar(&logLevel, "loglevel", "info", "log level to use")
	flag.Parse()

	c := conf{}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	lvl, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		logger.Fatal().Msgf("unable to parse log level %s: %v", logLevel, err)
	}

	logger = logger.Level(lvl).With().Logger()


	conf, err := os.Open(confFile)
	if err != nil {
		logger.Fatal().Msgf("unable to open config %s: %v", confFile, err)
	}

	err = c.getConf(conf)
	if err != nil {
		logger.Fatal().Msgf("unable to read config %s: %v", confFile, err)
	}

	logger.Info().Msgf("%d targets found in %s", len(c.Targets), confFile)

	// targetList is an array that will contain each instance of up target found in conf file
	targetList := []*scan.Target{}
	for _, target := range c.Targets {
		t, err := scan.New(
			target.Name,
			target.Period,
			target.IP,
			scan.WithPorts("tcp", target.TCP.Range, target.TCP.Expected),
			scan.WithPorts("udp", target.UDP.Range, target.UDP.Expected),
			scan.WithLogger(logger),
		)

		if err != nil {
			logger.Fatal().Msgf("error with target %q: %v", target.Name, err)
		}

		targetList = append(targetList, t)
	}

	for i := 0; i < len(targetList); i++ {
		t := targetList[i]
		logger.Info().Msgf("Starting %s scan", t.Name())
		t.Scan()
	}
}

// getConf reads confFile and unmarshall it
func (c *conf) getConf(r io.Reader) error {
	yamlConf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	if err = yaml.Unmarshal(yamlConf, &c); err != nil {
		return err
	}

	return nil
}
