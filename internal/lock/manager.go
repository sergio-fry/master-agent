package lock

import (
	"fmt"

	"master-agent/internal/store"
)

// Manager enforces one active run per project and recovers stale locks.
type Manager struct {
	Store   *store.Store
	Checker ProcessChecker
}

// NewManager returns a Manager. If checker is nil, OSProcessChecker is used.
func NewManager(s *store.Store, checker ProcessChecker) *Manager {
	if checker == nil {
		checker = OSProcessChecker{}
	}
	return &Manager{Store: s, Checker: checker}
}

// Acquire creates a running run and project lock, or returns store.ErrLocked.
func (m *Manager) Acquire(projectID, taskID string, pid *int) (*store.Run, error) {
	run, _, err := m.Store.AcquireRunLock(projectID, taskID, pid)
	return run, err
}

// Release updates the run and removes the project lock (success or error).
func (m *Manager) Release(projectID string, run *store.Run) error {
	return m.Store.ReleaseRunLock(projectID, run)
}

// RecoverStale clears locks whose local SSH client PID is dead and marks runs as error.
// Locks without a PID are left unchanged.
func (m *Manager) RecoverStale() (int, error) {
	locks, err := m.Store.ListLocks()
	if err != nil {
		return 0, err
	}

	cleared := 0
	for _, l := range locks {
		if l.PID == nil {
			continue
		}
		if m.Checker.Alive(*l.PID) {
			continue
		}
		if err := m.Store.ClearStaleLock(l.ProjectID, store.ProcessLostMessage); err != nil {
			return cleared, fmt.Errorf("clear stale lock %s: %w", l.ProjectID, err)
		}
		cleared++
	}
	return cleared, nil
}
