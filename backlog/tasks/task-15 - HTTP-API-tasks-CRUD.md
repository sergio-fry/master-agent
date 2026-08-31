---
id: TASK-15
title: 'HTTP API: tasks CRUD'
status: Done
assignee:
  - '@root'
created_date: '2026-08-31 17:26'
updated_date: '2026-08-31 18:39'
labels: []
milestone: m-1
dependencies:
  - TASK-13
references:
  - specs/07-web-ui.md
  - specs/01-data-model.md
modified_files:
  - internal/api/tasks.go
  - internal/api/tasks_test.go
  - internal/api/server.go
priority: high
type: feature
ordinal: 220
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
JSON API for Tasks under a project: schedule (interval_seconds), command, prompt, enabled. No SSH fields on tasks.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Create/list tasks for a project; get/patch by task id
- [x] #2 API rejects SSH-related fields on tasks if supplied
- [x] #3 Handler tests cover CRUD and disable so task stops scheduling when daemon runs
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Review store Task CRUD and projects API handlers as the pattern.
2. Add GET/POST /api/v1/projects/{id}/tasks and GET/PATCH /api/v1/tasks/{id} wired to SQLite.
3. Reject SSH-related fields on task create/patch with a clear error.
4. Handler tests: CRUD, SSH field rejection, disable → task excluded from ListDueTasks.
5. Run go test / build to verify.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Starting implementation: tasks handlers on existing internal/api + store Task CRUD.

Implemented tasks handlers + routes; writing/running handler tests.

Verification: go test ./internal/api/... (CRUD, SSH reject, disable→ListDueTasks empty); go test ./...; go build ./cmd/master-agent — all passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added GET/POST /api/v1/projects/{id}/tasks and GET/PATCH /api/v1/tasks/{id} on SQLite. Tasks accept schedule/command/prompt/enabled only; SSH fields are rejected. Handler tests cover CRUD and disable excluding the task from ListDueTasks. Verified with go test ./internal/api/..., go test ./..., and go build.
<!-- SECTION:FINAL_SUMMARY:END -->
