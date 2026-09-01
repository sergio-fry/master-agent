package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"master-agent/internal/store"
)

// SSH test error codes returned by the HTTP API.
const (
	SSHCodeUnreachable     = "ssh_unreachable"
	SSHCodeAuthFailed      = "ssh_auth_failed"
	SSHCodeHostKeyMismatch = "ssh_host_key_mismatch"
	SSHCodePathNotFound    = "ssh_path_not_found"
)

// SSHTestResult is a successful SSH connection test.
type SSHTestResult struct {
	HostKeyType        string
	HostKeyFingerprint string
	HostKeyPublic      string
	NewlyPinned        bool
}

// SSHTestError is a failed SSH connection test with a stable code for clients.
type SSHTestError struct {
	Code    string
	Message string
}

func (e *SSHTestError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// SSHTester performs SSH connection tests and host key discovery.
type SSHTester struct {
	SSHPath     string
	KeyScanPath string
	KeyGenPath  string

	commandContext func(ctx context.Context, name string, arg ...string) *exec.Cmd
}

// Test connects to the project host, verifies credentials and path, and optionally
// compares or captures the server host public key line ("ssh-ed25519 AAAA...").
func (t *SSHTester) Test(ctx context.Context, project store.Project) (SSHTestResult, error) {
	if err := validateSSHTestProject(project); err != nil {
		return SSHTestResult{}, err
	}

	keyPath, cleanupKey, err := WriteTempSSHKey(project.SSHPrivateKey)
	if err != nil {
		return SSHTestResult{}, err
	}
	defer cleanupKey()

	scanned, err := t.scanHostKey(ctx, project)
	if err != nil {
		return SSHTestResult{}, err
	}

	keyLine := strings.TrimSpace(project.SSHHostKey)
	newlyPinned := false
	switch {
	case keyLine == "":
		keyLine = scanned
		newlyPinned = true
	case normalizeHostKeyLine(keyLine) != normalizeHostKeyLine(scanned):
		return SSHTestResult{}, &SSHTestError{
			Code:    SSHCodeHostKeyMismatch,
			Message: "server host key does not match pinned key; re-test after verifying the host",
		}
	}

	khPath, cleanupKH, err := WriteTempKnownHosts(project, keyLine)
	if err != nil {
		return SSHTestResult{}, err
	}
	defer cleanupKH()

	if err := t.verifyAuth(ctx, project, keyPath, khPath); err != nil {
		return SSHTestResult{}, err
	}
	if err := t.verifyPath(ctx, project, keyPath, khPath); err != nil {
		return SSHTestResult{}, err
	}

	fp, err := HostKeyFingerprint(t.keyGen(), keyLine)
	if err != nil {
		return SSHTestResult{}, err
	}
	keyType, _ := splitHostKeyLine(keyLine)
	return SSHTestResult{
		HostKeyType:        keyType,
		HostKeyFingerprint: fp,
		HostKeyPublic:      keyLine,
		NewlyPinned:        newlyPinned,
	}, nil
}

func validateSSHTestProject(p store.Project) error {
	if strings.TrimSpace(p.SSHHost) == "" {
		return fmt.Errorf("project ssh_host is required")
	}
	if strings.TrimSpace(p.SSHUser) == "" {
		return fmt.Errorf("project ssh_user is required")
	}
	if strings.TrimSpace(p.Path) == "" {
		return fmt.Errorf("project path is required")
	}
	if err := store.ValidateSSHPrivateKey(p.SSHPrivateKey); err != nil {
		return err
	}
	return nil
}

func (t *SSHTester) sshBin() string {
	if t.SSHPath != "" {
		return t.SSHPath
	}
	return "ssh"
}

func (t *SSHTester) keyScan() string {
	if t.KeyScanPath != "" {
		return t.KeyScanPath
	}
	return "ssh-keyscan"
}

func (t *SSHTester) keyGen() string {
	if t.KeyGenPath != "" {
		return t.KeyGenPath
	}
	return "ssh-keygen"
}

func (t *SSHTester) command(ctx context.Context, name string, arg ...string) *exec.Cmd {
	if t.commandContext != nil {
		return t.commandContext(ctx, name, arg...)
	}
	return exec.CommandContext(ctx, name, arg...)
}

func (t *SSHTester) scanHostKey(ctx context.Context, project store.Project) (string, error) {
	port := project.SSHPort
	if port == 0 {
		port = 22
	}
	args := []string{
		"-p", fmt.Sprintf("%d", port),
		"-T", "5",
		"-t", "ed25519,ecdsa,rsa",
		project.SSHHost,
	}
	cmd := t.command(ctx, t.keyScan(), args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = "host unreachable or ssh-keyscan failed"
		}
		return "", &SSHTestError{Code: SSHCodeUnreachable, Message: msg}
	}

	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if keyLine, ok := parseKeyScanLine(line); ok {
			return keyLine, nil
		}
	}
	return "", &SSHTestError{Code: SSHCodeUnreachable, Message: "no host key returned by ssh-keyscan"}
}

func parseKeyScanLine(line string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return "", false
	}
	keyTypeIdx := 1
	if strings.HasPrefix(fields[0], "|") {
		return "", false
	}
	if strings.HasPrefix(fields[1], "ssh-") {
		keyTypeIdx = 1
	} else if len(fields) >= 3 && strings.HasPrefix(fields[2], "ssh-") {
		keyTypeIdx = 2
	} else {
		return "", false
	}
	return strings.Join(fields[keyTypeIdx:keyTypeIdx+2], " "), true
}

func (t *SSHTester) verifyAuth(ctx context.Context, project store.Project, keyPath, khPath string) error {
	args, err := buildTestSSHArgs(project, keyPath, khPath, "echo ok")
	if err != nil {
		return err
	}
	cmd := t.command(ctx, t.sshBin(), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(out)) == "ok" {
		return nil
	}
	msg := classifySSHFailure(stderr.String(), err)
	return &SSHTestError{Code: SSHCodeAuthFailed, Message: msg}
}

func (t *SSHTester) verifyPath(ctx context.Context, project store.Project, keyPath, khPath string) error {
	remote := fmt.Sprintf("test -d %s", shellSingleQuote(project.Path))
	args, err := buildTestSSHArgs(project, keyPath, khPath, remote)
	if err != nil {
		return err
	}
	cmd := t.command(ctx, t.sshBin(), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if isAuthLike(stderr.String(), err) {
			return &SSHTestError{Code: SSHCodeAuthFailed, Message: classifySSHFailure(stderr.String(), err)}
		}
		return &SSHTestError{Code: SSHCodePathNotFound, Message: "remote project path does not exist or is not accessible"}
	}
	return nil
}

func buildTestSSHArgs(project store.Project, identityFile, knownHostsFile, remote string) ([]string, error) {
	port := project.SSHPort
	if port == 0 {
		port = 22
	}
	return []string{
		"-i", identityFile,
		"-p", fmt.Sprintf("%d", port),
		"-o", "BatchMode=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		"-o", "UserKnownHostsFile=" + knownHostsFile,
		"-o", "GlobalKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		fmt.Sprintf("%s@%s", project.SSHUser, project.SSHHost),
		remote,
	}, nil
}

func classifySSHFailure(stderr string, err error) string {
	msg := strings.TrimSpace(stderr)
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "permission denied"):
		return "SSH authentication failed"
	case strings.Contains(lower, "no such identity"):
		return "SSH private key rejected"
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "connection timed out"),
		strings.Contains(lower, "no route to host"),
		strings.Contains(lower, "could not resolve hostname"):
		return "SSH host unreachable"
	case strings.Contains(lower, "host key verification failed"):
		return "host key verification failed"
	case msg != "":
		return msg
	case err != nil:
		return err.Error()
	default:
		return "SSH connection failed"
	}
}

func isAuthLike(stderr string, err error) bool {
	msg := strings.ToLower(stderr)
	return strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "no such identity") ||
		strings.Contains(msg, "authentication failed")
}

// WriteTempKnownHosts writes a single-host known_hosts file for OpenSSH.
func WriteTempKnownHosts(project store.Project, keyLine string) (path string, cleanup func(), err error) {
	line, err := KnownHostsEntry(project, keyLine)
	if err != nil {
		return "", nil, err
	}
	f, err := os.CreateTemp("", "master-agent-known-hosts-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp known_hosts: %w", err)
	}
	path = f.Name()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("chmod temp known_hosts: %w", err)
	}
	if _, err := f.WriteString(line + "\n"); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("write temp known_hosts: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("close temp known_hosts: %w", err)
	}
	return path, func() { _ = os.Remove(path) }, nil
}

// KnownHostsEntry formats an OpenSSH known_hosts line for the project host/port.
func KnownHostsEntry(project store.Project, keyLine string) (string, error) {
	keyLine = strings.TrimSpace(keyLine)
	if keyLine == "" {
		return "", fmt.Errorf("host key is required")
	}
	host := strings.TrimSpace(project.SSHHost)
	if host == "" {
		return "", fmt.Errorf("project ssh_host is required")
	}
	port := project.SSHPort
	if port == 0 {
		port = 22
	}
	hostPart := host
	if port != 22 {
		hostPart = fmt.Sprintf("[%s]:%d", host, port)
	}
	return hostPart + " " + keyLine, nil
}

// HostKeyFingerprint returns the SHA256 fingerprint for a public key line.
func HostKeyFingerprint(keyGenPath, keyLine string) (string, error) {
	keyLine = strings.TrimSpace(keyLine)
	if keyLine == "" {
		return "", fmt.Errorf("host key is required")
	}
	f, err := os.CreateTemp("", "master-agent-hostkey-*")
	if err != nil {
		return "", fmt.Errorf("create temp host key: %w", err)
	}
	path := f.Name()
	defer os.Remove(path)
	if _, err := f.WriteString(keyLine + "\n"); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("write temp host key: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close temp host key: %w", err)
	}

	if keyGenPath == "" {
		keyGenPath = "ssh-keygen"
	}
	cmd := exec.Command(keyGenPath, "-lf", path)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ssh-keygen fingerprint: %w", err)
	}
	fields := strings.Fields(string(out))
	if len(fields) < 2 {
		return "", fmt.Errorf("unexpected ssh-keygen output")
	}
	return fields[1], nil
}

func normalizeHostKeyLine(line string) string {
	line = strings.TrimSpace(line)
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	return line
}

func splitHostKeyLine(line string) (keyType, keyData string) {
	parts := strings.Fields(strings.TrimSpace(line))
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", line
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
