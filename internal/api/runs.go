package api

import (
	"errors"
	"io"
	"net/http"
	"os"

	"master-agent/internal/store"
)

const maxRunLogBytes = 1 << 20 // 1MB tail/limit for run log responses

// runJSON is the API representation of a Run.
type runJSON struct {
	ID           string  `json:"id"`
	TaskID       string  `json:"task_id"`
	ProjectID    string  `json:"project_id"`
	StartedAt    string  `json:"started_at"`
	FinishedAt   *string `json:"finished_at"`
	ExitCode     *int    `json:"exit_code"`
	Status       string  `json:"status"`
	ErrorMessage *string `json:"error_message"`
	LogPath      *string `json:"log_path"`
}

func runToJSON(r *store.Run) runJSON {
	return runJSON{
		ID:           r.ID,
		TaskID:       r.TaskID,
		ProjectID:    r.ProjectID,
		StartedAt:    r.StartedAt,
		FinishedAt:   r.FinishedAt,
		ExitCode:     r.ExitCode,
		Status:       r.Status,
		ErrorMessage: r.ErrorMessage,
		LogPath:      r.LogPath,
	}
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}

	projectID := r.URL.Query().Get("project_id")
	taskID := r.URL.Query().Get("task_id")
	status := r.URL.Query().Get("status")
	if status != "" && !validRunStatus(status) {
		WriteError(w, http.StatusBadRequest, "invalid status")
		return
	}

	runs, err := s.cfg.Store.ListRunsFilter(store.RunFilter{
		ProjectID: projectID,
		TaskID:    taskID,
		Status:    status,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "list runs failed")
		return
	}

	out := make([]runJSON, 0, len(runs))
	for i := range runs {
		out = append(out, runToJSON(&runs[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "run id is required")
		return
	}

	run, err := s.cfg.Store.GetRun(id)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "run not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "get run failed")
		return
	}
	writeJSON(w, http.StatusOK, runToJSON(run))
}

func (s *Server) handleGetRunLog(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "run id is required")
		return
	}

	run, err := s.cfg.Store.GetRun(id)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "run not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "get run failed")
		return
	}
	if run.LogPath == nil || *run.LogPath == "" {
		WriteError(w, http.StatusNotFound, "log not found")
		return
	}

	f, err := os.Open(*run.LogPath)
	if err != nil {
		if os.IsNotExist(err) {
			WriteError(w, http.StatusNotFound, "log not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "read log failed")
		return
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxRunLogBytes))
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "read log failed")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func validRunStatus(status string) bool {
	switch status {
	case store.RunStatusRunning, store.RunStatusSuccess, store.RunStatusError:
		return true
	default:
		return false
	}
}
