package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSHBrowserListDirsStub(t *testing.T) {
	stubs := writeSSHTestStubs(t)
	dir := t.TempDir()
	sshPath := filepath.Join(dir, "ssh-browse")
	require.NoError(t, os.WriteFile(sshPath, []byte(`#!/bin/sh
case "$1" in
  home) echo /home/worker; exit 0 ;;
  list) echo workspace; exit 0 ;;
  *) exit 99 ;;
esac
`), 0o755))

	browser := &SSHBrowser{
		SSHPath: sshPath,
		commandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			base := filepath.Base(name)
			if base == "ssh-keyscan" {
				return exec.CommandContext(ctx, stubs.keyscan, "scan")
			}
			remote := arg[len(arg)-1]
			if strings.Contains(remote, "pwd") {
				return exec.CommandContext(ctx, sshPath, "home")
			}
			return exec.CommandContext(ctx, sshPath, "list")
		},
	}

	p := sampleProject()
	p.SSHHostKey = testHostKey
	res, err := browser.ListDirs(context.Background(), p, "")
	require.NoError(t, err)
	assert.Equal(t, "/home/worker", res.Path)
	assert.Len(t, res.Entries, 1)
	assert.Equal(t, "workspace", res.Entries[0].Name)
}
