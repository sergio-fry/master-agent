package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestProject(t *testing.T, tsURL string) projectJSON {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"name": "my-app", "path": "/home/dev/my-app",
		"ssh_host": "dev-box", "ssh_user": "dev", "ssh_key_path": "/secrets/k",
	})
	require.NoError(t, err)
	resp, err := http.Post(tsURL+"/api/v1/projects", "application/json", bytes.NewReader(raw))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var p projectJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&p))
	return p
}

func TestTasksCreateListGetPatch(t *testing.T) {
	ts, st := startTestServerWithStore(t, "")
	p := createTestProject(t, ts.URL)

	createBody := map[string]any{
		"name":             "drain",
		"prompt":           "Process backlog",
		"command":          "cursor agent -p {{prompt}}",
		"interval_seconds": 1800,
		"enabled":          true,
	}
	raw, err := json.Marshal(createBody)
	require.NoError(t, err)

	resp, err := http.Post(ts.URL+"/api/v1/projects/"+p.ID+"/tasks", "application/json", bytes.NewReader(raw))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var created taskJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, p.ID, created.ProjectID)
	assert.Equal(t, "drain", created.Name)
	assert.Equal(t, "Process backlog", created.Prompt)
	assert.Equal(t, "cursor agent -p {{prompt}}", created.Command)
	assert.Equal(t, 1800, created.IntervalSeconds)
	assert.True(t, created.Enabled)
	assert.NotEmpty(t, created.CreatedAt)
	assert.NotEmpty(t, created.UpdatedAt)

	// Response must not include SSH fields.
	rawBytes, err := json.Marshal(created)
	require.NoError(t, err)
	var rawMap map[string]any
	require.NoError(t, json.Unmarshal(rawBytes, &rawMap))
	assert.NotContains(t, rawMap, "ssh_host")
	assert.NotContains(t, rawMap, "ssh_user")
	assert.NotContains(t, rawMap, "ssh_port")
	assert.NotContains(t, rawMap, "ssh_key_path")

	listResp, err := http.Get(ts.URL + "/api/v1/projects/" + p.ID + "/tasks")
	require.NoError(t, err)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	var listed []taskJSON
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&listed))
	require.Len(t, listed, 1)
	assert.Equal(t, created.ID, listed[0].ID)

	getResp, err := http.Get(ts.URL + "/api/v1/tasks/" + created.ID)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var got taskJSON
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&got))
	assert.Equal(t, created, got)

	patchRaw, err := json.Marshal(map[string]any{
		"prompt":           "Updated prompt",
		"interval_seconds": 3600,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/tasks/"+created.ID, bytes.NewReader(patchRaw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer patchResp.Body.Close()
	require.Equal(t, http.StatusOK, patchResp.StatusCode)
	var patched taskJSON
	require.NoError(t, json.NewDecoder(patchResp.Body).Decode(&patched))
	assert.Equal(t, "Updated prompt", patched.Prompt)
	assert.Equal(t, 3600, patched.IntervalSeconds)
	assert.Equal(t, created.Name, patched.Name)

	// Sanity: store still has the task.
	stored, err := st.GetTask(created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated prompt", stored.Prompt)
}

func TestTasksDisableStopsScheduling(t *testing.T) {
	ts, st := startTestServerWithStore(t, "")
	p := createTestProject(t, ts.URL)

	raw, err := json.Marshal(map[string]any{
		"name": "due-task", "prompt": "do work", "command": "echo ok",
		"interval_seconds": 60, "enabled": true,
	})
	require.NoError(t, err)
	resp, err := http.Post(ts.URL+"/api/v1/projects/"+p.ID+"/tasks", "application/json", bytes.NewReader(raw))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created taskJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))

	now := time.Now().UTC()
	due, err := st.ListDueTasks(now)
	require.NoError(t, err)
	require.Len(t, due, 1)
	assert.Equal(t, created.ID, due[0].ID)

	patchRaw, err := json.Marshal(map[string]any{"enabled": false})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/tasks/"+created.ID, bytes.NewReader(patchRaw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer patchResp.Body.Close()
	require.Equal(t, http.StatusOK, patchResp.StatusCode)
	var patched taskJSON
	require.NoError(t, json.NewDecoder(patchResp.Body).Decode(&patched))
	assert.False(t, patched.Enabled)

	dueAfter, err := st.ListDueTasks(now)
	require.NoError(t, err)
	assert.Empty(t, dueAfter, "disabled task must not be due for the daemon scheduler")
}

func TestTasksRejectSSHFields(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")
	p := createTestProject(t, ts.URL)

	cases := []struct {
		name string
		body map[string]any
	}{
		{
			name: "ssh_host on create",
			body: map[string]any{
				"name": "t", "prompt": "p", "command": "c", "interval_seconds": 10,
				"ssh_host": "evil",
			},
		},
		{
			name: "ssh_key_path on create",
			body: map[string]any{
				"name": "t", "prompt": "p", "command": "c", "interval_seconds": 10,
				"ssh_key_path": "/secrets/x",
			},
		},
		{
			name: "private_key on create",
			body: map[string]any{
				"name": "t", "prompt": "p", "command": "c", "interval_seconds": 10,
				"private_key": "SECRET",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.body)
			require.NoError(t, err)
			resp, err := http.Post(ts.URL+"/api/v1/projects/"+p.ID+"/tasks", "application/json", bytes.NewReader(raw))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assertJSONError(t, resp, "ssh fields are not allowed on tasks")
		})
	}

	// Create a valid task, then patch with SSH field.
	createRaw, err := json.Marshal(map[string]any{
		"name": "ok", "prompt": "p", "command": "c", "interval_seconds": 10,
	})
	require.NoError(t, err)
	createResp, err := http.Post(ts.URL+"/api/v1/projects/"+p.ID+"/tasks", "application/json", bytes.NewReader(createRaw))
	require.NoError(t, err)
	defer createResp.Body.Close()
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var created taskJSON
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

	patchRaw, err := json.Marshal(map[string]any{"enabled": true, "ssh_user": "root"})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/tasks/"+created.ID, bytes.NewReader(patchRaw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer patchResp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, patchResp.StatusCode)
	assertJSONError(t, patchResp, "ssh fields are not allowed on tasks")
}

func TestTasksCreateValidationErrors(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")
	p := createTestProject(t, ts.URL)

	cases := []struct {
		name string
		body map[string]any
		want string
	}{
		{
			name: "missing name",
			body: map[string]any{"prompt": "p", "command": "c", "interval_seconds": 10},
			want: "name is required",
		},
		{
			name: "missing prompt",
			body: map[string]any{"name": "n", "command": "c", "interval_seconds": 10},
			want: "prompt is required",
		},
		{
			name: "missing command",
			body: map[string]any{"name": "n", "prompt": "p", "interval_seconds": 10},
			want: "command is required",
		},
		{
			name: "missing interval",
			body: map[string]any{"name": "n", "prompt": "p", "command": "c"},
			want: "interval_seconds is required",
		},
		{
			name: "invalid interval",
			body: map[string]any{"name": "n", "prompt": "p", "command": "c", "interval_seconds": 0},
			want: "interval_seconds must be positive",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.body)
			require.NoError(t, err)
			resp, err := http.Post(ts.URL+"/api/v1/projects/"+p.ID+"/tasks", "application/json", bytes.NewReader(raw))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assertJSONError(t, resp, tc.want)
		})
	}
}

func TestTasksProjectNotFound(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")

	raw, err := json.Marshal(map[string]any{
		"name": "t", "prompt": "p", "command": "c", "interval_seconds": 10,
	})
	require.NoError(t, err)
	resp, err := http.Post(ts.URL+"/api/v1/projects/missing/tasks", "application/json", bytes.NewReader(raw))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assertJSONError(t, resp, "project not found")

	listResp, err := http.Get(ts.URL + "/api/v1/projects/missing/tasks")
	require.NoError(t, err)
	defer listResp.Body.Close()
	assert.Equal(t, http.StatusNotFound, listResp.StatusCode)
	assertJSONError(t, listResp, "project not found")
}

func TestTasksGetNotFound(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")
	resp, err := http.Get(ts.URL + "/api/v1/tasks/does-not-exist")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assertJSONError(t, resp, "task not found")
}

func TestTasksDefaultEnabled(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")
	p := createTestProject(t, ts.URL)
	raw, err := json.Marshal(map[string]any{
		"name": "defaults", "prompt": "p", "command": "c", "interval_seconds": 30,
	})
	require.NoError(t, err)
	resp, err := http.Post(ts.URL+"/api/v1/projects/"+p.ID+"/tasks", "application/json", bytes.NewReader(raw))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created taskJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	assert.True(t, created.Enabled)
}
