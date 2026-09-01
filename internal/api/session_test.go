package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startTestServerAuth(t *testing.T, cfg Config) *httptest.Server {
	t.Helper()
	if cfg.Store == nil {
		cfg.Store = openTempStore(t)
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	srv := New(cfg)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func TestLoginSessionAuth(t *testing.T) {
	const user = "admin"
	const pass = "secret-pass"
	ts := startTestServerAuth(t, Config{
		Auth: AuthConfig{
			AdminUsername: user,
			AdminPassword: pass,
			SessionSecret: []byte("test-secret"),
			SessionTTL:    time.Hour,
		},
	})

	t.Run("unauthenticated API rejected", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/status")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("login page static without auth", func(t *testing.T) {
		srv := New(Config{
			Store: openTempStore(t),
			Auth: AuthConfig{
				AdminUsername: user,
				AdminPassword: pass,
				SessionSecret: []byte("test-secret"),
				SessionTTL:    time.Hour,
			},
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		})
		uiTS := httptest.NewServer(srv.HandlerWithUI(testUIHandler()))
		t.Cleanup(uiTS.Close)

		resp, err := http.Get(uiTS.URL + "/login.html")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json",
			bytes.NewReader([]byte(`{"username":"admin","password":"wrong"}`)))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assertJSONError(t, resp, "invalid username or password")
	})

	client := &http.Client{}
	var sessionCookies []*http.Cookie

	t.Run("valid login sets cookie", func(t *testing.T) {
		resp, err := client.Post(ts.URL+"/api/v1/auth/login", "application/json",
			bytes.NewReader([]byte(`{"username":"admin","password":"secret-pass"}`)))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		sessionCookies = resp.Cookies()
		require.NotEmpty(t, sessionCookies)
	})

	t.Run("session cookie grants API access", func(t *testing.T) {
		require.NotEmpty(t, sessionCookies)
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/status", nil)
		require.NoError(t, err)
		for _, c := range sessionCookies {
			req.AddCookie(c)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("logout clears session", func(t *testing.T) {
		require.NotEmpty(t, sessionCookies)
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/auth/logout", nil)
		require.NoError(t, err)
		for _, c := range sessionCookies {
			req.AddCookie(c)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		req2, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/status", nil)
		require.NoError(t, err)
		for _, c := range sessionCookies {
			req2.AddCookie(c)
		}
		resp2, err := client.Do(req2)
		require.NoError(t, err)
		defer resp2.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp2.StatusCode)
	})
}

func TestAdminAuthAlsoAcceptsBearerToken(t *testing.T) {
	const user = "admin"
	const pass = "secret-pass"
	const token = "api-token"
	ts := startTestServerAuth(t, Config{
		Token: token,
		Auth: AuthConfig{
			AdminUsername: user,
			AdminPassword: pass,
			SessionSecret: []byte("test-secret"),
			SessionTTL:    time.Hour,
		},
	})

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/status", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLoginNotConfiguredReturns404(t *testing.T) {
	ts := startTestServerAuth(t, Config{})
	resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json",
		bytes.NewReader([]byte(`{"username":"a","password":"b"}`)))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestLoginResponseNeverIncludesPassword(t *testing.T) {
	ts := startTestServerAuth(t, Config{
		Auth: AuthConfig{
			AdminUsername: "admin",
			AdminPassword: "secret-pass",
			SessionSecret: []byte("test-secret"),
			SessionTTL:    time.Hour,
		},
	})
	resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json",
		bytes.NewReader([]byte(`{"username":"admin","password":"secret-pass"}`)))
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "secret-pass")
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	assert.Equal(t, true, body["ok"])
}
