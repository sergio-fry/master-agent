---
id: TASK-12
title: Structured logging and SIGTERM graceful shutdown
status: Done
assignee:
  - '@auto'
created_date: '2026-08-31 15:56'
updated_date: '2026-08-31 18:00'
labels: []
milestone: m-0
dependencies:
  - TASK-8
  - TASK-4
references:
  - specs/02-scheduler.md
  - specs/04-acceptance-criteria.md
priority: high
type: enhancement
ordinal: 125
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Emit structured logs for each run (project, task, ssh_host, pid, duration, exit_code). On SIGTERM, finish the current SSH run then exit cleanly.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Logs include required run fields at info/error levels as appropriate
- [x] #2 SIGTERM waits for in-flight run before process exit
- [x] #3 Covered by unit test (signal/fake runner) and/or acceptance where practical
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Review specs and current daemon/scheduler/SSH run path. 2. Add structured run logging (project, task, ssh_host, pid, duration, exit_code). 3. Wire SIGTERM to finish in-flight SSH run then exit cleanly. 4. Add unit tests with fake runner; verify build. 5. Finalize AC and commit.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Ordered after TASK-4; completes MVP observability before Web UI foundation (TASK-13).

Explored: daemon already uses signal.NotifyContext, but Tick passes cancelable ctx into SSHRunner (CommandContext) so SIGTERM would kill in-flight SSH. Current run log lacks ssh_host/pid/duration. Plan: slog fields + WithoutCancel for runs + Result.PID from ssh Start/Wait.

Implemented: Result.PID via ssh Start/Wait; slog run fields (project/task/ssh_host/pid/duration_ms/exit_code); context.WithoutCancel for in-flight runs; unit tests for log fields + cancel-waits.

Validation: go test ./... and go build ./... passed (PATH=/usr/local/go/bin). Covered by TestTickStructuredRunLogFields, TestTickStructuredRunLogErrorLevel, TestRunWaitsForInFlightOnCancel.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added structured slog run logs (project, task, ssh_host, pid, duration_ms, exit_code) and SIGTERM-safe shutdown that finishes the in-flight SSH run via context.WithoutCancel. Verified with unit tests and go build ./...
<!-- SECTION:FINAL_SUMMARY:END -->
