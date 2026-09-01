package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeSSHPrivateKey(t *testing.T) {
	raw := "-----BEGIN OPENSSH PRIVATE KEY-----\ndata\n-----END OPENSSH PRIVATE KEY-----"
	assert.Equal(t, raw+"\n", NormalizeSSHPrivateKey(raw))
	assert.Equal(t, raw+"\n", NormalizeSSHPrivateKey(raw+"\n"))
	assert.Equal(t, "", NormalizeSSHPrivateKey("  \n"))
}
