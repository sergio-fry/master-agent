package lock

import (
	"os"
	"syscall"
)

// ProcessChecker reports whether a local process PID is still alive.
type ProcessChecker interface {
	Alive(pid int) bool
}

// OSProcessChecker uses a zero-signal probe on the local process table.
type OSProcessChecker struct{}

// Alive reports whether pid exists and is signalable by this process.
func (OSProcessChecker) Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil
}

// FakeProcessChecker is a test double mapping PIDs to liveness.
type FakeProcessChecker struct {
	AlivePIDs map[int]bool
}

// Alive returns the configured liveness for pid (false if unset).
func (f FakeProcessChecker) Alive(pid int) bool {
	if f.AlivePIDs == nil {
		return false
	}
	return f.AlivePIDs[pid]
}
