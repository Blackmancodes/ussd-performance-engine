package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"performance-engine/internal/api"
	"performance-engine/internal/engine"
)

func main() {
	listenAddr := flag.String("listen-addr", ":8080", "API listen address")
	receiverURL := flag.String("receiver-url", "http://127.0.0.1:8080/api/v1/receive", "receiver URL used by /api/v1/send")
	apiKey := flag.String("api-key", "", "optional X-API-Key required on API routes")
	flag.Parse()

	metrics := api.NewMetrics()
	handler := api.Handler{
		Sender:  engine.HTTPSender{URL: *receiverURL, APIKey: *apiKey, APIKeyName: "X-API-Key"},
		APIKey:  *apiKey,
		Metrics: metrics,
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}))
	mux.Handle("/", handler)
	log.Printf("API listening on %s", *listenAddr)
	log.Fatal(http.ListenAndServe(*listenAddr, mux))
}
