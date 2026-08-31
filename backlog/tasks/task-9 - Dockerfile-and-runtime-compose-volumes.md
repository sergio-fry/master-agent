---
id: TASK-9
title: Dockerfile and runtime compose volumes
status: Done
assignee:
  - '@root'
created_date: '2026-08-31 15:56'
updated_date: '2026-08-31 16:59'
labels: []
milestone: m-0
dependencies:
  - TASK-8
references:
  - specs/05-tech-stack.md
  - specs/03-agent-invocation.md
  - specs/04-acceptance-criteria.md
modified_files:
  - Dockerfile
  - docker-compose.yml
  - .dockerignore
  - .gitignore
  - README.md
  - specs/05-tech-stack.md
  - data/.gitkeep
  - secrets/.gitkeep
  - ssh/known_hosts.example
priority: high
type: feature
ordinal: 90
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Ship a Docker image with the Go binary and openssh-client. Document/compose mounts for /data (SQLite, logs), per-project SSH keys (ro), and known_hosts. Secrets must not be baked into the image.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Dockerfile multi-stage build produces an image that runs master-agent daemon
- [x] #2 Image includes openssh-client; does not include Cursor/MCP/backlog
- [x] #3 Compose or docs show volumes for DB, keys, known_hosts with StrictHostKeyChecking-ready setup
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Review current Go layout, CLI entrypoints, and data/SSH path conventions.
2. Add multi-stage Dockerfile (build Go binary → minimal runtime with openssh-client only).
3. Add docker-compose.yml (or compose snippet) mounting /data, SSH keys (ro), known_hosts; document StrictHostKeyChecking setup.
4. Update README with run/compose volume guidance; ensure secrets stay out of the image.
5. Verify docker build and that the image can run master-agent daemon.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Started: reviewing CLI defaults (/data/master-agent.db), SSH StrictHostKeyChecking=yes; implementing Dockerfile + compose volumes.

Added Dockerfile (multi-stage), docker-compose.yml volumes, .dockerignore, README Docker section, data/secrets placeholders, ssh/known_hosts.example. Next: docker compose build + daemon smoke.

Verified: docker compose build OK; image has /usr/bin/ssh + master-agent, no cursor/backlog; daemon container Running creates /data/master-agent.db; mounts /data /secrets /etc/ssh/ssh_known_hosts OK; go test ./... OK in golang:1.26-alpine.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added multi-stage Dockerfile (Go binary + openssh-client only), docker-compose.yml with /data, /secrets:ro, known_hosts mounts, and README/spec volume docs. Verified with docker compose build, daemon smoke (DB created, mounts OK), and go test ./...
<!-- SECTION:FINAL_SUMMARY:END -->
