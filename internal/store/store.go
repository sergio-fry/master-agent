package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

const schemaVersion = 2

// Store is SQLite persistence for projects, tasks, locks, and runs.
type Store struct {
	db *sql.DB
}

// Open creates the DB file (and parent directories) if needed, opens SQLite,
// and applies schema migrations.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign_keys: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var current int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if current >= schemaVersion {
		return nil
	}

	switch current {
	case 0:
		sqlBytes, err := schemaFS.ReadFile("schema.sql")
		if err != nil {
			return fmt.Errorf("read schema: %w", err)
		}
		if _, err := s.db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}
	case 1:
		if err := s.migrateV1ToV2(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported schema version %d", current)
	}

	if _, err := s.db.Exec(`INSERT INTO schema_migrations(version) VALUES (?)`, schemaVersion); err != nil {
		return fmt.Errorf("record schema version: %w", err)
	}
	return nil
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func newID() string {
	return uuid.NewString()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func intToBool(v int) bool {
	return v != 0
}
