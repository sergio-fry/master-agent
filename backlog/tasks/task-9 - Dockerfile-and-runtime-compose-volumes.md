---
id: TASK-9
title: Dockerfile and runtime compose volumes
status: To Do
assignee: []
created_date: '2026-08-31 15:56'
labels: []
milestone: m-0
dependencies:
  - TASK-8
references:
  - specs/05-tech-stack.md
  - specs/03-agent-invocation.md
  - specs/04-acceptance-criteria.md
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
- [ ] #1 Dockerfile multi-stage build produces an image that runs master-agent daemon
- [ ] #2 Image includes openssh-client; does not include Cursor/MCP/backlog
- [ ] #3 Compose or docs show volumes for DB, keys, known_hosts with StrictHostKeyChecking-ready setup
<!-- AC:END -->
