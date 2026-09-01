package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"master-agent/internal/runner"
	"master-agent/internal/store"
)

type sshTestRequest struct {
	Path          *string `json:"path"`
	SSHHost       *string `json:"ssh_host"`
	SSHUser       *string `json:"ssh_user"`
	SSHPort       *int    `json:"ssh_port"`
	SSHPrivateKey *string `json:"ssh_private_key"`
}

type sshTestResponse struct {
	OK                 bool   `json:"ok"`
	HostKeyType        string `json:"host_key_type"`
	HostKeyFingerprint string `json:"host_key_fingerprint"`
	HostKeyPinned      bool   `json:"host_key_pinned"`
}

func (s *Server) handleProjectSSHTest(w http.ResponseWriter, r *http.Request) {
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

	var req sshTestRequest
	if r.ContentLength != 0 {
		if err := decodeJSONBody(r, &req); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	applySSHTestOverrides(p, req)

	s.runSSHTest(w, r, *p, true)
}

func (s *Server) handleDraftSSHTest(w http.ResponseWriter, r *http.Request) {
	var req sshTestRequest
	if err := decodeJSONBody(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	p := store.Project{SSHPort: 22}
	if err := applyDraftSSHTest(&p, req); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.runSSHTest(w, r, p, false)
}

func applySSHTestOverrides(p *store.Project, req sshTestRequest) {
	if req.Path != nil {
		p.Path = strings.TrimSpace(*req.Path)
	}
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

func applyDraftSSHTest(p *store.Project, req sshTestRequest) error {
	if req.Path == nil || strings.TrimSpace(*req.Path) == "" {
		return errors.New("path is required")
	}
	if req.SSHHost == nil || strings.TrimSpace(*req.SSHHost) == "" {
		return errors.New("ssh_host is required")
	}
	if req.SSHUser == nil || strings.TrimSpace(*req.SSHUser) == "" {
		return errors.New("ssh_user is required")
	}
	if req.SSHPrivateKey == nil {
		return errors.New("ssh_private_key is required")
	}
	p.Path = strings.TrimSpace(*req.Path)
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

func (s *Server) runSSHTest(w http.ResponseWriter, r *http.Request, project store.Project, persist bool) {
	testFn := s.cfg.SSHTest
	if testFn == nil {
		testFn = func(ctx context.Context, project store.Project) (runner.SSHTestResult, error) {
			return (&runner.SSHTester{}).Test(ctx, project)
		}
	}
	result, err := testFn(r.Context(), project)
	if err != nil {
		writeSSHTestError(w, err)
		return
	}

	hostKeyPinned := project.HostKeyPinned() || result.NewlyPinned
	if persist && result.NewlyPinned {
		project.SSHHostKey = result.HostKeyPublic
		if err := s.cfg.Store.UpdateProject(&project); err != nil {
			WriteError(w, http.StatusInternalServerError, "persist host key failed")
			return
		}
		hostKeyPinned = true
	}

	writeJSON(w, http.StatusOK, sshTestResponse{
		OK:                 true,
		HostKeyType:        result.HostKeyType,
		HostKeyFingerprint: result.HostKeyFingerprint,
		HostKeyPinned:      hostKeyPinned,
	})
}

func writeSSHTestError(w http.ResponseWriter, err error) {
	var sshErr *runner.SSHTestError
	if errors.As(err, &sshErr) {
		status := http.StatusBadGateway
		switch sshErr.Code {
		case runner.SSHCodeHostKeyMismatch, runner.SSHCodePathNotFound:
			status = http.StatusUnprocessableEntity
		case runner.SSHCodeAuthFailed:
			status = http.StatusUnauthorized
		}
		WriteErrorCode(w, status, sshErr.Message, sshErr.Code)
		return
	}
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	WriteError(w, http.StatusBadGateway, "ssh test failed")
}
