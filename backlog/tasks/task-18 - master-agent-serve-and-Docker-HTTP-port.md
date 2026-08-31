---
id: TASK-18
title: master-agent serve and Docker HTTP port
status: To Do
assignee: []
created_date: '2026-08-31 17:26'
labels: []
milestone: m-1
dependencies:
  - TASK-13
references:
  - specs/07-web-ui.md
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
- [ ] #1 serve starts HTTP on configured addr and serves /api/v1 health or status
- [ ] #2 daemon --http-addr mounts the same server without breaking the tick loop
- [ ] #3 docker-compose documents port publish, MASTER_AGENT_TOKEN, and secrets volume writability for uploads
<!-- AC:END -->
