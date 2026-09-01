//go:build acceptance

package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/store"
)

func TestSSHTesterLiveWorker(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	keyPath := filepath.Join(root, "test", "fixtures", "ssh", "id_ed25519")
	key, err := os.ReadFile(keyPath)
	require.NoError(t, err)

	p := store.Project{
		Path:          "/home/worker/workspace",
		SSHHost:       "worker",
		SSHUser:       "worker",
		SSHPort:       22,
		SSHPrivateKey: string(key),
	}

	res, err := (&SSHTester{}).Test(context.Background(), p)
	require.NoError(t, err, "ssh test against compose worker")
	assert.Equal(t, "ssh-ed25519", res.HostKeyType)
	assert.NotEmpty(t, res.HostKeyFingerprint)
	assert.True(t, res.NewlyPinned)
}
