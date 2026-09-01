package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"master-agent/internal/store"
)

// projectJSON is the API representation of a Project for list/create/patch responses.
// SSH private key material is never returned on list or mutating responses.
type projectJSON struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	SSHHost       string `json:"ssh_host"`
	SSHUser       string `json:"ssh_user"`
	SSHPort       int    `json:"ssh_port"`
	KeyConfigured bool   `json:"key_configured"`
	HostKeyPinned bool   `json:"host_key_pinned"`
	Enabled       bool   `json:"enabled"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// projectDetailJSON is the single-project GET payload including the stored private key for edit forms.
type projectDetailJSON struct {
	projectJSON
	SSHPrivateKey string `json:"ssh_private_key"`
}

type createProjectRequest struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	SSHHost       string `json:"ssh_host"`
	SSHUser       string `json:"ssh_user"`
	SSHPort       *int   `json:"ssh_port"`
	SSHPrivateKey string `json:"ssh_private_key"`
	Enabled       *bool  `json:"enabled"`
}

type patchProjectRequest struct {
	Name          *string `json:"name"`
	Path          *string `json:"path"`
	SSHHost       *string `json:"ssh_host"`
	SSHUser       *string `json:"ssh_user"`
	SSHPort       *int    `json:"ssh_port"`
	SSHPrivateKey *string `json:"ssh_private_key"`
	Enabled       *bool   `json:"enabled"`
}

func projectToJSON(p *store.Project) projectJSON {
	return projectJSON{
		ID:            p.ID,
		Name:          p.Name,
		Path:          p.Path,
		SSHHost:       p.SSHHost,
		SSHUser:       p.SSHUser,
		SSHPort:       p.SSHPort,
		KeyConfigured: p.KeyConfigured(),
		HostKeyPinned: p.HostKeyPinned(),
		Enabled:       p.Enabled,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func projectToDetailJSON(p *store.Project) projectDetailJSON {
	return projectDetailJSON{
		projectJSON:   projectToJSON(p),
		SSHPrivateKey: p.SSHPrivateKey,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}
	projects, err := s.cfg.Store.ListProjects()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "list projects failed")
		return
	}
	out := make([]projectJSON, 0, len(projects))
	for i := range projects {
		out = append(out, projectToJSON(&projects[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}

	var req createProjectRequest
	if err := decodeJSONBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Path = strings.TrimSpace(req.Path)
	req.SSHHost = strings.TrimSpace(req.SSHHost)
	req.SSHUser = strings.TrimSpace(req.SSHUser)
	req.SSHPrivateKey = store.NormalizeSSHPrivateKey(req.SSHPrivateKey)

	if err := validateCreateProject(req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	port := 22
	if req.SSHPort != nil {
		port = *req.SSHPort
	}

	p := &store.Project{
		Name:          req.Name,
		Path:          req.Path,
		SSHHost:       req.SSHHost,
		SSHUser:       req.SSHUser,
		SSHPort:       port,
		SSHPrivateKey: req.SSHPrivateKey,
		Enabled:       enabled,
	}
	if err := s.cfg.Store.CreateProject(p); err != nil {
		WriteError(w, http.StatusInternalServerError, "create project failed")
		return
	}
	writeJSON(w, http.StatusCreated, projectToJSON(p))
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "project id is required")
		return
	}
	p, err := s.cfg.Store.GetProject(id)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "get project failed")
		return
	}
	writeJSON(w, http.StatusOK, projectToDetailJSON(p))
}

func (s *Server) handlePatchProject(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Store == nil {
		WriteError(w, http.StatusInternalServerError, "store not configured")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "project id is required")
		return
	}

	var req patchProjectRequest
	if err := decodeJSONBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	p, err := s.cfg.Store.GetProject(id)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "project not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "get project failed")
		return
	}

	if err := applyProjectPatch(p, req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.cfg.Store.UpdateProject(p); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "project not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "update project failed")
		return
	}
	writeJSON(w, http.StatusOK, projectToJSON(p))
}

func validateCreateProject(req createProjectRequest) error {
	switch {
	case req.Name == "":
		return errors.New("name is required")
	case req.Path == "":
		return errors.New("path is required")
	case req.SSHHost == "":
		return errors.New("ssh_host is required")
	case req.SSHUser == "":
		return errors.New("ssh_user is required")
	}
	if err := store.ValidateSSHPrivateKey(req.SSHPrivateKey); err != nil {
		return err
	}
	if req.SSHPort != nil && *req.SSHPort <= 0 {
		return errors.New("ssh_port must be positive")
	}
	return nil
}

func applyProjectPatch(p *store.Project, req patchProjectRequest) error {
	if req.Name == nil && req.Path == nil && req.SSHHost == nil && req.SSHUser == nil &&
		req.SSHPort == nil && req.SSHPrivateKey == nil && req.Enabled == nil {
		return errors.New("no fields to update")
	}
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		if v == "" {
			return errors.New("name must not be empty")
		}
		p.Name = v
	}
	if req.Path != nil {
		v := strings.TrimSpace(*req.Path)
		if v == "" {
			return errors.New("path must not be empty")
		}
		p.Path = v
	}
	if req.SSHHost != nil {
		v := strings.TrimSpace(*req.SSHHost)
		if v == "" {
			return errors.New("ssh_host must not be empty")
		}
		p.SSHHost = v
	}
	if req.SSHUser != nil {
		v := strings.TrimSpace(*req.SSHUser)
		if v == "" {
			return errors.New("ssh_user must not be empty")
		}
		p.SSHUser = v
	}
	if req.SSHPrivateKey != nil {
		v := store.NormalizeSSHPrivateKey(*req.SSHPrivateKey)
		if err := store.ValidateSSHPrivateKey(v); err != nil {
			return err
		}
		p.SSHPrivateKey = v
	}
	if req.SSHPort != nil {
		if *req.SSHPort <= 0 {
			return errors.New("ssh_port must be positive")
		}
		p.SSHPort = *req.SSHPort
	}
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}
	return nil
}

func decodeJSONBody(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is required")
		}
		return errors.New("invalid JSON body")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("invalid JSON body")
	}
	return nil
}
