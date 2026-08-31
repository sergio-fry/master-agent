package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/store"
)

func insertSampleRuns(t *testing.T, st *store.Store, projectID string, taskIDs []string) []*store.Run {
	t.Helper()
	runs := []*store.Run{
		{
			TaskID:    taskIDs[0],
			ProjectID: projectID,
			StartedAt: "2026-08-31T12:00:00Z",
			Status:    store.RunStatusSuccess,
		},
		{
			TaskID:    taskIDs[1],
			ProjectID: projectID,
			StartedAt: "2026-08-31T13:00:00Z",
			Status:    store.RunStatusError,
		},
	}
	for _, r := range runs {
		require.NoError(t, st.InsertRun(r))
	}
	return runs
}

func TestRunsListFilters(t *testing.T) {
	ts, st := startTestServerWithStore(t, "")
	p := createTestProject(t, ts.URL)

	t1 := &store.Task{
		ProjectID:       p.ID,
		Name:            "drain",
		Prompt:          "p",
		Command:         "echo",
		IntervalSeconds: 60,
		Enabled:         true,
	}
	t2 := &store.Task{
		ProjectID:       p.ID,
		Name:            "audit",
		Prompt:          "p",
		Command:         "echo",
		IntervalSeconds: 60,
		Enabled:         true,
	}
	require.NoError(t, st.CreateTask(t1))
	require.NoError(t, st.CreateTask(t2))

	runs := insertSampleRuns(t, st, p.ID, []string{t1.ID, t2.ID})

	t.Run("all for project", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/runs?project_id=" + p.ID)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var listed []runJSON
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&listed))
		require.Len(t, listed, 2)
		assert.Equal(t, runs[1].ID, listed[0].ID)
		assert.Equal(t, runs[0].ID, listed[1].ID)
	})

	t.Run("filter by task_id", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/runs?project_id=" + p.ID + "&task_id=" + t1.ID)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var listed []runJSON
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&listed))
		require.Len(t, listed, 1)
		assert.Equal(t, runs[0].ID, listed[0].ID)
		assert.Equal(t, t1.ID, listed[0].TaskID)
	})

	t.Run("filter by status", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/runs?project_id=" + p.ID + "&status=error")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var listed []runJSON
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&listed))
		require.Len(t, listed, 1)
		assert.Equal(t, runs[1].ID, listed[0].ID)
		assert.Equal(t, store.RunStatusError, listed[0].Status)
	})

	t.Run("invalid status", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/runs?status=bogus")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assertJSONError(t, resp, "invalid status")
	})
}

func TestRunsGetAndLog(t *testing.T) {
	ts, st := startTestServerWithStore(t, "")
	p := createTestProject(t, ts.URL)

	task := &store.Task{
		ProjectID:       p.ID,
		Name:            "drain",
		Prompt:          "p",
		Command:         "echo",
		IntervalSeconds: 60,
		Enabled:         true,
	}
	require.NoError(t, st.CreateTask(task))

	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "run.log")
	require.NoError(t, os.WriteFile(logPath, []byte("hello run log\nline two"), 0o644))

	run := &store.Run{
		TaskID:    task.ID,
		ProjectID: p.ID,
		StartedAt: "2026-08-31T12:00:00Z",
		Status:    store.RunStatusSuccess,
		LogPath:   &logPath,
	}
	require.NoError(t, st.InsertRun(run))

	getResp, err := http.Get(ts.URL + "/api/v1/runs/" + run.ID)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)

	var got runJSON
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&got))
	assert.Equal(t, run.ID, got.ID)
	assert.Equal(t, task.ID, got.TaskID)
	assert.Equal(t, p.ID, got.ProjectID)
	assert.Equal(t, store.RunStatusSuccess, got.Status)
	require.NotNil(t, got.LogPath)
	assert.Equal(t, logPath, *got.LogPath)

	logResp, err := http.Get(ts.URL + "/api/v1/runs/" + run.ID + "/log")
	require.NoError(t, err)
	defer logResp.Body.Close()
	require.Equal(t, http.StatusOK, logResp.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", logResp.Header.Get("Content-Type"))

	body, err := io.ReadAll(logResp.Body)
	require.NoError(t, err)
	assert.Equal(t, "hello run log\nline two", string(body))
}

func TestRunsLogNotFound(t *testing.T) {
	ts, st := startTestServerWithStore(t, "")
	p := createTestProject(t, ts.URL)

	task := &store.Task{
		ProjectID:       p.ID,
		Name:            "drain",
		Prompt:          "p",
		Command:         "echo",
		IntervalSeconds: 60,
		Enabled:         true,
	}
	require.NoError(t, st.CreateTask(task))

	run := &store.Run{
		TaskID:    task.ID,
		ProjectID: p.ID,
		StartedAt: "2026-08-31T12:00:00Z",
		Status:    store.RunStatusRunning,
	}
	require.NoError(t, st.InsertRun(run))

	resp, err := http.Get(ts.URL + "/api/v1/runs/" + run.ID + "/log")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assertJSONError(t, resp, "log not found")

	missingPath := filepath.Join(t.TempDir(), "missing.log")
	run.LogPath = &missingPath
	require.NoError(t, st.UpdateRun(run))

	resp2, err := http.Get(ts.URL + "/api/v1/runs/" + run.ID + "/log")
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
	assertJSONError(t, resp2, "log not found")
}

func TestRunsGetNotFound(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")
	resp, err := http.Get(ts.URL + "/api/v1/runs/nonexistent-id")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assertJSONError(t, resp, "run not found")
}
