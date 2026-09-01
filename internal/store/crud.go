package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// CreateProject inserts a project. ID and timestamps are set if empty.
func (s *Store) CreateProject(p *Project) error {
	if p.ID == "" {
		p.ID = newID()
	}
	now := nowISO()
	if p.CreatedAt == "" {
		p.CreatedAt = now
	}
	if p.UpdatedAt == "" {
		p.UpdatedAt = now
	}
	if p.SSHPort == 0 {
		p.SSHPort = 22
	}

	_, err := s.db.Exec(`
		INSERT INTO projects (
			id, name, path, ssh_host, ssh_user, ssh_port, ssh_private_key, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Path, p.SSHHost, p.SSHUser, p.SSHPort, p.SSHPrivateKey,
		boolToInt(p.Enabled), p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}
	return nil
}

// GetProject returns a project by id.
func (s *Store) GetProject(id string) (*Project, error) {
	row := s.db.QueryRow(`
		SELECT id, name, path, ssh_host, ssh_user, ssh_port, ssh_private_key, enabled, created_at, updated_at
		FROM projects WHERE id = ?`, id)

	var p Project
	var enabled int
	err := row.Scan(
		&p.ID, &p.Name, &p.Path, &p.SSHHost, &p.SSHUser, &p.SSHPort, &p.SSHPrivateKey,
		&enabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	p.Enabled = intToBool(enabled)
	return &p, nil
}

// UpdateProject updates mutable project fields and bumps updated_at.
func (s *Store) UpdateProject(p *Project) error {
	p.UpdatedAt = nowISO()
	res, err := s.db.Exec(`
		UPDATE projects SET
			name = ?, path = ?, ssh_host = ?, ssh_user = ?, ssh_port = ?,
			ssh_private_key = ?, enabled = ?, updated_at = ?
		WHERE id = ?`,
		p.Name, p.Path, p.SSHHost, p.SSHUser, p.SSHPort, p.SSHPrivateKey,
		boolToInt(p.Enabled), p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update project rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListProjects returns all projects ordered by name.
func (s *Store) ListProjects() ([]Project, error) {
	rows, err := s.db.Query(`
		SELECT id, name, path, ssh_host, ssh_user, ssh_port, ssh_private_key, enabled, created_at, updated_at
		FROM projects ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		var enabled int
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Path, &p.SSHHost, &p.SSHUser, &p.SSHPort, &p.SSHPrivateKey,
			&enabled, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		p.Enabled = intToBool(enabled)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list projects rows: %w", err)
	}
	return out, nil
}

// GetProjectByName returns a project by exact name.
// If multiple rows match, returns an error.
func (s *Store) GetProjectByName(name string) (*Project, error) {
	rows, err := s.db.Query(`
		SELECT id, name, path, ssh_host, ssh_user, ssh_port, ssh_private_key, enabled, created_at, updated_at
		FROM projects WHERE name = ? ORDER BY created_at, id`, name)
	if err != nil {
		return nil, fmt.Errorf("get project by name: %w", err)
	}
	defer rows.Close()

	var matches []Project
	for rows.Next() {
		var p Project
		var enabled int
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Path, &p.SSHHost, &p.SSHUser, &p.SSHPort, &p.SSHPrivateKey,
			&enabled, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		p.Enabled = intToBool(enabled)
		matches = append(matches, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get project by name rows: %w", err)
	}
	switch len(matches) {
	case 0:
		return nil, ErrNotFound
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("multiple projects named %q; use id", name)
	}
}

// CreateTask inserts a task. ID and timestamps are set if empty.
func (s *Store) CreateTask(t *Task) error {
	if t.ID == "" {
		t.ID = newID()
	}
	now := nowISO()
	if t.CreatedAt == "" {
		t.CreatedAt = now
	}
	if t.UpdatedAt == "" {
		t.UpdatedAt = now
	}

	_, err := s.db.Exec(`
		INSERT INTO tasks (
			id, project_id, name, prompt, command, interval_seconds,
			enabled, last_run_at, next_run_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.Name, t.Prompt, t.Command, t.IntervalSeconds,
		boolToInt(t.Enabled), nullString(t.LastRunAt), nullString(t.NextRunAt),
		t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

// GetTask returns a task by id.
func (s *Store) GetTask(id string) (*Task, error) {
	row := s.db.QueryRow(`
		SELECT id, project_id, name, prompt, command, interval_seconds,
			enabled, last_run_at, next_run_at, created_at, updated_at
		FROM tasks WHERE id = ?`, id)

	var t Task
	var enabled int
	var lastRun, nextRun sql.NullString
	err := row.Scan(
		&t.ID, &t.ProjectID, &t.Name, &t.Prompt, &t.Command, &t.IntervalSeconds,
		&enabled, &lastRun, &nextRun, &t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	t.Enabled = intToBool(enabled)
	t.LastRunAt = fromNullString(lastRun)
	t.NextRunAt = fromNullString(nextRun)
	return &t, nil
}

// UpdateTask updates mutable task fields and bumps updated_at.
func (s *Store) UpdateTask(t *Task) error {
	t.UpdatedAt = nowISO()
	res, err := s.db.Exec(`
		UPDATE tasks SET
			project_id = ?, name = ?, prompt = ?, command = ?, interval_seconds = ?,
			enabled = ?, last_run_at = ?, next_run_at = ?, updated_at = ?
		WHERE id = ?`,
		t.ProjectID, t.Name, t.Prompt, t.Command, t.IntervalSeconds,
		boolToInt(t.Enabled), nullString(t.LastRunAt), nullString(t.NextRunAt),
		t.UpdatedAt, t.ID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update task rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListTasks returns tasks, optionally filtered by project ID, ordered by name.
func (s *Store) ListTasks(projectID string) ([]Task, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if projectID == "" {
		rows, err = s.db.Query(`
			SELECT id, project_id, name, prompt, command, interval_seconds,
				enabled, last_run_at, next_run_at, created_at, updated_at
			FROM tasks ORDER BY name, id`)
	} else {
		rows, err = s.db.Query(`
			SELECT id, project_id, name, prompt, command, interval_seconds,
				enabled, last_run_at, next_run_at, created_at, updated_at
			FROM tasks WHERE project_id = ? ORDER BY name, id`, projectID)
	}
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
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
			return nil, fmt.Errorf("scan task: %w", err)
		}
		t.Enabled = intToBool(enabled)
		t.LastRunAt = fromNullString(lastRun)
		t.NextRunAt = fromNullString(nextRun)
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tasks rows: %w", err)
	}
	return out, nil
}

// GetTaskByProjectAndName returns a task by project id and task name.
// If multiple rows match, returns an error.
func (s *Store) GetTaskByProjectAndName(projectID, name string) (*Task, error) {
	rows, err := s.db.Query(`
		SELECT id, project_id, name, prompt, command, interval_seconds,
			enabled, last_run_at, next_run_at, created_at, updated_at
		FROM tasks WHERE project_id = ? AND name = ? ORDER BY created_at, id`,
		projectID, name)
	if err != nil {
		return nil, fmt.Errorf("get task by name: %w", err)
	}
	defer rows.Close()

	var matches []Task
	for rows.Next() {
		var t Task
		var enabled int
		var lastRun, nextRun sql.NullString
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Name, &t.Prompt, &t.Command, &t.IntervalSeconds,
			&enabled, &lastRun, &nextRun, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		t.Enabled = intToBool(enabled)
		t.LastRunAt = fromNullString(lastRun)
		t.NextRunAt = fromNullString(nextRun)
		matches = append(matches, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get task by name rows: %w", err)
	}
	switch len(matches) {
	case 0:
		return nil, ErrNotFound
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("multiple tasks named %q on project; use id", name)
	}
}

// InsertLock acquires a project lock. Fails if a lock for the project already exists.
func (s *Store) InsertLock(l *Lock) error {
	if l.AcquiredAt == "" {
		l.AcquiredAt = nowISO()
	}
	_, err := s.db.Exec(`
		INSERT INTO locks (project_id, task_id, run_id, pid, acquired_at)
		VALUES (?, ?, ?, ?, ?)`,
		l.ProjectID, l.TaskID, l.RunID, nullInt(l.PID), l.AcquiredAt,
	)
	if err != nil {
		return fmt.Errorf("insert lock: %w", err)
	}
	return nil
}

// GetLock returns the lock for a project, if any.
func (s *Store) GetLock(projectID string) (*Lock, error) {
	row := s.db.QueryRow(`
		SELECT project_id, task_id, run_id, pid, acquired_at
		FROM locks WHERE project_id = ?`, projectID)

	var l Lock
	var pid sql.NullInt64
	err := row.Scan(&l.ProjectID, &l.TaskID, &l.RunID, &pid, &l.AcquiredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get lock: %w", err)
	}
	l.PID = fromNullInt(pid)
	return &l, nil
}

// DeleteLock releases the lock for a project.
func (s *Store) DeleteLock(projectID string) error {
	res, err := s.db.Exec(`DELETE FROM locks WHERE project_id = ?`, projectID)
	if err != nil {
		return fmt.Errorf("delete lock: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete lock rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// InsertRun creates a run record.
func (s *Store) InsertRun(r *Run) error {
	if r.ID == "" {
		r.ID = newID()
	}
	if r.StartedAt == "" {
		r.StartedAt = nowISO()
	}
	if r.Status == "" {
		r.Status = RunStatusRunning
	}

	_, err := s.db.Exec(`
		INSERT INTO runs (
			id, task_id, project_id, started_at, finished_at, exit_code,
			status, error_message, log_path
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.TaskID, r.ProjectID, r.StartedAt,
		nullString(r.FinishedAt), nullInt(r.ExitCode),
		r.Status, nullString(r.ErrorMessage), nullString(r.LogPath),
	)
	if err != nil {
		return fmt.Errorf("insert run: %w", err)
	}
	return nil
}

// GetRun returns a run by id.
func (s *Store) GetRun(id string) (*Run, error) {
	row := s.db.QueryRow(`
		SELECT id, task_id, project_id, started_at, finished_at, exit_code,
			status, error_message, log_path
		FROM runs WHERE id = ?`, id)

	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}
	return r, nil
}

// ListRuns returns runs for a project, optionally filtered by task_id.
// Results are ordered by started_at descending (newest first).
func (s *Store) ListRuns(projectID, taskID string) ([]Run, error) {
	if projectID == "" {
		return nil, fmt.Errorf("list runs: project id is required")
	}

	var (
		rows *sql.Rows
		err  error
	)
	const cols = `id, task_id, project_id, started_at, finished_at, exit_code,
			status, error_message, log_path`
	if taskID == "" {
		rows, err = s.db.Query(`
			SELECT `+cols+`
			FROM runs WHERE project_id = ?
			ORDER BY started_at DESC, id DESC`, projectID)
	} else {
		rows, err = s.db.Query(`
			SELECT `+cols+`
			FROM runs WHERE project_id = ? AND task_id = ?
			ORDER BY started_at DESC, id DESC`, projectID, taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()

	var out []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		out = append(out, *r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list runs rows: %w", err)
	}
	return out, nil
}

// RunFilter holds optional filters for listing runs.
type RunFilter struct {
	ProjectID string
	TaskID    string
	Status    string
}

// ListRunsFilter returns runs matching optional filters, newest first.
func (s *Store) ListRunsFilter(f RunFilter) ([]Run, error) {
	const cols = `id, task_id, project_id, started_at, finished_at, exit_code,
			status, error_message, log_path`
	query := `SELECT ` + cols + ` FROM runs WHERE 1=1`
	var args []any
	if f.ProjectID != "" {
		query += ` AND project_id = ?`
		args = append(args, f.ProjectID)
	}
	if f.TaskID != "" {
		query += ` AND task_id = ?`
		args = append(args, f.TaskID)
	}
	if f.Status != "" {
		query += ` AND status = ?`
		args = append(args, f.Status)
	}
	query += ` ORDER BY started_at DESC, id DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list runs filter: %w", err)
	}
	defer rows.Close()

	var out []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		out = append(out, *r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list runs filter rows: %w", err)
	}
	return out, nil
}

type runScanner interface {
	Scan(dest ...any) error
}

func scanRun(row runScanner) (*Run, error) {
	var r Run
	var finished, errMsg, logPath sql.NullString
	var exitCode sql.NullInt64
	err := row.Scan(
		&r.ID, &r.TaskID, &r.ProjectID, &r.StartedAt,
		&finished, &exitCode, &r.Status, &errMsg, &logPath,
	)
	if err != nil {
		return nil, err
	}
	r.FinishedAt = fromNullString(finished)
	r.ExitCode = fromNullInt(exitCode)
	r.ErrorMessage = fromNullString(errMsg)
	r.LogPath = fromNullString(logPath)
	return &r, nil
}

// UpdateRun updates lifecycle fields of a run.
func (s *Store) UpdateRun(r *Run) error {
	res, err := s.db.Exec(`
		UPDATE runs SET
			finished_at = ?, exit_code = ?, status = ?, error_message = ?, log_path = ?
		WHERE id = ?`,
		nullString(r.FinishedAt), nullInt(r.ExitCode), r.Status,
		nullString(r.ErrorMessage), nullString(r.LogPath), r.ID,
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
	return nil
}

func nullString(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func fromNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}

func nullInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func fromNullInt(ni sql.NullInt64) *int {
	if !ni.Valid {
		return nil
	}
	v := int(ni.Int64)
	return &v
}
