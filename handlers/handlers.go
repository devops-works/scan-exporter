package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HandleFunc handles the metrics path.
func HandleFunc() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())

	r.NotFoundHandler = http.HandlerFunc(defaultPage)

	return r
}

// defaultPage set the response header to 404 status and prints an error message.
func defaultPage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "404 page not found")
}
