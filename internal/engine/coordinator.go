package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"performance-engine/internal/config"
)

type Worker struct {
	ID      int
	Count   int
	Config  config.Config
	Sender  Sender
	Metrics *Metrics
}

func (w Worker) Run(ctx context.Context, totalSessions int) ([]SessionResult, error) {
	runner := Runner{Config: w.Config, Sender: w.Sender, Metrics: w.Metrics}
	return w.runSessions(ctx, runner, WorkerSessions(totalSessions, w.ID, w.Count))
}

func (w Worker) RunSessions(ctx context.Context, sessionNumbers []int) ([]SessionResult, error) {
	runner := Runner{Config: w.Config, Sender: w.Sender, Metrics: w.Metrics}
	return w.runSessions(ctx, runner, sessionNumbers)
}

func (w Worker) runSessions(ctx context.Context, runner Runner, sessionNumbers []int) ([]SessionResult, error) {
	results := make([]SessionResult, 0)
	var firstErr error
	for _, sessionNumber := range sessionNumbers {
		result, err := runner.Run(ctx, sessionNumber)
		results = append(results, result)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("worker %d session %d: %w", w.ID, sessionNumber, err)
			}
		}
	}
	return results, firstErr
}

type Coordinator struct {
	Config  config.Config
	Sender  Sender
	Metrics *Metrics
}

func (c Coordinator) Run(ctx context.Context, totalSessions, workerCount int) ([]SessionResult, error) {
	if workerCount <= 0 {
		return nil, fmt.Errorf("worker count must be positive")
	}
	results := make([]SessionResult, 0, totalSessions)
	resultCh := make(chan []SessionResult, workerCount)
	errCh := make(chan error, workerCount)
	var waitGroup sync.WaitGroup
	for workerID := 0; workerID < workerCount; workerID++ {
		workerConfig := c.Config
		workerConfig.Scale.WorkerID = workerID
		workerConfig.Scale.WorkerCount = workerCount
		worker := Worker{ID: workerID, Count: workerCount, Config: workerConfig, Sender: c.Sender, Metrics: c.Metrics}
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			workerResults, err := worker.Run(ctx, totalSessions)
			resultCh <- workerResults
			if err != nil {
				errCh <- err
			}
		}()
	}
	waitGroup.Wait()
	close(resultCh)
	close(errCh)
	for workerResults := range resultCh {
		results = append(results, workerResults...)
	}
	sort.Slice(results, func(first, second int) bool {
		return results[first].SessionID < results[second].SessionID
	})
	if len(errCh) > 0 {
		return results, <-errCh
	}
	return results, nil
}
