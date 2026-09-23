package report

import "testing"

func TestParsePrometheusMetrics(t *testing.T) {
	metricsText := `
# HELP ussd_requests_total Total sender requests emitted by the engine.
# TYPE ussd_requests_total counter
ussd_requests_total{journey="check_balance",network="MTN",stage="begin",status="success"} 10
ussd_requests_total{journey="check_balance",network="MTN",stage="end",status="complete"} 5
# HELP ussd_sessions_total Total sessions by terminal state and journey.
# TYPE ussd_sessions_total counter
ussd_sessions_total{journey="check_balance",state="COMPLETED"} 8
ussd_sessions_total{journey="check_balance",state="FAILED"} 2
# HELP ussd_request_duration_seconds Sender request duration in seconds.
# TYPE ussd_request_duration_seconds histogram
ussd_request_duration_seconds_sum{journey="check_balance",network="MTN",stage="begin"} 12.5
ussd_request_duration_seconds_count{journey="check_balance",network="MTN",stage="begin"} 10
`

	summary, err := ParseMetricsText(metricsText)
	if err != nil {
		t.Fatalf("ParseMetricsText() error = %v", err)
	}
	if summary.TotalRequests != 15 {
		t.Fatalf("TotalRequests = %v, want 15", summary.TotalRequests)
	}
	if summary.CompletedSessions != 8 {
		t.Fatalf("CompletedSessions = %v, want 8", summary.CompletedSessions)
	}
	if summary.FailedSessions != 2 {
		t.Fatalf("FailedSessions = %v, want 2", summary.FailedSessions)
	}
	if summary.AverageLatencySeconds != 1.25 {
		t.Fatalf("AverageLatencySeconds = %v, want 1.25", summary.AverageLatencySeconds)
	}
}

func TestCompareSummaries(t *testing.T) {
	current := Summary{TotalRequests: 120, CompletedSessions: 96, FailedSessions: 4, AverageLatencySeconds: 1.5}
	baseline := Summary{TotalRequests: 100, CompletedSessions: 88, FailedSessions: 2, AverageLatencySeconds: 1.2}

	delta := CompareSummaries(current, baseline)
	if delta.RequestDelta != 20 {
		t.Fatalf("RequestDelta = %v, want 20", delta.RequestDelta)
	}
	if delta.CompletedDelta != 8 {
		t.Fatalf("CompletedDelta = %v, want 8", delta.CompletedDelta)
	}
	if delta.FailedDelta != 2 {
		t.Fatalf("FailedDelta = %v, want 2", delta.FailedDelta)
	}
	if delta.LatencyDeltaSec < 0.29 || delta.LatencyDeltaSec > 0.31 {
		t.Fatalf("LatencyDeltaSec = %v, want ~0.3", delta.LatencyDeltaSec)
	}
}

func TestParsePrometheusAPIResponse(t *testing.T) {
	payload := `{
		"status":"success",
		"data":{"resultType":"vector","result":[
			{"metric":{"__name__":"ussd_requests_total","journey":"check_balance","network":"MTN","stage":"begin","status":"success"},"value":[1710000000,"10"]},
			{"metric":{"__name__":"ussd_sessions_total","journey":"check_balance","state":"COMPLETED"},"value":[1710000000,"8"]},
			{"metric":{"__name__":"ussd_sessions_total","journey":"check_balance","state":"FAILED"},"value":[1710000000,"2"]},
			{"metric":{"__name__":"ussd_request_duration_seconds_sum","journey":"check_balance","network":"MTN","stage":"begin"},"value":[1710000000,"12.5"]},
			{"metric":{"__name__":"ussd_request_duration_seconds_count","journey":"check_balance","network":"MTN","stage":"begin"},"value":[1710000000,"10"]}
		]}}
	`

	summary, err := ParsePrometheusAPIResponse(payload)
	if err != nil {
		t.Fatalf("ParsePrometheusAPIResponse() error = %v", err)
	}
	if summary.TotalRequests != 10 {
		t.Fatalf("TotalRequests = %v, want 10", summary.TotalRequests)
	}
	if summary.CompletedSessions != 8 {
		t.Fatalf("CompletedSessions = %v, want 8", summary.CompletedSessions)
	}
	if summary.FailedSessions != 2 {
		t.Fatalf("FailedSessions = %v, want 2", summary.FailedSessions)
	}
	if summary.AverageLatencySeconds != 1.25 {
		t.Fatalf("AverageLatencySeconds = %v, want 1.25", summary.AverageLatencySeconds)
	}
}
