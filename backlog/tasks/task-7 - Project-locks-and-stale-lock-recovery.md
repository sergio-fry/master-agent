---
id: TASK-7
title: Project locks and stale lock recovery
status: To Do
assignee: []
created_date: '2026-08-31 15:56'
labels: []
milestone: m-0
dependencies:
  - TASK-2
references:
  - specs/01-data-model.md
  - specs/02-scheduler.md
  - specs/06-testing.md
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
- [ ] #1 Acquire lock in a transaction; if project already locked, acquire fails and run is skipped
- [ ] #2 Release lock after run finishes (success or error)
- [ ] #3 Stale recovery clears lock and marks run as error when local pid is dead
- [ ] #4 Unit tests cover acquire/skip/release and stale recovery with a fake process checker
<!-- AC:END -->
