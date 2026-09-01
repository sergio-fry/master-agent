package api

import (
	"net/http"

	"master-agent/internal/store"
)

type statusLockJSON struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name,omitempty"`
	TaskID      string `json:"task_id"`
	TaskName    string `json:"task_name,omitempty"`
	RunID       string `json:"run_id"`
	AcquiredAt  string `json:"acquired_at"`
	PID         *int   `json:"pid,omitempty"`
}

type statusResponse struct {
	OK         bool             `json:"ok"`
	DBPath     string           `json:"db_path,omitempty"`
	DBOK       bool             `json:"db_ok"`
	LockActive bool             `json:"lock_active"`
	Locks      []statusLockJSON `json:"locks"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	resp := statusResponse{
		OK:    true,
		Locks: []statusLockJSON{},
	}
	if s.cfg.DBPath != "" {
		resp.DBPath = s.cfg.DBPath
	}

	if s.cfg.Store == nil {
		writeJSON(w, http.StatusOK, resp)
		return
	}

	if err := s.cfg.Store.Ping(); err != nil {
		writeJSON(w, http.StatusOK, resp)
		return
	}
	resp.DBOK = true

	locks, err := s.cfg.Store.ListLocks()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "list locks failed")
		return
	}

	for _, l := range locks {
		resp.Locks = append(resp.Locks, lockToStatusJSON(s.cfg.Store, l))
	}
	resp.LockActive = len(resp.Locks) > 0

	writeJSON(w, http.StatusOK, resp)
}

func lockToStatusJSON(st *store.Store, l store.Lock) statusLockJSON {
	out := statusLockJSON{
		ProjectID:  l.ProjectID,
		TaskID:     l.TaskID,
		RunID:      l.RunID,
		AcquiredAt: l.AcquiredAt,
		PID:        l.PID,
	}
	if p, err := st.GetProject(l.ProjectID); err == nil {
		out.ProjectName = p.Name
	}
	if t, err := st.GetTask(l.TaskID); err == nil {
		out.TaskName = t.Name
	}
	return out
}
