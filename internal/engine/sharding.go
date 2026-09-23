package engine

// WorkerSessions returns the one-based session numbers assigned to a worker.
// Round-robin assignment keeps every worker's work disjoint and balanced.
func WorkerSessions(totalSessions, workerID, workerCount int) []int {
	if workerCount == 0 {
		workerCount = 1
	}
	if totalSessions <= 0 || workerCount < 0 || workerID < 0 || workerID >= workerCount {
		return nil
	}
	sessions := make([]int, 0, (totalSessions+workerCount-workerID-1)/workerCount)
	for session := workerID + 1; session <= totalSessions; session += workerCount {
		sessions = append(sessions, session)
	}
	return sessions
}
