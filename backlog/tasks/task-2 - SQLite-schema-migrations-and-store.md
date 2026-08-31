---
id: TASK-2
title: 'SQLite schema, migrations, and store'
status: Done
assignee:
  - '@Sergei'
created_date: '2026-08-31 15:55'
updated_date: '2026-08-31 16:17'
labels: []
milestone: m-0
dependencies:
  - TASK-1
references:
  - specs/01-data-model.md
  - specs/04-acceptance-criteria.md
  - specs/06-testing.md
modified_files:
  - go.mod
  - go.sum
  - internal/store/store.go
  - internal/store/models.go
  - internal/store/crud.go
  - internal/store/schema.sql
  - internal/store/store_test.go
priority: high
type: feature
ordinal: 20
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement SQLite persistence for projects, tasks, locks, and runs per specs/01-data-model.md. Auto-create DB file and migrate schema on startup. Provide a store API used by CLI and daemon (no HTTP).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 On first start, DB file is created and tables projects, tasks, locks, runs exist
- [x] #2 Store can create/read/update Project with path, ssh_host, ssh_user, ssh_port, ssh_key_path, enabled
- [x] #3 Store can create/read/update Task bound to project with prompt, command, interval_seconds, enabled, last_run_at, next_run_at
- [x] #4 Store supports lock insert/delete and run insert/update for lifecycle fields
- [x] #5 Unit/integration tests cover schema init and basic CRUD against a temp SQLite file
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add modernc.org/sqlite driver and Open(path) that creates parent dirs, DB file, and runs embedded schema migrations.
2. Define Project/Task/Lock/Run types matching specs/01-data-model.md.
3. Implement store CRUD: project create/get/update; task create/get/update; lock insert/delete; run insert/update.
4. Add unit/integration tests against a temp SQLite file covering schema init and CRUD.
5. Verify go build ./... and go test ./...
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Started implementation: adding modernc.org/sqlite and store package.

Verified go build ./... and go test ./... (store CRUD + schema init).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Implemented SQLite store with modernc.org/sqlite (no CGO): Open creates DB file + parent dirs and applies embedded schema (projects, tasks, locks, runs). CRUD covers Project/Task create-read-update, Lock insert/delete, Run insert/update for lifecycle fields. Unit/integration tests against temp SQLite pass. Verified: go build ./... and go test ./...
<!-- SECTION:FINAL_SUMMARY:END -->
