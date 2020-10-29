package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/devops-works/scan-exporter/config"
	"github.com/devops-works/scan-exporter/metrics/prometheus"
	"github.com/devops-works/scan-exporter/pprof"
	"github.com/devops-works/scan-exporter/scan"
	"github.com/devops-works/scan-exporter/storage/redis"
)

var (
	Version   string
	BuildDate string
)

func main() {
	var confFile, logLevel, redisURL, pprofAddr string
	flag.StringVar(&confFile, "config", "config.yaml", "path to config file")
	flag.StringVar(&logLevel, "log.level", "info", "log level to use")
	flag.StringVar(&redisURL, "redis.url", "", "Redis URL (default: redis://127.0.0.1:6379/0)")
	flag.StringVar(&pprofAddr, "pprof.addr", "127.0.0.1:6060", "pprof addr")
	flag.Parse()

	fmt.Printf("scan-exporter version %s (built %s)\n", Version, BuildDate)

	pprofServer, err := pprof.New(pprofAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create pprof server")
	}
	log.Info().Msgf("pprof started on 'http://%s'", pprofServer.Addr)

	go pprofServer.Run()

	// Priority to flags
	if redisEnv := os.Getenv("REDIS_URL"); redisEnv != "" && redisURL == "" {
		redisURL = redisEnv
	}
	// If nothing is provided in both env and flag, set a default value
	if redisURL == "" {
		redisURL = "redis://127.0.0.1:6379/0"
	}

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	lvl, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		logger.Fatal().Msgf("unable to parse log level %s: %v", logLevel, err)
	}

	logger = logger.Level(lvl).With().Logger()

	// Read config file.
	c, err := config.New(confFile)
	if err != nil {
		logger.Fatal().Msgf("unable to read config %s: %v", confFile, err)
	}

	logger.Info().Msgf("%d target(s) found in %s", len(c.Targets), confFile)

	storage, err := redis.New(redisURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("error while initializing redis")
	}
	m := prometheus.New(storage)

	// targetList is an array that will contain each instance of up target found in conf file
	targetList := []*scan.Target{}
	for _, target := range c.Targets {
		t, err := scan.New(
			target.Name,
			target.IP,
			target.Workers,
			m,
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
		go t.Run()
	}

	// Start Prometheus server and wait forever
	err = m.StartServ(len(targetList))
	logger.Error().Err(err).Msg("error while running metric server")
}
