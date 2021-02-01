package metrics

import (
	"net/http"
	"time"

	"github.com/devops-works/scan-exporter/common"
	"github.com/devops-works/scan-exporter/handlers"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// Server is the metrics server. It contains all the Prometheus metrics
type Server struct {
	notRespondingList                                    map[string]bool
	numOfTargets, pendingScans, numOfDownTargets, uptime prometheus.Gauge
	unexpectedPorts, openPorts, closedPorts, diffPorts   *prometheus.GaugeVec
}

// NewMetrics is the type that will transit between scan and metrics. It carries
// informations that will be used for calculation, such as expected ports.
type NewMetrics struct {
	Name     string
	IP       string
	Diff     int
	Open     []string
	Closed   []string
	Expected []string
}

// PingInfo holds the ping update of a specific target
type PingInfo struct {
	Name         string
	IP           string
	IsResponding bool
	RTT          time.Duration
}

// Init initialize the metrics
func Init() *Server {
	s := Server{
		numOfTargets: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scanexporter_targets_number_total",
			Help: "Number of targets detected in config file.",
		}),

		pendingScans: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scanexporter_pending_scans",
			Help: "Number of scans in the waiting line.",
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
		}, []string{"name", "ip"}),
		openPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_open_ports_total",
			Help: "Number of ports that are open.",
		}, []string{"name", "ip"}),

		closedPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_unexpected_closed_ports_total",
			Help: "Number of ports that are closed and shouldn't be.",
		}, []string{"name", "ip"}),

		diffPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_diff_ports_total",
			Help: "Number of ports that are different from previous scan.",
		}, []string{"name", "ip"}),
	}

	prometheus.MustRegister(
		s.numOfTargets,
		s.pendingScans,
		s.uptime,
		s.numOfDownTargets,
		s.unexpectedPorts,
		s.openPorts,
		s.closedPorts,
		s.diffPorts,
	)

	// Initialize the map
	s.notRespondingList = make(map[string]bool)

	// Start uptime counter
	go s.uptimeCounter()

	return &s
}

// StartServ starts the prometheus server.
func (s *Server) StartServ(nTargets int) error {
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
	return srv.ListenAndServe()
}

// Updater updates metrics
func (s *Server) Updater(metChan chan NewMetrics, pingChan chan PingInfo, pending chan int) {
	var unexpectedPorts, closedPorts []string
	for {
		select {
		case nm := <-metChan:
			s.diffPorts.WithLabelValues(nm.Name, nm.IP).Set(float64(nm.Diff))
			log.Info().Str("name", nm.Name).Str("ip", nm.IP).Msgf("%s (%s) open ports: %s", nm.Name, nm.IP, nm.Open)

			s.openPorts.WithLabelValues(nm.Name, nm.IP).Set(float64(len(nm.Open)))

			// If the port is open but not expected
			for _, port := range nm.Open {
				if !common.StringInSlice(port, nm.Expected) {
					unexpectedPorts = append(unexpectedPorts, port)
				}
			}
			s.unexpectedPorts.WithLabelValues(nm.Name, nm.IP).Set(float64(len(unexpectedPorts)))
			if len(unexpectedPorts) > 0 {
				log.Warn().Str("name", nm.Name).Str("ip", nm.IP).Msgf("%s (%s) unexpected open ports: %s", nm.Name, nm.IP, unexpectedPorts)
			} else {
				log.Info().Str("name", nm.Name).Str("ip", nm.IP).Msgf("%s (%s) unexpected open ports: %s", nm.Name, nm.IP, unexpectedPorts)
			}

			unexpectedPorts = nil

			// If the port is expected but not open
			for _, port := range nm.Expected {
				if !common.StringInSlice(port, nm.Open) {
					closedPorts = append(closedPorts, port)
				}
			}
			s.closedPorts.WithLabelValues(nm.Name, nm.IP).Set(float64(len(closedPorts)))
			if len(closedPorts) > 0 {
				log.Warn().Str("name", nm.Name).Str("ip", nm.IP).Msgf("%s (%s) unexpected closed ports: %s", nm.Name, nm.IP, closedPorts)
			} else {
				log.Info().Str("name", nm.Name).Str("ip", nm.IP).Msgf("%s (%s) unexpected closed ports: %s", nm.Name, nm.IP, closedPorts)
			}

			closedPorts = nil
		case pm := <-pingChan:
			if pm.IsResponding {
				log.Info().Str("name", pm.Name).Str("ip", pm.IP).Str("rtt", pm.RTT.String()).Msgf("%s (%s) responds to ICMP requests", pm.Name, pm.IP)
			} else {
				log.Warn().Str("name", pm.Name).Str("ip", pm.IP).Str("rtt", "nil").Msgf("%s (%s) does not respond to ICMP requests", pm.Name, pm.IP)
			}

			// Check if the IP is already in the map.
			_, ok := s.notRespondingList[pm.IP]
			if !ok {
				// If not, add it as responding.
				s.notRespondingList[pm.IP] = false
			}

			// Check if the target didn't respond in the previous scan.
			alreadyNotResponding := s.notRespondingList[pm.IP]

			if pm.IsResponding && alreadyNotResponding {
				// Wasn't responding, but now is ok
				s.numOfDownTargets.Dec()
				s.notRespondingList[pm.IP] = false

			} else if !pm.IsResponding && !alreadyNotResponding {
				// First time it doesn't respond.
				// Increment the number of down targets.
				s.numOfDownTargets.Inc()
				s.notRespondingList[pm.IP] = true
			}
			// Else, everything is good, do nothing or everything is as bad as it was, so do nothing too.
		case pending := <-pending:
			s.pendingScans.Set(float64(pending))
			log.Debug().Int("pending", pending).Msgf("%d pending scans", pending)
		}
	}
}

// uptime metric
func (s *Server) uptimeCounter() {
	for {
		s.uptime.Add(5)
		time.Sleep(5 * time.Second)
	}
}
