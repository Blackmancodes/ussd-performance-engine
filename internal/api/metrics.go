package api

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	Requests *prometheus.CounterVec
	Registry *prometheus.Registry
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	metrics := &Metrics{
		Requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "postman_requests_total", Help: "Requests received by the Postman-facing API."}, []string{"route", "status"}),
		Registry: registry,
	}
	registry.MustRegister(metrics.Requests)
	return metrics
}

func (m *Metrics) Observe(route, status string) {
	m.Requests.WithLabelValues(route, status).Inc()
}
