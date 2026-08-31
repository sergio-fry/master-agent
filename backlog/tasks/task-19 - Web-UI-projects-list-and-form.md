---
id: TASK-19
title: 'Web UI: projects list and form'
status: Done
assignee:
  - '@agent'
created_date: '2026-08-31 17:27'
updated_date: '2026-08-31 20:01'
labels: []
milestone: m-1
dependencies:
  - TASK-14
  - TASK-18
references:
  - specs/07-web-ui.md
priority: high
type: feature
ordinal: 260
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Browser UI to list projects and create/edit path, SSH host/user/port, key path, enabled flag.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Projects page lists all projects from the API
- [x] #2 Create and edit forms persist via API and reflect validation errors
- [x] #3 Enable/disable is available from the UI
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Explore serve/API and embed pattern
2. Add static UI shell (projects list + create/edit form)
3. Wire forms to /api/v1/projects CRUD
4. Enable/disable toggle in UI
5. Run tests and verify build
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Starting implementation: internal/webui with embedded static assets, compose with API handler in serve/daemon.

Verified: go test ./... passes; docker compose build succeeds. UI lists/creates/edits projects via /api/v1; enable/disable via PATCH toggle.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added embedded Web UI (internal/webui) with projects table, create/edit dialog, and enable/disable toggle wired to /api/v1/projects. serve and daemon --http-addr now mount UI via HandlerWithUI; API auth applies only to /api/ paths so the browser can load the token prompt. Verified with go test ./... and docker compose build.
<!-- SECTION:FINAL_SUMMARY:END -->
