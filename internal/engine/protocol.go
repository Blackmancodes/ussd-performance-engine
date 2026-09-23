package engine

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"performance-engine/internal/config"
)

const (
	registerPath  = "/v1/workers/register"
	workPath      = "/v1/work"
	resultPath    = "/v1/workers/results"
	heartbeatPath = "/v1/workers/heartbeat"
	statusPath    = "/v1/status"
)

type WorkerRegistration struct {
	WorkerID    int `json:"worker_id"`
	WorkerCount int `json:"worker_count"`
}

type WorkAssignment struct {
	WorkerID       int   `json:"worker_id"`
	SessionNumbers []int `json:"session_numbers"`
	Done           bool  `json:"done"`
}

type WorkerResult struct {
	WorkerID int             `json:"worker_id"`
	Results  []SessionResult `json:"results"`
}

type CoordinatorStatus struct {
	RegisteredWorkers int  `json:"registered_workers"`
	SubmittedWorkers  int  `json:"submitted_workers"`
	TotalWorkers      int  `json:"total_workers"`
	TotalSessions     int  `json:"total_sessions"`
	StaleWorkers      int  `json:"stale_workers"`
	Completed         bool `json:"completed"`
}

type HTTPCoordinator struct {
	Config           config.Config
	TotalSessions    int
	WorkerCount      int
	AuthToken        string
	HeartbeatTimeout time.Duration

	mu          sync.Mutex
	registered  map[int]bool
	assignments map[int]WorkAssignment
	results     map[int]WorkerResult
	lastSeen    map[int]time.Time
	done        chan struct{}
	doneOnce    sync.Once
}

func NewHTTPCoordinator(cfg config.Config, totalSessions, workerCount int) (*HTTPCoordinator, error) {
	if totalSessions < 0 {
		return nil, fmt.Errorf("total sessions must not be negative")
	}
	if workerCount <= 0 {
		return nil, fmt.Errorf("worker count must be positive")
	}
	return &HTTPCoordinator{
		Config:           cfg,
		TotalSessions:    totalSessions,
		WorkerCount:      workerCount,
		HeartbeatTimeout: 30 * time.Second,
		registered:       make(map[int]bool),
		assignments:      make(map[int]WorkAssignment),
		results:          make(map[int]WorkerResult),
		lastSeen:         make(map[int]time.Time),
		done:             make(chan struct{}),
	}, nil
}

func (c *HTTPCoordinator) Wait(ctx context.Context) error {
	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *HTTPCoordinator) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if !c.authorized(request) {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch {
	case request.Method == http.MethodPost && request.URL.Path == registerPath:
		c.register(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == workPath:
		c.assign(writer, request)
	case request.Method == http.MethodPost && request.URL.Path == resultPath:
		c.submit(writer, request)
	case request.Method == http.MethodPost && request.URL.Path == heartbeatPath:
		c.heartbeat(writer, request)
	case request.Method == http.MethodGet && request.URL.Path == statusPath:
		c.status(writer)
	default:
		http.NotFound(writer, request)
	}
}

func (c *HTTPCoordinator) register(writer http.ResponseWriter, request *http.Request) {
	var registration WorkerRegistration
	if !decodeJSON(writer, request, &registration) {
		return
	}
	if registration.WorkerCount != c.WorkerCount || registration.WorkerID < 0 || registration.WorkerID >= c.WorkerCount {
		http.Error(writer, "worker registration does not match coordinator", http.StatusBadRequest)
		return
	}
	c.mu.Lock()
	c.registered[registration.WorkerID] = true
	c.lastSeen[registration.WorkerID] = time.Now()
	c.mu.Unlock()
	writeJSON(writer, http.StatusOK, map[string]bool{"accepted": true})
}

func (c *HTTPCoordinator) heartbeat(writer http.ResponseWriter, request *http.Request) {
	var registration WorkerRegistration
	if !decodeJSON(writer, request, &registration) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.registered[registration.WorkerID] || registration.WorkerCount != c.WorkerCount {
		http.Error(writer, "worker is not registered", http.StatusConflict)
		return
	}
	c.lastSeen[registration.WorkerID] = time.Now()
	writeJSON(writer, http.StatusOK, map[string]bool{"accepted": true})
}

func (c *HTTPCoordinator) assign(writer http.ResponseWriter, request *http.Request) {
	workerID, err := strconv.Atoi(request.URL.Query().Get("worker_id"))
	if err != nil {
		http.Error(writer, "worker_id is required", http.StatusBadRequest)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.registered[workerID] {
		http.Error(writer, "worker is not registered", http.StatusConflict)
		return
	}
	assignment, exists := c.assignments[workerID]
	if !exists {
		assignment = WorkAssignment{WorkerID: workerID, SessionNumbers: WorkerSessions(c.TotalSessions, workerID, c.WorkerCount), Done: true}
		c.assignments[workerID] = assignment
	}
	writeJSON(writer, http.StatusOK, assignment)
}

func (c *HTTPCoordinator) submit(writer http.ResponseWriter, request *http.Request) {
	var result WorkerResult
	if !decodeJSON(writer, request, &result) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.registered[result.WorkerID] {
		http.Error(writer, "worker is not registered", http.StatusConflict)
		return
	}
	if assignment, exists := c.assignments[result.WorkerID]; !exists || !sameSessionNumbers(assignment.SessionNumbers, result.Results) {
		http.Error(writer, "submitted results do not match assignment", http.StatusBadRequest)
		return
	}
	c.results[result.WorkerID] = result
	if len(c.results) == c.WorkerCount {
		c.doneOnce.Do(func() { close(c.done) })
	}
	writeJSON(writer, http.StatusOK, map[string]bool{"accepted": true})
}

func (c *HTTPCoordinator) status(writer http.ResponseWriter) {
	c.mu.Lock()
	status := CoordinatorStatus{RegisteredWorkers: len(c.registered), SubmittedWorkers: len(c.results), TotalWorkers: c.WorkerCount, TotalSessions: c.TotalSessions, Completed: len(c.results) == c.WorkerCount}
	for workerID := range c.registered {
		if time.Since(c.lastSeen[workerID]) > c.HeartbeatTimeout {
			status.StaleWorkers++
		}
	}
	c.mu.Unlock()
	writeJSON(writer, http.StatusOK, status)
}

type HTTPWorker struct {
	BaseURL    string
	Client     *http.Client
	AuthToken  string
	MaxRetries int
	RetryDelay time.Duration
}

func (w HTTPWorker) Run(ctx context.Context, worker Worker) ([]SessionResult, error) {
	client := w.Client
	if client == nil {
		client = http.DefaultClient
	}
	if w.MaxRetries == 0 {
		w.MaxRetries = 3
	}
	if w.RetryDelay == 0 {
		w.RetryDelay = 100 * time.Millisecond
	}
	registration := WorkerRegistration{WorkerID: worker.ID, WorkerCount: worker.Count}
	if err := w.call(ctx, client, http.MethodPost, registerPath, registration, nil); err != nil {
		return nil, err
	}
	var assignment WorkAssignment
	workURL, err := url.Parse(strings.TrimRight(w.BaseURL, "/") + workPath)
	if err != nil {
		return nil, err
	}
	query := workURL.Query()
	query.Set("worker_id", strconv.Itoa(worker.ID))
	workURL.RawQuery = query.Encode()
	if err := w.callURL(ctx, client, http.MethodGet, workURL.String(), nil, &assignment); err != nil {
		return nil, err
	}
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()
	go w.heartbeatLoop(heartbeatCtx, client, registration)
	results, err := worker.RunSessions(ctx, assignment.SessionNumbers)
	if err != nil {
		return results, err
	}
	if err := w.call(ctx, client, http.MethodPost, resultPath, WorkerResult{WorkerID: worker.ID, Results: results}, nil); err != nil {
		return results, err
	}
	return results, nil
}

func (w HTTPWorker) heartbeatLoop(ctx context.Context, client *http.Client, registration WorkerRegistration) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = w.call(ctx, client, http.MethodPost, heartbeatPath, registration, nil)
		case <-ctx.Done():
			return
		}
	}
}

func (w HTTPWorker) call(ctx context.Context, client *http.Client, method, path string, body, response any) error {
	return w.callURL(ctx, client, method, strings.TrimRight(w.BaseURL, "/")+path, body, response)
}

func (w HTTPWorker) callURL(ctx context.Context, client *http.Client, method, endpoint string, body, response any) error {
	var requestData []byte
	if body == nil {
		requestData = []byte{}
	} else {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestData = data
	}
	var lastErr error
	for attempt := 0; attempt <= w.MaxRetries; attempt++ {
		request, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(string(requestData)))
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/json")
		if w.AuthToken != "" {
			request.Header.Set("Authorization", "Bearer "+w.AuthToken)
		}
		result, err := client.Do(request)
		if err == nil && result.StatusCode >= 200 && result.StatusCode < 300 {
			defer result.Body.Close()
			if response != nil && json.NewDecoder(result.Body).Decode(response) != nil {
				return fmt.Errorf("decode coordinator response")
			}
			return nil
		}
		if result != nil {
			result.Body.Close()
			lastErr = fmt.Errorf("coordinator returned HTTP %d", result.StatusCode)
			if result.StatusCode < 500 {
				return lastErr
			}
		} else {
			lastErr = err
		}
		if attempt < w.MaxRetries {
			select {
			case <-time.After(w.RetryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return lastErr
}

func (c *HTTPCoordinator) authorized(request *http.Request) bool {
	if c.AuthToken == "" {
		return true
	}
	expected := "Bearer " + c.AuthToken
	return subtle.ConstantTimeCompare([]byte(request.Header.Get("Authorization")), []byte(expected)) == 1
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, target any) bool {
	if err := json.NewDecoder(request.Body).Decode(target); err != nil {
		http.Error(writer, "invalid JSON", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func sameSessionNumbers(expected []int, results []SessionResult) bool {
	if len(expected) != len(results) {
		return false
	}
	for index, sessionNumber := range expected {
		if results[index].SessionID != fmt.Sprintf("ATUid_%d", sessionNumber) {
			return false
		}
	}
	return true
}
