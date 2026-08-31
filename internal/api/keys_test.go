package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/store"
)

const testPrivateKeyPEM = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACDqrHomBqUVD1Zc8lJbnQ7cYr6LFrSu+HOACCFeN/RYgAAAAKCw0JP1sNCT
9QAAAAtzc2gtZWQyNTUxOQAAACDqrHomBqUVD1Zc8lJbnQ7cYr6LFrSu+HOACCFeN/RYgA
AAAEAApELQqIXI1lytCJYiPFlg1TdMFElAEkYI88srl1WV8OqseiYGpRUPVlzyUludDtxi
vosWtK74c4AIIV438FiAAAAAAECAwQF
-----END OPENSSH PRIVATE KEY-----
`

func startTestServerWithSecrets(t *testing.T, token string) (*httptest.Server, *store.Store, string) {
	t.Helper()
	st := openTempStore(t)
	secretsDir := filepath.Join(t.TempDir(), "secrets")
	srv := New(Config{
		Store:      st,
		SecretsDir: secretsDir,
		Token:      token,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, st, secretsDir
}

func uploadProjectKey(t *testing.T, ts *httptest.Server, projectID, keyContent string) *http.Response {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("key", "id_ed25519")
	require.NoError(t, err)
	_, err = io.WriteString(part, keyContent)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/projects/"+projectID+"/key", &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func TestProjectKeyUploadAndStatus(t *testing.T) {
	ts, _, secretsDir := startTestServerWithSecrets(t, "")
	created := createTestProject(t, ts.URL)

	statusResp, err := http.Get(ts.URL + "/api/v1/projects/" + created.ID + "/key")
	require.NoError(t, err)
	defer statusResp.Body.Close()
	require.Equal(t, http.StatusOK, statusResp.StatusCode)
	var before keyStatusJSON
	require.NoError(t, json.NewDecoder(statusResp.Body).Decode(&before))
	assert.False(t, before.Present)

	uploadResp := uploadProjectKey(t, ts, created.ID, testPrivateKeyPEM)
	defer uploadResp.Body.Close()
	require.Equal(t, http.StatusOK, uploadResp.StatusCode)

	uploadBody, err := io.ReadAll(uploadResp.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(uploadBody), "BEGIN OPENSSH PRIVATE KEY")
	assert.NotContains(t, string(uploadBody), "b3BlbnNzaC1rZXk")

	var uploaded keyStatusJSON
	require.NoError(t, json.Unmarshal(uploadBody, &uploaded))
	assert.True(t, uploaded.Present)

	keyPath := filepath.Join(secretsDir, "projects", created.ID, "id_ed25519")
	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(keyFilePerm), info.Mode().Perm())

	onDisk, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, testPrivateKeyPEM, string(onDisk))

	getResp, err := http.Get(ts.URL + "/api/v1/projects/" + created.ID)
	require.NoError(t, err)
	defer getResp.Body.Close()
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var updated projectJSON
	require.NoError(t, json.NewDecoder(getResp.Body).Decode(&updated))
	assert.Equal(t, keyPath, updated.SSHKeyPath)

	afterResp, err := http.Get(ts.URL + "/api/v1/projects/" + created.ID + "/key")
	require.NoError(t, err)
	defer afterResp.Body.Close()
	require.Equal(t, http.StatusOK, afterResp.StatusCode)
	afterBody, err := io.ReadAll(afterResp.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(afterBody), "BEGIN OPENSSH PRIVATE KEY")

	var after keyStatusJSON
	require.NoError(t, json.Unmarshal(afterBody, &after))
	assert.True(t, after.Present)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(afterBody, &raw))
	assert.Equal(t, map[string]any{"present": true}, raw)
}

func TestProjectKeyUploadReplacesExisting(t *testing.T) {
	ts, _, secretsDir := startTestServerWithSecrets(t, "")
	created := createTestProject(t, ts.URL)

	first := uploadProjectKey(t, ts, created.ID, testPrivateKeyPEM)
	first.Body.Close()
	require.Equal(t, http.StatusOK, first.StatusCode)

	replaced := uploadProjectKey(t, ts, created.ID, strings.TrimSpace(testPrivateKeyPEM)+"\n")
	replaced.Body.Close()
	require.Equal(t, http.StatusOK, replaced.StatusCode)

	keyPath := filepath.Join(secretsDir, "projects", created.ID, "id_ed25519")
	info, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(keyFilePerm), info.Mode().Perm())
}

func TestProjectKeyUploadValidationErrors(t *testing.T) {
	ts, _, _ := startTestServerWithSecrets(t, "")
	created := createTestProject(t, ts.URL)

	t.Run("not found", func(t *testing.T) {
		resp := uploadProjectKey(t, ts, "missing-id", testPrivateKeyPEM)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assertJSONError(t, resp, "project not found")
	})

	t.Run("missing file", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/projects/"+created.ID+"/key", strings.NewReader("not-multipart"))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assertJSONError(t, resp, "invalid multipart form")
	})

	t.Run("empty key", func(t *testing.T) {
		resp := uploadProjectKey(t, ts, created.ID, "")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assertJSONError(t, resp, "key file is empty")
	})
}

func TestProjectKeyStatusNotFound(t *testing.T) {
	ts, _, _ := startTestServerWithSecrets(t, "")
	resp, err := http.Get(ts.URL + "/api/v1/projects/missing-id/key")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assertJSONError(t, resp, "project not found")
}
