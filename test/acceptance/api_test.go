//go:build acceptance

package acceptance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAPIToken     = "acceptance-test-token"
	apiSecretsDir    = "/data/secrets"
	defaultAPIPort   = "18080"
)

func apiBaseURL() string {
	if u := os.Getenv("ACCEPTANCE_API_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	port := os.Getenv("ACCEPTANCE_API_PORT")
	if port == "" {
		port = defaultAPIPort
	}
	return "http://127.0.0.1:" + port
}

type apiClient struct {
	base   string
	token  string
	client *http.Client
}

func newAPIClient(token string) *apiClient {
	return &apiClient{
		base:   apiBaseURL(),
		token:  token,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *apiClient) do(method, path string, body io.Reader, contentType string) (*http.Response, []byte) {
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		panic(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return resp, raw
}

type apiProject struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	SSHHost    string `json:"ssh_host"`
	SSHUser    string `json:"ssh_user"`
	SSHPort    int    `json:"ssh_port"`
	SSHKeyPath string `json:"ssh_key_path"`
	Enabled    bool   `json:"enabled"`
}

type apiTask struct {
	ID              string `json:"id"`
	ProjectID       string `json:"project_id"`
	Name            string `json:"name"`
	Prompt          string `json:"prompt"`
	Command         string `json:"command"`
	IntervalSeconds int    `json:"interval_seconds"`
	Enabled         bool   `json:"enabled"`
}

type apiRun struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	ProjectID string `json:"project_id"`
	Status    string `json:"status"`
}

type apiKeyStatus struct {
	Present bool `json:"present"`
}

type apiError struct {
	Error string `json:"error"`
}

func stopServe(t *testing.T, root string) {
	t.Helper()
	cmd := composeCmd(root, "exec", "-T", "master-agent", "sh", "-c",
		"pkill -f '[m]aster-agent serve' || true")
	_ = cmd.Run()
	time.Sleep(300 * time.Millisecond)
}

func startServe(t *testing.T, root, token string) {
	t.Helper()
	stopServe(t, root)
	script := "mkdir -p " + apiSecretsDir + " && master-agent serve --addr 0.0.0.0:8080 --secrets-dir " + apiSecretsDir
	cmd := composeCmd(root, "exec", "-d", "-T", "master-agent", "sh", "-c", script)
	if token != "" {
		cmd = composeCmd(root, "exec", "-d", "-T", "-e", "MASTER_AGENT_TOKEN="+token,
			"master-agent", "sh", "-c", script)
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "start serve: %s", out)

	client := newAPIClient(token)
	waitUntil(t, 30*time.Second, "HTTP API ready", func() bool {
		resp, err := http.NewRequest(http.MethodGet, client.base+"/api/v1/status", nil)
		if err != nil {
			return false
		}
		req := resp
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		httpResp, err := client.client.Do(req)
		if err != nil {
			return false
		}
		defer httpResp.Body.Close()
		return httpResp.StatusCode == http.StatusOK
	})
}

func resetAPISecrets(t *testing.T, root string) {
	t.Helper()
	execOnMaster(t, root, "sh", "-c", "rm -rf "+apiSecretsDir)
}

func apiScenarioEnv(t *testing.T, token string) (string, *apiClient) {
	t.Helper()
	root := scenarioEnv(t)
	resetAPISecrets(t, root)
	stopServe(t, root)
	startServe(t, root, token)
	t.Cleanup(func() { stopServe(t, root) })
	return root, newAPIClient(token)
}

func fixtureSSHKey(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "test/fixtures/ssh/id_ed25519"))
	require.NoError(t, err)
	return data
}

func (c *apiClient) createProject(t *testing.T, body map[string]any) apiProject {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	resp, data := c.do(http.MethodPost, "/api/v1/projects", bytes.NewReader(raw), "application/json")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "body: %s", data)
	var p apiProject
	require.NoError(t, json.Unmarshal(data, &p))
	return p
}

func (c *apiClient) uploadProjectKey(t *testing.T, projectID string, key []byte) apiKeyStatus {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("key", "id_ed25519")
	require.NoError(t, err)
	_, err = part.Write(key)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	resp, data := c.do(http.MethodPost, "/api/v1/projects/"+projectID+"/key", &buf, w.FormDataContentType())
	require.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", data)
	assert.NotContains(t, string(data), "BEGIN OPENSSH PRIVATE KEY")

	var status apiKeyStatus
	require.NoError(t, json.Unmarshal(data, &status))
	return status
}

func (c *apiClient) createTask(t *testing.T, projectID string, body map[string]any) apiTask {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	path := fmt.Sprintf("/api/v1/projects/%s/tasks", projectID)
	resp, data := c.do(http.MethodPost, path, bytes.NewReader(raw), "application/json")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "body: %s", data)
	var task apiTask
	require.NoError(t, json.Unmarshal(data, &task))
	return task
}

func TestScenarioAPIProjectTaskCRUD(t *testing.T) {
	_, client := apiScenarioEnv(t, "")

	created := client.createProject(t, map[string]any{
		"name":         "api-proj",
		"path":         workspacePath,
		"ssh_host":     workerHost,
		"ssh_user":     workerUser,
		"ssh_key_path": apiSecretsDir + "/placeholder/id_ed25519",
		"enabled":      true,
	})
	require.NotEmpty(t, created.ID)
	assert.Equal(t, "api-proj", created.Name)
	assert.Equal(t, workerHost, created.SSHHost)

	resp, listData := client.do(http.MethodGet, "/api/v1/projects", nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var projects []apiProject
	require.NoError(t, json.Unmarshal(listData, &projects))
	require.Len(t, projects, 1)
	assert.Equal(t, created.ID, projects[0].ID)

	resp, getData := client.do(http.MethodGet, "/api/v1/projects/"+created.ID, nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var got apiProject
	require.NoError(t, json.Unmarshal(getData, &got))
	assert.Equal(t, created.Name, got.Name)

	patchRaw, err := json.Marshal(map[string]any{"enabled": false})
	require.NoError(t, err)
	resp, patchData := client.do(http.MethodPatch, "/api/v1/projects/"+created.ID, bytes.NewReader(patchRaw), "application/json")
	require.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", patchData)
	require.NoError(t, json.Unmarshal(patchData, &got))
	assert.False(t, got.Enabled)

	task := client.createTask(t, created.ID, map[string]any{
		"name":             "api-task",
		"prompt":           "acceptance stub",
		"command":          "touch api-task-flag",
		"interval_seconds": 3600,
		"enabled":          true,
	})
	assert.Equal(t, created.ID, task.ProjectID)
	assert.Equal(t, "api-task", task.Name)

	resp, tasksData := client.do(http.MethodGet, "/api/v1/projects/"+created.ID+"/tasks", nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var tasks []apiTask
	require.NoError(t, json.Unmarshal(tasksData, &tasks))
	require.Len(t, tasks, 1)

	resp, taskGetData := client.do(http.MethodGet, "/api/v1/tasks/"+task.ID, nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var gotTask apiTask
	require.NoError(t, json.Unmarshal(taskGetData, &gotTask))
	assert.Equal(t, task.ID, gotTask.ID)

	taskPatch, err := json.Marshal(map[string]any{"prompt": "updated prompt"})
	require.NoError(t, err)
	resp, taskPatchData := client.do(http.MethodPatch, "/api/v1/tasks/"+task.ID, bytes.NewReader(taskPatch), "application/json")
	require.Equal(t, http.StatusOK, resp.StatusCode, "body: %s", taskPatchData)
	require.NoError(t, json.Unmarshal(taskPatchData, &gotTask))
	assert.Equal(t, "updated prompt", gotTask.Prompt)
}

func TestScenarioAPIKeyUpload(t *testing.T) {
	root, client := apiScenarioEnv(t, "")

	created := client.createProject(t, map[string]any{
		"name":         "key-proj",
		"path":         workspacePath,
		"ssh_host":     workerHost,
		"ssh_user":     workerUser,
		"ssh_key_path": apiSecretsDir + "/placeholder/id_ed25519",
	})

	resp, beforeData := client.do(http.MethodGet, "/api/v1/projects/"+created.ID+"/key", nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var before apiKeyStatus
	require.NoError(t, json.Unmarshal(beforeData, &before))
	assert.False(t, before.Present)

	key := fixtureSSHKey(t)
	status := client.uploadProjectKey(t, created.ID, key)
	assert.True(t, status.Present)

	keyPath := filepath.ToSlash(filepath.Join(apiSecretsDir, "projects", created.ID, "id_ed25519"))
	out := execOnMaster(t, root, "sh", "-c", "test -f "+shellQuote(keyPath)+" && echo present")
	assert.Equal(t, "present", out)

	resp, projData := client.do(http.MethodGet, "/api/v1/projects/"+created.ID, nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var updated apiProject
	require.NoError(t, json.Unmarshal(projData, &updated))
	assert.Equal(t, keyPath, updated.SSHKeyPath)

	resp, afterData := client.do(http.MethodGet, "/api/v1/projects/"+created.ID+"/key", nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotContains(t, string(afterData), "BEGIN OPENSSH PRIVATE KEY")
	require.NoError(t, json.Unmarshal(afterData, &status))
	assert.True(t, status.Present)
}

func TestScenarioAPIRunsAndLogs(t *testing.T) {
	root, client := apiScenarioEnv(t, "")

	project := client.createProject(t, map[string]any{
		"name":         "runs-proj",
		"path":         workspacePath,
		"ssh_host":     workerHost,
		"ssh_user":     workerUser,
		"ssh_key_path": apiSecretsDir + "/placeholder/id_ed25519",
	})
	client.uploadProjectKey(t, project.ID, fixtureSSHKey(t))

	flag := "api-run-flag"
	task := client.createTask(t, project.ID, map[string]any{
		"name":             "runs-task",
		"prompt":           "acceptance stub",
		"command":          fmt.Sprintf("touch %s", flag),
		"interval_seconds": 3600,
		"enabled":          true,
	})

	stopServe(t, root)
	startDaemon(t, root)
	waitUntil(t, 30*time.Second, "run via daemon after API setup", func() bool {
		return latestRunStatus(t, root, task.ID) == "success"
	})
	stopDaemon(t, root)

	runID := sqlQuery(t, root, fmt.Sprintf(
		`SELECT id FROM runs WHERE task_id = %s ORDER BY started_at DESC, id DESC LIMIT 1;`,
		sqlString(task.ID)))
	logPath := "/data/acceptance-run.log"
	execOnMaster(t, root, "sh", "-c", "printf 'acceptance log line\\n' > "+shellQuote(logPath))
	sqlExec(t, root, fmt.Sprintf(
		`UPDATE runs SET log_path = %s WHERE id = %s;`, sqlString(logPath), sqlString(runID)))

	startServe(t, root, "")

	resp, runsData := client.do(http.MethodGet, "/api/v1/runs?project_id="+project.ID, nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var runs []apiRun
	require.NoError(t, json.Unmarshal(runsData, &runs))
	require.NotEmpty(t, runs)
	assert.Equal(t, "success", runs[0].Status)
	assert.Equal(t, task.ID, runs[0].TaskID)

	runID = runs[0].ID
	resp, runData := client.do(http.MethodGet, "/api/v1/runs/"+runID, nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var runDetail apiRun
	require.NoError(t, json.Unmarshal(runData, &runDetail))
	assert.Equal(t, runID, runDetail.ID)

	resp, logData := client.do(http.MethodGet, "/api/v1/runs/"+runID+"/log", nil, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Contains(t, string(logData), "acceptance log line")

	assert.True(t, workspaceFlagOnMaster(t, root, "/workspace", flag))
}

func TestScenarioAPIAuthRejection(t *testing.T) {
	root := scenarioEnv(t)
	resetAPISecrets(t, root)
	stopServe(t, root)
	startServe(t, root, testAPIToken)
	t.Cleanup(func() { stopServe(t, root) })

	anon := newAPIClient("")
	resp, data := anon.do(http.MethodGet, "/api/v1/status", nil, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	var errBody apiError
	require.NoError(t, json.Unmarshal(data, &errBody))
	assert.Equal(t, "unauthorized", errBody.Error)

	wrong := newAPIClient("wrong-token")
	resp, data = wrong.do(http.MethodGet, "/api/v1/status", nil, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.NoError(t, json.Unmarshal(data, &errBody))
	assert.Equal(t, "unauthorized", errBody.Error)

	authed := newAPIClient(testAPIToken)
	resp, data = authed.do(http.MethodGet, "/api/v1/status", nil, "")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var status map[string]any
	require.NoError(t, json.Unmarshal(data, &status))
	assert.Equal(t, true, status["ok"])

	// Static UI remains reachable without API token.
	uiResp, err := http.Get(apiBaseURL() + "/")
	require.NoError(t, err)
	defer uiResp.Body.Close()
	assert.Equal(t, http.StatusOK, uiResp.StatusCode)
}
