package store

import "fmt"

func (s *Store) migrateV3ToV4() error {
	has, err := s.hasColumn("projects", "ssh_host_key")
	if err != nil {
		return err
	}
	if has {
		return nil
	}
	if _, err := s.db.Exec(`ALTER TABLE projects ADD COLUMN ssh_host_key TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add ssh_host_key column: %w", err)
	}
	return nil
}
