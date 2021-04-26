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
	Addr                                                    string
	NotRespondingList                                       map[string]bool
	NumOfTargets, PendingScans, NumOfDownTargets, Uptime    prometheus.Gauge
	UnexpectedPorts, OpenPorts, ClosedPorts, DiffPorts, Rtt *prometheus.GaugeVec
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
		NumOfTargets: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scanexporter_targets_number_total",
			Help: "Number of targets detected in config file.",
		}),

		PendingScans: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scanexporter_pending_scans",
			Help: "Number of scans in the waiting line.",
		}),

		Uptime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scanexporter_uptime_sec",
			Help: "Scan exporter uptime, in seconds.",
		}),

		NumOfDownTargets: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scanexporter_icmp_not_responding_total",
			Help: "Number of targets that doesn't respond to pings.",
		}),
		UnexpectedPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_unexpected_open_ports_total",
			Help: "Number of ports that are open, and shouldn't be.",
		}, []string{"name", "ip"}),
		OpenPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_open_ports_total",
			Help: "Number of ports that are open.",
		}, []string{"name", "ip"}),

		ClosedPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_unexpected_closed_ports_total",
			Help: "Number of ports that are closed and shouldn't be.",
		}, []string{"name", "ip"}),

		DiffPorts: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_diff_ports_total",
			Help: "Number of ports that are different from previous scan.",
		}, []string{"name", "ip"}),

		Rtt: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "scanexporter_rtt_total",
			Help: "Response time of the target.",
		}, []string{"name", "ip"}),
	}

	prometheus.MustRegister(
		s.NumOfTargets,
		s.PendingScans,
		s.Uptime,
		s.NumOfDownTargets,
		s.UnexpectedPorts,
		s.OpenPorts,
		s.ClosedPorts,
		s.DiffPorts,
		s.Rtt,
	)

	// Initialize the map
	s.NotRespondingList = make(map[string]bool)

	// Start uptime counter
	go s.uptimeCounter()

	return &s
}

// Start starts the prometheus server
func (s *Server) Start() error {
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
			// New metrics set has been receievd

			s.DiffPorts.WithLabelValues(nm.Name, nm.IP).Set(float64(nm.Diff))
			log.Info().Str("name", nm.Name).Str("ip", nm.IP).Msgf("%s (%s) open ports: %s", nm.Name, nm.IP, nm.Open)

			s.OpenPorts.WithLabelValues(nm.Name, nm.IP).Set(float64(len(nm.Open)))

			// If the port is open but not expected
			for _, port := range nm.Open {
				if !common.StringInSlice(port, nm.Expected) {
					unexpectedPorts = append(unexpectedPorts, port)
				}
			}
			s.UnexpectedPorts.WithLabelValues(nm.Name, nm.IP).Set(float64(len(unexpectedPorts)))
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
			s.ClosedPorts.WithLabelValues(nm.Name, nm.IP).Set(float64(len(closedPorts)))
			if len(closedPorts) > 0 {
				log.Warn().Str("name", nm.Name).Str("ip", nm.IP).Msgf("%s (%s) unexpected closed ports: %s", nm.Name, nm.IP, closedPorts)
			} else {
				log.Info().Str("name", nm.Name).Str("ip", nm.IP).Msgf("%s (%s) unexpected closed ports: %s", nm.Name, nm.IP, closedPorts)
			}

			closedPorts = nil
		case pm := <-pingChan:
			log.Debug().Str("name", pm.Name).Str("ip", pm.IP).Msg("received new ping result")

			// New ping metric has been received
			if pm.IsResponding {
				log.Debug().Str("name", pm.Name).Str("ip", pm.IP).Str("rtt", pm.RTT.String()).Msgf("%s (%s) responds to ICMP requests", pm.Name, pm.IP)
			} else {
				log.Warn().Str("name", pm.Name).Str("ip", pm.IP).Str("rtt", "nil").Msgf("%s (%s) does not respond to ICMP requests", pm.Name, pm.IP)
			}

			// Update target's RTT metric
			s.Rtt.WithLabelValues(pm.Name, pm.IP).Set(float64(pm.RTT))

			// Check if the IP is already in the map.
			_, ok := s.NotRespondingList[pm.IP]
			if !ok {
				// If not, add it as responding.
				s.NotRespondingList[pm.IP] = false
			}

			// Check if the target didn't respond in the previous scan.
			alreadyNotResponding := s.NotRespondingList[pm.IP]

			if pm.IsResponding && alreadyNotResponding {
				// Wasn't responding, but now is ok
				s.NumOfDownTargets.Dec()
				s.NotRespondingList[pm.IP] = false

			} else if !pm.IsResponding && !alreadyNotResponding {
				// First time it doesn't respond.
				// Increment the number of down targets.
				s.NumOfDownTargets.Inc()
				s.NotRespondingList[pm.IP] = true
			}
			// Else, everything is good, do nothing or everything is as bad as it was, so do nothing too.
		case pending := <-pending:
			// New pending metric has been received

			s.PendingScans.Set(float64(pending))
			log.Trace().Int("pending", pending).Msgf("%d pending scans", pending)
		}
	}
}

// uptime metric
func (s *Server) uptimeCounter() {
	for {
		s.Uptime.Add(5)
		time.Sleep(5 * time.Second)
	}
}
