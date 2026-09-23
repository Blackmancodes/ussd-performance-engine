package engine

import (
	"context"
	"sync"
	"testing"

	"performance-engine/internal/config"
)

type recordingSender struct {
	mu       sync.Mutex
	requests []Request
}

func (s *recordingSender) Send(_ context.Context, request Request) (Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, request)
	return Response{Response: "CON next", Continue: true}, nil
}

func TestRunnerEmitsConfiguredJourneyStages(t *testing.T) {
	sender := &recordingSender{}
	runner := Runner{Config: testConfig(), Sender: sender}
	result, err := runner.Run(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != "COMPLETED" || len(sender.requests) != 3 {
		t.Fatalf("unexpected result: state=%s requests=%d", result.State, len(sender.requests))
	}
	for index, expected := range []string{"begin", "continue", "end"} {
		if sender.requests[index].Stage != expected {
			t.Errorf("request %d stage = %q, want %q", index, sender.requests[index].Stage, expected)
		}
	}
}

func TestSeedProducesSameSequence(t *testing.T) {
	first := &recordingSender{}
	second := &recordingSender{}
	firstRunner := Runner{Config: testConfig(), Sender: first}
	secondRunner := Runner{Config: testConfig(), Sender: second}
	for session := 1; session <= 10; session++ {
		_, _ = firstRunner.Run(context.Background(), session)
		_, _ = secondRunner.Run(context.Background(), session)
	}
	for index := range first.requests {
		if first.requests[index] != second.requests[index] {
			t.Fatalf("request %d differs for identical seed", index)
		}
	}
}

func testConfig() config.Config {
	return config.Config{
		Seed:     42,
		MNOPools: map[string]config.MNO{"MTN": {Prefixes: []string{"0803"}, Weight: 1}},
		Journeys: []config.Journey{{Name: "balance", Weight: 1, Steps: []config.Step{{USSD: "*737#"}, {Input: "1"}, {Input: "1"}}}},
	}
}

func TestWorkerSessionsPartitionTheLoad(t *testing.T) {
	const totalSessions = 11
	const workerCount = 3
	seen := make(map[int]int)
	for workerID := 0; workerID < workerCount; workerID++ {
		for _, session := range WorkerSessions(totalSessions, workerID, workerCount) {
			seen[session]++
		}
	}
	if len(seen) != totalSessions {
		t.Fatalf("assigned %d sessions, want %d", len(seen), totalSessions)
	}
	for session := 1; session <= totalSessions; session++ {
		if seen[session] != 1 {
			t.Errorf("session %d assigned %d times, want once", session, seen[session])
		}
	}
}

func TestWorkerSessionsRejectInvalidWorker(t *testing.T) {
	if sessions := WorkerSessions(10, 3, 3); sessions != nil {
		t.Fatalf("invalid worker received sessions: %v", sessions)
	}
}

func TestCoordinatorRunsEachSessionOnce(t *testing.T) {
	sender := &recordingSender{}
	coordinator := Coordinator{Config: testConfig(), Sender: sender}
	results, err := coordinator.Run(context.Background(), 11, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 11 {
		t.Fatalf("got %d results, want 11", len(results))
	}
	for sessionNumber, result := range results {
		if result.SessionID == "" {
			t.Errorf("result %d has empty session ID", sessionNumber)
		}
		if len(result.Requests) == 0 {
			t.Errorf("result %d has no requests", sessionNumber)
		}
	}
	if len(sender.requests) != 11*3 {
		t.Fatalf("got %d requests, want 33", len(sender.requests))
	}
	seen := make(map[string]int)
	for _, request := range sender.requests {
		seen[request.SessionID]++
	}
	if len(seen) != 11 {
		t.Fatalf("got %d unique sessions, want 11", len(seen))
	}
	for sessionID, requestCount := range seen {
		if requestCount != 3 {
			t.Errorf("session %s emitted %d requests, want 3", sessionID, requestCount)
		}
	}
	msisdns := make(map[string]string)
	for _, request := range sender.requests {
		if previousSession, exists := msisdns[request.MSISDN]; exists && previousSession != request.SessionID {
			t.Errorf("MSISDN %s was assigned to sessions %s and %s", request.MSISDN, previousSession, request.SessionID)
		}
		msisdns[request.MSISDN] = request.SessionID
	}
}
