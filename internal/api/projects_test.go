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
		"name":         "my-app",
		"path":         "/home/dev/my-app",
		"ssh_host":     "dev-box",
		"ssh_user":     "dev",
		"ssh_port":     2222,
		"ssh_key_path": "/secrets/projects/my-app/id_ed25519",
		"enabled":      true,
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
	assert.Equal(t, "/home/dev/my-app", created.Path)
	assert.Equal(t, "dev-box", created.SSHHost)
	assert.Equal(t, "dev", created.SSHUser)
	assert.Equal(t, 2222, created.SSHPort)
	assert.Equal(t, "/secrets/projects/my-app/id_ed25519", created.SSHKeyPath)
	assert.True(t, created.Enabled)
	assert.NotEmpty(t, created.CreatedAt)
	assert.NotEmpty(t, created.UpdatedAt)

	// Response must never include private key material — only path.
	var rawMap map[string]any
	rawBytes, err := json.Marshal(created)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(rawBytes, &rawMap))
	assert.NotContains(t, rawMap, "private_key")
	assert.NotContains(t, rawMap, "key")
	assert.NotContains(t, rawMap, "ssh_private_key")
	assert.Equal(t, "/secrets/projects/my-app/id_ed25519", rawMap["ssh_key_path"])

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
	var got projectJSON
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&got))
	assert.Equal(t, created, got)

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
	assert.Equal(t, created.Name, patched.Name)
	assert.Equal(t, created.SSHKeyPath, patched.SSHKeyPath)
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
				"path": "/p", "ssh_host": "h", "ssh_user": "u", "ssh_key_path": "/k",
			},
			want: "name is required",
		},
		{
			name: "missing path",
			body: map[string]any{
				"name": "n", "ssh_host": "h", "ssh_user": "u", "ssh_key_path": "/k",
			},
			want: "path is required",
		},
		{
			name: "missing ssh_host",
			body: map[string]any{
				"name": "n", "path": "/p", "ssh_user": "u", "ssh_key_path": "/k",
			},
			want: "ssh_host is required",
		},
		{
			name: "missing ssh_user",
			body: map[string]any{
				"name": "n", "path": "/p", "ssh_host": "h", "ssh_key_path": "/k",
			},
			want: "ssh_user is required",
		},
		{
			name: "missing ssh_key_path",
			body: map[string]any{
				"name": "n", "path": "/p", "ssh_host": "h", "ssh_user": "u",
			},
			want: "ssh_key_path is required",
		},
		{
			name: "invalid ssh_port",
			body: map[string]any{
				"name": "n", "path": "/p", "ssh_host": "h", "ssh_user": "u",
				"ssh_key_path": "/k", "ssh_port": 0,
			},
			want: "ssh_port must be positive",
		},
		{
			name: "empty body",
			body: nil,
			want: "request body is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var reader io.Reader
			if tc.body != nil {
				raw, err := json.Marshal(tc.body)
				require.NoError(t, err)
				reader = bytes.NewReader(raw)
			} else {
				reader = bytes.NewReader(nil)
			}
			resp, err := http.Post(ts.URL+"/api/v1/projects", "application/json", reader)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assertJSONError(t, resp, tc.want)
		})
	}
}

func TestProjectsPatchNotFoundAndValidation(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")

	t.Run("not found", func(t *testing.T) {
		raw, err := json.Marshal(map[string]any{"enabled": false})
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/projects/missing-id", bytes.NewReader(raw))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assertJSONError(t, resp, "project not found")
	})

	t.Run("empty patch", func(t *testing.T) {
		createRaw, err := json.Marshal(map[string]any{
			"name": "n", "path": "/p", "ssh_host": "h", "ssh_user": "u", "ssh_key_path": "/k",
		})
		require.NoError(t, err)
		createResp, err := http.Post(ts.URL+"/api/v1/projects", "application/json", bytes.NewReader(createRaw))
		require.NoError(t, err)
		defer createResp.Body.Close()
		require.Equal(t, http.StatusCreated, createResp.StatusCode)
		var created projectJSON
		require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))

		req, err := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/projects/"+created.ID, bytes.NewReader([]byte(`{}`)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assertJSONError(t, resp, "no fields to update")
	})
}

func TestProjectsGetNotFound(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")
	resp, err := http.Get(ts.URL + "/api/v1/projects/does-not-exist")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assertJSONError(t, resp, "project not found")
}

func TestProjectsDefaultSSHPortAndEnabled(t *testing.T) {
	ts, _ := startTestServerWithStore(t, "")
	raw, err := json.Marshal(map[string]any{
		"name": "defaults", "path": "/p", "ssh_host": "h", "ssh_user": "u", "ssh_key_path": "/k",
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
