// Package api implements the shared HTTP JSON API under /api/v1.
//
// Error responses use standard HTTP status codes and body:
//
//	{"error":"<message>"}
//
// See specs/07-web-ui.md. When MASTER_AGENT_TOKEN is set, requests must include
// Authorization: Bearer <token>. When unset, the API allows all callers
// (operators should bind to loopback or a trusted network).
package api

import (
	"log/slog"
	"net/http"
	"os"

	"master-agent/internal/store"
)

// EnvToken is the environment variable for optional Bearer auth.
const EnvToken = "MASTER_AGENT_TOKEN"

// Config configures the HTTP API server.
type Config struct {
	// Store is the SQLite store; required for handlers that touch persistence.
	Store *store.Store
	// Token, if non-empty, requires Authorization: Bearer <token> on /api/v1.
	// Typically loaded from MASTER_AGENT_TOKEN via TokenFromEnv.
	Token string
	// Logger receives request logs; nil uses slog.Default().
	Logger *slog.Logger
}

// Server is the HTTP API for the Web UI control plane.
type Server struct {
	cfg Config
	mux *http.ServeMux
}

// TokenFromEnv returns the value of MASTER_AGENT_TOKEN (may be empty).
func TokenFromEnv() string {
	return os.Getenv(EnvToken)
}

// New builds a Server with routes registered under /api/v1.
func New(cfg Config) *Server {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	s := &Server{
		cfg: cfg,
		mux: http.NewServeMux(),
	}
	s.mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	s.mux.HandleFunc("GET /api/v1/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /api/v1/projects", s.handleCreateProject)
	s.mux.HandleFunc("GET /api/v1/projects/{id}", s.handleGetProject)
	s.mux.HandleFunc("PATCH /api/v1/projects/{id}", s.handlePatchProject)
	return s
}

// Handler returns the HTTP handler with request-ID, auth, and logging middleware.
func (s *Server) Handler() http.Handler {
	var h http.Handler = s.mux
	h = withAuth(s.cfg.Token, h)
	h = withLogging(s.cfg.Logger, h)
	h = withRequestID(h)
	return h
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}
