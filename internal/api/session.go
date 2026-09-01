package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	EnvAdminUsername = "ADMIN_USERNAME"
	EnvAdminPassword = "ADMIN_PASSWORD"
	EnvSessionSecret = "SESSION_SECRET"
	EnvSessionTTL    = "SESSION_TTL"

	sessionCookieName = "master_agent_session"
	defaultSessionTTL = 7 * 24 * time.Hour
)

// AuthConfig holds optional admin login and session settings.
type AuthConfig struct {
	AdminUsername string
	AdminPassword string
	SessionSecret []byte
	SessionTTL    time.Duration
}

// AdminEnabled reports whether login/password auth is required.
func (a AuthConfig) AdminEnabled() bool {
	return a.AdminUsername != "" && a.AdminPassword != ""
}

// AuthFromEnv loads admin/session settings from the environment.
func AuthFromEnv() AuthConfig {
	ttl := defaultSessionTTL
	if raw := strings.TrimSpace(os.Getenv(EnvSessionTTL)); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			ttl = d
		}
	}
	cfg := AuthConfig{
		AdminUsername: strings.TrimSpace(os.Getenv(EnvAdminUsername)),
		AdminPassword: os.Getenv(EnvAdminPassword),
		SessionTTL:    ttl,
	}
	if secret := os.Getenv(EnvSessionSecret); secret != "" {
		cfg.SessionSecret = []byte(secret)
	} else if cfg.AdminEnabled() {
		sum := sha256.Sum256([]byte("master-agent-session:" + cfg.AdminUsername + "\x00" + cfg.AdminPassword))
		cfg.SessionSecret = sum[:]
	}
	return cfg
}

type sessionManager struct {
	secret []byte
	ttl    time.Duration
	mu     sync.Mutex
	active map[string]int64 // session id -> expiry unix
}

func newSessionManager(cfg AuthConfig) *sessionManager {
	if !cfg.AdminEnabled() {
		return nil
	}
	return &sessionManager{
		secret: cfg.SessionSecret,
		ttl:    cfg.SessionTTL,
		active: make(map[string]int64),
	}
}

func (m *sessionManager) issue(w http.ResponseWriter, username string) {
	if m == nil {
		return
	}
	id := uuid.NewString()
	exp := time.Now().Add(m.ttl).Unix()
	m.mu.Lock()
	m.active[id] = exp
	m.mu.Unlock()
	value := m.sign(username, exp, id)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(m.ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *sessionManager) clear(w http.ResponseWriter, r *http.Request) {
	if m != nil {
		m.revokeRequest(r)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *sessionManager) revokeRequest(r *http.Request) {
	if m == nil {
		return
	}
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return
	}
	_, _, id, ok := m.verify(c.Value)
	if !ok || id == "" {
		return
	}
	m.mu.Lock()
	delete(m.active, id)
	m.mu.Unlock()
}

func (m *sessionManager) valid(r *http.Request) bool {
	if m == nil {
		return false
	}
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return false
	}
	username, exp, id, ok := m.verify(c.Value)
	if !ok || id == "" {
		return false
	}
	if time.Now().Unix() > exp {
		m.mu.Lock()
		delete(m.active, id)
		m.mu.Unlock()
		return false
	}
	m.mu.Lock()
	_, live := m.active[id]
	m.mu.Unlock()
	if !live {
		return false
	}
	return username != ""
}

func (m *sessionManager) sign(username string, exp int64, id string) string {
	payload := username + "|" + strconv.FormatInt(exp, 10) + "|" + id
	payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payloadB64))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payloadB64 + "." + sig
}

func (m *sessionManager) verify(value string) (username string, exp int64, id string, ok bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return "", 0, "", false
	}
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || subtle.ConstantTimeCompare(expected, got) != 1 {
		return "", 0, "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", 0, "", false
	}
	seg := strings.SplitN(string(raw), "|", 3)
	if len(seg) != 3 {
		return "", 0, "", false
	}
	exp, err = strconv.ParseInt(seg[1], 10, 64)
	if err != nil {
		return "", 0, "", false
	}
	return seg[0], exp, seg[2], true
}

func adminCredentialsMatch(cfg AuthConfig, username, password string) bool {
	if !cfg.AdminEnabled() {
		return false
	}
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(cfg.AdminUsername)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(cfg.AdminPassword)) == 1
	return userOK && passOK
}

func authRequired(cfg AuthConfig, token string) bool {
	if cfg.AdminEnabled() {
		return true
	}
	return token != ""
}

func requestAuthenticated(r *http.Request, cfg AuthConfig, token string, sessions *sessionManager) bool {
	if sessions != nil && sessions.valid(r) {
		return true
	}
	if token != "" {
		got := bearerToken(r.Header.Get("Authorization"))
		if got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1 {
			return true
		}
	}
	return false
}

func isAuthExemptPath(path, method string) bool {
	if path == "/api/v1/auth/login" && method == http.MethodPost {
		return true
	}
	return false
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	OK bool `json:"ok"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.Auth.AdminEnabled() {
		WriteError(w, http.StatusNotFound, "login not configured")
		return
	}
	var req loginRequest
	if err := decodeJSONBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !adminCredentialsMatch(s.cfg.Auth, strings.TrimSpace(req.Username), req.Password) {
		WriteError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}
	if s.sessions != nil {
		s.sessions.issue(w, s.cfg.Auth.AdminUsername)
	}
	writeJSON(w, http.StatusOK, loginResponse{OK: true})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if s.sessions != nil {
		s.sessions.clear(w, r)
	}
	writeJSON(w, http.StatusOK, loginResponse{OK: true})
}
