package wire

import (
	"pa/internal/jobs"
	"sync"
)

// JobsRuntimePhase is the explicit scheduled-jobs runtime lifecycle (EP-042).
type JobsRuntimePhase int

const (
	// JobsRuntimeInitializing: DB open / manager construction in progress.
	JobsRuntimeInitializing JobsRuntimePhase = iota
	// JobsRuntimeReady: manager available for /jobs and create_scheduled_job.
	JobsRuntimeReady
	// JobsRuntimeFailed: initialization failed; management surfaces are gated.
	JobsRuntimeFailed
)

// JobsRuntimeState holds the live jobs manager and its initialization phase.
type JobsRuntimeState struct {
	mu      sync.RWMutex
	manager *jobs.Manager
	phase   JobsRuntimePhase
}

// NewJobsRuntimeState returns state in JobsRuntimeInitializing.
func NewJobsRuntimeState() *JobsRuntimeState {
	return &JobsRuntimeState{phase: JobsRuntimeInitializing}
}

// SetReady marks the runtime ready with a constructed manager.
func (s *JobsRuntimeState) SetReady(manager *jobs.Manager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manager = manager
	s.phase = JobsRuntimeReady
}

// SetFailed marks initialization as failed (management gated, readiness not OK).
func (s *JobsRuntimeState) SetFailed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manager = nil
	s.phase = JobsRuntimeFailed
}

// Snapshot returns the manager (if any) and the current phase.
func (s *JobsRuntimeState) Snapshot() (*jobs.Manager, JobsRuntimePhase) {
	if s == nil {
		return nil, JobsRuntimeInitializing
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.manager, s.phase
}

// SnapshotLegacy maps phase to (manager, ready, initFailed) for create_scheduled_job runtime lookup.
func (s *JobsRuntimeState) SnapshotLegacy() (*jobs.Manager, bool, bool) {
	mgr, phase := s.Snapshot()
	switch phase {
	case JobsRuntimeReady:
		return mgr, true, false
	case JobsRuntimeFailed:
		// ready=true preserves create_scheduled_job lookup order (failed checked after !ready).
		return mgr, true, true
	default:
		return nil, false, false
	}
}
