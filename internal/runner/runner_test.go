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

	"master-agent/internal/placeholder"
	"master-agent/internal/store"
)

func sampleProject() store.Project {
	return store.Project{
		ID:         "proj-1",
		Name:       "my-app",
		Path:       "/home/dev/my-app",
		SSHHost:    "dev-box",
		SSHUser:    "dev",
		SSHPort:    22,
		SSHKeyPath: "/secrets/projects/my-app/id_ed25519",
		Enabled:    true,
	}
}

func TestBuildSSHArgsFromProject(t *testing.T) {
	args, err := BuildSSHArgs(sampleProject(), "touch flag", DefaultServerAliveInterval, DefaultServerAliveCountMax)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"-i", "/secrets/projects/my-app/id_ed25519",
		"-p", "22",
		"-o", "BatchMode=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"dev@dev-box",
	}, args[:len(args)-1])

	remote := args[len(args)-1]
	wantRemote, err := BuildRemoteCommand("/home/dev/my-app", "touch flag")
	require.NoError(t, err)
	assert.Equal(t, wantRemote, remote)
	assert.True(t, strings.HasPrefix(remote, "bash -lc "))
	assert.Contains(t, remote, "/home/dev/my-app")
	assert.Contains(t, remote, "touch flag")
	// SSH options precede the remote script; never a local agent binary.
	assert.Equal(t, "-i", args[0])
}

func TestBuildSSHArgsCustomPortAndKeepalive(t *testing.T) {
	p := sampleProject()
	p.SSHPort = 2222
	args, err := BuildSSHArgs(p, "echo ok", 60, 5)
	require.NoError(t, err)
	assert.Contains(t, args, "2222")
	assert.Contains(t, args, "ServerAliveInterval=60")
	assert.Contains(t, args, "ServerAliveCountMax=5")
}

func TestBuildSSHArgsDefaultPort(t *testing.T) {
	p := sampleProject()
	p.SSHPort = 0
	args, err := BuildSSHArgs(p, "true", 0, 0)
	require.NoError(t, err)
	assert.Equal(t, "22", args[3])
	assert.Contains(t, args, "ServerAliveInterval=30")
}

func TestBuildSSHArgsRequiresFields(t *testing.T) {
	tests := []struct {
		name string
		mut  func(*store.Project)
	}{
		{"host", func(p *store.Project) { p.SSHHost = "" }},
		{"user", func(p *store.Project) { p.SSHUser = "" }},
		{"key", func(p *store.Project) { p.SSHKeyPath = "" }},
		{"path", func(p *store.Project) { p.Path = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := sampleProject()
			tt.mut(&p)
			_, err := BuildSSHArgs(p, "true", 0, 0)
			require.Error(t, err)
		})
	}
}

func TestBuildRemoteCommandShellAndJSON(t *testing.T) {
	path := `/tmp/dir with spaces`
	shell, err := BuildRemoteCommand(path, `echo hello`)
	require.NoError(t, err)
	inner := "cd " + placeholder.ShellQuote(path) + " && echo hello"
	assert.Equal(t, "bash -lc "+placeholder.ShellQuote(inner), shell)

	jsonCmd, err := BuildRemoteCommand("/home/dev/app", `["cursor","agent","-p","Do the work"]`)
	require.NoError(t, err)
	jsonInner := "cd " + placeholder.ShellQuote("/home/dev/app") +
		" && exec " + placeholder.ShellQuote("cursor") +
		" " + placeholder.ShellQuote("agent") +
		" " + placeholder.ShellQuote("-p") +
		" " + placeholder.ShellQuote("Do the work")
	assert.Equal(t, "bash -lc "+placeholder.ShellQuote(jsonInner), jsonCmd)
	assert.Contains(t, jsonCmd, "exec")
	assert.Contains(t, jsonCmd, "cursor")
}

func TestBuildRemoteCommandEmptyJSON(t *testing.T) {
	_, err := BuildRemoteCommand("/p", `[]`)
	require.Error(t, err)
}

func TestResultErrorMessage(t *testing.T) {
	assert.Equal(t, "", Result{ExitCode: 0}.ErrorMessage())
	assert.Equal(t, "remote command exited with code 1", Result{ExitCode: 1}.ErrorMessage())
	assert.Equal(t, "Permission denied", Result{ExitCode: 255, Stderr: "Permission denied\n"}.ErrorMessage())

	long := strings.Repeat("x", DefaultErrorMessageBytes+10)
	got := Result{ExitCode: 1, Stderr: long}.ErrorMessage()
	assert.Len(t, got, DefaultErrorMessageBytes)
	assert.True(t, Result{ExitCode: 1}.Failed())
	assert.False(t, Result{ExitCode: 0}.Failed())
}

func TestFakeRunnerRecordsCalls(t *testing.T) {
	fake := &FakeRunner{
		Result: Result{ExitCode: 0, Stdout: "ok"},
	}
	p := sampleProject()
	res, err := fake.Run(context.Background(), p, "echo ok")
	require.NoError(t, err)
	assert.Equal(t, 0, res.ExitCode)
	assert.Equal(t, "ok", res.Stdout)
	require.Len(t, fake.Calls, 1)
	assert.Equal(t, "echo ok", fake.Calls[0].Command)
	assert.Equal(t, p.SSHHost, fake.Calls[0].Project.SSHHost)

	fake.ResultFunc = func(_ context.Context, _ store.Project, _ string) (Result, error) {
		return Result{ExitCode: 1, Stderr: "boom"}, nil
	}
	res, err = fake.Run(context.Background(), p, "exit 1")
	require.NoError(t, err)
	assert.True(t, res.Failed())
	assert.Equal(t, "boom", res.ErrorMessage())
	assert.Len(t, fake.Calls, 2)
}

func TestSSHRunnerCapturesStdoutStderrAndExit(t *testing.T) {
	stub := writeSSHStub(t, `
case "$1" in
  success)
    printf %s out-ok
    printf %s err-ok >&2
    exit 0
    ;;
  fail)
    printf %s out-fail
    printf %s remote-failed >&2
    exit 7
    ;;
  auth)
    printf %s 'Permission denied' >&2
    exit 255
    ;;
  *)
    echo "unexpected: $*" >&2
    exit 99
    ;;
esac
`)

	r := &SSHRunner{
		SSHPath: stub,
		commandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			mode := "success"
			remote := arg[len(arg)-1]
			switch {
			case strings.Contains(remote, "exit 1"):
				mode = "fail"
			case strings.Contains(remote, "auth-check"):
				mode = "auth"
			}
			return exec.CommandContext(ctx, name, mode)
		},
	}

	t.Run("success", func(t *testing.T) {
		res, err := r.Run(context.Background(), sampleProject(), "true")
		require.NoError(t, err)
		assert.Equal(t, 0, res.ExitCode)
		assert.Equal(t, "out-ok", res.Stdout)
		assert.Equal(t, "err-ok", res.Stderr)
		assert.False(t, res.Failed())
		assert.Greater(t, res.PID, 0)
	})

	t.Run("nonzero_remote", func(t *testing.T) {
		res, err := r.Run(context.Background(), sampleProject(), "exit 1")
		require.NoError(t, err)
		assert.Equal(t, 7, res.ExitCode)
		assert.Equal(t, "out-fail", res.Stdout)
		assert.Equal(t, "remote-failed", res.Stderr)
		assert.True(t, res.Failed())
		assert.Equal(t, "remote-failed", res.ErrorMessage())
	})

	t.Run("ssh_auth_failure", func(t *testing.T) {
		res, err := r.Run(context.Background(), sampleProject(), "auth-check")
		require.NoError(t, err)
		assert.Equal(t, 255, res.ExitCode)
		assert.True(t, res.Failed())
		assert.Equal(t, "Permission denied", res.ErrorMessage())
	})
}

func TestSSHRunnerInvokesOnlySSHBinary(t *testing.T) {
	var sawName string
	var sawArgs []string
	stub := writeSSHStub(t, `exit 0`)

	r := &SSHRunner{
		SSHPath: stub,
		commandContext: func(ctx context.Context, name string, arg ...string) *exec.Cmd {
			sawName = name
			sawArgs = append([]string{}, arg...)
			return exec.CommandContext(ctx, name, arg...)
		},
	}

	_, err := r.Run(context.Background(), sampleProject(), `["cursor","agent","-p","hi"]`)
	require.NoError(t, err)
	assert.Equal(t, stub, sawName)
	require.NotEmpty(t, sawArgs)
	assert.Equal(t, "-i", sawArgs[0])
	assert.Equal(t, "dev@dev-box", sawArgs[len(sawArgs)-2])
	assert.Contains(t, sawArgs[len(sawArgs)-1], "cursor")
	assert.Contains(t, sawArgs[len(sawArgs)-1], "exec")
}

func writeSSHStub(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ssh-stub")
	script := "#!/bin/sh\n" + body + "\n"
	require.NoError(t, os.WriteFile(path, []byte(script), 0o755))
	return path
}
