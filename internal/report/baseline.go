package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Baseline represents a persisted summary used for regression detection.
type Baseline struct {
	CreatedAt string  `json:"created_at"`
	Summary   Summary `json:"summary"`
}

// BaselineStore persists run baselines to disk.
type BaselineStore struct {
	Dir string
}

func NewBaselineStore(dir string) *BaselineStore {
	return &BaselineStore{Dir: dir}
}

func (s *BaselineStore) Save(name string, summary Summary) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("create baseline directory: %w", err)
	}

	payload := Baseline{
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Summary:   summary,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal baseline: %w", err)
	}
	path := filepath.Join(s.Dir, name+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write baseline file: %w", err)
	}
	return nil
}

func (s *BaselineStore) Load(name string) (Summary, error) {
	path := filepath.Join(s.Dir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Summary{}, fmt.Errorf("read baseline %q: %w", name, err)
	}
	var baseline Baseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return Summary{}, fmt.Errorf("unmarshal baseline %q: %w", name, err)
	}
	return baseline.Summary, nil
}

// RegressionThresholds describe the acceptable delta ranges for a compare operation.
type RegressionThresholds struct {
	MaxFailedDelta     int
	MaxLatencyDeltaSec float64
	MinCompletedDelta  int
}

// RegressionReport summarizes whether a run is within accepted thresholds.
type RegressionReport struct {
	WithinThresholds bool
	Current          Summary
	Baseline         Summary
	Delta            Delta
	Thresholds       RegressionThresholds
}

func DetectRegression(current, baseline Summary, thresholds RegressionThresholds) RegressionReport {
	delta := CompareSummaries(current, baseline)
	report := RegressionReport{
		Current:    current,
		Baseline:   baseline,
		Delta:      delta,
		Thresholds: thresholds,
	}
	report.WithinThresholds = true
	if delta.FailedDelta > thresholds.MaxFailedDelta {
		report.WithinThresholds = false
	}
	if delta.LatencyDeltaSec > thresholds.MaxLatencyDeltaSec {
		report.WithinThresholds = false
	}
	if delta.CompletedDelta < thresholds.MinCompletedDelta {
		report.WithinThresholds = false
	}
	return report
}
