package report

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBaselineStoreRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "baselines")
	store := NewBaselineStore(dir)
	summary := Summary{TotalRequests: 250, CompletedSessions: 220, FailedSessions: 5, AverageLatencySeconds: 1.7}

	if err := store.Save("prod-baseline", summary); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := store.Load("prod-baseline")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.TotalRequests != summary.TotalRequests || loaded.CompletedSessions != summary.CompletedSessions || loaded.FailedSessions != summary.FailedSessions {
		t.Fatalf("loaded baseline = %#v, want %#v", loaded, summary)
	}
	if _, err := os.Stat(filepath.Join(dir, "prod-baseline.json")); err != nil {
		t.Fatalf("baseline file missing: %v", err)
	}
}

func TestDetectRegression(t *testing.T) {
	current := Summary{TotalRequests: 120, CompletedSessions: 96, FailedSessions: 4, AverageLatencySeconds: 1.5}
	baseline := Summary{TotalRequests: 100, CompletedSessions: 100, FailedSessions: 2, AverageLatencySeconds: 1.2}

	report := DetectRegression(current, baseline, RegressionThresholds{MaxFailedDelta: 2, MaxLatencyDeltaSec: 0.5, MinCompletedDelta: 0})
	if report.WithinThresholds {
		t.Fatal("expected regression to be detected")
	}
	if report.Delta.CompletedDelta != -4 {
		t.Fatalf("CompletedDelta = %v, want -4", report.Delta.CompletedDelta)
	}
}
