---
id: TASK-13
title: 'HTTP API foundation: router, auth, JSON errors'
status: Done
assignee:
  - '@agent'
created_date: '2026-08-31 17:25'
updated_date: '2026-08-31 18:33'
labels: []
milestone: m-1
dependencies:
  - TASK-4
  - TASK-12
references:
  - specs/07-web-ui.md
  - specs/05-tech-stack.md
modified_files:
  - internal/api/server.go
  - internal/api/middleware.go
  - internal/api/errors.go
  - internal/api/server_test.go
  - specs/07-web-ui.md
priority: high
type: feature
ordinal: 200
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add shared HTTP layer for /api/v1: router, request IDs/logging hooks, Bearer token auth when MASTER_AGENT_TOKEN is set, consistent JSON error responses. Used by all Web UI API tasks. Depends on MVP run-list and logging polish so store/observability are ready.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 HTTP server package exists and can be started from tests with a temp DB
- [x] #2 When MASTER_AGENT_TOKEN is set, unauthenticated API requests are rejected; with token they succeed
- [x] #3 JSON error body shape is documented and covered by unit tests
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Explore existing CLI/store/logging layout and HTTP-related conventions.
2. Add internal/api (or http) package: Server with mux under /api/v1, request-ID + logging middleware, Bearer auth when MASTER_AGENT_TOKEN set, JSON error helper {error: "..."}.
3. Wire a minimal health/status or placeholder route so auth and errors can be tested end-to-end.
4. Unit tests: start server with temp DB; auth reject/allow; JSON error shape.
5. go test ./... and go build; finalize AC and commit.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Starting implementation: internal/api with mux /api/v1, Bearer auth, JSON errors, GET /status stub.

Verification: go test ./internal/api/... (auth + JSON error + temp DB httptest); go test ./...; go build ./cmd/master-agent. Added internal/api (server, middleware, ErrorBody), GET /api/v1/status stub, documented error/request-id in specs/07-web-ui.md.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added shared HTTP API foundation in internal/api: /api/v1 mux, X-Request-ID + slog request logging, optional Bearer auth via MASTER_AGENT_TOKEN, and JSON {"error":"..."} helper. Minimal GET /api/v1/status for smoke/auth tests. Verified with go test ./internal/api/... and go test ./... plus go build.
<!-- SECTION:FINAL_SUMMARY:END -->
