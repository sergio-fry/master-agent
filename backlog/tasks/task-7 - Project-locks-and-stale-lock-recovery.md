---
id: TASK-7
title: Project locks and stale lock recovery
status: Done
assignee:
  - '@auto'
created_date: '2026-08-31 15:56'
updated_date: '2026-08-31 16:37'
labels: []
milestone: m-0
dependencies:
  - TASK-2
references:
  - specs/01-data-model.md
  - specs/02-scheduler.md
  - specs/06-testing.md
modified_files:
  - internal/store/lock.go
  - internal/lock/manager.go
  - internal/lock/process.go
  - internal/lock/manager_test.go
priority: high
type: feature
ordinal: 70
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Enforce one active run per project via SQLite locks. Recover stale locks when the local SSH client PID is gone. Support future parallel-by-project; MVP daemon still runs one global run.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Acquire lock in a transaction; if project already locked, acquire fails and run is skipped
- [x] #2 Release lock after run finishes (success or error)
- [x] #3 Stale recovery clears lock and marks run as error when local pid is dead
- [x] #4 Unit tests cover acquire/skip/release and stale recovery with a fake process checker
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Review existing SQLite schema/store for Lock and Run entities
2. Implement AcquireLock (transaction: fail if locked) and ReleaseLock
3. Implement stale lock recovery via ProcessChecker (alive PID check)
4. Unit tests: acquire/skip/release + stale recovery with fake process checker
5. Run go test ./... and verify build
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Starting implementation: transactional Acquire/Release on store + ProcessChecker stale recovery in internal/lock.

Implemented store.AcquireRunLock/ReleaseRunLock/ListLocks/ClearStaleLock and lock.Manager with ProcessChecker + RecoverStale. Writing/running unit tests.

Verified: go build ./... and go test ./... (internal/lock acquire/skip/release + stale recovery with FakeProcessChecker).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added transactional project lock lifecycle: store.AcquireRunLock fails with ErrLocked when already locked (no run created); ReleaseRunLock updates the run and deletes the lock on success or error; ClearStaleLock + lock.Manager.RecoverStale mark runs as error ("process lost") when ProcessChecker reports a dead local PID. Unit tests in internal/lock cover acquire/skip/release and stale recovery via FakeProcessChecker. Verified with go build ./... and go test ./....
<!-- SECTION:FINAL_SUMMARY:END -->
