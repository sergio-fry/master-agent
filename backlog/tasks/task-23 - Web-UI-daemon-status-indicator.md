---
id: TASK-23
title: 'Web UI: daemon status indicator'
status: To Do
assignee: []
created_date: '2026-08-31 17:27'
labels: []
milestone: m-1
dependencies:
  - TASK-18
  - TASK-19
references:
  - specs/07-web-ui.md
  - specs/02-scheduler.md
priority: medium
type: enhancement
ordinal: 300
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Small status view: whether HTTP is up, optional active lock/run summary from GET /status when daemon shares process or status is available from DB locks.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Status endpoint or page shows DB connectivity and any active project lock from SQLite
- [ ] #2 UI surfaces this without requiring Cursor or SSH
<!-- AC:END -->
