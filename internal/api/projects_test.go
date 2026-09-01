package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/store"
)

func startTestServerWithStore(t *testing.T, token string) (*httptest.Server, *store.Store) {
	t.Helper()
	st := openTempStore(t)
	srv := New(Config{
		Store:  st,
		Token:  token,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, st
}

func TestProjectsCreateListGetPatch(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")

	createBody := map[string]any{
		"name":            "my-app",
		"path":            "/home/dev/my-app",
		"ssh_host":        "dev-box",
		"ssh_user":        "dev",
		"ssh_port":        2222,
		"ssh_private_key": store.TestSSHPrivateKey,
		"enabled":         true,
	}
	raw, err := json.Marshal(createBody)
	require.NoError(t, err)

	resp, err := http.Post(ts.URL+"/api/v1/projects", "application/json", bytes.NewReader(raw))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var created projectJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "my-app", created.Name)
	assert.True(t, created.KeyConfigured)
	assert.True(t, created.Enabled)

	var rawMap map[string]any
	rawBytes, err := json.Marshal(created)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(rawBytes, &rawMap))
	assert.NotContains(t, rawMap, "ssh_private_key")
	assert.Equal(t, true, rawMap["key_configured"])

	listResp, err := http.Get(ts.URL + "/api/v1/projects")
	require.NoError(t, err)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)
	var listed []projectJSON
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&listed))
	require.Len(t, listed, 1)
	assert.Equal(t, created.ID, listed[0].ID)

	getResp, err := http.Get(ts.URL + "/api/v1/projects/" + created.ID)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var gotDetail map[string]any
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&gotDetail))
	assert.Equal(t, created.Name, gotDetail["name"])
	assert.Contains(t, gotDetail, "ssh_private_key")
	assert.Equal(t, store.TestSSHPrivateKey, gotDetail["ssh_private_key"])

	patchRaw, err := json.Marshal(map[string]any{"enabled": false})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/projects/"+created.ID, bytes.NewReader(patchRaw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer patchResp.Body.Close()
	require.Equal(t, http.StatusOK, patchResp.StatusCode)
	var patched projectJSON
	require.NoError(t, json.NewDecoder(patchResp.Body).Decode(&patched))
	assert.False(t, patched.Enabled)
	assert.True(t, patched.KeyConfigured)
}

func TestProjectsCreateValidationErrors(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")

	cases := []struct {
		name string
		body map[string]any
		want string
	}{
		{
			name: "missing name",
			body: map[string]any{
				"path": "/p", "ssh_host": "h", "ssh_user": "u", "ssh_private_key": store.TestSSHPrivateKey,
			},
			want: "name is required",
		},
		{
			name: "missing path",
			body: map[string]any{
				"name": "n", "ssh_host": "h", "ssh_user": "u", "ssh_private_key": store.TestSSHPrivateKey,
			},
			want: "path is required",
		},
		{
			name: "missing ssh_host",
			body: map[string]any{
				"name": "n", "path": "/p", "ssh_user": "u", "ssh_private_key": store.TestSSHPrivateKey,
			},
			want: "ssh_host is required",
		},
		{
			name: "missing ssh_user",
			body: map[string]any{
				"name": "n", "path": "/p", "ssh_host": "h", "ssh_private_key": store.TestSSHPrivateKey,
			},
			want: "ssh_user is required",
		},
		{
			name: "missing ssh_private_key",
			body: map[string]any{
				"name": "n", "path": "/p", "ssh_host": "h", "ssh_user": "u",
			},
			want: "ssh_private_key is required",
		},
		{
			name: "invalid ssh_port",
			body: map[string]any{
				"name": "n", "path": "/p", "ssh_host": "h", "ssh_user": "u",
				"ssh_private_key": store.TestSSHPrivateKey, "ssh_port": 0,
			},
			want: "ssh_port must be positive",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.body)
			require.NoError(t, err)
			resp, err := http.Post(ts.URL+"/api/v1/projects", "application/json", bytes.NewReader(raw))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assertJSONError(t, resp, tc.want)
		})
	}
}

func TestProjectsPatchUpdatesSSHKey(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")

	createRaw, err := json.Marshal(map[string]any{
		"name": "n", "path": "/p", "ssh_host": "h", "ssh_user": "u",
		"ssh_private_key": store.TestSSHPrivateKey,
	})
	require.NoError(t, err)
	resp, err := http.Post(ts.URL+"/api/v1/projects", "application/json", bytes.NewReader(createRaw))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created projectJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))

	newKey := "-----BEGIN OPENSSH PRIVATE KEY-----\nupdated\n-----END OPENSSH PRIVATE KEY-----\n"
	patchRaw, err := json.Marshal(map[string]any{"ssh_private_key": newKey})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/projects/"+created.ID, bytes.NewReader(patchRaw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer patchResp.Body.Close()
	require.Equal(t, http.StatusOK, patchResp.StatusCode)
	body, err := io.ReadAll(patchResp.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "BEGIN OPENSSH")
	assert.NotContains(t, string(body), "ssh_private_key")
}

func TestProjectsNotFound(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")

	resp, err := http.Get(ts.URL + "/api/v1/projects/nope")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestProjectsCreateDefaults(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")

	raw, err := json.Marshal(map[string]any{
		"name": "defaults", "path": "/p", "ssh_host": "h", "ssh_user": "u",
		"ssh_private_key": store.TestSSHPrivateKey,
	})
	require.NoError(t, err)
	resp, err := http.Post(ts.URL+"/api/v1/projects", "application/json", bytes.NewReader(raw))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created projectJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
	assert.Equal(t, 22, created.SSHPort)
	assert.True(t, created.Enabled)
}

func TestProjectsCRUDAfterV1Migration(t *testing.T) {
	st := openStoreMigratedFromV1(t)
	srv := New(Config{
		Store:  st,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	createBody := map[string]any{
		"name":            "migrated-db",
		"path":            "/home/dev/app",
		"ssh_host":        "worker",
		"ssh_user":        "dev",
		"ssh_private_key": store.TestSSHPrivateKey,
	}
	raw, err := json.Marshal(createBody)
	require.NoError(t, err)

	resp, err := http.Post(ts.URL+"/api/v1/projects", "application/json", bytes.NewReader(raw))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, "project create must work after v1→v2 migration")

	listResp, err := http.Get(ts.URL + "/api/v1/projects")
	require.NoError(t, err)
	defer listResp.Body.Close()
	require.Equal(t, http.StatusOK, listResp.StatusCode)
}
