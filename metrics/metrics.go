package metrics

import (
	"devops-works/scan-exporter/handlers"
	"fmt"
	"net/http"
	"time"
)

type resMsg struct {
	id              string
	ip              string
	protocol        string
	openPorts       []string
	unexpectedPorts []string
	closedPorts     []string
}

// var (
// 	unexpectedPorts = promauto.NewCounter(prometheus.CounterOpts{
// 		Name: "scanexporter_unexpected_ports",
// 		Help: "Represents the fact that some ports are unexpected.",
// 	})
// )

// Handle receives data from a finished scan. It also receive the number of targets declared in config file
func Handle(res resMsg, nTargets int) {

	// go startServ()

	// check if there is already some entries in redis
	// write data in target:ip:proto:1 if there is something, else in target:ip:proto:0
	// compare
	// expose
}

// StartServ starts the prometheus server
func StartServ() {

	srv := &http.Server{
		Addr:         ":2112",
		Handler:      handlers.HandleFunc(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	fmt.Println(srv.ListenAndServe())
}
