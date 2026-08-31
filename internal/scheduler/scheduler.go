// Package scheduler implements the daemon tick loop and task due checks.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"master-agent/internal/lock"
	"master-agent/internal/placeholder"
	"master-agent/internal/runner"
	"master-agent/internal/store"
)

// DefaultTickInterval is used when Config.TickInterval is unset or non-positive.
const DefaultTickInterval = 30 * time.Second

// Config holds daemon runtime settings.
type Config struct {
	// TickInterval is how often the daemon polls for due tasks.
	TickInterval time.Duration
}

// Daemon runs the scheduler loop: recover stale locks, start at most one global run per tick.
type Daemon struct {
	Store  *store.Store
	Locks  *lock.Manager
	Runner runner.Runner
	Config Config

	// Now is the clock; nil uses time.Now.
	Now func() time.Time

	// Logger receives operational messages; nil uses the standard library logger.
	Logger *log.Logger
}

func (d *Daemon) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d *Daemon) logf(format string, args ...any) {
	if d.Logger != nil {
		d.Logger.Printf(format, args...)
		return
	}
	log.Printf(format, args...)
}

func (d *Daemon) tickInterval() time.Duration {
	if d.Config.TickInterval > 0 {
		return d.Config.TickInterval
	}
	return DefaultTickInterval
}

// Run loops until ctx is cancelled: Tick then sleep TickInterval.
func (d *Daemon) Run(ctx context.Context) error {
	for {
		if err := d.Tick(ctx); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			d.logf("tick error: %v", err)
		}

		timer := time.NewTimer(d.tickInterval())
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// Tick performs one scheduler iteration: stale recovery, then at most one run when due.
func (d *Daemon) Tick(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d.Store == nil || d.Locks == nil || d.Runner == nil {
		return fmt.Errorf("scheduler: Store, Locks, and Runner are required")
	}

	if _, err := d.Locks.RecoverStale(); err != nil {
		return fmt.Errorf("recover stale locks: %w", err)
	}

	busy, err := d.Store.HasAnyLock()
	if err != nil {
		return err
	}
	if busy {
		// MVP: one active run globally — skip starting another.
		return nil
	}

	due, err := d.Store.ListDueTasks(d.now())
	if err != nil {
		return err
	}
	if len(due) == 0 {
		return nil
	}

	for i := range due {
		started, err := d.tryStart(ctx, &due[i])
		if err != nil {
			return err
		}
		if started {
			return nil
		}
	}
	return nil
}

// tryStart attempts to run one due task. started is false when the project lock was held
// (race) or the task/project disappeared; true when a run was completed (success or error).
func (d *Daemon) tryStart(ctx context.Context, task *store.Task) (started bool, err error) {
	project, err := d.Store.GetProject(task.ProjectID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if !project.Enabled || !task.Enabled {
		return false, nil
	}

	command, expandErr := placeholder.Expand(task.Command, *project, *task)

	run, err := d.Locks.Acquire(task.ProjectID, task.ID, nil)
	if err != nil {
		if errors.Is(err, store.ErrLocked) {
			return false, nil
		}
		return false, err
	}

	finishedAt := d.now()
	exitCode := 0
	status := store.RunStatusSuccess
	var errMsg *string

	if expandErr != nil {
		finishedAt = d.now()
		exitCode = -1
		status = store.RunStatusError
		msg := expandErr.Error()
		errMsg = &msg
	} else {
		res, runErr := d.Runner.Run(ctx, *project, command)
		finishedAt = d.now()
		if runErr != nil {
			exitCode = res.ExitCode
			if exitCode == 0 {
				exitCode = -1
			}
			status = store.RunStatusError
			msg := runErr.Error()
			if em := res.ErrorMessage(); em != "" {
				msg = em + ": " + msg
			}
			errMsg = &msg
		} else if res.Failed() {
			exitCode = res.ExitCode
			status = store.RunStatusError
			msg := res.ErrorMessage()
			errMsg = &msg
		} else {
			exitCode = res.ExitCode
			status = store.RunStatusSuccess
		}
	}

	finishedStr := finishedAt.UTC().Format(time.RFC3339Nano)
	run.FinishedAt = &finishedStr
	run.ExitCode = &exitCode
	run.Status = status
	run.ErrorMessage = errMsg

	if err := d.Locks.Release(task.ProjectID, run); err != nil {
		return true, fmt.Errorf("release lock after run %s: %w", run.ID, err)
	}
	if err := d.Store.ScheduleTaskAfterRun(task.ID, finishedAt, task.IntervalSeconds); err != nil {
		return true, fmt.Errorf("schedule next run for task %s: %w", task.ID, err)
	}

	d.logf("run %s task=%s project=%s status=%s exit=%d", run.ID, task.Name, project.Name, status, exitCode)
	return true, nil
}
