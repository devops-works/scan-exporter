package metrics

import (
	"devops-works/scan-exporter/handlers"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

// ResMsg holds all the data received from a scan.
type ResMsg struct {
	ID              string
	IP              string
	Protocol        string
	OpenPorts       []string
	UnexpectedPorts []string
	ClosedPorts     []string
}

var (
	numOfTargets = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "scanexporter_targets_number_total",
		Help: "Number of targets detected in config file.",
	})
)

// Handle receives data from a finished scan. It also receive the number of targets declared in config file.
func Handle(res ResMsg, nTargets int) {
	// Set the number of targets. This is done once.
	numOfTargets.Set(float64(nTargets))

	// check if there is already some entries in redis
	// write data in target:ip:proto:1 if there is something, else in target:ip:proto:0
	// compare
	// expose
}

// StartServ starts the prometheus server.
func StartServ(l zerolog.Logger) {
	srv := &http.Server{
		Addr:         ":2112",
		Handler:      handlers.HandleFunc(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	l.Error().Msgf("server error : %s", srv.ListenAndServe())
}

// init is called at package initialisation.
func init() {
	prometheus.MustRegister(numOfTargets)
}
