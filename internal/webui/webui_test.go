package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerServesIndexHTML(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "master-agent")
	assert.Contains(t, string(body), "Projects")
}

func TestHandlerServesStaticAssets(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	for _, path := range []string{"/app.js", "/auth.js", "/login.js", "/tasks.js", "/runs.js", "/style.css", "/tasks.html", "/runs.html", "/login.html"} {
		t.Run(path, func(t *testing.T) {
			resp, err := http.Get(ts.URL + path)
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.NotEmpty(t, body)
		})
	}
}

func TestHandlerAppJSHasAPIClient(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/app.js")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	s := string(body)
	assert.True(t, strings.Contains(s, "'/projects'"))
	assert.True(t, strings.Contains(s, "MAAuth"))
	assert.NotContains(t, s, "sessionStorage")
}

func TestHandlerIndexHasKeyUploadUI(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	s := string(body)
	assert.Contains(t, s, "key-dialog")
	assert.Contains(t, s, `type="file"`)
	assert.Contains(t, s, "SSH key")
}

func TestHandlerAppJSHasKeyUploadClient(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/app.js")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	s := string(body)
	assert.Contains(t, s, "/key")
	assert.Contains(t, s, "FormData")
	assert.Contains(t, s, "key_present")
	assert.NotContains(t, s, "BEGIN OPENSSH PRIVATE KEY")
	assert.NotContains(t, s, "readAsText")
}

func TestHandlerServesTasksHTML(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/tasks.html")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	s := string(body)
	assert.Contains(t, s, "Tasks")
	assert.Contains(t, s, "interval_seconds")
	assert.Contains(t, s, "task-form")
	assert.NotContains(t, s, "ssh_host")
	assert.NotContains(t, s, "ssh_key_path")
}

func TestHandlerTasksJSHasAPIClient(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/tasks.js")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	s := string(body)
	assert.Contains(t, s, "/projects/")
	assert.Contains(t, s, "/tasks/")
	assert.Contains(t, s, "last_run_at")
	assert.Contains(t, s, "next_run_at")
	assert.Contains(t, s, "interval_seconds")
	assert.NotContains(t, s, "ssh_host")
}

func TestHandlerIndexNeverShowsKeyMaterialHint(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	s := string(body)
	assert.Contains(t, s, "never shown")
	assert.NotContains(t, s, "BEGIN OPENSSH PRIVATE KEY")
}

func TestHandlerServesRunsHTML(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/runs.html")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	s := string(body)
	assert.Contains(t, s, "Runs")
	assert.Contains(t, s, "project-filter")
	assert.Contains(t, s, "status-filter")
	assert.Contains(t, s, "run-detail-dialog")
	assert.Contains(t, s, "detail-log")
}

func TestHandlerRunsJSHasAPIClient(t *testing.T) {
	ts := httptest.NewServer(Handler())
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/runs.js")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	s := string(body)
	assert.Contains(t, s, "/runs")
	assert.Contains(t, s, "project_id")
	assert.Contains(t, s, "status")
	assert.Contains(t, s, "/log")
	assert.Contains(t, s, "exit_code")
	assert.Contains(t, s, "error_message")
}
