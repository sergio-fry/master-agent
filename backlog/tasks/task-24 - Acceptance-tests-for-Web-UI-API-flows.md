---
id: TASK-24
title: Acceptance tests for Web UI API flows
status: Done
assignee:
  - '@agent'
created_date: '2026-08-31 17:27'
updated_date: '2026-08-31 21:03'
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
- [x] #1 Acceptance (or integration) tests create project+task via API, upload a key to temp secrets, list runs/logs
- [x] #2 Auth rejection when token required is covered
- [x] #3 Tests do not depend on Cursor, backlog CLI, or MCP
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Explore existing acceptance harness and HTTP API handlers
2. Add acceptance tests for API: project/task CRUD, key upload, runs/logs
3. Add auth rejection test when MASTER_AGENT_TOKEN is set
4. Run unit + acceptance tests and verify build
5. Finalize task and commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added test/acceptance/api_test.go and published port 8080 in docker-compose.test.yml.

Fixed harness TestMain: wait for master-agent container before sqlite install; install sqlite when ACCEPTANCE_SKIP_COMPOSE=1. Verified: go test ./..., go build ./..., go test -tags=acceptance ./test/acceptance/... (all PASS, ~199s).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added acceptance API tests (project/task CRUD, SSH key upload, runs/logs, auth rejection) in test/acceptance/api_test.go; published port 18080 in docker-compose.test.yml; hardened harness TestMain with container-ready wait and sqlite install for SKIP_COMPOSE. Verified: go test ./..., go build ./..., go test -tags=acceptance ./test/acceptance/... (all PASS).
<!-- SECTION:FINAL_SUMMARY:END -->
