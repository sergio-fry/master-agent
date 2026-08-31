package lock_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/lock"
	"master-agent/internal/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "master-agent.db")
	s, err := store.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func seedProjectTask(t *testing.T, s *store.Store) (projectID, taskID string) {
	t.Helper()
	p := &store.Project{
		Name: "app", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHKeyPath: "/k", Enabled: true,
	}
	require.NoError(t, s.CreateProject(p))
	task := &store.Task{
		ProjectID: p.ID, Name: "drain", Prompt: "do", Command: "echo ok",
		IntervalSeconds: 60, Enabled: true,
	}
	require.NoError(t, s.CreateTask(task))
	return p.ID, task.ID
}

func TestAcquireSkipAndRelease(t *testing.T) {
	s := openStore(t)
	projectID, taskID := seedProjectTask(t, s)
	mgr := lock.NewManager(s, lock.FakeProcessChecker{})

	pid := 1111
	run, err := mgr.Acquire(projectID, taskID, &pid)
	require.NoError(t, err)
	require.NotNil(t, run)
	assert.Equal(t, store.RunStatusRunning, run.Status)
	assert.Equal(t, projectID, run.ProjectID)
	assert.Equal(t, taskID, run.TaskID)

	gotLock, err := s.GetLock(projectID)
	require.NoError(t, err)
	assert.Equal(t, run.ID, gotLock.RunID)
	require.NotNil(t, gotLock.PID)
	assert.Equal(t, 1111, *gotLock.PID)

	// Second acquire must fail and must not create another run.
	_, err = mgr.Acquire(projectID, taskID, nil)
	require.ErrorIs(t, err, store.ErrLocked)

	runsBeforeRelease := mustListRunning(t, s)
	assert.Len(t, runsBeforeRelease, 1)

	exit := 0
	run.Status = store.RunStatusSuccess
	run.ExitCode = &exit
	require.NoError(t, mgr.Release(projectID, run))

	_, err = s.GetLock(projectID)
	assert.ErrorIs(t, err, store.ErrNotFound)

	finished, err := s.GetRun(run.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RunStatusSuccess, finished.Status)
	require.NotNil(t, finished.FinishedAt)
	require.NotNil(t, finished.ExitCode)
	assert.Equal(t, 0, *finished.ExitCode)

	// After release, acquire succeeds again.
	run2, err := mgr.Acquire(projectID, taskID, nil)
	require.NoError(t, err)
	assert.NotEqual(t, run.ID, run2.ID)

	errMsg := "remote exit 1"
	code := 1
	run2.Status = store.RunStatusError
	run2.ExitCode = &code
	run2.ErrorMessage = &errMsg
	require.NoError(t, mgr.Release(projectID, run2))

	_, err = s.GetLock(projectID)
	assert.ErrorIs(t, err, store.ErrNotFound)
	errored, err := s.GetRun(run2.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RunStatusError, errored.Status)
	require.NotNil(t, errored.ErrorMessage)
	assert.Equal(t, errMsg, *errored.ErrorMessage)
}

func TestRecoverStaleClearsDeadPID(t *testing.T) {
	s := openStore(t)
	projectID, taskID := seedProjectTask(t, s)

	alivePID := 2001
	deadPID := 2002
	checker := lock.FakeProcessChecker{AlivePIDs: map[int]bool{alivePID: true}}
	mgr := lock.NewManager(s, checker)

	aliveRun, err := mgr.Acquire(projectID, taskID, &alivePID)
	require.NoError(t, err)

	// Simulate a second project with a dead SSH client PID.
	p2 := &store.Project{
		Name: "other", Path: "/o", SSHHost: "h", SSHUser: "u",
		SSHKeyPath: "/k2", Enabled: true,
	}
	require.NoError(t, s.CreateProject(p2))
	t2 := &store.Task{
		ProjectID: p2.ID, Name: "t", Prompt: "p", Command: "c",
		IntervalSeconds: 30, Enabled: true,
	}
	require.NoError(t, s.CreateTask(t2))
	deadRun, err := mgr.Acquire(p2.ID, t2.ID, &deadPID)
	require.NoError(t, err)

	cleared, err := mgr.RecoverStale()
	require.NoError(t, err)
	assert.Equal(t, 1, cleared)

	// Alive lock remains.
	_, err = s.GetLock(projectID)
	require.NoError(t, err)
	stillRunning, err := s.GetRun(aliveRun.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RunStatusRunning, stillRunning.Status)

	// Dead lock cleared; run marked error with process lost.
	_, err = s.GetLock(p2.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
	staleRun, err := s.GetRun(deadRun.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RunStatusError, staleRun.Status)
	require.NotNil(t, staleRun.FinishedAt)
	require.NotNil(t, staleRun.ErrorMessage)
	assert.Equal(t, store.ProcessLostMessage, *staleRun.ErrorMessage)

	// Project with dead lock can be acquired again.
	_, err = mgr.Acquire(p2.ID, t2.ID, &alivePID)
	require.NoError(t, err)
}

func TestRecoverStaleSkipsNilPID(t *testing.T) {
	s := openStore(t)
	projectID, taskID := seedProjectTask(t, s)
	mgr := lock.NewManager(s, lock.FakeProcessChecker{})

	run, err := mgr.Acquire(projectID, taskID, nil)
	require.NoError(t, err)

	cleared, err := mgr.RecoverStale()
	require.NoError(t, err)
	assert.Equal(t, 0, cleared)

	_, err = s.GetLock(projectID)
	require.NoError(t, err)
	got, err := s.GetRun(run.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RunStatusRunning, got.Status)
}

func mustListRunning(t *testing.T, s *store.Store) []store.Run {
	t.Helper()
	// Acquire creates at most one running run in these tests; use Get via lock.
	locks, err := s.ListLocks()
	require.NoError(t, err)
	var runs []store.Run
	for _, l := range locks {
		r, err := s.GetRun(l.RunID)
		require.NoError(t, err)
		if r.Status == store.RunStatusRunning {
			runs = append(runs, *r)
		}
	}
	return runs
}
