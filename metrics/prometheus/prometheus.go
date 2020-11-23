package prometheus

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/devops-works/scan-exporter/common"
	"github.com/devops-works/scan-exporter/handlers"
	"github.com/devops-works/scan-exporter/metrics"
	"github.com/devops-works/scan-exporter/storage"
)

// Server holds a metrics server configuration
type Server struct {
	storage                                            storage.ListManager
	notRespondingList                                  map[string]bool
	numOfTargets, numOfDownTargets                     prometheus.Gauge
	unexpectedPorts, openPorts, closedPorts, diffPorts *prometheus.GaugeVec
}

// New instance of server
func New(store storage.ListManager, numOfTargets int) *Server {
	s := Server{
		storage: store,
		numOfTargets: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "scanexporter_targets_number_total",
			Help: "Number of targets detected in config file.",
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
	prometheus.MustRegister(s.numOfDownTargets)
	prometheus.MustRegister(s.unexpectedPorts)
	prometheus.MustRegister(s.openPorts)
	prometheus.MustRegister(s.closedPorts)
	prometheus.MustRegister(s.diffPorts)

	// Initialize the map
	s.notRespondingList = make(map[string]bool)

	return &s
}

// ReceiveResults receives data from a finished scan. It also receive the number of targets declared in config file.
func (s *Server) ReceiveResults(res metrics.ResMsg) error {
	var m sync.Mutex
	if res.Protocol == "icmp" {
		s.icmpNotResponding(res.OpenPorts, res.IP, &m)
		return nil
	}

	setName := res.IP + ":" + res.Protocol

	// Expose the number of unexpected ports.
	s.unexpectedPorts.WithLabelValues(res.Protocol, res.Name).Set(float64(len(res.UnexpectedPorts)))

	// Expose the number of open ports.
	s.openPorts.WithLabelValues(res.Protocol, res.Name).Set(float64(len(res.OpenPorts)))

	// Expose the number of closed ports.
	s.closedPorts.WithLabelValues(res.Protocol, res.Name).Set(float64(len(res.ClosedPorts)))

	// Redis
	prev, err := s.storage.ReadList(setName)
	if err != nil {
		return err
	}

	diff := common.CompareStringSlices(prev, res.OpenPorts)
	s.diffPorts.WithLabelValues(res.Protocol, res.Name).Set(float64(diff))

	if err = s.storage.ReplaceList(setName, res.OpenPorts); err != nil {
		return err
	}

	return nil
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

// icmpNotResponding adjust the numOfDownTargets variable depending of the current and the previous
// status of the target.
func (s *Server) icmpNotResponding(ports []string, IP string, m *sync.Mutex) {
	m.Lock()
	defer m.Unlock()

	// Check if the IP is already in the map.
	_, ok := s.notRespondingList[IP]
	if !ok {
		// If not, add it as responding.
		s.notRespondingList[IP] = false
	}

	var isResponding bool
	// When a target responds, the ports array contains a value.
	if len(ports) == 0 {
		isResponding = false
	} else {
		isResponding = true
	}

	// Check if the target didn't respond in the previous scan.
	alreadyNotResponding := s.notRespondingList[IP]

	if isResponding && alreadyNotResponding {
		// Wasn't responding, but now is ok
		s.numOfDownTargets.Dec()
		s.notRespondingList[IP] = false

	} else if !isResponding && !alreadyNotResponding {
		// First time it doesn't respond.
		// Increment the number of down targets.
		s.numOfDownTargets.Inc()
		s.notRespondingList[IP] = true
	}
	// Else, everything is good, do nothing or everything is as bad as it was, so do nothing too.
}
