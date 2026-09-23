package engine

import (
	"context"
	"strings"
	"testing"
)

func TestRunnerEmitsPrometheusMetrics(t *testing.T) {
	metrics := NewMetrics()
	runner := Runner{Config: testConfig(), Sender: &recordingSender{}, Metrics: metrics}
	if _, err := runner.Run(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	metricFamilies, err := metrics.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, family := range metricFamilies {
		names = append(names, family.GetName())
	}
	joined := strings.Join(names, ",")
	for _, expected := range []string{"ussd_requests_total", "ussd_request_duration_seconds", "ussd_sessions_total"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("metrics missing %q: %s", expected, joined)
		}
	}
}