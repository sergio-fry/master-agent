---
id: TASK-6
title: SSH runner with Runner interface
status: To Do
assignee: []
created_date: '2026-08-31 15:55'
labels: []
milestone: m-0
dependencies:
  - TASK-5
references:
  - specs/03-agent-invocation.md
  - specs/05-tech-stack.md
  - specs/06-testing.md
priority: high
type: feature
ordinal: 60
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement remote execution only via SSH using Project host/user/port/key and remote cwd Project.path. Introduce a Runner interface so unit tests can fake execution. Use BatchMode, IdentitiesOnly, StrictHostKeyChecking, and keepalive options per specs/03.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Runner builds SSH invocation from Project fields and substituted command; no local agent spawn in container
- [ ] #2 Remote working directory is Project.path; stdout/stderr are captured for logging
- [ ] #3 Non-zero remote exit and SSH connect/auth failures return error results usable by daemon
- [ ] #4 Unit tests use a fake Runner; no Cursor/AI dependency
<!-- AC:END -->
