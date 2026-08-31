---
id: TASK-10
title: Acceptance test harness with sshd worker
status: To Do
assignee: []
created_date: '2026-08-31 15:56'
labels: []
milestone: m-0
dependencies:
  - TASK-9
references:
  - specs/06-testing.md
  - specs/04-acceptance-criteria.md
priority: high
type: feature
ordinal: 100
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add docker-compose.test.yml (or equivalent) with master-agent + sshd worker, test SSH keys, and a Go acceptance package (build tag acceptance) that drives E2E without Cursor/AI. Stub remote commands only (touch, echo, exit 1, sleep).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Compose brings up sshd worker and master-agent (or test runner) with working key-based SSH
- [ ] #2 go test -tags=acceptance ./test/acceptance/... can run against the harness
- [ ] #3 Harness does not depend on Cursor, backlog CLI, or MCP
- [ ] #4 Document how to run acceptance tests in README or AGENTS
<!-- AC:END -->
