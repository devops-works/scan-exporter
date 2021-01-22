package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HandleFunc fills the router.
func HandleFunc() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())
	r.Handle("/health", http.HandlerFunc(healthCheckPage))
	r.NotFoundHandler = http.HandlerFunc(notFoundPage)

	return r
}

// notFoundPage set the response header to 404 status and prints an error message.
func notFoundPage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "<h1>404 page not found</h1>")
}

// healthCheckPage handles the /health page.
func healthCheckPage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `
{
	"alive": "true",
	"motd","%s"
}`, motd())
}

func motd() string {
	messages := []string{
		"Who the f*ck is Jeff, and why does he have nuclear weapons ?",
		"Working as a dancing monkey doesn't make you an anarchist.",
		"Seek cake now",
		"How can you ensure yourself that a hairdresser isn't a robot?",
		"Accept a monkey",
	}
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(len(messages))
	return messages[n]
}
