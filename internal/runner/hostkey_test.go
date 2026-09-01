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

const testHostKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIM46IgN5zAabJIE6wrmR6RAqNtsAe50LgH1BNOsptMei"

func TestBuildSSHArgsWithKnownHosts(t *testing.T) {
	p := sampleProject()
	p.SSHHostKey = testHostKey
	args, err := BuildSSHArgs(p, testIdentityFile, "true", 0, 0, "/tmp/kh")
	require.NoError(t, err)
	assert.Contains(t, args, "UserKnownHostsFile=/tmp/kh")
	assert.Contains(t, args, "GlobalKnownHostsFile=/dev/null")
}

func TestKnownHostsEntryDefaultPort(t *testing.T) {
	line, err := KnownHostsEntry(sampleProject(), testHostKey)
	require.NoError(t, err)
	assert.Equal(t, "dev-box "+testHostKey, line)
}

func TestKnownHostsEntryCustomPort(t *testing.T) {
	p := sampleProject()
	p.SSHPort = 2222
	line, err := KnownHostsEntry(p, testHostKey)
	require.NoError(t, err)
	assert.Equal(t, "[dev-box]:2222 "+testHostKey, line)
}

func TestSSHTesterSuccessPinsHostKey(t *testing.T) {
	stubs := writeSSHTestStubs(t)
	tester := sshTesterWithStubs(stubs)

	res, err := tester.Test(context.Background(), sampleProject())
	require.NoError(t, err)
	assert.Equal(t, "ssh-ed25519", res.HostKeyType)
	assert.Equal(t, testHostKey, res.HostKeyPublic)
	assert.True(t, res.NewlyPinned)
	assert.Equal(t, "SHA256:stubfingerprint", res.HostKeyFingerprint)
}

func TestSSHTesterHostKeyMismatch(t *testing.T) {
	stubs := writeSSHTestStubs(t)
	tester := sshTesterWithStubs(stubs)

	p := sampleProject()
	p.SSHHostKey = "ssh-ed25519 AAAAotherkeydata"
	_, err := tester.Test(context.Background(), p)
	require.Error(t, err)
	var sshErr *SSHTestError
	require.ErrorAs(t, err, &sshErr)
	assert.Equal(t, SSHCodeHostKeyMismatch, sshErr.Code)
}

func TestSSHTesterAuthFailure(t *testing.T) {
	stubs := writeSSHTestStubs(t)
	tester := sshTesterWithStubs(stubs)
	tester.commandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		base := filepathBase(name)
		if base == "ssh-keyscan" {
			return exec.CommandContext(ctx, stubs.keyscan, "scan")
		}
		if base == "ssh-keygen" {
			return exec.CommandContext(ctx, stubs.keygen, arg...)
		}
		return exec.CommandContext(ctx, stubs.ssh, "auth-fail")
	}

	_, err := tester.Test(context.Background(), sampleProject())
	require.Error(t, err)
	var sshErr *SSHTestError
	require.ErrorAs(t, err, &sshErr)
	assert.Equal(t, SSHCodeAuthFailed, sshErr.Code)
}

func TestSSHRunnerUsesPinnedHostKey(t *testing.T) {
	var sawArgs []string
	stub := writeSSHStub(t, `exit 0`)
	p := sampleProject()
	p.SSHHostKey = testHostKey

	r := &SSHRunner{
		SSHPath: stub,
		commandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			sawArgs = append([]string{}, arg...)
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	_, err := r.Run(context.Background(), p, "true")
	require.NoError(t, err)
	assert.Contains(t, strings.Join(sawArgs, " "), "UserKnownHostsFile=")
}

func TestParseKeyScanLine(t *testing.T) {
	line, ok := parseKeyScanLine("worker " + testHostKey)
	require.True(t, ok)
	assert.Equal(t, testHostKey, line)

	line, ok = parseKeyScanLine("[worker]:2222 " + testHostKey)
	require.True(t, ok)
	assert.Equal(t, testHostKey, line)
}

type sshTestStubs struct {
	ssh     string
	keyscan string
	keygen  string
}

func sshTesterWithStubs(stubs sshTestStubs) *SSHTester {
	return &SSHTester{
		SSHPath:     stubs.ssh,
		KeyScanPath: stubs.keyscan,
		KeyGenPath:  stubs.keygen,
		commandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			base := filepathBase(name)
			switch base {
			case "ssh-keyscan":
				return exec.CommandContext(ctx, stubs.keyscan, "scan")
			case "ssh":
				mode := "auth"
				if len(arg) > 0 {
					remote := arg[len(arg)-1]
					if strings.Contains(remote, "test -d") {
						mode = "path"
					}
				}
				return exec.CommandContext(ctx, stubs.ssh, mode)
			case "ssh-keygen":
				return exec.CommandContext(ctx, stubs.keygen, arg...)
			default:
				return exec.CommandContext(ctx, name, arg...)
			}
		},
	}
}

func writeSSHTestStubs(t *testing.T) sshTestStubs {
	t.Helper()
	dir := t.TempDir()
	sshPath := filepath.Join(dir, "ssh")
	keyscanPath := filepath.Join(dir, "ssh-keyscan")
	keygenPath := filepath.Join(dir, "ssh-keygen")

	require.NoError(t, os.WriteFile(sshPath, []byte(`#!/bin/sh
case "$1" in
  auth) echo ok; exit 0 ;;
  path) exit 0 ;;
  auth-fail) echo Permission denied >&2; exit 255 ;;
  *) exit 99 ;;
esac
`), 0o755))

	require.NoError(t, os.WriteFile(keyscanPath, []byte(`#!/bin/sh
echo "worker `+testHostKey+`"
exit 0
`), 0o755))

	require.NoError(t, os.WriteFile(keygenPath, []byte(`#!/bin/sh
echo "256 SHA256:stubfingerprint worker (ED25519)"
exit 0
`), 0o755))

	return sshTestStubs{ssh: sshPath, keyscan: keyscanPath, keygen: keygenPath}
}

func filepathBase(path string) string {
	return filepath.Base(path)
}
