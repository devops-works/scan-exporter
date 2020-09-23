package main

import (
	"flag"
	"os"

	"devops-works/scan-exporter/config"
	"devops-works/scan-exporter/scan"

	"github.com/rs/zerolog"
)

var logPath = flag.String("logpath", "./", "Path to save log files")

func main() {
	var confFile, logLevel string
	flag.StringVar(&confFile, "config", "config.yaml", "path to config file")
	flag.StringVar(&logLevel, "loglevel", "info", "log level to use")
	flag.Parse()

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	lvl, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		logger.Fatal().Msgf("unable to parse log level %s: %v", logLevel, err)
	}

	logger = logger.Level(lvl).With().Logger()

	c, err := config.New(confFile)
	if err != nil {
		logger.Fatal().Msgf("unable to read config %s: %v", confFile, err)
	}

	logger.Info().Msgf("%d target(s) found in %s", len(c.Targets), confFile)

	// targetList is an array that will contain each instance of up target found in conf file
	targetList := []*scan.Target{}
	for _, target := range c.Targets {
		t, err := scan.New(
			target.Name,
			target.IP,
			scan.WithPorts("tcp", target.TCP.Period, target.TCP.Range, target.TCP.Expected),
			scan.WithPorts("udp", target.UDP.Period, target.UDP.Range, target.UDP.Expected),
			scan.WithPorts("icmp", target.ICMP.Period, target.ICMP.Range, target.ICMP.Expected),
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
		t.Run()
	}
}
