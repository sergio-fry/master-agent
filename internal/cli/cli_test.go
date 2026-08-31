package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/store"
)

func runCLI(t *testing.T, dbPath string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := RunWithOptions(append([]string{"--db", dbPath}, args...), Options{
		Stdout: &out,
		Stderr: &out,
	})
	return out.String(), err
}

func TestProjectAddPersistsToSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	out, err := runCLI(t, dbPath,
		"project", "add",
		"--name", "my-app",
		"--path", "/home/dev/my-app",
		"--ssh-host", "dev-box",
		"--ssh-user", "dev",
		"--ssh-key", "/secrets/projects/my-app/id_ed25519",
	)
	require.NoError(t, err)
	assert.Contains(t, out, "project created: my-app")

	s, err := store.Open(dbPath)
	require.NoError(t, err)
	defer s.Close()

	p, err := s.GetProjectByName("my-app")
	require.NoError(t, err)
	assert.Equal(t, "/home/dev/my-app", p.Path)
	assert.Equal(t, "dev-box", p.SSHHost)
	assert.Equal(t, "dev", p.SSHUser)
	assert.Equal(t, 22, p.SSHPort)
	assert.Equal(t, "/secrets/projects/my-app/id_ed25519", p.SSHKeyPath)
	assert.True(t, p.Enabled)
}

func TestProjectAddCustomSSHPort(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	_, err := runCLI(t, dbPath,
		"project", "add",
		"--name", "app",
		"--path", "/p",
		"--ssh-host", "h",
		"--ssh-user", "u",
		"--ssh-key", "/k",
		"--ssh-port", "2222",
	)
	require.NoError(t, err)

	s, err := store.Open(dbPath)
	require.NoError(t, err)
	defer s.Close()
	p, err := s.GetProjectByName("app")
	require.NoError(t, err)
	assert.Equal(t, 2222, p.SSHPort)
}

func TestTaskAddRequiresProjectFieldsAndNoSSH(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	_, err := runCLI(t, dbPath,
		"project", "add",
		"--name", "my-app", "--path", "/p", "--ssh-host", "h",
		"--ssh-user", "u", "--ssh-key", "/k",
	)
	require.NoError(t, err)

	out, err := runCLI(t, dbPath,
		"task", "add",
		"--project", "my-app",
		"--name", "drain",
		"--interval", "1800",
		"--command", `cursor agent -p "{{prompt}}"`,
		"--prompt", "do work",
	)
	require.NoError(t, err)
	assert.Contains(t, out, "task created: drain")

	// Task command must not expose SSH flags.
	helpOut, err := runCLI(t, dbPath, "task", "add", "--help")
	require.NoError(t, err)
	assert.NotContains(t, helpOut, "ssh-host")
	assert.NotContains(t, helpOut, "ssh-user")
	assert.NotContains(t, helpOut, "ssh-key")
	assert.NotContains(t, helpOut, "ssh-port")

	s, err := store.Open(dbPath)
	require.NoError(t, err)
	defer s.Close()
	proj, err := s.GetProjectByName("my-app")
	require.NoError(t, err)
	tasks, err := s.ListTasks(proj.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "drain", tasks[0].Name)
	assert.Equal(t, 1800, tasks[0].IntervalSeconds)
	assert.Equal(t, "do work", tasks[0].Prompt)
	assert.Equal(t, `cursor agent -p "{{prompt}}"`, tasks[0].Command)
}

func TestListAndDisableProjectAndTask(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	_, err := runCLI(t, dbPath,
		"project", "add",
		"--name", "my-app", "--path", "/p", "--ssh-host", "h",
		"--ssh-user", "u", "--ssh-key", "/k",
	)
	require.NoError(t, err)
	_, err = runCLI(t, dbPath,
		"task", "add",
		"--project", "my-app", "--name", "drain",
		"--interval", "1800", "--command", "echo", "--prompt", "p",
	)
	require.NoError(t, err)

	listOut, err := runCLI(t, dbPath, "project", "list")
	require.NoError(t, err)
	assert.Contains(t, listOut, "my-app")
	assert.Contains(t, listOut, "true")

	taskList, err := runCLI(t, dbPath, "task", "list", "--project", "my-app")
	require.NoError(t, err)
	assert.Contains(t, taskList, "drain")

	_, err = runCLI(t, dbPath, "project", "disable", "--name", "my-app")
	require.NoError(t, err)
	_, err = runCLI(t, dbPath, "task", "disable", "--project", "my-app", "--name", "drain")
	require.NoError(t, err)

	s, err := store.Open(dbPath)
	require.NoError(t, err)
	defer s.Close()
	p, err := s.GetProjectByName("my-app")
	require.NoError(t, err)
	assert.False(t, p.Enabled)
	tasks, err := s.ListTasks(p.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.False(t, tasks[0].Enabled)

	_, err = runCLI(t, dbPath, "project", "enable", "--name", "my-app")
	require.NoError(t, err)
	_, err = runCLI(t, dbPath, "task", "enable", "--project", "my-app", "--name", "drain")
	require.NoError(t, err)
	p, err = s.GetProjectByName("my-app")
	require.NoError(t, err)
	assert.True(t, p.Enabled)
}

func TestMultipleTasksOnOneProject(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	_, err := runCLI(t, dbPath,
		"project", "add",
		"--name", "my-app", "--path", "/p", "--ssh-host", "h",
		"--ssh-user", "u", "--ssh-key", "/k",
	)
	require.NoError(t, err)

	_, err = runCLI(t, dbPath,
		"task", "add",
		"--project", "my-app", "--name", "drain",
		"--interval", "1800", "--command", "c1", "--prompt", "every 30m",
	)
	require.NoError(t, err)
	_, err = runCLI(t, dbPath,
		"task", "add",
		"--project", "my-app", "--name", "audit",
		"--interval", "86400", "--command", "c2", "--prompt", "daily",
	)
	require.NoError(t, err)

	out, err := runCLI(t, dbPath, "task", "list", "--project", "my-app")
	require.NoError(t, err)
	assert.True(t, strings.Contains(out, "drain") && strings.Contains(out, "audit"))
	assert.Contains(t, out, "1800")
	assert.Contains(t, out, "86400")

	s, err := store.Open(dbPath)
	require.NoError(t, err)
	defer s.Close()
	proj, err := s.GetProjectByName("my-app")
	require.NoError(t, err)
	tasks, err := s.ListTasks(proj.ID)
	require.NoError(t, err)
	require.Len(t, tasks, 2)
}

func TestTaskAddMissingRequiredFlags(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	_, err := runCLI(t, dbPath, "task", "add", "--name", "x")
	require.Error(t, err)
}
