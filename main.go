package main

import (
	"flag"
	"fmt"

	"github.com/devops-works/scan-exporter/config"
	"github.com/devops-works/scan-exporter/pprof"
	"github.com/devops-works/scan-exporter/scan"
	"github.com/rs/zerolog/log"
)

var (
	// Version holds the build version
	Version string
	// BuildDate holds the build date
	BuildDate string
)

func main() {
	var confFile, logLevel, dbURL, pprofAddr string
	flag.StringVar(&confFile, "config", "config.yaml", "path to config file")
	flag.StringVar(&logLevel, "log.level", "info", "log level to use")
	flag.StringVar(&dbURL, "db.url", "", "datastore URL (default: redis://127.0.0.1:6379/0)")
	flag.StringVar(&pprofAddr, "pprof.addr", "", "pprof addr")
	flag.Parse()

	fmt.Printf("scan-exporter version %s (built %s)\n", Version, BuildDate)

	// Start  pprof server is asked.
	if pprofAddr != "" {
		pprofServer, err := pprof.New(pprofAddr)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to create pprof server")
		}
		log.Info().Msgf("pprof started on 'http://%s'", pprofServer.Addr)

		go pprofServer.Run()
	}

	// Parse configuration file
	c, err := config.New(confFile)
	if err != nil {
		log.Fatal().Msgf("error reading %s: %s", confFile, err)
	}

	if err := scan.Start(c); err != nil {
		log.Fatal().Err(err).Msg("error with scanner")
	}
}
