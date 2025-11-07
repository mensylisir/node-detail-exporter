package server

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Start starts the HTTP server and exposes the /metrics endpoint.
func Start(gatherer prometheus.Gatherer) {
	http.Handle("/metrics", promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))
	log.Println("Starting node-prober on :9101")
	log.Fatal(http.ListenAndServe(":9101", nil))
}
