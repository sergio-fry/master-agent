---
id: TASK-16
title: 'HTTP API: runs list, detail, and log tail'
status: Done
assignee:
  - '@agent'
created_date: '2026-08-31 17:26'
updated_date: '2026-08-31 19:33'
labels: []
milestone: m-1
dependencies:
  - TASK-13
  - TASK-4
references:
  - specs/07-web-ui.md
  - specs/01-data-model.md
modified_files:
  - internal/api/runs.go
  - internal/api/runs_test.go
  - internal/api/server.go
  - internal/store/crud.go
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
- [x] #1 GET /api/v1/runs supports filter by project_id, task_id, status
- [x] #2 GET /api/v1/runs/{id} returns metadata; GET .../log returns log text with size limit and 404 if missing
- [x] #3 Tests cover listing sample runs and reading a temp log file
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add store ListRunsFilter (project_id, task_id, status).\n2. Add GET /api/v1/runs, /runs/{id}, /runs/{id}/log handlers.\n3. Handler tests: list filters + temp log file read.\n4. go test / build.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Starting implementation.

Implemented ListRunsFilter, runs API handlers, and handler tests.

Verification: go test ./internal/api/... ./internal/store/... ./... and go build ./cmd/master-agent — all passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added GET /api/v1/runs (filters: project_id, task_id, status), GET /api/v1/runs/{id}, and GET /api/v1/runs/{id}/log (1MB limit, text/plain). Store ListRunsFilter backs optional query filters. Handler tests cover listing filters and temp log file read/404. Verified with go test ./internal/api/... ./internal/store/... ./... and go build ./cmd/master-agent.
<!-- SECTION:FINAL_SUMMARY:END -->
