package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"master-agent/internal/store"
)

// taskJSON is the API representation of a Task.
// Tasks have no SSH fields — connection settings come from the Project.
type taskJSON struct {
	ID              string  `json:"id"`
	ProjectID       string  `json:"project_id"`
	Name            string  `json:"name"`
	Prompt          string  `json:"prompt"`
	Command         string  `json:"command"`
	IntervalSeconds int     `json:"interval_seconds"`
	Enabled         bool    `json:"enabled"`
	LastRunAt       *string `json:"last_run_at"`
	NextRunAt       *string `json:"next_run_at"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type createTaskRequest struct {
	Name            string `json:"name"`
	Prompt          string `json:"prompt"`
	Command         string `json:"command"`
	IntervalSeconds *int   `json:"interval_seconds"`
	Enabled         *bool  `json:"enabled"`
}

type patchTaskRequest struct {
	Name            *string `json:"name"`
	Prompt          *string `json:"prompt"`
	Command         *string `json:"command"`
	IntervalSeconds *int    `json:"interval_seconds"`
	Enabled         *bool   `json:"enabled"`
}

var taskSSHFieldNames = []string{
	"ssh_host", "ssh_user", "ssh_port", "ssh_key_path",
	"private_key", "ssh_private_key", "key",
}

func taskToJSON(t *store.Task) taskJSON {
	return taskJSON{
		ID:              t.ID,
		ProjectID:       t.ProjectID,
		Name:            t.Name,
		Prompt:          t.Prompt,
		Command:         t.Command,
		IntervalSeconds: t.IntervalSeconds,
		Enabled:         t.Enabled,
		LastRunAt:       t.LastRunAt,
		NextRunAt:       t.NextRunAt,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}

func (s *Server) handleListProjectTasks(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}
	projectID := r.PathValue("id")
	if projectID == "" {
		WriteError(w, http.StatusBadRequest, "project id is required")
		return
	}
	if _, err := s.cfg.Store.GetProject(projectID); errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "project not found")
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "get project failed")
		return
	}

	tasks, err := s.cfg.Store.ListTasks(projectID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "list tasks failed")
		return
	}
	out := make([]taskJSON, 0, len(tasks))
	for i := range tasks {
		out = append(out, taskToJSON(&tasks[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateProjectTask(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}
	projectID := r.PathValue("id")
	if projectID == "" {
		WriteError(w, http.StatusBadRequest, "project id is required")
		return
	}
	if _, err := s.cfg.Store.GetProject(projectID); errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "project not found")
		return
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "get project failed")
		return
	}

	body, err := readLimitedBody(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := rejectTaskSSHFields(body); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req createTaskRequest
	if err := decodeJSONBytes(body, &req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.Command = strings.TrimSpace(req.Command)

	if err := validateCreateTask(req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	t := &store.Task{
		ProjectID:       projectID,
		Name:            req.Name,
		Prompt:          req.Prompt,
		Command:         req.Command,
		IntervalSeconds: *req.IntervalSeconds,
		Enabled:         enabled,
	}
	if err := s.cfg.Store.CreateTask(t); err != nil {
		WriteError(w, http.StatusInternalServerError, "create task failed")
		return
	}
	writeJSON(w, http.StatusCreated, taskToJSON(t))
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "task id is required")
		return
	}
	t, err := s.cfg.Store.GetTask(id)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "get task failed")
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func (s *Server) handlePatchTask(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "task id is required")
		return
	}

	body, err := readLimitedBody(r)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := rejectTaskSSHFields(body); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req patchTaskRequest
	if err := decodeJSONBytes(body, &req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	t, err := s.cfg.Store.GetTask(id)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "get task failed")
		return
	}

	if err := applyTaskPatch(t, req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.cfg.Store.UpdateTask(t); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "task not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "update task failed")
		return
	}
	writeJSON(w, http.StatusOK, taskToJSON(t))
}

func validateCreateTask(req createTaskRequest) error {
	switch {
	case req.Name == "":
		return errors.New("name is required")
	case req.Prompt == "":
		return errors.New("prompt is required")
	case req.Command == "":
		return errors.New("command is required")
	case req.IntervalSeconds == nil:
		return errors.New("interval_seconds is required")
	case *req.IntervalSeconds <= 0:
		return errors.New("interval_seconds must be positive")
	}
	return nil
}

func applyTaskPatch(t *store.Task, req patchTaskRequest) error {
	if req.Name == nil && req.Prompt == nil && req.Command == nil &&
		req.IntervalSeconds == nil && req.Enabled == nil {
		return errors.New("no fields to update")
	}
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		if v == "" {
			return errors.New("name must not be empty")
		}
		t.Name = v
	}
	if req.Prompt != nil {
		v := strings.TrimSpace(*req.Prompt)
		if v == "" {
			return errors.New("prompt must not be empty")
		}
		t.Prompt = v
	}
	if req.Command != nil {
		v := strings.TrimSpace(*req.Command)
		if v == "" {
			return errors.New("command must not be empty")
		}
		t.Command = v
	}
	if req.IntervalSeconds != nil {
		if *req.IntervalSeconds <= 0 {
			return errors.New("interval_seconds must be positive")
		}
		t.IntervalSeconds = *req.IntervalSeconds
	}
	if req.Enabled != nil {
		t.Enabled = *req.Enabled
	}
	return nil
}

func rejectTaskSSHFields(body []byte) error {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil // typed decode will report invalid JSON
	}
	for _, name := range taskSSHFieldNames {
		if _, ok := raw[name]; ok {
			return errors.New("ssh fields are not allowed on tasks")
		}
	}
	return nil
}

func readLimitedBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, errors.New("failed to read request body")
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, errors.New("request body is required")
	}
	return body, nil
}

func decodeJSONBytes(body []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid JSON body")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid JSON body")
	}
	return nil
}
