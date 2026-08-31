---
id: TASK-14
title: 'HTTP API: projects CRUD'
status: Done
assignee:
  - '@Sergei'
created_date: '2026-08-31 17:26'
updated_date: '2026-08-31 18:36'
labels: []
milestone: m-1
dependencies:
  - TASK-13
references:
  - specs/07-web-ui.md
  - specs/01-data-model.md
modified_files:
  - internal/api/projects.go
  - internal/api/projects_test.go
  - internal/api/server.go
priority: high
type: feature
ordinal: 210
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
JSON API for listing, creating, updating, and enabling/disabling Projects (path, ssh_host, ssh_user, ssh_port, ssh_key_path, enabled) per specs/01 and specs/07.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 GET/POST /api/v1/projects and GET/PATCH /api/v1/projects/{id} work against SQLite store
- [x] #2 Responses include SSH connection fields but never private key contents
- [x] #3 Handler tests cover create, list, patch enabled, and validation errors
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Review existing internal/api scaffolding and store Project CRUD.
2. Implement GET/POST /api/v1/projects and GET/PATCH /api/v1/projects/{id} handlers wired to SQLite.
3. Ensure JSON responses expose SSH connection fields but never private key contents.
4. Add handler tests: create, list, patch enabled, validation errors.
5. Run go test / build to verify.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Starting implementation: projects handlers on existing internal/api + store CRUD.

Verification: go test ./internal/api/... (create/list/get/patch enabled, validation errors, no key material); go test ./...; go build ./cmd/master-agent — all passed.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added GET/POST /api/v1/projects and GET/PATCH /api/v1/projects/{id} on SQLite via internal/api. Responses expose SSH connection fields and ssh_key_path only (never key contents). Handler tests cover create, list, patch enabled, and validation. Verified with go test ./internal/api/..., go test ./..., and go build.
<!-- SECTION:FINAL_SUMMARY:END -->
