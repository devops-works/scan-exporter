package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/devops-works/scan-exporter/config"
	"github.com/devops-works/scan-exporter/logger"
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
	if err := run(os.Args, os.Stdout); err != nil {
		log.Fatal().Err(err).Msgf("error running %s", os.Args[0])
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	var confFile, pprofAddr, loglvl string
	flag.StringVar(&confFile, "config", "config.yaml", "path to config file")
	flag.StringVar(&pprofAddr, "pprof.addr", "", "pprof addr")
	flag.StringVar(&loglvl, "log.lvl", "info", "log level. Can be {trace,debug,info,warn,error,fatal}")
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

	// Set global loglevel
	// Overwrite the flag loglevel by the one given in configuration
	if c.LogLevel != "" {
		log.Info().Msgf("log level from configuration file found: %s", c.LogLevel)
		loglvl = c.LogLevel
	}

	logger := logger.New(loglvl)

	if err := scan.Start(c, logger); err != nil {
		return err
	}
	return nil
}
