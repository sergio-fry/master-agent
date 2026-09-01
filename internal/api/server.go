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
	"context"
	"log/slog"
	"net/http"
	"os"

	"master-agent/internal/runner"
	"master-agent/internal/store"
)

// EnvToken is the environment variable for optional Bearer auth.
const EnvToken = "MASTER_AGENT_TOKEN"

// Config configures the HTTP API server.
type Config struct {
	// Store is the SQLite store; required for handlers that touch persistence.
	Store *store.Store
	// DBPath is the SQLite file path shown on GET /status (informational).
	DBPath string
	// Token, if non-empty, requires Authorization: Bearer <token> on /api/v1 when
	// admin login is not configured. With admin login, Bearer remains supported for API clients.
	Token string
	// Auth configures optional admin username/password and session cookies.
	Auth AuthConfig
	// Logger receives request logs; nil uses slog.Default().
	Logger *slog.Logger
	// SSHTest overrides SSH connection testing (for unit tests). Nil uses runner.SSHTester.
	SSHTest func(ctx context.Context, project store.Project) (runner.SSHTestResult, error)
	// ListDirs overrides remote directory listing (for unit tests). Nil uses runner.SSHBrowser.
	ListDirs func(ctx context.Context, project store.Project, path string) (runner.RemoteDirListing, error)
}

// Server is the HTTP API for the Web UI control plane.
type Server struct {
	cfg      Config
	mux      *http.ServeMux
	sessions *sessionManager
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
		cfg:      cfg,
		mux:      http.NewServeMux(),
		sessions: newSessionManager(cfg.Auth),
	}
	s.mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.handleLogout)
	s.mux.HandleFunc("GET /api/v1/status", s.handleStatus)
	s.mux.HandleFunc("GET /api/v1/projects", s.handleListProjects)
	s.mux.HandleFunc("POST /api/v1/projects", s.handleCreateProject)
	s.mux.HandleFunc("GET /api/v1/projects/{id}", s.handleGetProject)
	s.mux.HandleFunc("PATCH /api/v1/projects/{id}", s.handlePatchProject)
	s.mux.HandleFunc("POST /api/v1/projects/{id}/ssh/test", s.handleProjectSSHTest)
	s.mux.HandleFunc("POST /api/v1/projects/{id}/ssh/list-dirs", s.handleProjectListDirs)
	s.mux.HandleFunc("POST /api/v1/ssh/test", s.handleDraftSSHTest)
	s.mux.HandleFunc("POST /api/v1/ssh/list-dirs", s.handleDraftListDirs)
	s.mux.HandleFunc("GET /api/v1/projects/{id}/tasks", s.handleListProjectTasks)
	s.mux.HandleFunc("POST /api/v1/projects/{id}/tasks", s.handleCreateProjectTask)
	s.mux.HandleFunc("GET /api/v1/tasks/{id}", s.handleGetTask)
	s.mux.HandleFunc("PATCH /api/v1/tasks/{id}", s.handlePatchTask)
	s.mux.HandleFunc("GET /api/v1/runs", s.handleListRuns)
	s.mux.HandleFunc("GET /api/v1/runs/{id}/log", s.handleGetRunLog)
	s.mux.HandleFunc("GET /api/v1/runs/{id}", s.handleGetRun)
	return s
}

// Handler returns the HTTP handler with request-ID, auth, and logging middleware.
// Only /api/v1 routes are registered (no embedded UI).
func (s *Server) Handler() http.Handler {
	return s.wrapMiddleware(s.mux)
}

// HandlerWithUI returns the API plus embedded static Web UI at /.
func (s *Server) HandlerWithUI(ui http.Handler) http.Handler {
	root := http.NewServeMux()
	root.Handle("/api/", s.mux)
	if ui != nil {
		root.Handle("/", ui)
	}
	return s.wrapMiddleware(root)
}

func (s *Server) wrapMiddleware(h http.Handler) http.Handler {
	h = withAuth(s.cfg.Auth, s.cfg.Token, s.sessions, h)
	h = withLogging(s.cfg.Logger, h)
	h = withRequestID(h)
	return h
}

