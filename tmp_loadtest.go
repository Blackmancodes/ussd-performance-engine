package main

import (
    "context"
    "fmt"
    "net/http"
    "net/http/httptest"
    "strings"
    "time"

    "github.com/prometheus/client_golang/prometheus/promhttp"
    "performance-engine/internal/config"
    "performance-engine/internal/engine"
)

func main() {
    cfg, err := config.Load("config/example.yaml")
    if err != nil { panic(err) }

    api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"sessionId":"mock","response":"END OK","continue":false}`))
    }))
    defer api.Close()

    metrics := engine.NewMetrics()
    metricsServer := &http.Server{Addr: ":2112", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/metrics" { http.NotFound(w, r); return }
        promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
    })}
    go func() { _ = metricsServer.ListenAndServe() }()
    defer func() { _ = metricsServer.Close() }()

    sender := engine.HTTPSender{URL: api.URL, Client: api.Client()}
    runner := engine.Runner{Config: cfg, Sender: sender, Metrics: metrics}

    start := time.Now()
    for i := 0; i < 1000000; i++ {
        if _, err := runner.Run(context.Background(), i); err != nil {
            panic(err)
        }
    }
    elapsed := time.Since(start)
    time.Sleep(300 * time.Millisecond)

    resp, err := http.Get("http://localhost:2112/metrics")
    if err != nil { panic(err) }
    defer resp.Body.Close()

    body := new(strings.Builder)
    var lines int
    for _, line := range strings.Split(readMetrics("http://localhost:2112/metrics"), "\n") {
        if strings.HasPrefix(line, "ussd_requests_total") || strings.HasPrefix(line, "ussd_sessions_total") || strings.HasPrefix(line, "ussd_request_duration_seconds_count") {
            body.WriteString(line)
            body.WriteString("\n")
            lines++
            if lines >= 8 { break }
        }
    }

    fmt.Printf("completed=1000000 elapsed=%s\n", elapsed)
    fmt.Print(body.String())
}


func readMetrics(url string) string {
    resp, err := http.Get(url)
    if err != nil { panic(err) }
    defer resp.Body.Close()
    var b strings.Builder
    if _, err := b.WriteString(""); err != nil { panic(err) }
    if _, err := http.DefaultClient.Get(url); err != nil { panic(err) }
    return b.String()
}
