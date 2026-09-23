package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"performance-engine/internal/config"
	"performance-engine/internal/engine"
)

func main() {
	configPath := flag.String("config", "config/example.yaml", "path to YAML config")
	mock := flag.Bool("mock", false, "use the local mock receiver")
	count := flag.Int("sessions", 1, "number of sessions to run")
	role := flag.String("role", "coordinator", "execution role: coordinator or worker")
	workerID := flag.Int("worker-id", -1, "zero-based worker ID; defaults to config")
	workerCount := flag.Int("worker-count", -1, "number of workers; defaults to config")
	mockErrorRate := flag.Float64("mock-error-rate", -1, "mock receiver error rate between 0 and 1; defaults to config")
	coordinatorHTTPAddr := flag.String("coordinator-http-addr", "", "serve the coordinator worker protocol on this address")
	coordinatorURL := flag.String("coordinator-url", "", "connect a worker to a coordinator HTTP URL")
	coordinatorToken := flag.String("coordinator-token", "", "shared bearer token for coordinator HTTP requests")
	metricsAddr := flag.String("metrics-addr", "", "serve Prometheus metrics on this address and keep the process alive")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	if *workerID >= 0 {
		cfg.Scale.WorkerID = *workerID
	}
	if *workerCount >= 0 {
		cfg.Scale.WorkerCount = *workerCount
	}
	if *mockErrorRate >= 0 {
		cfg.Behavior.MockErrorRate = *mockErrorRate
	}
	if err := cfg.Validate(); err != nil {
		log.Fatal(err)
	}
	var sender engine.Sender = engine.HTTPSender{URL: cfg.Target.SenderURL, APIKey: cfg.Target.APIKey, APIKeyName: cfg.Target.APIKeyHeader}
	var server *httptest.Server
	if *mock {
		var mockRequestCount uint64
		server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			requestNumber := atomic.AddUint64(&mockRequestCount, 1)
			if cfg.Behavior.MockErrorRate > 0 && float64(requestNumber%10000)/10000 < cfg.Behavior.MockErrorRate {
				http.Error(writer, `{"error":"simulated receiver failure"}`, http.StatusBadGateway)
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"sessionId":"mock","response":"END OK","continue":false}`))
		}))
		defer server.Close()
		sender = engine.HTTPSender{URL: server.URL, APIKeyName: cfg.Target.APIKeyHeader}
	}
	metrics := engine.NewMetrics()
	if *metricsAddr != "" {
		metricsServer := &http.Server{Addr: *metricsAddr, Handler: metricsHandler(metrics)}
		go func() {
			if serveErr := metricsServer.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
				log.Printf("metrics server: %v", serveErr)
			}
		}()
	}
	if *role != "coordinator" && *role != "worker" {
		log.Fatalf("invalid role %q: want coordinator or worker", *role)
	}
	configuredWorkerCount := cfg.Scale.WorkerCount
	if configuredWorkerCount == 0 {
		configuredWorkerCount = 1
	}
	protocolServer := false
	var protocolCoordinator *engine.HTTPCoordinator
	var protocolHTTPServer *http.Server
	if *role == "coordinator" && *coordinatorHTTPAddr != "" {
		coordinator, coordinatorErr := engine.NewHTTPCoordinator(cfg, *count, configuredWorkerCount)
		if coordinatorErr != nil {
			log.Fatal(coordinatorErr)
		}
		coordinator.AuthToken = *coordinatorToken
		protocolCoordinator = coordinator
		protocolHTTPServer = &http.Server{Addr: *coordinatorHTTPAddr, Handler: coordinator}
		protocolServer = true
		go func() {
			if serveErr := protocolHTTPServer.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
				log.Printf("coordinator server: %v", serveErr)
			}
		}()
		log.Printf("coordinator protocol listening on %s", *coordinatorHTTPAddr)
	}
	var results []engine.SessionResult
	if protocolServer {
		results = nil
	} else if *role == "coordinator" {
		coordinator := engine.Coordinator{Config: cfg, Sender: sender, Metrics: metrics}
		results, err = coordinator.Run(context.Background(), *count, configuredWorkerCount)
	} else if *coordinatorURL != "" {
		worker := engine.Worker{ID: cfg.Scale.WorkerID, Count: configuredWorkerCount, Config: cfg, Sender: sender, Metrics: metrics}
		results, err = (engine.HTTPWorker{BaseURL: *coordinatorURL, AuthToken: *coordinatorToken}).Run(context.Background(), worker)
	} else {
		worker := engine.Worker{ID: cfg.Scale.WorkerID, Count: configuredWorkerCount, Config: cfg, Sender: sender, Metrics: metrics}
		results, err = worker.Run(context.Background(), *count)
	}
	if err != nil {
		log.Print(err)
	}
	for _, result := range results {
		if result.State == "FAILED" {
			log.Printf("session=%s state=%s", result.SessionID, result.State)
			continue
		}
		fmt.Printf("session=%s journey=%s network=%s msisdn=%s requests=%d state=%s\n", result.SessionID, result.Journey, result.Network, result.MSISDN, len(result.Requests), result.State)
	}
	if protocolServer {
		if waitErr := protocolCoordinator.Wait(context.Background()); waitErr != nil {
			log.Print(waitErr)
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = protocolHTTPServer.Shutdown(shutdownCtx)
	} else if *metricsAddr != "" {
		for {
			time.Sleep(time.Hour)
		}
	}
}

func metricsHandler(metrics *engine.Metrics) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/metrics" {
			http.NotFound(writer, request)
			return
		}
		handler := promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})
		handler.ServeHTTP(writer, request)
	})
}
