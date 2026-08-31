// Package runner executes remote task commands over SSH.
package runner

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"master-agent/internal/store"
)

// DefaultErrorMessageBytes is the max stderr/error text kept for Run.error_message.
const DefaultErrorMessageBytes = 8 * 1024

// Result is the outcome of one remote execution attempt.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Failed reports whether the daemon should record the run as error.
func (r Result) Failed() bool {
	return r.ExitCode != 0
}

// ErrorMessage returns a daemon-usable error string (stderr tail, or exit summary).
func (r Result) ErrorMessage() string {
	msg := strings.TrimSpace(r.Stderr)
	if msg == "" {
		if r.ExitCode == 0 {
			return ""
		}
		return fmt.Sprintf("remote command exited with code %d", r.ExitCode)
	}
	return trimTail(msg, DefaultErrorMessageBytes)
}

func trimTail(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}

// Runner runs an already-substituted command on a Project's SSH target.
// Implementations must not spawn local CLI agents in the orchestrator container.
type Runner interface {
	Run(ctx context.Context, project store.Project, command string) (Result, error)
}

// FakeCall records one FakeRunner.Run invocation.
type FakeCall struct {
	Project store.Project
	Command string
}

// FakeRunner is a test double for scheduler and other unit tests.
type FakeRunner struct {
	mu sync.Mutex

	Calls []FakeCall

	// Result and Err are returned when ResultFunc is nil.
	Result Result
	Err    error

	// ResultFunc overrides Result/Err when set.
	ResultFunc func(ctx context.Context, project store.Project, command string) (Result, error)
}

// Run records the call and returns the configured result.
func (f *FakeRunner) Run(ctx context.Context, project store.Project, command string) (Result, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, FakeCall{Project: project, Command: command})
	fn := f.ResultFunc
	res, err := f.Result, f.Err
	f.mu.Unlock()

	if fn != nil {
		return fn(ctx, project, command)
	}
	return res, err
}
