package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"syscall"

	"master-agent/internal/placeholder"
	"master-agent/internal/store"
)

// OpenSSH keepalive defaults for long-running remote agents (specs/03).
const (
	DefaultServerAliveInterval = 30
	DefaultServerAliveCountMax = 3
)

// SSHRunner executes commands via the local OpenSSH client only.
type SSHRunner struct {
	// SSHPath is the ssh binary; empty means "ssh".
	SSHPath string

	// ServerAliveInterval / ServerAliveCountMax override keepalive defaults when > 0.
	ServerAliveInterval int
	ServerAliveCountMax int

	// commandContext builds the process; nil uses exec.CommandContext.
	// Tests inject a stub to avoid a real SSH session.
	commandContext func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

// Run invokes ssh with Project credentials and the substituted remote command.
func (r *SSHRunner) Run(ctx context.Context, project store.Project, command string) (Result, error) {
	interval, countMax := r.keepalive()
	args, err := BuildSSHArgs(project, command, interval, countMax)
	if err != nil {
		return Result{}, err
	}

	sshBin := r.SSHPath
	if sshBin == "" {
		sshBin = "ssh"
	}

	cc := r.commandContext
	if cc == nil {
		cc = exec.CommandContext
	}
	cmd := cc(ctx, sshBin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	res := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if runErr == nil {
		res.ExitCode = 0
		return res, nil
	}

	var ee *exec.ExitError
	if errors.As(runErr, &ee) {
		res.ExitCode = exitStatus(ee)
		// Non-zero remote / SSH client exit is a Result for the daemon, not a Go error.
		return res, nil
	}

	// Failed to start ssh or context canceled before a process exit.
	if res.Stderr == "" {
		res.Stderr = runErr.Error()
	}
	res.ExitCode = -1
	return res, fmt.Errorf("ssh: %w", runErr)
}

func (r *SSHRunner) keepalive() (interval, countMax int) {
	interval = r.ServerAliveInterval
	if interval <= 0 {
		interval = DefaultServerAliveInterval
	}
	countMax = r.ServerAliveCountMax
	if countMax <= 0 {
		countMax = DefaultServerAliveCountMax
	}
	return interval, countMax
}

func exitStatus(ee *exec.ExitError) int {
	if status, ok := ee.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus()
	}
	return ee.ExitCode()
}

// BuildSSHArgs returns OpenSSH client arguments (without the binary name)
// for the given project and already-substituted command.
func BuildSSHArgs(project store.Project, command string, aliveInterval, aliveCountMax int) ([]string, error) {
	if project.SSHHost == "" {
		return nil, fmt.Errorf("project ssh_host is required")
	}
	if project.SSHUser == "" {
		return nil, fmt.Errorf("project ssh_user is required")
	}
	if project.SSHKeyPath == "" {
		return nil, fmt.Errorf("project ssh_key_path is required")
	}
	if project.Path == "" {
		return nil, fmt.Errorf("project path is required")
	}

	port := project.SSHPort
	if port == 0 {
		port = 22
	}

	remote, err := BuildRemoteCommand(project.Path, command)
	if err != nil {
		return nil, err
	}

	if aliveInterval <= 0 {
		aliveInterval = DefaultServerAliveInterval
	}
	if aliveCountMax <= 0 {
		aliveCountMax = DefaultServerAliveCountMax
	}

	return []string{
		"-i", project.SSHKeyPath,
		"-p", fmt.Sprintf("%d", port),
		"-o", "BatchMode=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", fmt.Sprintf("ServerAliveInterval=%d", aliveInterval),
		"-o", fmt.Sprintf("ServerAliveCountMax=%d", aliveCountMax),
		fmt.Sprintf("%s@%s", project.SSHUser, project.SSHHost),
		remote,
	}, nil
}

// BuildRemoteCommand builds the remote shell snippet: cd into project path, then run command.
// Substituted shell strings are used as-is; JSON argv becomes exec with shell-quoted args.
// Wrapped in bash -lc so worker login PATH (cursor, backlog, …) is available.
func BuildRemoteCommand(projectPath, substituted string) (string, error) {
	inner, err := remoteInner(projectPath, substituted)
	if err != nil {
		return "", err
	}
	return "bash -lc " + placeholder.ShellQuote(inner), nil
}

func remoteInner(projectPath, substituted string) (string, error) {
	trimmed := bytes.TrimSpace([]byte(substituted))
	var argv []string
	if err := json.Unmarshal(trimmed, &argv); err == nil {
		if len(argv) == 0 {
			return "", fmt.Errorf("empty JSON argv command")
		}
		var b bytes.Buffer
		b.WriteString("cd ")
		b.WriteString(placeholder.ShellQuote(projectPath))
		b.WriteString(" && exec")
		for _, a := range argv {
			b.WriteByte(' ')
			b.WriteString(placeholder.ShellQuote(a))
		}
		return b.String(), nil
	}

	return "cd " + placeholder.ShellQuote(projectPath) + " && " + substituted, nil
}
