package store

import (
	"errors"
	"strings"
)

// TestSSHPrivateKey is placeholder key material for unit tests.
const TestSSHPrivateKey = "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----\n"

func ValidateSSHPrivateKey(key string) error {
	k := strings.TrimSpace(key)
	if k == "" {
		return errors.New("ssh_private_key is required")
	}
	if !strings.Contains(k, "PRIVATE KEY") {
		return errors.New("invalid ssh private key format")
	}
	return nil
}

// KeyConfigured reports whether the project has inline SSH key material.
func (p *Project) KeyConfigured() bool {
	return strings.TrimSpace(p.SSHPrivateKey) != ""
}
