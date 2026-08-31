package scheduler_test

import (
	"bytes"
	"context"
	"log"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/lock"
	"master-agent/internal/runner"
	"master-agent/internal/scheduler"
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

func seed(t *testing.T, s *store.Store, nextRun *string, interval int) (store.Project, store.Task) {
	t.Helper()
	p := store.Project{
		Name: "app", Path: "/work", SSHHost: "host", SSHUser: "user",
		SSHKeyPath: "/key", Enabled: true,
	}
	require.NoError(t, s.CreateProject(&p))
	task := store.Task{
		ProjectID: p.ID, Name: "drain", Prompt: "do work",
		Command: "echo {{prompt}}", IntervalSeconds: interval, Enabled: true,
		NextRunAt: nextRun,
	}
	require.NoError(t, s.CreateTask(&task))
	return p, task
}

func newDaemon(s *store.Store, fake *runner.FakeRunner, now time.Time) *scheduler.Daemon {
	var buf bytes.Buffer
	return &scheduler.Daemon{
		Store:  s,
		Locks:  lock.NewManager(s, lock.FakeProcessChecker{}),
		Runner: fake,
		Config: scheduler.Config{TickInterval: time.Second},
		Now:    func() time.Time { return now },
		Logger: log.New(&buf, "", 0),
	}
}

func TestTickStartsDueTask(t *testing.T) {
	s := openStore(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute).Format(time.RFC3339Nano)
	_, task := seed(t, s, &past, 60)

	fake := &runner.FakeRunner{Result: runner.Result{ExitCode: 0, Stdout: "ok"}}
	d := newDaemon(s, fake, now)

	require.NoError(t, d.Tick(context.Background()))

	require.Len(t, fake.Calls, 1)
	assert.Contains(t, fake.Calls[0].Command, "do work")

	updated, err := s.GetTask(task.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.LastRunAt)
	require.NotNil(t, updated.NextRunAt)
	assert.Equal(t, now.Format(time.RFC3339Nano), *updated.LastRunAt)
	assert.Equal(t, now.Add(60*time.Second).Format(time.RFC3339Nano), *updated.NextRunAt)

	_, err = s.GetLock(task.ProjectID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestTickStartsWhenNextRunAtNull(t *testing.T) {
	s := openStore(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	_, _ = seed(t, s, nil, 30)

	fake := &runner.FakeRunner{Result: runner.Result{ExitCode: 0}}
	d := newDaemon(s, fake, now)
	require.NoError(t, d.Tick(context.Background()))
	assert.Len(t, fake.Calls, 1)
}

func TestTickIdleWhenNoDueTasks(t *testing.T) {
	s := openStore(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour).Format(time.RFC3339Nano)
	_, _ = seed(t, s, &future, 60)

	fake := &runner.FakeRunner{Result: runner.Result{ExitCode: 0}}
	d := newDaemon(s, fake, now)
	require.NoError(t, d.Tick(context.Background()))
	assert.Empty(t, fake.Calls)
}

func TestTickSkipsDisabledTaskAndProject(t *testing.T) {
	s := openStore(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)

	p := store.Project{
		Name: "off", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHKeyPath: "/k", Enabled: false,
	}
	require.NoError(t, s.CreateProject(&p))
	past := now.Add(-time.Second).Format(time.RFC3339Nano)
	require.NoError(t, s.CreateTask(&store.Task{
		ProjectID: p.ID, Name: "t", Prompt: "p", Command: "echo x",
		IntervalSeconds: 10, Enabled: true, NextRunAt: &past,
	}))

	p2 := store.Project{
		Name: "on", Path: "/p2", SSHHost: "h", SSHUser: "u",
		SSHKeyPath: "/k", Enabled: true,
	}
	require.NoError(t, s.CreateProject(&p2))
	require.NoError(t, s.CreateTask(&store.Task{
		ProjectID: p2.ID, Name: "disabled", Prompt: "p", Command: "echo y",
		IntervalSeconds: 10, Enabled: false, NextRunAt: &past,
	}))

	fake := &runner.FakeRunner{Result: runner.Result{ExitCode: 0}}
	d := newDaemon(s, fake, now)
	require.NoError(t, d.Tick(context.Background()))
	assert.Empty(t, fake.Calls)
}

func TestTickSkipsWhenGlobalRunActive(t *testing.T) {
	s := openStore(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute).Format(time.RFC3339Nano)
	p, task := seed(t, s, &past, 60)

	mgr := lock.NewManager(s, lock.FakeProcessChecker{})
	_, err := mgr.Acquire(p.ID, task.ID, nil)
	require.NoError(t, err)

	p2 := store.Project{
		Name: "other", Path: "/o", SSHHost: "h", SSHUser: "u",
		SSHKeyPath: "/k2", Enabled: true,
	}
	require.NoError(t, s.CreateProject(&p2))
	require.NoError(t, s.CreateTask(&store.Task{
		ProjectID: p2.ID, Name: "also-due", Prompt: "p", Command: "echo z",
		IntervalSeconds: 30, Enabled: true, NextRunAt: &past,
	}))

	fake := &runner.FakeRunner{Result: runner.Result{ExitCode: 0}}
	d := newDaemon(s, fake, now)
	require.NoError(t, d.Tick(context.Background()))
	assert.Empty(t, fake.Calls, "must not start a second global run")
}

func TestTickFailureSchedulesNextWithoutImmediateRetry(t *testing.T) {
	s := openStore(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute).Format(time.RFC3339Nano)
	_, task := seed(t, s, &past, 120)

	fake := &runner.FakeRunner{
		Result: runner.Result{ExitCode: 1, Stderr: "boom"},
	}
	d := newDaemon(s, fake, now)
	require.NoError(t, d.Tick(context.Background()))

	require.Len(t, fake.Calls, 1)

	updated, err := s.GetTask(task.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.NextRunAt)
	assert.Equal(t, now.Add(120*time.Second).Format(time.RFC3339Nano), *updated.NextRunAt)
	require.NotNil(t, updated.LastRunAt)

	_, err = s.GetLock(task.ProjectID)
	assert.ErrorIs(t, err, store.ErrNotFound)

	fake.Calls = nil
	require.NoError(t, d.Tick(context.Background()))
	assert.Empty(t, fake.Calls)

	run, err := s.LatestRunForTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RunStatusError, run.Status)
	require.NotNil(t, run.ExitCode)
	assert.Equal(t, 1, *run.ExitCode)
}

func TestTickSuccessRunStatus(t *testing.T) {
	s := openStore(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	_, task := seed(t, s, nil, 45)

	fake := &runner.FakeRunner{Result: runner.Result{ExitCode: 0}}
	d := newDaemon(s, fake, now)
	require.NoError(t, d.Tick(context.Background()))

	run, err := s.LatestRunForTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, store.RunStatusSuccess, run.Status)
	require.NotNil(t, run.ExitCode)
	assert.Equal(t, 0, *run.ExitCode)
	require.NotNil(t, run.FinishedAt)
}
