---
id: TASK-21
title: 'Web UI: tasks list and form'
status: Done
assignee:
  - '@agent'
created_date: '2026-08-31 17:27'
updated_date: '2026-08-31 20:06'
labels: []
milestone: m-1
dependencies:
  - TASK-15
  - TASK-19
references:
  - specs/07-web-ui.md
  - specs/01-data-model.md
modified_files:
  - internal/webui/static/tasks.html
  - internal/webui/static/tasks.js
  - internal/webui/static/index.html
  - internal/webui/static/app.js
  - internal/webui/static/style.css
  - internal/webui/webui_test.go
priority: high
type: feature
ordinal: 280
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Per-project UI for tasks: interval, command, prompt, enabled; show last_run_at / next_run_at when present.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Tasks for a project are listed and editable via API-backed forms
- [x] #2 Create task requires command, prompt, interval; no SSH fields in the form
- [x] #3 Disable task is available from the UI
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add tasks.html with project selector, table, and create/edit dialog
2. Add tasks.js wired to /api/v1/projects/{id}/tasks and /api/v1/tasks/{id}
3. Link from projects page; show last_run_at / next_run_at
4. Enable/disable toggle; no SSH fields in form
5. Run go test ./... and go build; finalize task
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added tasks.html/tasks.js with project selector, CRUD forms, enable/disable, last/next run columns. Linked from projects page. webui tests cover static assets and no SSH fields in task form.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added per-project tasks UI (tasks.html, tasks.js): list with last_run_at/next_run_at, create/edit dialog (name, interval, command, prompt, enabled), enable/disable toggle, project links from index. No SSH fields in task form. Verified with go test ./internal/webui/..., go test ./..., go build ./cmd/master-agent, docker compose -f docker-compose.test.yml build.
<!-- SECTION:FINAL_SUMMARY:END -->
