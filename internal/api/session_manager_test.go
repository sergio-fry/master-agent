package api

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionManagerRevoke(t *testing.T) {
	m := newSessionManager(AuthConfig{
		AdminUsername: "admin",
		AdminPassword: "pass",
		SessionSecret: []byte("secret"),
		SessionTTL:    time.Hour,
	})
	rec := httptest.NewRecorder()
	m.issue(rec, "admin")
	require.NotEmpty(t, rec.Result().Cookies())

	req := httptest.NewRequest("GET", "/", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	assert.True(t, m.valid(req))

	clearRec := httptest.NewRecorder()
	m.clear(clearRec, req)
	assert.False(t, m.valid(req))
}
