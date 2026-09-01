package store

import (
	"fmt"
	"os"
)

func (s *Store) migrateV1ToV2() error {
	if _, err := s.db.Exec(`ALTER TABLE projects ADD COLUMN ssh_private_key TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add ssh_private_key column: %w", err)
	}

	rows, err := s.db.Query(`SELECT id, ssh_key_path FROM projects`)
	if err != nil {
		return fmt.Errorf("list projects for key migration: %w", err)
	}

	type projectKey struct {
		id      string
		keyPath string
	}
	var projects []projectKey
	for rows.Next() {
		var p projectKey
		if err := rows.Scan(&p.id, &p.keyPath); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scan project key path: %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close project rows: %w", err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate projects for key migration: %w", err)
	}

	for _, p := range projects {
		if p.keyPath == "" {
			continue
		}
		data, err := os.ReadFile(p.keyPath)
		if err != nil {
			continue
		}
		if _, err := s.db.Exec(`UPDATE projects SET ssh_private_key = ? WHERE id = ?`, string(data), p.id); err != nil {
			return fmt.Errorf("migrate project %s key: %w", p.id, err)
		}
	}

	if _, err := s.db.Exec(`ALTER TABLE projects DROP COLUMN ssh_key_path`); err != nil {
		return fmt.Errorf("drop ssh_key_path column: %w", err)
	}
	return nil
}
