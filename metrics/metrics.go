package metrics

import (
	"net/http"
	"time"

	"github.com/devops-works/scan-exporter/handlers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// Server holds a metrics server configuration
type Server struct {
	notRespondingList                                  map[string]bool
	numOfTargets, numOfDownTargets, uptime             prometheus.Gauge
	unexpectedPorts, openPorts, closedPorts, diffPorts *prometheus.GaugeVec
}

// New instance of server
func New(numOfTargets int) *Server {
	s := Server{
		numOfTargets: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scanexporter_targets_number_total",
			Help: "Number of targets detected in config file.",
		}),

		uptime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scanexporter_uptime_sec",
			Help: "Scan exporter uptime, in seconds.",
		}),

		numOfDownTargets: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scanexporter_icmp_not_responding_total",
			Help: "Number of targets that doesn't respond to pings.",
		}),
		unexpectedPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_unexpected_open_ports_total",
			Help: "Number of ports that are open, and shouldn't be.",
		}, []string{"proto", "name"}),
		openPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_open_ports_total",
			Help: "Number of ports that are open.",
		}, []string{"proto", "name"}),

		closedPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_unexpected_closed_ports_total",
			Help: "Number of ports that are closed and shouldn't be.",
		}, []string{"proto", "name"}),

		diffPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_diff_ports_total",
			Help: "Number of ports that are different from previous scan.",
		}, []string{"proto", "name"}),
	}

	prometheus.MustRegister(s.numOfTargets)
	prometheus.MustRegister(s.uptime)
	prometheus.MustRegister(s.numOfDownTargets)
	prometheus.MustRegister(s.unexpectedPorts)
	prometheus.MustRegister(s.openPorts)
	prometheus.MustRegister(s.closedPorts)
	prometheus.MustRegister(s.diffPorts)

	// Initialize the map
	s.notRespondingList = make(map[string]bool)

	return &s
}

// StartServ starts the prometheus server.
func (s *Server) StartServ(nTargets int) {
	// Set the number of targets. This is done once.
	s.numOfTargets.Set(float64(nTargets))

	// Set the number of hosts that doesn't respond to ping to 0.
	s.numOfDownTargets.Set(0)

	srv := &http.Server{
		Addr:         ":2112",
		Handler:      handlers.HandleFunc(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal().Err(err).Msg("http server failed")
	}
}
