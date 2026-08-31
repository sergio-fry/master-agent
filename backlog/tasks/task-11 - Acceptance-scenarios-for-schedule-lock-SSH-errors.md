---
id: TASK-11
title: 'Acceptance scenarios for schedule, lock, SSH, errors'
status: To Do
assignee: []
created_date: '2026-08-31 15:56'
labels: []
milestone: m-0
dependencies:
  - TASK-10
references:
  - specs/04-acceptance-criteria.md
  - specs/06-testing.md
priority: high
type: feature
ordinal: 110
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement acceptance tests for the Gherkin scenarios in specs/04: idle, due run over SSH, skip when locked, agent/SSH failure, per-project SSH identity. Use stub commands and assert DB state and/or remote side effects (flag files).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Idle: no SSH when nothing is due
- [ ] #2 Due task: SSH to project host/key, cwd=path, lock during run, run status and next_run_at updated
- [ ] #3 Locked project: second due task on same project does not start
- [ ] #4 Remote exit 1 or SSH failure: run error, lock released, next_run_at scheduled, daemon continues
- [ ] #5 Two projects with different keys/hosts use the correct identity each
<!-- AC:END -->
