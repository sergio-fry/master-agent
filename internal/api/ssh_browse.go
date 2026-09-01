package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"master-agent/internal/runner"
	"master-agent/internal/store"
)

type listDirsRequest struct {
	Path          *string `json:"path"`
	SSHHost       *string `json:"ssh_host"`
	SSHUser       *string `json:"ssh_user"`
	SSHPort       *int    `json:"ssh_port"`
	SSHPrivateKey *string `json:"ssh_private_key"`
}

func (s *Server) handleProjectListDirs(w http.ResponseWriter, r *http.Request) {
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

	var req listDirsRequest
	if r.ContentLength != 0 {
		if err := decodeJSONBody(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	applyListDirsOverrides(p, req)
	dirPath := ""
	if req.Path != nil {
		dirPath = strings.TrimSpace(*req.Path)
	}
	s.listDirsWithPath(w, r, *p, dirPath)
}

func (s *Server) handleDraftListDirs(w http.ResponseWriter, r *http.Request) {
	var req listDirsRequest
	if err := decodeJSONBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	p := store.Project{SSHPort: 22, Path: "/"}
	if err := applyDraftListDirs(&p, req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	dirPath := ""
	if req.Path != nil {
		dirPath = strings.TrimSpace(*req.Path)
	}
	s.listDirsWithPath(w, r, p, dirPath)
}

func applyListDirsOverrides(p *store.Project, req listDirsRequest) {
	if req.SSHHost != nil {
		p.SSHHost = strings.TrimSpace(*req.SSHHost)
	}
	if req.SSHUser != nil {
		p.SSHUser = strings.TrimSpace(*req.SSHUser)
	}
	if req.SSHPort != nil {
		p.SSHPort = *req.SSHPort
	}
	if req.SSHPrivateKey != nil {
		p.SSHPrivateKey = store.NormalizeSSHPrivateKey(*req.SSHPrivateKey)
	}
}

func applyDraftListDirs(p *store.Project, req listDirsRequest) error {
	if req.SSHHost == nil || strings.TrimSpace(*req.SSHHost) == "" {
		return errors.New("ssh_host is required")
	}
	if req.SSHUser == nil || strings.TrimSpace(*req.SSHUser) == "" {
		return errors.New("ssh_user is required")
	}
	if req.SSHPrivateKey == nil {
		return errors.New("ssh_private_key is required")
	}
	p.SSHHost = strings.TrimSpace(*req.SSHHost)
	p.SSHUser = strings.TrimSpace(*req.SSHUser)
	p.SSHPrivateKey = store.NormalizeSSHPrivateKey(*req.SSHPrivateKey)
	if req.SSHPort != nil {
		if *req.SSHPort <= 0 {
			return errors.New("ssh_port must be positive")
		}
		p.SSHPort = *req.SSHPort
	}
	return store.ValidateSSHPrivateKey(p.SSHPrivateKey)
}

func (s *Server) listDirsWithPath(w http.ResponseWriter, r *http.Request, project store.Project, dirPath string) {
	listFn := s.cfg.ListDirs
	if listFn == nil {
		listFn = func(ctx context.Context, project store.Project, path string) (runner.RemoteDirListing, error) {
			return (&runner.SSHBrowser{}).ListDirs(ctx, project, path)
		}
	}
	result, err := listFn(r.Context(), project, dirPath)
	if err != nil {
		writeSSHTestError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
