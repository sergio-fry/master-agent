//go:build acceptance

package acceptance

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	composeFile    = "docker-compose.test.yml"
	composeProject = "master-agent-acceptance"
	workerUser     = "worker"
	workerHost     = "worker"
	workerBHost    = "worker-b"
	keyPath        = "/secrets/runtime/id_ed25519"
	keyPathB       = "/secrets/runtime/id_ed25519_b"
	workspacePath  = "/home/worker/workspace"
	containerDB    = "/data/master-agent.db"
)

// repoRoot returns the repository root (directory containing docker-compose.test.yml).
func repoRoot(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	if _, err := os.Stat(filepath.Join(root, composeFile)); err != nil {
		t.Fatalf("compose file not found at %s: %v", root, err)
	}
	return root
}

func composeCmd(root string, args ...string) *exec.Cmd {
	all := append([]string{"compose", "-f", composeFile, "-p", composeProject}, args...)
	cmd := exec.Command("docker", all...)
	cmd.Dir = root
	return cmd
}

func runCompose(t testing.TB, root string, args ...string) string {
	t.Helper()
	cmd := composeCmd(root, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("docker compose %v: %v\n%s", args, err, buf.String())
	}
	return buf.String()
}

func waitForSSHHost(t testing.TB, root, host, identity string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		cmd := composeCmd(root, "exec", "-T", "master-agent",
			"ssh",
			"-i", identity,
			"-o", "BatchMode=yes",
			"-o", "IdentitiesOnly=yes",
			"-o", "StrictHostKeyChecking=yes",
			"-o", "ConnectTimeout=2",
			fmt.Sprintf("%s@%s", workerUser, host),
			"echo ready",
		)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		if err := cmd.Run(); err == nil && strings.Contains(buf.String(), "ready") {
			return
		}
		last = buf.String()
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("SSH to %s not ready within %s; last output:\n%s", host, timeout, last)
}

func waitForSSH(t testing.TB, root string, timeout time.Duration) {
	t.Helper()
	waitForSSHHost(t, root, workerHost, keyPath, timeout)
}

func waitForAllWorkers(t testing.TB, root string, timeout time.Duration) {
	t.Helper()
	waitForSSHHost(t, root, workerHost, keyPath, timeout)
	waitForSSHHost(t, root, workerBHost, keyPathB, timeout)
}

func execOnMaster(t testing.TB, root string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"exec", "-T", "master-agent"}, args...)
	return strings.TrimSpace(runCompose(t, root, cmdArgs...))
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func execSSH(t testing.TB, root string, remoteCmd string) string {
	t.Helper()
	return execSSHHost(t, root, workerHost, keyPath, remoteCmd)
}

func execSSHHost(t testing.TB, root, host, identity, remoteCmd string) string {
	t.Helper()
	// Match production SSHRunner: single remote argv "bash -lc '<inner>'".
	return execOnMaster(t, root,
		"ssh",
		"-i", identity,
		"-o", "BatchMode=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		fmt.Sprintf("%s@%s", workerUser, host),
		"bash -lc "+shellQuote(remoteCmd),
	)
}

func waitForMasterAgentContainer(root string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last string
	for time.Now().Before(deadline) {
		cmd := composeCmd(root, "exec", "-T", "master-agent", "echo", "ready")
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		if err := cmd.Run(); err == nil && strings.TrimSpace(buf.String()) == "ready" {
			return nil
		}
		last = buf.String()
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("master-agent container not ready within %s; last output:\n%s", timeout, last)
}

func installSqliteCLI(root string) error {
	cmd := composeCmd(root, "exec", "-T", "master-agent", "sh", "-c",
		"command -v sqlite3 >/dev/null || apk add --no-cache sqlite")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install sqlite3: %w\n%s", err, buf.String())
	}
	return nil
}

func resetAcceptanceVolume(root string) error {
	cmd := composeCmd(root, "exec", "-T", "master-agent", "sh", "-c",
		"rm -f /data/master-agent.db /data/master-agent.db-wal /data/master-agent.db-shm /data/master-agent.db-journal")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reset db volume: %w\n%s", err, buf.String())
	}
	return nil
}

func TestMain(m *testing.M) {
	root := filepath.Clean(filepath.Join(func() string {
		_, file, _, _ := runtime.Caller(0)
		return filepath.Dir(file)
	}(), "../.."))

	if os.Getenv("ACCEPTANCE_SKIP_COMPOSE") == "1" {
		if err := waitForMasterAgentContainer(root, 60*time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		if err := installSqliteCLI(root); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		os.Exit(m.Run())
	}

	up := composeCmd(root, "up", "-d", "--build")
	up.Stdout = os.Stdout
	up.Stderr = os.Stderr
	if err := up.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "compose up failed: %v\n", err)
		os.Exit(1)
	}

	if err := waitForMasterAgentContainer(root, 60*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		_ = composeCmd(root, "down", "-v", "--remove-orphans").Run()
		os.Exit(1)
	}

	if err := installSqliteCLI(root); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		_ = composeCmd(root, "down", "-v", "--remove-orphans").Run()
		os.Exit(1)
	}

	code := m.Run()

	down := composeCmd(root, "down", "-v", "--remove-orphans")
	down.Stdout = os.Stdout
	down.Stderr = os.Stderr
	_ = down.Run()

	os.Exit(code)
}

func TestHarnessComposeAndKeySSH(t *testing.T) {
	root := repoRoot(t)
	waitForAllWorkers(t, root, 60*time.Second)

	out := execSSH(t, root, "whoami && hostname")
	if !strings.Contains(out, workerUser) {
		t.Fatalf("expected whoami=%s, got %q", workerUser, out)
	}
	if !strings.Contains(out, "worker") {
		t.Fatalf("expected hostname to contain worker, got %q", out)
	}

	outB := execSSHHost(t, root, workerBHost, keyPathB, "whoami && hostname")
	if !strings.Contains(outB, workerUser) {
		t.Fatalf("worker-b whoami: got %q", outB)
	}
	if !strings.Contains(outB, "worker-b") {
		t.Fatalf("expected hostname worker-b, got %q", outB)
	}
}

func TestHarnessStubRemoteCommands(t *testing.T) {
	root := repoRoot(t)
	waitForSSH(t, root, 60*time.Second)

	flag := filepath.ToSlash(filepath.Join(workspacePath, "acceptance-flag"))
	echoFile := filepath.ToSlash(filepath.Join(workspacePath, "acceptance-echo.txt"))

	execSSH(t, root, fmt.Sprintf("rm -f %s %s", flag, echoFile))

	execSSH(t, root, fmt.Sprintf("touch %s", flag))
	stat := execSSH(t, root, fmt.Sprintf("test -f %s && echo present", flag))
	if stat != "present" {
		t.Fatalf("touch flag missing: %q", stat)
	}

	echoOut := execSSH(t, root, fmt.Sprintf("echo ok > %s && cat %s", echoFile, echoFile))
	if echoOut != "ok" {
		t.Fatalf("echo stub: want ok, got %q", echoOut)
	}

	cmd := composeCmd(root, "exec", "-T", "master-agent",
		"ssh",
		"-i", keyPath,
		"-o", "BatchMode=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=yes",
		fmt.Sprintf("%s@%s", workerUser, workerHost),
		"bash -lc "+shellQuote("exit 1"),
	)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected exit 1 from remote stub")
	}
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit from exit 1 stub, got %v", err)
	}

	start := time.Now()
	execSSH(t, root, "sleep 1")
	if time.Since(start) < time.Second {
		t.Fatalf("sleep stub returned too quickly: %s", time.Since(start))
	}
}

func TestHarnessMasterAgentBinaryPresent(t *testing.T) {
	root := repoRoot(t)
	out := execOnMaster(t, root, "master-agent", "--help")
	if !strings.Contains(out, "Schedule SSH runs") && !strings.Contains(out, "daemon") {
		t.Fatalf("master-agent --help unexpected:\n%s", out)
	}
	which := execOnMaster(t, root, "which", "ssh")
	if which == "" {
		t.Fatal("ssh client missing in master-agent image")
	}
}

func TestHarnessNoCursorBacklogMCP(t *testing.T) {
	root := repoRoot(t)
	for _, bin := range []string{"cursor", "backlog", "mcp"} {
		cmd := composeCmd(root, "exec", "-T", "master-agent", "sh", "-c", "command -v "+bin)
		if err := cmd.Run(); err == nil {
			t.Fatalf("harness image must not contain %q", bin)
		}
		cmd = composeCmd(root, "exec", "-T", "worker", "sh", "-c", "command -v "+bin)
		if err := cmd.Run(); err == nil {
			t.Fatalf("worker image must not contain %q", bin)
		}
	}
}
