package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"performance-engine/internal/report"
)

func main() {
	metricsURL := flag.String("prom-url", "http://localhost:2112/metrics", "Prometheus metrics URL or query URL")
	metricsFile := flag.String("file", "", "Path to a Prometheus exposition file")
	metricsAlias := flag.String("metrics", "", "Alias for --file")
	baselineFile := flag.String("baseline", "", "Optional baseline metrics file to compare against")
	baselineDir := flag.String("baseline-dir", "./reports/baselines", "Directory for persisted baselines")
	outputDir := flag.String("output-dir", "./reports/runs", "Directory for generated comparison artifacts")
	baselineName := flag.String("baseline-name", "default", "Persisted baseline name")
	thresholdFailed := flag.Int("threshold-failed", 2, "Max allowed failed-session delta before regression")
	thresholdLatency := flag.Float64("threshold-latency", 0.5, "Max allowed latency increase in seconds before regression")
	thresholdCompleted := flag.Int("threshold-completed", 0, "Minimum completed-session delta to remain acceptable")
	output := flag.String("output", "", "Optional path to write the summary report (default: stdout)")
	outputFormat := flag.String("format", "json", "Output format: json, html, csv")
	flag.Parse()
	_ = outputDir

	if *metricsAlias != "" {
		*metricsFile = *metricsAlias
	}
	current, err := loadSummary(*metricsURL, *metricsFile)
	if err != nil {
		fatalf("load metrics: %v", err)
	}

	var base *report.Summary
	if *baselineFile != "" {
		baselineText, err := os.ReadFile(*baselineFile)
		if err != nil {
			fatalf("read baseline: %v", err)
		}
		baseline, err := report.ParseMetricsText(string(baselineText))
		if err != nil {
			fatalf("parse baseline: %v", err)
		}
		base = &baseline
	} else {
		store := report.NewBaselineStore(*baselineDir)
		baselinePath := filepath.Join(*baselineDir, *baselineName+".json")
		if _, statErr := os.Stat(baselinePath); statErr == nil {
			baseline, loadErr := store.Load(*baselineName)
			if loadErr != nil {
				fatalf("load baseline: %v", loadErr)
			}
			base = &baseline
			comparison := report.DetectRegression(current, baseline, report.RegressionThresholds{
				MaxFailedDelta:     *thresholdFailed,
				MaxLatencyDeltaSec: *thresholdLatency,
				MinCompletedDelta:  *thresholdCompleted,
			})
			if mkdirErr := os.MkdirAll(*outputDir, 0o755); mkdirErr != nil {
				fatalf("create output dir: %v", mkdirErr)
			}
			comparisonData, marshalErr := report.NewReport(current, &baseline).JSON()
			if marshalErr != nil {
				fatalf("marshal comparison report: %v", marshalErr)
			}
			if writeErr := os.WriteFile(filepath.Join(*outputDir, "comparison.json"), comparisonData, 0o644); writeErr != nil {
				fatalf("write comparison report: %v", writeErr)
			}
			if err := os.MkdirAll(*outputDir, 0o755); err != nil {
				fatalf("create output dir: %v", err)
			}
			comparisonData, err := report.NewReport(current, &baseline).JSON()
			if err != nil {
				fatalf("marshal comparison report: %v", err)
			}
			if err := os.WriteFile(filepath.Join(*outputDir, "comparison.json"), comparisonData, 0o644); err != nil {
				fatalf("write comparison report: %v", err)
			}
			fmt.Printf("within_thresholds=%v\n", comparison.WithinThresholds)
			fmt.Printf("request_delta=%d completed_delta=%d failed_delta=%d latency_delta=%0.6f\n", comparison.Delta.RequestDelta, comparison.Delta.CompletedDelta, comparison.Delta.FailedDelta, comparison.Delta.LatencyDeltaSec)
		} else if saveErr := store.Save(*baselineName, current); saveErr != nil {
			fatalf("save baseline: %v", saveErr)
		} else {
			if mkdirErr := os.MkdirAll(*outputDir, 0o755); mkdirErr != nil {
				fatalf("create output dir: %v", mkdirErr)
			}
			baselineData, marshalErr := report.NewReport(current, nil).JSON()
			if marshalErr != nil {
				fatalf("marshal baseline report: %v", marshalErr)
			}
			if writeErr := os.WriteFile(filepath.Join(*outputDir, "baseline.json"), baselineData, 0o644); writeErr != nil {
				fatalf("write baseline report: %v", writeErr)
			}
			if err := os.MkdirAll(*outputDir, 0o755); err != nil {
				fatalf("create output dir: %v", err)
			}
			baselineData, err := report.NewReport(current, nil).JSON()
			if err != nil {
				fatalf("marshal baseline report: %v", err)
			}
			if err := os.WriteFile(filepath.Join(*outputDir, "baseline.json"), baselineData, 0o644); err != nil {
				fatalf("write baseline report: %v", err)
			}
			fmt.Printf("baseline_saved=%s\n", *baselineName)
		}
	}

	reportDoc := report.NewReport(current, base)
	switch strings.ToLower(*outputFormat) {
	case "html":
		payload := reportDoc.HTML()
		if *output == "" {
			fmt.Print(payload)
			return
		}
		if err := os.WriteFile(*output, []byte(payload), 0o644); err != nil {
			fatalf("write html report: %v", err)
		}
		fmt.Printf("report written to %s\n", *output)
	case "csv":
		payload := reportDoc.CSV()
		if *output == "" {
			fmt.Print(payload)
			return
		}
		if err := os.WriteFile(*output, []byte(payload), 0o644); err != nil {
			fatalf("write csv report: %v", err)
		}
		fmt.Printf("report written to %s\n", *output)
	default:
		payload, err := reportDoc.JSON()
		if err != nil {
			fatalf("marshal report: %v", err)
		}
		if *output == "" {
			fmt.Println(string(payload))
			return
		}
		if err := os.WriteFile(*output, payload, 0o644); err != nil {
			fatalf("write json report: %v", err)
		}
		fmt.Printf("report written to %s\n", *output)
	}
}

func loadSummary(promURL, metricsFile string) (report.Summary, error) {
	metricsText, err := loadMetrics(promURL, metricsFile)
	if err != nil {
		return report.Summary{}, err
	}
	if strings.HasPrefix(strings.TrimSpace(metricsText), "{") {
		return report.ParsePrometheusAPIResponse(metricsText)
	}
	return report.ParseMetricsText(metricsText)
}

func loadMetrics(promURL, metricsFile string) (string, error) {
	if metricsFile != "" {
		body, err := os.ReadFile(metricsFile)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
	if promURL == "" {
		return "", fmt.Errorf("no metrics source provided")
	}
	if strings.HasPrefix(promURL, "http://") || strings.HasPrefix(promURL, "https://") {
		resp, err := http.Get(promURL)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", fmt.Errorf("status %s", resp.Status)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		return string(body), nil
	}
	return promURL, nil
}

func writeOutput(path string, payload any) {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fatalf("marshal report: %v", err)
	}
	if path == "" {
		fmt.Println(string(data))
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fatalf("write report: %v", err)
	}
	fmt.Printf("report written to %s\n", path)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
