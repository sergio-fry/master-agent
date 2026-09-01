package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/runner"
	"master-agent/internal/store"
)

func TestProjectListDirsReturnsEntries(t *testing.T) {
	st := openTempStore(t)
	ts := startSSHTestServer(t, st, nil)
	srv := New(Config{
		Store: st,
		ListDirs: func(ctx context.Context, project store.Project, path string) (runner.RemoteDirListing, error) {
			return runner.RemoteDirListing{
				Path: "/home/worker",
				Parent: "/home",
				Entries: []runner.RemoteDirEntry{
					{Name: "workspace", Path: "/home/worker/workspace"},
				},
			}, nil
		},
	})
	ts2 := httptest.NewServer(srv.Handler())
	t.Cleanup(ts2.Close)
	_ = ts

	p := &store.Project{
		Name: "p", Path: "/p", SSHHost: "h", SSHUser: "u",
		SSHPrivateKey: store.TestSSHPrivateKey, Enabled: true,
	}
	require.NoError(t, st.CreateProject(p))

	resp, err := http.Post(ts2.URL+"/api/v1/projects/"+p.ID+"/ssh/list-dirs", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "/home/worker", body["path"])
	entries, ok := body["entries"].([]any)
	require.True(t, ok)
	require.Len(t, entries, 1)
}
