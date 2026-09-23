package engine

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPWorkerProtocolCompletesAllAssignments(t *testing.T) {
	coordinator, err := NewHTTPCoordinator(testConfig(), 6, 2)
	if err != nil {
		t.Fatal(err)
	}
	coordinator.AuthToken = "test-token"
	server := httptest.NewServer(coordinator)
	defer server.Close()

	sender := &recordingSender{}
	for workerID := 0; workerID < 2; workerID++ {
		worker := Worker{ID: workerID, Count: 2, Config: testConfig(), Sender: sender}
		results, runErr := (HTTPWorker{BaseURL: server.URL, AuthToken: "test-token"}).Run(context.Background(), worker)
		if runErr != nil {
			t.Fatalf("worker %d failed: %v", workerID, runErr)
		}
		if len(results) != 3 {
			t.Fatalf("worker %d processed %d sessions, want 3", workerID, len(results))
		}
	}

	statusRequest, err := http.NewRequest(http.MethodGet, server.URL+statusPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	statusRequest.Header.Set("Authorization", "Bearer test-token")
	response, err := http.DefaultClient.Do(statusRequest)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var status CoordinatorStatus
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.RegisteredWorkers != 2 || status.SubmittedWorkers != 2 || status.TotalSessions != 6 || !status.Completed {
		t.Fatalf("unexpected coordinator status: %+v", status)
	}
}

type flakyCoordinatorTransport struct{ calls int }

func (t *flakyCoordinatorTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	t.calls++
	if t.calls < 3 {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("busy"))}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accepted":true}`))}, nil
}

func TestHTTPWorkerRetriesTemporaryCoordinatorFailure(t *testing.T) {
	transport := &flakyCoordinatorTransport{}
	worker := HTTPWorker{BaseURL: "http://coordinator", Client: &http.Client{Transport: transport}, MaxRetries: 2}
	if err := worker.call(context.Background(), worker.Client, http.MethodPost, registerPath, WorkerRegistration{}, nil); err != nil {
		t.Fatal(err)
	}
	if transport.calls != 3 {
		t.Fatalf("got %d requests, want 3", transport.calls)
	}
}
