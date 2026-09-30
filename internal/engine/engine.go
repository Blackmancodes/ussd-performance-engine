package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"performance-engine/internal/config"
)

type Request struct {
	SessionID string `json:"sessionId"`
	MSISDN    string `json:"msisdn"`
	Network   string `json:"network"`
	USSD      string `json:"ussdString"`
	Input     string `json:"input"`
	Stage     string `json:"stage"`
}

type Response struct {
	SessionID string `json:"sessionId"`
	Response  string `json:"response"`
	Continue  bool   `json:"continue"`
}

type Sender interface {
	Send(context.Context, Request) (Response, error)
}

type HTTPSender struct {
	URL        string
	APIKey     string
	APIKeyName string
	Client     *http.Client
}

func (s HTTPSender) Send(ctx context.Context, request Request) (Response, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return Response{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL, strings.NewReader(string(body)))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.APIKey != "" {
		req.Header.Set(s.APIKeyName, s.APIKey)
	}
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Response{}, fmt.Errorf("sender returned HTTP %d", res.StatusCode)
	}
	var response Response
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return Response{}, err
	}
	return response, nil
}

type SessionResult struct {
	SessionID string
	MSISDN    string
	Network   string
	Journey   string
	Requests  []Request
	State     string
}

type Runner struct {
	Config  config.Config
	Sender  Sender
	Random  *rand.Rand
	Metrics *Metrics
}

func (r *Runner) Run(ctx context.Context, sessionNumber int) (SessionResult, error) {
	random := r.Random
	if random == nil {
		random = rand.New(rand.NewSource(r.Config.Seed + int64(sessionNumber)))
	}
	network, msisdn := pickMSISDN(r.Config, random, sessionNumber)
	journey := pickJourney(r.Config.Journeys, random.Float64())
	result := SessionResult{SessionID: fmt.Sprintf("ATUid_%d", sessionNumber), MSISDN: msisdn, Network: network, Journey: journey.Name, State: "IDLE"}
	for stepIndex, step := range journey.Steps {
		stage := "continue"
		if stepIndex == 0 {
			stage = "begin"
		} else if stepIndex == len(journey.Steps)-1 {
			stage = "end"
		}
		request := Request{SessionID: result.SessionID, MSISDN: msisdn, Network: network, USSD: step.USSD, Input: step.Input, Stage: stage}
		started := time.Now()
		response, err := r.Sender.Send(ctx, request)
		result.Requests = append(result.Requests, request)
		if err != nil {
			result.State = "FAILED"
			if r.Metrics != nil {
				r.Metrics.ObserveRequest(network, journey.Name, stage, "error", time.Since(started))
				r.Metrics.ObserveSession(result.State, journey.Name)
				r.Metrics.ObserveFailure(result, err)
			}
			return result, err
		}
		if r.Metrics != nil {
			status := "success"
			if !response.Continue || strings.HasPrefix(response.Response, "END") {
				status = "complete"
			}
			r.Metrics.ObserveRequest(network, journey.Name, stage, status, time.Since(started))
		}
		if !response.Continue || strings.HasPrefix(response.Response, "END") {
			result.State = "COMPLETED"
			if r.Metrics != nil {
				r.Metrics.ObserveSession(result.State, journey.Name)
			}
			return result, nil
		}
	}
	result.State = "COMPLETED"
	if r.Metrics != nil {
		r.Metrics.ObserveSession(result.State, journey.Name)
	}
	return result, nil
}

type Metrics struct {
	Requests *prometheus.CounterVec
	Latency  *prometheus.HistogramVec
	Sessions *prometheus.CounterVec
	Failures *prometheus.CounterVec
	Registry *prometheus.Registry
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	metrics := &Metrics{
		Requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ussd_requests_total", Help: "Total sender requests emitted by the engine."}, []string{"network", "journey", "stage", "status"}),
		Latency:  prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "ussd_request_duration_seconds", Help: "Sender request duration in seconds.", Buckets: prometheus.DefBuckets}, []string{"network", "journey", "stage"}),
		Sessions: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ussd_sessions_total", Help: "Total sessions by terminal state and journey."}, []string{"state", "journey"}),
		Failures: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ussd_failure_events_total", Help: "Total session failures by bounded category."}, []string{"network", "journey", "failure_type"}),
		Registry: registry,
	}
	registry.MustRegister(metrics.Requests, metrics.Latency, metrics.Sessions, metrics.Failures)
	return metrics
}

func (m *Metrics) ObserveRequest(network, journey, stage, status string, duration time.Duration) {
	m.Requests.WithLabelValues(network, journey, stage, status).Inc()
	m.Latency.WithLabelValues(network, journey, stage).Observe(duration.Seconds())
}

func (m *Metrics) ObserveSession(state, journey string) {
	m.Sessions.WithLabelValues(state, journey).Inc()
}

func (m *Metrics) ObserveFailure(result SessionResult, failure error) {
	failureType := "unknown"
	switch {
	case errors.Is(failure, context.DeadlineExceeded):
		failureType = "timeout"
	case errors.Is(failure, context.Canceled):
		failureType = "canceled"
	case failure != nil:
		failureType = "sender_error"
	}
	m.Failures.WithLabelValues(result.Network, result.Journey, failureType).Inc()
}

func pickJourney(journeys []config.Journey, sample float64) config.Journey {
	for _, journey := range journeys {
		sample -= journey.Weight
		if sample <= 0 {
			return journey
		}
	}
	return journeys[len(journeys)-1]
}

func pickMSISDN(cfg config.Config, random *rand.Rand, sessionNumber int) (string, string) {
	total := 0.0
	for _, mno := range cfg.MNOPools {
		total += mno.Weight
	}
	sample := random.Float64() * total
	for network, mno := range cfg.MNOPools {
		sample -= mno.Weight
		if sample <= 0 && len(mno.Prefixes) > 0 {
			prefix := mno.Prefixes[random.Intn(len(mno.Prefixes))]
			suffix := 1000000 + (sessionNumber-1)%9000000
			return network, "234" + prefix[1:] + strconv.Itoa(suffix)
		}
	}
	return "UNKNOWN", "2348000000000"
}
