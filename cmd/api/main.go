package main

import (
	"flag"
	"log"
	"net/http"

	"performance-engine/internal/api"
	"performance-engine/internal/engine"
)

func main() {
	listenAddr := flag.String("listen-addr", ":8080", "API listen address")
	receiverURL := flag.String("receiver-url", "http://127.0.0.1:8080/api/v1/receive", "receiver URL used by /api/v1/send")
	apiKey := flag.String("api-key", "", "optional X-API-Key required on API routes")
	flag.Parse()

	handler := api.Handler{
		Sender: engine.HTTPSender{URL: *receiverURL, APIKey: *apiKey, APIKeyName: "X-API-Key"},
		APIKey: *apiKey,
	}
	log.Printf("API listening on %s", *listenAddr)
	log.Fatal(http.ListenAndServe(*listenAddr, handler))
}
