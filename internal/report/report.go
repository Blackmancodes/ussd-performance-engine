package report

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Summary is the compact run-level view produced from Prometheus metrics.
type Summary struct {
	TotalRequests         int
	CompletedSessions     int
	FailedSessions        int
	AverageLatencySeconds float64
}

// Delta captures changes between a current run and a baseline run.
type Delta struct {
	RequestDelta    int
	CompletedDelta  int
	FailedDelta     int
	LatencyDeltaSec float64
}

// Report is the stage-5/6 artifact written from a completed run.
type Report struct {
	GeneratedAt string
	Summary     Summary
	Baseline    *Summary
	Delta       *Delta
}

var requestsRe = regexp.MustCompile(`^ussd_requests_total\{.*\}\s+([0-9.]+)$`)
var sessionsRe = regexp.MustCompile(`^ussd_sessions_total\{.*state="([A-Z]+)".*\}\s+([0-9.]+)$`)
var histogramSumRe = regexp.MustCompile(`^ussd_request_duration_seconds_sum\{.*\}\s+([0-9.]+)$`)
var histogramCountRe = regexp.MustCompile(`^ussd_request_duration_seconds_count\{.*\}\s+([0-9.]+)$`)

// ParseMetricsText extracts the run summary from the Prometheus text exposition format.
func ParseMetricsText(metricsText string) (Summary, error) {
	summary := Summary{}
	lineCount := 0
	latencySum := 0.0
	latencyCount := 0.0
	for _, line := range strings.Split(metricsText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "ussd_requests_total") {
			matches := requestsRe.FindStringSubmatch(line)
			if len(matches) == 2 {
				value, err := strconv.ParseFloat(matches[1], 64)
				if err != nil {
					return Summary{}, fmt.Errorf("parse request count: %w", err)
				}
				summary.TotalRequests += int(value)
				lineCount++
			}
			continue
		}
		if strings.HasPrefix(line, "ussd_sessions_total") {
			matches := sessionsRe.FindStringSubmatch(line)
			if len(matches) == 3 {
				value, err := strconv.ParseFloat(matches[2], 64)
				if err != nil {
					return Summary{}, fmt.Errorf("parse session count: %w", err)
				}
				switch matches[1] {
				case "COMPLETED":
					summary.CompletedSessions += int(value)
				case "FAILED":
					summary.FailedSessions += int(value)
				}
			}
			continue
		}
		if strings.HasPrefix(line, "ussd_request_duration_seconds_sum") {
			matches := histogramSumRe.FindStringSubmatch(line)
			if len(matches) == 2 {
				value, err := strconv.ParseFloat(matches[1], 64)
				if err != nil {
					return Summary{}, fmt.Errorf("parse latency sum: %w", err)
				}
				latencySum += value
			}
			continue
		}
		if strings.HasPrefix(line, "ussd_request_duration_seconds_count") {
			matches := histogramCountRe.FindStringSubmatch(line)
			if len(matches) == 2 {
				value, err := strconv.ParseFloat(matches[1], 64)
				if err != nil {
					return Summary{}, fmt.Errorf("parse latency count: %w", err)
				}
				latencyCount += value
			}
		}
	}
	if latencyCount > 0 {
		summary.AverageLatencySeconds = latencySum / latencyCount
	}
	if lineCount == 0 && summary.CompletedSessions == 0 && summary.FailedSessions == 0 {
		return Summary{}, fmt.Errorf("no parsable metrics found")
	}
	return summary, nil
}

// ParsePrometheusAPIResponse parses the JSON payload returned by Prometheus' /api/v1/query endpoint.
func ParsePrometheusAPIResponse(payload string) (Summary, error) {
	var response struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		return Summary{}, fmt.Errorf("unmarshal Prometheus response: %w", err)
	}
	if response.Status != "success" {
		return Summary{}, fmt.Errorf("Prometheus query status %q", response.Status)
	}

	summary := Summary{}
	latencySum := 0.0
	latencyCount := 0.0
	for _, result := range response.Data.Result {
		metricName := result.Metric["__name__"]
		if len(result.Value) < 2 {
			continue
		}
		valueStr, ok := result.Value[1].(string)
		if !ok {
			continue
		}
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}
		switch metricName {
		case "ussd_requests_total":
			summary.TotalRequests += int(value)
		case "ussd_sessions_total":
			switch result.Metric["state"] {
			case "COMPLETED":
				summary.CompletedSessions += int(value)
			case "FAILED":
				summary.FailedSessions += int(value)
			}
		case "ussd_request_duration_seconds_sum":
			latencySum += value
		case "ussd_request_duration_seconds_count":
			latencyCount += value
		}
	}
	if latencyCount > 0 {
		summary.AverageLatencySeconds = latencySum / latencyCount
	}
	if summary.TotalRequests == 0 && summary.CompletedSessions == 0 && summary.FailedSessions == 0 && latencyCount == 0 {
		return Summary{}, fmt.Errorf("no parsable metrics found in Prometheus API response")
	}
	return summary, nil
}

// CompareSummaries returns delta between the current and baseline summaries.
func CompareSummaries(current, baseline Summary) Delta {
	return Delta{
		RequestDelta:    current.TotalRequests - baseline.TotalRequests,
		CompletedDelta:  current.CompletedSessions - baseline.CompletedSessions,
		FailedDelta:     current.FailedSessions - baseline.FailedSessions,
		LatencyDeltaSec: current.AverageLatencySeconds - baseline.AverageLatencySeconds,
	}
}

// NewReport builds a report artifact from a current run and optional baseline.
func NewReport(current Summary, baseline *Summary) Report {
	report := Report{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Summary: current}
	if baseline != nil {
		report.Baseline = baseline
		delta := CompareSummaries(current, *baseline)
		report.Delta = &delta
	}
	return report
}

// JSON serializes the report as JSON.
func (r Report) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// HTML renders a readable executive summary for a report artifact.
func (r Report) HTML() string {
	var deltaHTML string
	if r.Delta != nil {
		deltaHTML = fmt.Sprintf(`
		<h2>Baseline delta</h2>
		<ul>
			<li>Request delta: %d</li>
			<li>Completed delta: %d</li>
			<li>Failed delta: %d</li>
			<li>Latency delta (s): %.3f</li>
		</ul>`, r.Delta.RequestDelta, r.Delta.CompletedDelta, r.Delta.FailedDelta, r.Delta.LatencyDeltaSec)
	}
	return fmt.Sprintf(`<!doctype html>
<html>
<head><meta charset="utf-8"><title>USSD Performance Report</title>
<style>body{font-family:Arial,sans-serif;padding:2rem;} h1,h2{color:#1f2937;} li{margin:0.4rem 0;} .card{border:1px solid #d1d5db;border-radius:8px;padding:1rem;margin:1rem 0;}</style>
</head>
<body>
<h1>USSD Performance Report</h1>
<div class="card"><strong>Generated:</strong> %s</div>
<div class="card">
<h2>Current run summary</h2>
<ul>
	<li>Total requests: %d</li>
	<li>Completed sessions: %d</li>
	<li>Failed sessions: %d</li>
	<li>Average latency (s): %.3f</li>
</ul>
</div>
%s
</body>
</html>`, r.GeneratedAt, r.Summary.TotalRequests, r.Summary.CompletedSessions, r.Summary.FailedSessions, r.Summary.AverageLatencySeconds, deltaHTML)
}

// CSV renders a compact row suitable for spreadsheets.
func (r Report) CSV() string {
	header := "generated_at,total_requests,completed_sessions,failed_sessions,average_latency_seconds,request_delta,completed_delta,failed_delta,latency_delta_seconds"
	row := fmt.Sprintf("%s,%d,%d,%d,%.6f", r.GeneratedAt, r.Summary.TotalRequests, r.Summary.CompletedSessions, r.Summary.FailedSessions, r.Summary.AverageLatencySeconds)
	if r.Delta != nil {
		row = fmt.Sprintf("%s,%d,%d,%d,%.6f", row, r.Delta.RequestDelta, r.Delta.CompletedDelta, r.Delta.FailedDelta, r.Delta.LatencyDeltaSec)
	} else {
		row = fmt.Sprintf("%s,%s,%s,%s,%s", row, "", "", "", "")
	}
	return header + "\n" + row + "\n"
}
