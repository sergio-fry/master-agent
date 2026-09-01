package store

import (
	"fmt"
	"os"
)

func (s *Store) migrateV1ToV2() error {
	if _, err := s.db.Exec(`ALTER TABLE projects ADD COLUMN ssh_private_key TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add ssh_private_key column: %w", err)
	}

	rows, err := s.db.Query(`SELECT id, ssh_private_key FROM projects`)
	if err != nil {
		return fmt.Errorf("list projects for key migration: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, keyPath string
		if err := rows.Scan(&id, &keyPath); err != nil {
			return fmt.Errorf("scan project key path: %w", err)
		}
		if keyPath == "" {
			continue
		}
		data, err := os.ReadFile(keyPath)
		if err != nil {
			continue
		}
		if _, err := s.db.Exec(`UPDATE projects SET ssh_private_key = ? WHERE id = ?`, string(data), id); err != nil {
			return fmt.Errorf("migrate project %s key: %w", id, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate projects for key migration: %w", err)
	}

	if _, err := s.db.Exec(`ALTER TABLE projects DROP COLUMN ssh_private_key`); err != nil {
		return fmt.Errorf("drop ssh_private_key column: %w", err)
	}
	return nil
}
