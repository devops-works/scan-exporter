package metrics

import (
	"devops-works/scan-exporter/common"
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

	numOfDownTargets = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "scanexporter_icmp_not_responding_total",
		Help: "Number of targets that doesn't respond to pings.",
	})

	notRespondingList = []string{}
)

// Handle receives data from a finished scan. It also receive the number of targets declared in config file.
func Handle(res ResMsg) {
	if res.Protocol == "icmp" {
		icmpNotResponding(res.OpenPorts, res.IP)
	}

	// check if there is already some entries in redis
	// write data in target:ip:proto:1 if there is something, else in target:ip:proto:0
	// compare
	// expose
}

// StartServ starts the prometheus server.
func StartServ(l zerolog.Logger, nTargets int) {
	// Set the number of targets. This is done once.
	numOfTargets.Set(float64(nTargets))

	// Set the number of hosts that doesn't respond to ping to 0.
	numOfDownTargets.Set(0)

	srv := &http.Server{
		Addr:         ":2112",
		Handler:      handlers.HandleFunc(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	l.Error().Msgf("server error : %s", srv.ListenAndServe())
}

func icmpNotResponding(ports []string, IP string) {
	isResponding := true
	if len(ports) == 0 {
		isResponding = !isResponding
	}

	alreadyNotResponding := common.StringInSlice(IP, notRespondingList)

	if isResponding && alreadyNotResponding {
		// Wasn't responding, but now is ok
		numOfDownTargets.Dec()

		for index := range notRespondingList {
			if notRespondingList[index] == IP {
				// Remove the element at index i from a.
				notRespondingList[index] = notRespondingList[len(notRespondingList)-1] // Copy last element to index i.
				notRespondingList[len(notRespondingList)-1] = ""                       // Erase last element (write zero value).
				notRespondingList = notRespondingList[:len(notRespondingList)-1]       // Truncate slice.
			}
		}

	} else if !isResponding && !alreadyNotResponding {
		// First time it doesn't respond
		// Increment the number of down targets
		numOfDownTargets.Inc()
		// Add IP to notRespondingList
		notRespondingList = append(notRespondingList, IP)
	} else {
		// Everything is good, do nothing or everything is as bad as it was, so do nothing too.
	}
}

// init is called at package initialisation.
func init() {
	prometheus.MustRegister(numOfTargets)
	prometheus.MustRegister(numOfDownTargets)
}
