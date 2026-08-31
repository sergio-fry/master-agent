package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrLocked is returned when a project already has an active lock.
var ErrLocked = errors.New("project locked")

// ProcessLostMessage is stored on runs cleared by stale lock recovery.
const ProcessLostMessage = "process lost"

// AcquireRunLock inserts a running run and project lock in one transaction.
// If the project is already locked, returns ErrLocked and does not create a run.
func (s *Store) AcquireRunLock(projectID, taskID string, pid *int) (*Run, *Lock, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, fmt.Errorf("begin acquire: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var existing string
	err = tx.QueryRow(`SELECT project_id FROM locks WHERE project_id = ?`, projectID).Scan(&existing)
	switch {
	case err == nil:
		return nil, nil, ErrLocked
	case !errors.Is(err, sql.ErrNoRows):
		return nil, nil, fmt.Errorf("check lock: %w", err)
	}

	run := &Run{
		ID:        newID(),
		TaskID:    taskID,
		ProjectID: projectID,
		StartedAt: nowISO(),
		Status:    RunStatusRunning,
	}
	_, err = tx.Exec(`
		INSERT INTO runs (
			id, task_id, project_id, started_at, finished_at, exit_code,
			status, error_message, log_path
		) VALUES (?, ?, ?, ?, NULL, NULL, ?, NULL, NULL)`,
		run.ID, run.TaskID, run.ProjectID, run.StartedAt, run.Status,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert run: %w", err)
	}

	lock := &Lock{
		ProjectID:  projectID,
		TaskID:     taskID,
		RunID:      run.ID,
		PID:        pid,
		AcquiredAt: nowISO(),
	}
	_, err = tx.Exec(`
		INSERT INTO locks (project_id, task_id, run_id, pid, acquired_at)
		VALUES (?, ?, ?, ?, ?)`,
		lock.ProjectID, lock.TaskID, lock.RunID, nullInt(lock.PID), lock.AcquiredAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert lock: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit acquire: %w", err)
	}
	return run, lock, nil
}

// ReleaseRunLock updates the run lifecycle fields and deletes the project lock.
// Safe to call on success or error paths; missing lock is not an error.
func (s *Store) ReleaseRunLock(projectID string, run *Run) error {
	if run == nil {
		return fmt.Errorf("release: run is nil")
	}
	if run.FinishedAt == nil || *run.FinishedAt == "" {
		now := nowISO()
		run.FinishedAt = &now
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin release: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`
		UPDATE runs SET
			finished_at = ?, exit_code = ?, status = ?, error_message = ?, log_path = ?
		WHERE id = ?`,
		nullString(run.FinishedAt), nullInt(run.ExitCode), run.Status,
		nullString(run.ErrorMessage), nullString(run.LogPath), run.ID,
	)
	if err != nil {
		return fmt.Errorf("update run: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update run rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}

	if _, err := tx.Exec(`DELETE FROM locks WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("delete lock: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit release: %w", err)
	}
	return nil
}

// ListLocks returns all project locks.
func (s *Store) ListLocks() ([]Lock, error) {
	rows, err := s.db.Query(`
		SELECT project_id, task_id, run_id, pid, acquired_at
		FROM locks ORDER BY acquired_at, project_id`)
	if err != nil {
		return nil, fmt.Errorf("list locks: %w", err)
	}
	defer rows.Close()

	var out []Lock
	for rows.Next() {
		var l Lock
		var pid sql.NullInt64
		if err := rows.Scan(&l.ProjectID, &l.TaskID, &l.RunID, &pid, &l.AcquiredAt); err != nil {
			return nil, fmt.Errorf("scan lock: %w", err)
		}
		l.PID = fromNullInt(pid)
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list locks rows: %w", err)
	}
	return out, nil
}

// ClearStaleLock marks the lock's run as error and deletes the lock in one transaction.
func (s *Store) ClearStaleLock(projectID, errorMessage string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin clear stale: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var runID string
	err = tx.QueryRow(`SELECT run_id FROM locks WHERE project_id = ?`, projectID).Scan(&runID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get stale lock: %w", err)
	}

	finished := nowISO()
	msg := errorMessage
	if msg == "" {
		msg = ProcessLostMessage
	}
	_, err = tx.Exec(`
		UPDATE runs SET finished_at = ?, status = ?, error_message = ?
		WHERE id = ?`,
		finished, RunStatusError, msg, runID,
	)
	if err != nil {
		return fmt.Errorf("mark stale run: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM locks WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("delete stale lock: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit clear stale: %w", err)
	}
	return nil
}
