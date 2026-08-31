package api

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"master-agent/internal/store"
)

const (
	defaultSecretsDir = "/secrets"
	projectKeyName    = "id_ed25519"
	maxKeyUploadBytes = 64 << 10 // 64 KiB — private keys are small
	keyDirPerm        = 0o700
	keyFilePerm       = 0o600
)

type keyStatusJSON struct {
	Present bool `json:"present"`
}

func (s *Server) secretsDir() string {
	if s.cfg.SecretsDir != "" {
		return s.cfg.SecretsDir
	}
	return defaultSecretsDir
}

func projectKeyPath(secretsDir, projectID string) string {
	return filepath.Join(secretsDir, "projects", projectID, projectKeyName)
}

func (s *Server) handleGetProjectKey(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, keyStatusJSON{Present: projectKeyPresent(s.secretsDir(), p)})
}

func (s *Server) handleUploadProjectKey(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(maxKeyUploadBytes); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, _, err := r.FormFile("key")
	if err != nil {
		WriteError(w, http.StatusBadRequest, "key file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxKeyUploadBytes+1))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "read key failed")
		return
	}
	if len(data) == 0 {
		WriteError(w, http.StatusBadRequest, "key file is empty")
		return
	}
	if len(data) > maxKeyUploadBytes {
		WriteError(w, http.StatusBadRequest, "key file is too large")
		return
	}

	keyPath, err := writeProjectKey(s.secretsDir(), id, data)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "write key failed")
		return
	}

	p.SSHKeyPath = keyPath
	if err := s.cfg.Store.UpdateProject(p); err != nil {
		WriteError(w, http.StatusInternalServerError, "update project failed")
		return
	}

	writeJSON(w, http.StatusOK, keyStatusJSON{Present: true})
}

func writeProjectKey(secretsDir, projectID string, data []byte) (string, error) {
	keyPath := projectKeyPath(secretsDir, projectID)
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, keyDirPerm); err != nil {
		return "", err
	}
	if err := os.WriteFile(keyPath, data, keyFilePerm); err != nil {
		return "", err
	}
	// Ensure permissions even when overwriting an existing file.
	if err := os.Chmod(keyPath, keyFilePerm); err != nil {
		return "", err
	}
	return keyPath, nil
}

func projectKeyPresent(secretsDir string, p *store.Project) bool {
	candidates := []string{
		projectKeyPath(secretsDir, p.ID),
	}
	if path := strings.TrimSpace(p.SSHKeyPath); path != "" {
		candidates = append(candidates, path)
	}
	for _, path := range candidates {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() && info.Size() > 0 {
			return true
		}
	}
	return false
}
