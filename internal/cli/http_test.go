package cli

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"master-agent/internal/lock"
	"master-agent/internal/runner"
	"master-agent/internal/scheduler"
	"master-agent/internal/store"
)

func TestStartHTTPServerServesStatus(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dbPath)
	require.NoError(t, err)
	defer s.Close()

	addr := freeTCPAddr(t)
	apiSrv := newAPIServer(s, t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = startHTTPServer(ctx, addr, apiSrv.Handler(), io.Discard)
	}()

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://" + addr + "/api/v1/status")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 3*time.Second, 25*time.Millisecond)

	resp, err := http.Get("http://" + addr + "/api/v1/status")
	require.NoError(t, err)
	defer resp.Body.Close()

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, true, body["ok"])
}

func TestDaemonHTTPAddrDoesNotBlockTick(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(dbPath)
	require.NoError(t, err)
	defer s.Close()

	addr := freeTCPAddr(t)
	apiSrv := newAPIServer(s, t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = startHTTPServer(ctx, addr, apiSrv.Handler(), io.Discard)
	}()

	d := &scheduler.Daemon{
		Store:  s,
		Locks:  lock.NewManager(s, nil),
		Runner: &runner.FakeRunner{},
		Config: scheduler.Config{TickInterval: time.Hour},
	}
	require.NoError(t, d.Tick(ctx))

	resp, err := http.Get("http://" + addr + "/api/v1/status")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServeCommandRegistered(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	out, err := runCLI(t, dbPath, "serve", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "--addr")
	assert.Contains(t, out, "HTTP API")
}

func TestDaemonHTTPAddrFlagRegistered(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	out, err := runCLI(t, dbPath, "daemon", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "--http-addr")
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}
