---
id: TASK-12
title: Structured logging and SIGTERM graceful shutdown
status: To Do
assignee: []
created_date: '2026-08-31 15:56'
labels: []
milestone: m-0
dependencies:
  - TASK-8
references:
  - specs/02-scheduler.md
  - specs/04-acceptance-criteria.md
priority: medium
type: enhancement
ordinal: 120
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Emit structured logs for each run (project, task, ssh_host, pid, duration, exit_code). On SIGTERM, finish the current SSH run then exit cleanly.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Logs include required run fields at info/error levels as appropriate
- [ ] #2 SIGTERM waits for in-flight run before process exit
- [ ] #3 Covered by unit test (signal/fake runner) and/or acceptance where practical
<!-- AC:END -->
