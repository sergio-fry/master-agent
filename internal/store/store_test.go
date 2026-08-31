package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTempStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "data", "master-agent.db")
	s, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s, path
}

func TestOpenCreatesDBAndTables(t *testing.T) {
	s, path := openTempStore(t)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.False(t, info.IsDir())

	for _, table := range []string{"projects", "tasks", "locks", "runs"} {
		var name string
		err := s.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		require.NoError(t, err, "table %s should exist", table)
		assert.Equal(t, table, name)
	}

	// Re-open is idempotent (migrations already applied).
	require.NoError(t, s.Close())
	s2, err := Open(path)
	require.NoError(t, err)
	defer s2.Close()
}

func TestProjectCRUD(t *testing.T) {
	s, _ := openTempStore(t)

	p := &Project{
		Name:       "my-app",
		Path:       "/home/dev/my-app",
		SSHHost:    "dev-box",
		SSHUser:    "dev",
		SSHPort:    2222,
		SSHKeyPath: "/secrets/projects/my-app/id_ed25519",
		Enabled:    true,
	}
	require.NoError(t, s.CreateProject(p))
	require.NotEmpty(t, p.ID)
	require.NotEmpty(t, p.CreatedAt)
	require.NotEmpty(t, p.UpdatedAt)

	got, err := s.GetProject(p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.Name, got.Name)
	assert.Equal(t, p.Path, got.Path)
	assert.Equal(t, p.SSHHost, got.SSHHost)
	assert.Equal(t, p.SSHUser, got.SSHUser)
	assert.Equal(t, 2222, got.SSHPort)
	assert.Equal(t, p.SSHKeyPath, got.SSHKeyPath)
	assert.True(t, got.Enabled)

	got.Enabled = false
	got.SSHPort = 22
	got.Path = "/home/dev/other"
	require.NoError(t, s.UpdateProject(got))

	updated, err := s.GetProject(p.ID)
	require.NoError(t, err)
	assert.False(t, updated.Enabled)
	assert.Equal(t, 22, updated.SSHPort)
	assert.Equal(t, "/home/dev/other", updated.Path)
	assert.GreaterOrEqual(t, updated.UpdatedAt, p.UpdatedAt)

	_, err = s.GetProject("missing")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProjectDefaultSSHPort(t *testing.T) {
	s, _ := openTempStore(t)
	p := &Project{
		Name: "x", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHKeyPath: "/k", Enabled: true,
	}
	require.NoError(t, s.CreateProject(p))
	got, err := s.GetProject(p.ID)
	require.NoError(t, err)
	assert.Equal(t, 22, got.SSHPort)
}

func TestTaskCRUD(t *testing.T) {
	s, _ := openTempStore(t)

	p := &Project{
		Name: "app", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHKeyPath: "/k", Enabled: true,
	}
	require.NoError(t, s.CreateProject(p))

	next := "2026-08-31T12:00:00Z"
	task := &Task{
		ProjectID:       p.ID,
		Name:            "drain",
		Prompt:          "do work",
		Command:         `cursor agent -p "{{prompt}}"`,
		IntervalSeconds: 1800,
		Enabled:         true,
		NextRunAt:       &next,
	}
	require.NoError(t, s.CreateTask(task))
	require.NotEmpty(t, task.ID)

	got, err := s.GetTask(task.ID)
	require.NoError(t, err)
	assert.Equal(t, p.ID, got.ProjectID)
	assert.Equal(t, "drain", got.Name)
	assert.Equal(t, "do work", got.Prompt)
	assert.Equal(t, `cursor agent -p "{{prompt}}"`, got.Command)
	assert.Equal(t, 1800, got.IntervalSeconds)
	assert.True(t, got.Enabled)
	require.NotNil(t, got.NextRunAt)
	assert.Equal(t, next, *got.NextRunAt)
	assert.Nil(t, got.LastRunAt)

	last := "2026-08-31T12:30:00Z"
	next2 := "2026-08-31T13:00:00Z"
	got.Enabled = false
	got.LastRunAt = &last
	got.NextRunAt = &next2
	got.IntervalSeconds = 3600
	require.NoError(t, s.UpdateTask(got))

	updated, err := s.GetTask(task.ID)
	require.NoError(t, err)
	assert.False(t, updated.Enabled)
	assert.Equal(t, 3600, updated.IntervalSeconds)
	require.NotNil(t, updated.LastRunAt)
	assert.Equal(t, last, *updated.LastRunAt)
	require.NotNil(t, updated.NextRunAt)
	assert.Equal(t, next2, *updated.NextRunAt)
}

func TestLockInsertDelete(t *testing.T) {
	s, _ := openTempStore(t)

	p := &Project{
		Name: "app", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHKeyPath: "/k", Enabled: true,
	}
	require.NoError(t, s.CreateProject(p))
	task := &Task{
		ProjectID: p.ID, Name: "t", Prompt: "p", Command: "echo",
		IntervalSeconds: 60, Enabled: true,
	}
	require.NoError(t, s.CreateTask(task))
	run := &Run{TaskID: task.ID, ProjectID: p.ID, Status: RunStatusRunning}
	require.NoError(t, s.InsertRun(run))

	pid := 4242
	lock := &Lock{ProjectID: p.ID, TaskID: task.ID, RunID: run.ID, PID: &pid}
	require.NoError(t, s.InsertLock(lock))
	require.NotEmpty(t, lock.AcquiredAt)

	got, err := s.GetLock(p.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, got.TaskID)
	assert.Equal(t, run.ID, got.RunID)
	require.NotNil(t, got.PID)
	assert.Equal(t, 4242, *got.PID)

	// Duplicate lock should fail (PK on project_id).
	err = s.InsertLock(&Lock{ProjectID: p.ID, TaskID: task.ID, RunID: run.ID})
	require.Error(t, err)

	require.NoError(t, s.DeleteLock(p.ID))
	_, err = s.GetLock(p.ID)
	assert.ErrorIs(t, err, ErrNotFound)

	err = s.DeleteLock(p.ID)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRunInsertUpdate(t *testing.T) {
	s, _ := openTempStore(t)

	p := &Project{
		Name: "app", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHKeyPath: "/k", Enabled: true,
	}
	require.NoError(t, s.CreateProject(p))
	task := &Task{
		ProjectID: p.ID, Name: "t", Prompt: "p", Command: "echo",
		IntervalSeconds: 60, Enabled: true,
	}
	require.NoError(t, s.CreateTask(task))

	run := &Run{TaskID: task.ID, ProjectID: p.ID}
	require.NoError(t, s.InsertRun(run))
	require.NotEmpty(t, run.ID)
	assert.Equal(t, RunStatusRunning, run.Status)

	got, err := s.GetRun(run.ID)
	require.NoError(t, err)
	assert.Equal(t, RunStatusRunning, got.Status)
	assert.Nil(t, got.FinishedAt)
	assert.Nil(t, got.ExitCode)

	finished := "2026-08-31T12:01:00Z"
	exitCode := 1
	errMsg := "remote exit 1"
	logPath := "/data/logs/run-1.log"
	got.FinishedAt = &finished
	got.ExitCode = &exitCode
	got.Status = RunStatusError
	got.ErrorMessage = &errMsg
	got.LogPath = &logPath
	require.NoError(t, s.UpdateRun(got))

	updated, err := s.GetRun(run.ID)
	require.NoError(t, err)
	assert.Equal(t, RunStatusError, updated.Status)
	require.NotNil(t, updated.FinishedAt)
	assert.Equal(t, finished, *updated.FinishedAt)
	require.NotNil(t, updated.ExitCode)
	assert.Equal(t, 1, *updated.ExitCode)
	require.NotNil(t, updated.ErrorMessage)
	assert.Equal(t, errMsg, *updated.ErrorMessage)
	require.NotNil(t, updated.LogPath)
	assert.Equal(t, logPath, *updated.LogPath)
}

func TestTaskRequiresExistingProject(t *testing.T) {
	s, _ := openTempStore(t)
	err := s.CreateTask(&Task{
		ProjectID: "missing", Name: "t", Prompt: "p", Command: "c",
		IntervalSeconds: 1, Enabled: true,
	})
	require.Error(t, err)
	assert.NotErrorIs(t, err, sql.ErrNoRows)
}
