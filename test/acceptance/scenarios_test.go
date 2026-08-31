//go:build acceptance

package acceptance

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scenarioEnv resets DB/daemon state and waits for SSH before each scenario.
func scenarioEnv(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	waitForAllWorkers(t, root, 60*time.Second)
	stopDaemon(t, root)
	require.NoError(t, resetAcceptanceVolume(root))
	cleanWorkspaces(t, root)
	t.Cleanup(func() { stopDaemon(t, root) })
	return root
}

func cleanWorkspaces(t *testing.T, root string) {
	t.Helper()
	execSSHHost(t, root, workerHost, keyPath, "rm -rf "+workspacePath+"/*")
	execSSHHost(t, root, workerBHost, keyPathB, "rm -rf "+workspacePath+"/*")
}

func stopDaemon(t *testing.T, root string) {
	t.Helper()
	cmd := composeCmd(root, "exec", "-T", "master-agent", "sh", "-c",
		"pkill -f '[m]aster-agent daemon' || true")
	_ = cmd.Run()
	time.Sleep(300 * time.Millisecond)
}

func startDaemon(t *testing.T, root string) {
	t.Helper()
	stopDaemon(t, root)
	cmd := composeCmd(root, "exec", "-d", "-T", "master-agent",
		"master-agent", "daemon", "--tick-interval", "1s")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "start daemon: %s", out)
}

func ma(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"master-agent"}, args...)
	return execOnMaster(t, root, cmdArgs...)
}

func addProject(t *testing.T, root, name, host, key string) {
	t.Helper()
	out := ma(t, root,
		"project", "add",
		"--name", name,
		"--path", workspacePath,
		"--ssh-host", host,
		"--ssh-user", workerUser,
		"--ssh-key", key,
	)
	require.Contains(t, out, "project created")
}

func addTask(t *testing.T, root, project, name, command string, intervalSec int) {
	t.Helper()
	out := ma(t, root,
		"task", "add",
		"--project", project,
		"--name", name,
		"--interval", fmt.Sprintf("%d", intervalSec),
		"--command", command,
		"--prompt", "acceptance stub",
	)
	require.Contains(t, out, "task created")
}

func sqlQuery(t *testing.T, root, query string) string {
	t.Helper()
	out, err := sqlQueryErr(root, query)
	require.NoError(t, err, "sqlite query %q", query)
	return out
}

func sqlQueryErr(root, query string) (string, error) {
	cmd := composeCmd(root, "exec", "-T", "master-agent",
		"sqlite3", "-cmd", ".timeout 5000", containerDB, query)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func sqlExec(t *testing.T, root, query string) {
	t.Helper()
	cmd := composeCmd(root, "exec", "-T", "master-agent",
		"sqlite3", "-cmd", ".timeout 5000", containerDB, query)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "sqlite exec %q: %s", query, out)
}

func waitUntil(t *testing.T, timeout time.Duration, desc string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s after %s", desc, timeout)
}

func workspaceFlagOnMaster(t *testing.T, root, mount, name string) bool {
	t.Helper()
	path := mount + "/" + name
	out := execOnMaster(t, root, "sh", "-c",
		"test -f "+shellQuote(path)+" && echo yes || echo no")
	return out == "yes"
}

func taskIDByName(t *testing.T, root, name string) string {
	t.Helper()
	id := sqlQuery(t, root, fmt.Sprintf(
		`SELECT id FROM tasks WHERE name = %s LIMIT 1;`, sqlString(name)))
	require.NotEmpty(t, id, "task %q", name)
	return id
}

func projectIDForTask(t *testing.T, root, taskID string) string {
	t.Helper()
	id := sqlQuery(t, root, fmt.Sprintf(
		`SELECT project_id FROM tasks WHERE id = %s;`, sqlString(taskID)))
	require.NotEmpty(t, id)
	return id
}

func sqlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func latestRunStatus(t *testing.T, root, taskID string) string {
	t.Helper()
	out, err := sqlQueryErr(root, fmt.Sprintf(
		`SELECT status FROM runs WHERE task_id = %s ORDER BY started_at DESC, id DESC LIMIT 1;`,
		sqlString(taskID)))
	if err != nil {
		return ""
	}
	return out
}

func latestRunExit(t *testing.T, root, taskID string) int {
	t.Helper()
	raw := sqlQuery(t, root, fmt.Sprintf(
		`SELECT exit_code FROM runs WHERE task_id = %s ORDER BY started_at DESC, id DESC LIMIT 1;`,
		sqlString(taskID)))
	n, err := strconv.Atoi(raw)
	require.NoError(t, err, "exit_code=%q", raw)
	return n
}

func runCountForTask(t *testing.T, root, taskID string) int {
	t.Helper()
	raw := sqlQuery(t, root, fmt.Sprintf(
		`SELECT COUNT(*) FROM runs WHERE task_id = %s;`, sqlString(taskID)))
	n, err := strconv.Atoi(raw)
	require.NoError(t, err)
	return n
}

func lockTaskID(t *testing.T, root, projectID string) string {
	t.Helper()
	out, err := sqlQueryErr(root, fmt.Sprintf(
		`SELECT task_id FROM locks WHERE project_id = %s;`, sqlString(projectID)))
	if err != nil {
		return ""
	}
	return out
}

func TestScenarioIdleWhenNoDueTasks(t *testing.T) {
	root := scenarioEnv(t)

	addProject(t, root, "idle-proj", workerHost, keyPath)
	addTask(t, root, "idle-proj", "idle-task", "touch idle-should-not-run", 3600)

	future := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339Nano)
	sqlExec(t, root, fmt.Sprintf(
		`UPDATE tasks SET next_run_at = %s WHERE name = 'idle-task';`, sqlString(future)))

	startDaemon(t, root)
	time.Sleep(3500 * time.Millisecond)
	stopDaemon(t, root)

	taskID := taskIDByName(t, root, "idle-task")
	assert.Equal(t, 0, runCountForTask(t, root, taskID))
	assert.False(t, workspaceFlagOnMaster(t, root, "/workspace", "idle-should-not-run"))
}

func TestScenarioDueTaskOverSSH(t *testing.T) {
	root := scenarioEnv(t)

	flag := "due-flag"
	addProject(t, root, "due-proj", workerHost, keyPath)
	addTask(t, root, "due-proj", "due-task",
		fmt.Sprintf("touch %s && pwd > due-pwd.txt", flag), 3600)

	startDaemon(t, root)
	waitUntil(t, 20*time.Second, "due run success", func() bool {
		id, err := sqlQueryErr(root, `SELECT id FROM tasks WHERE name = 'due-task' LIMIT 1;`)
		if err != nil || id == "" {
			return false
		}
		return latestRunStatus(t, root, id) == "success"
	})
	stopDaemon(t, root)

	taskID := taskIDByName(t, root, "due-task")
	projectID := projectIDForTask(t, root, taskID)
	assert.Equal(t, "success", latestRunStatus(t, root, taskID))
	assert.Equal(t, 0, latestRunExit(t, root, taskID))
	assert.Empty(t, lockTaskID(t, root, projectID))

	next := sqlQuery(t, root, fmt.Sprintf(
		`SELECT next_run_at FROM tasks WHERE id = %s;`, sqlString(taskID)))
	last := sqlQuery(t, root, fmt.Sprintf(
		`SELECT last_run_at FROM tasks WHERE id = %s;`, sqlString(taskID)))
	require.NotEmpty(t, next)
	require.NotEmpty(t, last)

	assert.True(t, workspaceFlagOnMaster(t, root, "/workspace", flag))
	pwd := execOnMaster(t, root, "cat", "/workspace/due-pwd.txt")
	assert.Equal(t, workspacePath, strings.TrimSpace(pwd))
}

func TestScenarioSkipWhenProjectLocked(t *testing.T) {
	root := scenarioEnv(t)

	addProject(t, root, "lock-proj", workerHost, keyPath)
	addTask(t, root, "lock-proj", "t1-hold", "sleep 8; touch t1-done", 3600)

	startDaemon(t, root)

	var projectID, t1ID string
	waitUntil(t, 15*time.Second, "lock held by t1", func() bool {
		id, err := sqlQueryErr(root, `SELECT id FROM tasks WHERE name = 't1-hold' LIMIT 1;`)
		if err != nil || id == "" {
			return false
		}
		t1ID = id
		pid, err := sqlQueryErr(root, fmt.Sprintf(
			`SELECT project_id FROM tasks WHERE id = %s;`, sqlString(t1ID)))
		if err != nil || pid == "" {
			return false
		}
		projectID = pid
		return lockTaskID(t, root, projectID) == t1ID
	})

	addTask(t, root, "lock-proj", "t2-skip", "touch t2-should-not-run", 3600)
	time.Sleep(2500 * time.Millisecond)

	t2ID := taskIDByName(t, root, "t2-skip")
	assert.Equal(t, 0, runCountForTask(t, root, t2ID), "T2 must not start while locked")
	assert.False(t, workspaceFlagOnMaster(t, root, "/workspace", "t2-should-not-run"))
	assert.Equal(t, t1ID, lockTaskID(t, root, projectID))

	waitUntil(t, 20*time.Second, "t1 finished and lock released", func() bool {
		if lockTaskID(t, root, projectID) != "" {
			return false
		}
		return latestRunStatus(t, root, t1ID) == "success"
	})
	// Stop before a later tick can start T2 — scenario only requires skip while locked.
	stopDaemon(t, root)

	assert.Equal(t, 0, runCountForTask(t, root, t2ID))
	assert.True(t, workspaceFlagOnMaster(t, root, "/workspace", "t1-done"))
}

func TestScenarioRemoteExitError(t *testing.T) {
	root := scenarioEnv(t)

	addProject(t, root, "err-proj", workerHost, keyPath)
	addTask(t, root, "err-proj", "fail-task", "exit 1", 60)

	startDaemon(t, root)
	waitUntil(t, 20*time.Second, "error run recorded", func() bool {
		id, err := sqlQueryErr(root, `SELECT id FROM tasks WHERE name = 'fail-task' LIMIT 1;`)
		if err != nil || id == "" {
			return false
		}
		return latestRunStatus(t, root, id) == "error"
	})

	addTask(t, root, "err-proj", "ok-after-error", "touch recovered", 3600)
	waitUntil(t, 20*time.Second, "daemon continues after error", func() bool {
		return workspaceFlagOnMaster(t, root, "/workspace", "recovered")
	})
	stopDaemon(t, root)

	taskID := taskIDByName(t, root, "fail-task")
	projectID := projectIDForTask(t, root, taskID)
	assert.Equal(t, "error", latestRunStatus(t, root, taskID))
	assert.Equal(t, 1, latestRunExit(t, root, taskID))
	assert.Empty(t, lockTaskID(t, root, projectID))

	last := sqlQuery(t, root, fmt.Sprintf(
		`SELECT last_run_at FROM tasks WHERE id = %s;`, sqlString(taskID)))
	next := sqlQuery(t, root, fmt.Sprintf(
		`SELECT next_run_at FROM tasks WHERE id = %s;`, sqlString(taskID)))
	require.NotEmpty(t, last)
	require.NotEmpty(t, next)
	lastT, err := time.Parse(time.RFC3339Nano, last)
	require.NoError(t, err)
	nextT, err := time.Parse(time.RFC3339Nano, next)
	require.NoError(t, err)
	assert.WithinDuration(t, lastT.Add(60*time.Second), nextT, 2*time.Second)
}

func TestScenarioSSHFailure(t *testing.T) {
	root := scenarioEnv(t)

	addProject(t, root, "sshfail-proj", workerHost, keyPathB)
	addTask(t, root, "sshfail-proj", "sshfail-task", "touch should-not-exist", 120)

	startDaemon(t, root)
	waitUntil(t, 20*time.Second, "ssh failure run", func() bool {
		id, err := sqlQueryErr(root, `SELECT id FROM tasks WHERE name = 'sshfail-task' LIMIT 1;`)
		if err != nil || id == "" {
			return false
		}
		return latestRunStatus(t, root, id) == "error"
	})
	stopDaemon(t, root)

	taskID := taskIDByName(t, root, "sshfail-task")
	projectID := projectIDForTask(t, root, taskID)
	assert.Equal(t, "error", latestRunStatus(t, root, taskID))
	assert.Empty(t, lockTaskID(t, root, projectID))
	assert.False(t, workspaceFlagOnMaster(t, root, "/workspace", "should-not-exist"))
	next := sqlQuery(t, root, fmt.Sprintf(
		`SELECT next_run_at FROM tasks WHERE id = %s;`, sqlString(taskID)))
	require.NotEmpty(t, next)
}

func TestScenarioPerProjectSSHIdentity(t *testing.T) {
	root := scenarioEnv(t)

	addProject(t, root, "proj-a", workerHost, keyPath)
	addProject(t, root, "proj-b", workerBHost, keyPathB)
	addTask(t, root, "proj-a", "task-a", "echo A > identity.txt", 3600)
	addTask(t, root, "proj-b", "task-b", "echo B > identity.txt", 3600)

	startDaemon(t, root)
	waitUntil(t, 45*time.Second, "both projects ran", func() bool {
		return workspaceFlagOnMaster(t, root, "/workspace", "identity.txt") &&
			workspaceFlagOnMaster(t, root, "/workspace-b", "identity.txt")
	})
	stopDaemon(t, root)

	assert.Equal(t, "A", strings.TrimSpace(execOnMaster(t, root, "cat", "/workspace/identity.txt")))
	assert.Equal(t, "B", strings.TrimSpace(execOnMaster(t, root, "cat", "/workspace-b/identity.txt")))

	for _, name := range []string{"task-a", "task-b"} {
		id := taskIDByName(t, root, name)
		assert.Equal(t, "success", latestRunStatus(t, root, id), name)
	}

	hostA := sqlQuery(t, root, `SELECT ssh_host FROM projects WHERE name = 'proj-a';`)
	keyA := sqlQuery(t, root, `SELECT ssh_key_path FROM projects WHERE name = 'proj-a';`)
	hostB := sqlQuery(t, root, `SELECT ssh_host FROM projects WHERE name = 'proj-b';`)
	keyB := sqlQuery(t, root, `SELECT ssh_key_path FROM projects WHERE name = 'proj-b';`)
	assert.Equal(t, workerHost, hostA)
	assert.Equal(t, keyPath, keyA)
	assert.Equal(t, workerBHost, hostB)
	assert.Equal(t, keyPathB, keyB)
}
