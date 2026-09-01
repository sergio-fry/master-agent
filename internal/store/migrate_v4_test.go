package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateV3ToV4_AddsSSHHostKeyColumn(t *testing.T) {
	st, _ := openTempStore(t)
	has, err := st.hasColumn("projects", "ssh_host_key")
	require.NoError(t, err)
	assert.True(t, has)

	p := &Project{
		Name: "host-key", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHPrivateKey: TestSSHPrivateKey,
		SSHHostKey:    "ssh-ed25519 AAAAB",
		Enabled:       true,
	}
	require.NoError(t, st.CreateProject(p))

	got, err := st.GetProject(p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.SSHHostKey, got.SSHHostKey)
}
