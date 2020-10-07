package metrics

import (
	"devops-works/scan-exporter/common"
	"devops-works/scan-exporter/handlers"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis"
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

	unexpectedPorts = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scanexporter_unexpected_ports_total",
		Help: "Number of ports that are open, and shouldn't be.",
	},
		[]string{
			"proto",
			"ip",
		},
	)

	openPorts = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scanexporter_open_ports_total",
		Help: "Number of ports that are open.",
	},
		[]string{
			"proto",
			"ip",
		},
	)

	closedPorts = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scanexporter_closed_ports_total",
		Help: "Number of ports that are closed and shouldn't be.",
	},
		[]string{
			"proto",
			"ip",
		},
	)

	diffPorts = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scanexporter_diff_ports_total",
		Help: "Number of ports that are different from previous scan.",
	},
		[]string{
			"proto",
			"ip",
		},
	)

	notRespondingList = []string{} // Improve this with mutex
	rdb               *redis.Client
)

// Handle receives data from a finished scan. It also receive the number of targets declared in config file.
func Handle(res ResMsg) {
	if res.Protocol == "icmp" {
		icmpNotResponding(res.OpenPorts, res.IP)
		return
	}

	// Expose the number of unexpected ports.
	unexpectedPorts.WithLabelValues(res.Protocol, res.IP).Set(float64(len(res.UnexpectedPorts)))

	// Expose the number of open ports.
	openPorts.WithLabelValues(res.Protocol, res.IP).Set(float64(len(res.OpenPorts)))

	// Expose the number of closed ports.
	closedPorts.WithLabelValues(res.Protocol, res.IP).Set(float64(len(res.ClosedPorts)))

	setName := res.IP + "/" + res.Protocol
	// Read dataset
	items := readSet(rdb, setName)
	fmt.Printf("dataset read from redis : %s\n", items)
	if len(items) == 0 {
		writeSet(rdb, setName, res.OpenPorts)
		diffPorts.WithLabelValues(res.Protocol, res.IP).Set(0)
	} else {
		diff := common.CompareStringSlices(items, res.OpenPorts)
		diffPorts.WithLabelValues(res.Protocol, res.IP).Set(float64(diff))
		writeSet(rdb, setName, res.OpenPorts)
	}
}

// StartServ starts the prometheus server.
func StartServ(l zerolog.Logger, nTargets int) {
	// Set the number of targets. This is done once.
	numOfTargets.Set(float64(nTargets))

	// Set the number of hosts that doesn't respond to ping to 0.
	numOfDownTargets.Set(0)

	// init rdb
	initRedisClient()

	srv := &http.Server{
		Addr:         ":2112",
		Handler:      handlers.HandleFunc(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	l.Error().Msgf("server error : %s", srv.ListenAndServe())
}

// icmpNotResponding adjust the numOfDownTargets variable depending of the current and the previous
// status of the target.
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
				notRespondingList[index] = notRespondingList[len(notRespondingList)-1]
				notRespondingList[len(notRespondingList)-1] = ""
				notRespondingList = notRespondingList[:len(notRespondingList)-1]
			}
		}

	} else if !isResponding && !alreadyNotResponding {
		// First time it doesn't respond.
		// Increment the number of down targets.
		numOfDownTargets.Inc()
		// Add IP to notRespondingList.
		notRespondingList = append(notRespondingList, IP)
	}
	// Else, everything is good, do nothing or everything is as bad as it was, so do nothing too.

}

// writeSet writes items in a Redis dataset called setName.
func writeSet(rdb *redis.Client, setName string, items []string) {
	for _, item := range items {
		err := rdb.SAdd(setName, item, 0).Err()
		if err != nil {
			panic(err) // TODO: change this :/
		}
	}
}

// readSet reads items from a Redis dataset called setName.
func readSet(rdb *redis.Client, setName string) []string {
	items, err := rdb.SMembers(setName).Result()
	if err != nil {
		panic(err)
	}
	return items
}

// initRedisClient initiates a new Redis client item.
func initRedisClient() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	pong, err := rdb.Ping().Result()
	if pong != "PONG" || err != nil {
		panic(err) // TODO: change this
	}
}

// init is called at package initialisation. It initialize prometheus variables.
func init() {
	prometheus.MustRegister(numOfTargets)
	prometheus.MustRegister(numOfDownTargets)
	prometheus.MustRegister(unexpectedPorts)
	prometheus.MustRegister(openPorts)
	prometheus.MustRegister(closedPorts)
	prometheus.MustRegister(diffPorts)
}
