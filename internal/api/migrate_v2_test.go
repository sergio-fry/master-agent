package api

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"master-agent/internal/store"

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
`

func openStoreMigratedFromV1(t *testing.T) *store.Store {
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
	require.NoError(t, db.Close())

	s, err := store.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s
}
