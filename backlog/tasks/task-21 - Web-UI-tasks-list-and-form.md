---
id: TASK-21
title: 'Web UI: tasks list and form'
status: To Do
assignee: []
created_date: '2026-08-31 17:27'
labels: []
milestone: m-1
dependencies:
  - TASK-15
  - TASK-19
references:
  - specs/07-web-ui.md
  - specs/01-data-model.md
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
- [ ] #1 Tasks for a project are listed and editable via API-backed forms
- [ ] #2 Create task requires command, prompt, interval; no SSH fields in the form
- [ ] #3 Disable task is available from the UI
<!-- AC:END -->
