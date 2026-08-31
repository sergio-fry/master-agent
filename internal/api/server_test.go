package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/store"
)

func openTempStore(t *testing.T) *store.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data", "master-agent.db")
	s, err := store.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func startTestServer(t *testing.T, token string) *httptest.Server {
	t.Helper()
	st := openTempStore(t)
	srv := New(Config{
		Store:  st,
		Token:  token,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestServerStartsWithTempDB(t *testing.T) {
	ts := startTestServer(t, "")
	resp, err := http.Get(ts.URL + "/api/v1/status")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, true, body["ok"])
}

func TestAuthRequiresBearerWhenTokenSet(t *testing.T) {
	const token = "secret-token"
	ts := startTestServer(t, token)

	t.Run("missing auth on API", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/status")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assertJSONError(t, resp, "unauthorized")
	})

	t.Run("static UI without auth", func(t *testing.T) {
		st := openTempStore(t)
		srv := New(Config{
			Store:  st,
			Token:  token,
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		})
		uiTS := httptest.NewServer(srv.HandlerWithUI(testUIHandler()))
		t.Cleanup(uiTS.Close)

		resp, err := http.Get(uiTS.URL + "/")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("wrong token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/status", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer wrong")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assertJSONError(t, resp, "unauthorized")
	})

	t.Run("valid token", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/status", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestAuthOpenWhenTokenEmpty(t *testing.T) {
	ts := startTestServer(t, "")
	resp, err := http.Get(ts.URL + "/api/v1/status")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestWriteErrorJSONShape(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusBadRequest, "bad request")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	rawBytes := rec.Body.Bytes()
	var body ErrorBody
	require.NoError(t, json.Unmarshal(rawBytes, &body))
	assert.Equal(t, "bad request", body.Error)

	// Ensure only the documented field is present.
	var raw map[string]any
	require.NoError(t, json.Unmarshal(rawBytes, &raw))
	assert.Equal(t, map[string]any{"error": "bad request"}, raw)
}

func TestNotFoundUsesMuxDefault(t *testing.T) {
	ts := startTestServer(t, "")
	resp, err := http.Get(ts.URL + "/api/v1/nope")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRequestIDPropagated(t *testing.T) {
	ts := startTestServer(t, "")
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/status", nil)
	require.NoError(t, err)
	req.Header.Set("X-Request-ID", "client-req-1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, "client-req-1", resp.Header.Get("X-Request-ID"))
}

func testUIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ui"))
	})
}

func TestHandlerWithUIServesAPIAndStatic(t *testing.T) {
	st := openTempStore(t)
	srv := New(Config{
		Store:  st,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	ts := httptest.NewServer(srv.HandlerWithUI(testUIHandler()))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/api/v1/status")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "ui", string(body))
}

func assertJSONError(t *testing.T, resp *http.Response, want string) {
	t.Helper()
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	var body ErrorBody
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, want, body.Error)
}
