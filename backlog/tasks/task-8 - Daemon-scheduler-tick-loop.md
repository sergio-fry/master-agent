---
id: TASK-8
title: Daemon scheduler tick loop
status: Done
assignee:
  - '@auto'
created_date: '2026-08-31 15:56'
updated_date: '2026-08-31 16:42'
labels: []
milestone: m-0
dependencies:
  - TASK-6
  - TASK-7
references:
  - specs/02-scheduler.md
  - specs/04-acceptance-criteria.md
  - specs/06-testing.md
modified_files:
  - internal/store/schedule.go
  - internal/scheduler/scheduler.go
  - internal/scheduler/scheduler_test.go
  - internal/cli/daemon.go
  - internal/cli/cli.go
priority: high
type: feature
ordinal: 80
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement master-agent daemon: configurable tick interval, find due tasks, global single active run for MVP, call Runner, update run status, schedule next_run_at after success or failure without immediate retry. Disabled projects/tasks are ignored.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Due task (enabled, project enabled, next_run_at null or past) starts a run when no global run is active
- [x] #2 While a run is in progress, no other run starts
- [x] #3 After exit 0 or non-zero, lock released, last_run_at/next_run_at updated, no immediate retry
- [x] #4 Idle when no due tasks: tick does not invoke Runner
- [x] #5 Unit tests with fake Runner cover due, idle, lock skip, failure scheduling
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add store helpers: ListDueTasks(now) and HasAnyLock (global single-run gate)
2. Implement scheduler.Daemon: Tick (recover stale → skip if locked → due → acquire → expand placeholders → Runner → release → schedule next_run_at) and Run loop with configurable tick interval
3. Wire CLI `daemon` command (--tick-interval / TICK_INTERVAL env)
4. Unit tests with FakeRunner: due start, idle, lock skip, failure scheduling
5. Verify go build ./... and go test ./...
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Starting implementation.

Implemented store ListDueTasks/HasAnyLock/ScheduleTaskAfterRun/LatestRunForTask, scheduler.Daemon Tick+Run, CLI daemon command. Writing/running unit tests.

Verified: go build ./... and go test ./... (scheduler: due/idle/lock-skip/failure scheduling with FakeRunner).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Implemented daemon tick loop: store ListDueTasks/HasAnyLock/ScheduleTaskAfterRun, scheduler.Daemon (stale recover → global lock gate → due task → placeholders → Runner → release → next_run_at), CLI `daemon` with --tick-interval/TICK_INTERVAL. Unit tests with FakeRunner cover due start, idle, global lock skip, and failure scheduling without immediate retry. Verified with go build ./... and go test ./....
<!-- SECTION:FINAL_SUMMARY:END -->
