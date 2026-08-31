---
id: TASK-24
title: Acceptance tests for Web UI API flows
status: To Do
assignee: []
created_date: '2026-08-31 17:27'
labels: []
milestone: m-1
dependencies:
  - TASK-17
  - TASK-20
  - TASK-21
  - TASK-22
references:
  - specs/07-web-ui.md
  - specs/06-testing.md
priority: high
type: feature
ordinal: 310
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Automated acceptance covering HTTP API (and critical UI-backed flows): project/task CRUD, key upload, runs/log read. No Cursor/AI. Extend harness as needed.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Acceptance (or integration) tests create project+task via API, upload a key to temp secrets, list runs/logs
- [ ] #2 Auth rejection when token required is covered
- [ ] #3 Tests do not depend on Cursor, backlog CLI, or MCP
<!-- AC:END -->
