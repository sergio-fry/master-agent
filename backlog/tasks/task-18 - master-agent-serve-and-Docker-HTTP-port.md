---
id: TASK-18
title: master-agent serve and Docker HTTP port
status: Done
assignee:
  - '@agent'
created_date: '2026-08-31 17:26'
updated_date: '2026-08-31 19:56'
labels: []
milestone: m-1
dependencies:
  - TASK-13
references:
  - specs/07-web-ui.md
  - specs/05-tech-stack.md
modified_files:
  - internal/cli/http.go
  - internal/cli/serve.go
  - internal/cli/http_test.go
  - internal/cli/daemon.go
  - internal/cli/cli.go
  - docker-compose.yml
  - README.md
  - specs/05-tech-stack.md
priority: high
type: feature
ordinal: 250
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Ship master-agent serve to run API+UI against --db, and optionally daemon --http-addr for one-container mode. Update compose to publish port and document rw secrets for key upload.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 serve starts HTTP on configured addr and serves /api/v1 health or status
- [x] #2 daemon --http-addr mounts the same server without breaking the tick loop
- [x] #3 docker-compose documents port publish, MASTER_AGENT_TOKEN, and secrets volume writability for uploads
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add shared HTTP server helper (api.New + graceful shutdown)
2. Implement master-agent serve --addr (default 127.0.0.1:8080)
3. Add daemon --http-addr to mount API in-process
4. Update docker-compose.yml (port, MASTER_AGENT_TOKEN, secrets rw)
5. Unit tests + verify build
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Verified: docker run golang:1.26-alpine go test ./... (all pass); docker compose build succeeds; serve --help and daemon --http-addr registered.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added master-agent serve (--addr, HTTP_ADDR) and daemon --http-addr with shared api.Server startup and graceful shutdown. Updated docker-compose.yml to publish 8080, document MASTER_AGENT_TOKEN, mount secrets rw, and run daemon with --http-addr 0.0.0.0:8080. Verified with go test ./... and docker compose build.
<!-- SECTION:FINAL_SUMMARY:END -->
