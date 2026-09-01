package runner

import (
	"fmt"
	"os"

	"master-agent/internal/store"
)

// WriteTempSSHKey writes private key material to a temp file (mode 0600) for OpenSSH -i.
func WriteTempSSHKey(key string) (path string, cleanup func(), err error) {
	k := store.NormalizeSSHPrivateKey(key)
	if k == "" {
		return "", nil, fmt.Errorf("project ssh private key is required")
	}
	f, err := os.CreateTemp("", "master-agent-ssh-key-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp key file: %w", err)
	}
	path = f.Name()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("chmod temp key file: %w", err)
	}
	if _, err := f.WriteString(k); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("write temp key file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("close temp key file: %w", err)
	}
	return path, func() { _ = os.Remove(path) }, nil
}
