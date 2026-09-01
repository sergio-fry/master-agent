package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

const schemaV1SQL = `
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY
);
CREATE TABLE projects (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    ssh_host TEXT NOT NULL,
    ssh_user TEXT NOT NULL,
    ssh_port INTEGER NOT NULL DEFAULT 22,
    ssh_key_path TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE tasks (
    id TEXT PRIMARY KEY NOT NULL,
    project_id TEXT NOT NULL REFERENCES projects(id),
    name TEXT NOT NULL,
    prompt TEXT NOT NULL,
    command TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    last_run_at TEXT,
    next_run_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE locks (
    project_id TEXT PRIMARY KEY NOT NULL REFERENCES projects(id),
    task_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    pid INTEGER,
    acquired_at TEXT NOT NULL
);
CREATE TABLE runs (
    id TEXT PRIMARY KEY NOT NULL,
    task_id TEXT NOT NULL REFERENCES tasks(id),
    project_id TEXT NOT NULL REFERENCES projects(id),
    started_at TEXT NOT NULL,
    finished_at TEXT,
    exit_code INTEGER,
    status TEXT NOT NULL,
    error_message TEXT,
    log_path TEXT
);
`

// openStoreMigratedFromV1 seeds a v1 SQLite file and runs Open(), which applies v2 migration.
func openStoreMigratedFromV1(t *testing.T, seed func(t *testing.T, db *sql.DB)) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "master-agent.db")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec(schemaV1SQL)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO schema_migrations(version) VALUES (1)`)
	require.NoError(t, err)
	if seed != nil {
		seed(t, db)
	}
	require.NoError(t, db.Close())

	s, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func projectColumnNames(t *testing.T, s *Store) []string {
	t.Helper()
	rows, err := s.db.Query(`PRAGMA table_info(projects)`)
	require.NoError(t, err)
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt any
		require.NoError(t, rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk))
		cols = append(cols, name)
	}
	require.NoError(t, rows.Err())
	return cols
}

func TestMigrateV1ToV2_ProjectsUseInlineSSHKey(t *testing.T) {
	s := openStoreMigratedFromV1(t, nil)

	cols := projectColumnNames(t, s)
	assert.Contains(t, cols, "ssh_private_key", "projects must store inline SSH key after v2 migration")
	assert.NotContains(t, cols, "ssh_key_path", "legacy ssh_key_path column must be removed")

	p := &Project{
		Name:          "my-app",
		Path:          "/home/dev/my-app",
		SSHHost:       "dev-box",
		SSHUser:       "dev",
		SSHPort:       22,
		SSHPrivateKey: TestSSHPrivateKey,
		Enabled:       true,
	}
	require.NoError(t, s.CreateProject(p))

	projects, err := s.ListProjects()
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.True(t, projects[0].KeyConfigured())
}

func TestMigrateV1ToV2_MigratesKeyFileToInline(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	require.NoError(t, os.WriteFile(keyPath, []byte(TestSSHPrivateKey), 0o600))

	const projectID = "11111111-1111-1111-1111-111111111111"
	s := openStoreMigratedFromV1(t, func(t *testing.T, db *sql.DB) {
		t.Helper()
		_, err := db.Exec(`
			INSERT INTO projects (
				id, name, path, ssh_host, ssh_user, ssh_port, ssh_key_path, enabled, created_at, updated_at
			) VALUES (?, 'legacy', '/p', 'h', 'u', 22, ?, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
			projectID, keyPath,
		)
		require.NoError(t, err)
	})

	got, err := s.GetProject(projectID)
	require.NoError(t, err)
	assert.Equal(t, TestSSHPrivateKey, got.SSHPrivateKey)
}

func TestMigrateV2ToV3_RepairsBrokenV2Schema(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	require.NoError(t, os.WriteFile(keyPath, []byte(TestSSHPrivateKey), 0o600))

	const projectID = "22222222-2222-2222-2222-222222222222"
	dir := t.TempDir()
	path := filepath.Join(dir, "master-agent.db")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = db.Exec(schemaV1SQL)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO projects (
			id, name, path, ssh_host, ssh_user, ssh_port, ssh_key_path, enabled, created_at, updated_at
		) VALUES (?, 'legacy', '/p', 'h', 'u', 22, ?, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		projectID, keyPath,
	)
	require.NoError(t, err)
	// Simulate the original buggy v2 migration outcome.
	require.NoError(t, sApplyBrokenV2Migration(db))
	_, err = db.Exec(`INSERT INTO schema_migrations(version) VALUES (1), (2)`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	s, err := Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	cols := projectColumnNames(t, s)
	assert.Contains(t, cols, "ssh_private_key")
	assert.NotContains(t, cols, "ssh_key_path")

	got, err := s.GetProject(projectID)
	require.NoError(t, err)
	assert.Equal(t, TestSSHPrivateKey, got.SSHPrivateKey)

	require.NoError(t, s.CreateProject(&Project{
		Name: "new", Path: "/n", SSHHost: "h", SSHUser: "u",
		SSHPrivateKey: TestSSHPrivateKey, Enabled: true,
	}))
}

// sApplyBrokenV2Migration reproduces the pre-fix v2 migration bug for repair tests.
func sApplyBrokenV2Migration(db *sql.DB) error {
	if _, err := db.Exec(`ALTER TABLE projects ADD COLUMN ssh_private_key TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE projects DROP COLUMN ssh_private_key`); err != nil {
		return err
	}
	return nil
}
