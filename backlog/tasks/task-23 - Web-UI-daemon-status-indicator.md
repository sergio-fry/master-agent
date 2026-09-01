---
id: TASK-23
title: 'Web UI: daemon status indicator'
status: Done
assignee:
  - '@agent'
created_date: '2026-08-31 17:27'
updated_date: '2026-09-01 13:28'
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
- [x] #1 Status endpoint or page shows DB connectivity and any active project lock from SQLite
- [x] #2 UI surfaces this without requiring Cursor or SSH
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Extend GET /api/v1/status: db connectivity, db_path, active locks from SQLite
2. Pass db path into api.Config from serve/daemon
3. Add status.html + status.js; Status nav link on all UI pages
4. Unit tests for status API and webui static assets
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Verification: go test ./internal/api/... ./internal/store/... ./internal/webui/... ./internal/cli/... — all pass. Status API returns db_ok, db_path, locks; status.html loads via embedded webui.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Extended GET /api/v1/status with db_ok, db_path, and active SQLite locks (with project/task names). Added status.html page and Status nav link on all UI pages. Verified with unit tests in internal/api, internal/webui, internal/cli.
<!-- SECTION:FINAL_SUMMARY:END -->
