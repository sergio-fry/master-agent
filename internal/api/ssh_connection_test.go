package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/runner"
	"master-agent/internal/store"
)

func startSSHTestServer(t *testing.T, st *store.Store, testFn func(ctx context.Context, project store.Project) (runner.SSHTestResult, error)) *httptest.Server {
	t.Helper()
	if st == nil {
		st = openTempStore(t)
	}
	srv := New(Config{
		Store:   st,
		SSHTest: testFn,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestProjectSSHTestSuccessPersistsHostKey(t *testing.T) {
	st := openTempStore(t)
	ts := startSSHTestServer(t, st, func(ctx context.Context, project store.Project) (runner.SSHTestResult, error) {
		return runner.SSHTestResult{
			HostKeyType:        "ssh-ed25519",
			HostKeyFingerprint: "SHA256:abc",
			HostKeyPublic:      "ssh-ed25519 AAAAB",
			NewlyPinned:        true,
		}, nil
	})

	p := &store.Project{
		Name: "p", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHPrivateKey: store.TestSSHPrivateKey, Enabled: true,
	}
	require.NoError(t, st.CreateProject(p))

	resp, err := http.Post(ts.URL+"/api/v1/projects/"+p.ID+"/ssh/test", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, true, body["ok"])
	assert.Equal(t, "SHA256:abc", body["host_key_fingerprint"])
	assert.Equal(t, true, body["host_key_pinned"])

	got, err := st.GetProject(p.ID)
	require.NoError(t, err)
	assert.Equal(t, "ssh-ed25519 AAAAB", got.SSHHostKey)

	getResp, err := http.Get(ts.URL + "/api/v1/projects/" + p.ID)
	require.NoError(t, err)
	defer getResp.Body.Close()
	raw, err := io.ReadAll(getResp.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), store.TestSSHPrivateKey)
}

func TestDraftSSHTestDoesNotPersist(t *testing.T) {
	ts := startSSHTestServer(t, nil, func(ctx context.Context, project store.Project) (runner.SSHTestResult, error) {
		return runner.SSHTestResult{
			HostKeyType:        "ssh-ed25519",
			HostKeyFingerprint: "SHA256:abc",
			HostKeyPublic:      "ssh-ed25519 AAAAB",
			NewlyPinned:        true,
		}, nil
	})

	raw, err := json.Marshal(map[string]any{
		"path": "/p", "ssh_host": "h", "ssh_user": "u",
		"ssh_private_key": store.TestSSHPrivateKey,
	})
	require.NoError(t, err)
	resp, err := http.Post(ts.URL+"/api/v1/ssh/test", "application/json", bytes.NewReader(raw))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "BEGIN OPENSSH")
	assert.NotContains(t, string(body), store.TestSSHPrivateKey)
}

func TestProjectSSHTestStructuredErrors(t *testing.T) {
	st := openTempStore(t)
	ts := startSSHTestServer(t, st, func(ctx context.Context, project store.Project) (runner.SSHTestResult, error) {
		return runner.SSHTestResult{}, &runner.SSHTestError{
			Code:    runner.SSHCodeAuthFailed,
			Message: "SSH authentication failed",
		}
	})

	p := &store.Project{
		Name: "p", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHPrivateKey: store.TestSSHPrivateKey, Enabled: true,
	}
	require.NoError(t, st.CreateProject(p))

	resp, err := http.Post(ts.URL+"/api/v1/projects/"+p.ID+"/ssh/test", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var errBody ErrorBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errBody))
	assert.Equal(t, "ssh_auth_failed", errBody.Code)
	assert.Equal(t, "SSH authentication failed", errBody.Error)
}

func TestDraftSSHTestValidation(t *testing.T) {
	ts := startSSHTestServer(t, nil, func(ctx context.Context, project store.Project) (runner.SSHTestResult, error) {
		return runner.SSHTestResult{}, nil
	})

	resp, err := http.Post(ts.URL+"/api/v1/ssh/test", "application/json", bytes.NewReader([]byte(`{}`)))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
