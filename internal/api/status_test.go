package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/store"
)

func TestStatusNoLocks(t *testing.T) {
	st := openTempStore(t)
	dbPath := t.TempDir() + "/master-agent.db"
	ts := httptestServerWithConfig(t, Config{Store: st, DBPath: dbPath})

	resp, err := http.Get(ts.URL + "/api/v1/status")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body statusResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.True(t, body.OK)
	assert.True(t, body.DBOK)
	assert.Equal(t, dbPath, body.DBPath)
	assert.False(t, body.LockActive)
	assert.Empty(t, body.Locks)
}

func TestStatusWithActiveLock(t *testing.T) {
	st := openTempStore(t)

	p := &store.Project{
		Name: "app", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHPrivateKey: store.TestSSHPrivateKey, Enabled: true,
	}
	require.NoError(t, st.CreateProject(p))
	task := &store.Task{
		ProjectID: p.ID, Name: "deploy", Prompt: "p", Command: "echo",
		IntervalSeconds: 60, Enabled: true,
	}
	require.NoError(t, st.CreateTask(task))
	run := &store.Run{TaskID: task.ID, ProjectID: p.ID, Status: store.RunStatusRunning}
	require.NoError(t, st.InsertRun(run))
	pid := 9999
	require.NoError(t, st.InsertLock(&store.Lock{
		ProjectID: p.ID, TaskID: task.ID, RunID: run.ID, PID: &pid,
	}))

	ts := httptestServerWithConfig(t, Config{Store: st})

	resp, err := http.Get(ts.URL + "/api/v1/status")
	require.NoError(t, err)
	defer resp.Body.Close()

	var body statusResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.True(t, body.DBOK)
	assert.True(t, body.LockActive)
	require.Len(t, body.Locks, 1)
	assert.Equal(t, p.ID, body.Locks[0].ProjectID)
	assert.Equal(t, "app", body.Locks[0].ProjectName)
	assert.Equal(t, task.ID, body.Locks[0].TaskID)
	assert.Equal(t, "deploy", body.Locks[0].TaskName)
	assert.Equal(t, run.ID, body.Locks[0].RunID)
	require.NotNil(t, body.Locks[0].PID)
	assert.Equal(t, 9999, *body.Locks[0].PID)
}

func httptestServerWithConfig(t *testing.T, cfg Config) *httptest.Server {
	t.Helper()
	srv := New(cfg)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}
