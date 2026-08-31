---
id: TASK-8
title: Daemon scheduler tick loop
status: To Do
assignee: []
created_date: '2026-08-31 15:56'
labels: []
milestone: m-0
dependencies:
  - TASK-6
  - TASK-7
references:
  - specs/02-scheduler.md
  - specs/04-acceptance-criteria.md
  - specs/06-testing.md
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
- [ ] #1 Due task (enabled, project enabled, next_run_at null or past) starts a run when no global run is active
- [ ] #2 While a run is in progress, no other run starts
- [ ] #3 After exit 0 or non-zero, lock released, last_run_at/next_run_at updated, no immediate retry
- [ ] #4 Idle when no due tasks: tick does not invoke Runner
- [ ] #5 Unit tests with fake Runner cover due, idle, lock skip, failure scheduling
<!-- AC:END -->
