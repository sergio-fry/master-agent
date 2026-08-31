---
id: TASK-16
title: 'HTTP API: runs list, detail, and log tail'
status: To Do
assignee: []
created_date: '2026-08-31 17:26'
labels: []
milestone: m-1
dependencies:
  - TASK-13
  - TASK-4
references:
  - specs/07-web-ui.md
  - specs/01-data-model.md
priority: high
type: feature
ordinal: 230
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Expose run history over HTTP with filters and safe log file reading (size limits). Builds on CLI run list behavior from TASK-4.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 GET /api/v1/runs supports filter by project_id, task_id, status
- [ ] #2 GET /api/v1/runs/{id} returns metadata; GET .../log returns log text with size limit and 404 if missing
- [ ] #3 Tests cover listing sample runs and reading a temp log file
<!-- AC:END -->
