package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ListDueTasks returns enabled tasks on enabled projects that are due at or before now.
// Ordering: NULL next_run_at first (first run), then next_run_at ASC, then id.
func (s *Store) ListDueTasks(now time.Time) ([]Task, error) {
	nowStr := now.UTC().Format(time.RFC3339Nano)
	rows, err := s.db.Query(`
		SELECT t.id, t.project_id, t.name, t.prompt, t.command, t.interval_seconds,
			t.enabled, t.last_run_at, t.next_run_at, t.created_at, t.updated_at
		FROM tasks t
		INNER JOIN projects p ON p.id = t.project_id
		WHERE t.enabled = 1
			AND p.enabled = 1
			AND (t.next_run_at IS NULL OR t.next_run_at <= ?)
		ORDER BY
			CASE WHEN t.next_run_at IS NULL THEN 0 ELSE 1 END,
			t.next_run_at ASC,
			t.id ASC`, nowStr)
	if err != nil {
		return nil, fmt.Errorf("list due tasks: %w", err)
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		var enabled int
		var lastRun, nextRun sql.NullString
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Name, &t.Prompt, &t.Command, &t.IntervalSeconds,
			&enabled, &lastRun, &nextRun, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan due task: %w", err)
		}
		t.Enabled = intToBool(enabled)
		t.LastRunAt = fromNullString(lastRun)
		t.NextRunAt = fromNullString(nextRun)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list due tasks rows: %w", err)
	}
	return out, nil
}

// HasAnyLock reports whether any project lock exists (MVP global single-run gate).
func (s *Store) HasAnyLock() (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM locks`).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("count locks: %w", err)
	}
	return n > 0, nil
}

// ScheduleTaskAfterRun sets last_run_at and next_run_at = finishedAt + interval_seconds.
func (s *Store) ScheduleTaskAfterRun(taskID string, finishedAt time.Time, intervalSeconds int) error {
	finished := finishedAt.UTC()
	last := finished.Format(time.RFC3339Nano)
	next := finished.Add(time.Duration(intervalSeconds) * time.Second).Format(time.RFC3339Nano)
	updated := nowISO()

	res, err := s.db.Exec(`
		UPDATE tasks SET last_run_at = ?, next_run_at = ?, updated_at = ?
		WHERE id = ?`, last, next, updated, taskID)
	if err != nil {
		return fmt.Errorf("schedule task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("schedule task rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// LatestRunForTask returns the most recently started run for a task.
func (s *Store) LatestRunForTask(taskID string) (*Run, error) {
	row := s.db.QueryRow(`
		SELECT id, task_id, project_id, started_at, finished_at, exit_code,
			status, error_message, log_path
		FROM runs WHERE task_id = ?
		ORDER BY started_at DESC, id DESC
		LIMIT 1`, taskID)

	var r Run
	var finished, errMsg, logPath sql.NullString
	var exitCode sql.NullInt64
	err := row.Scan(
		&r.ID, &r.TaskID, &r.ProjectID, &r.StartedAt,
		&finished, &exitCode, &r.Status, &errMsg, &logPath,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("latest run for task: %w", err)
	}
	r.FinishedAt = fromNullString(finished)
	r.ExitCode = fromNullInt(exitCode)
	r.ErrorMessage = fromNullString(errMsg)
	r.LogPath = fromNullString(logPath)
	return &r, nil
}
