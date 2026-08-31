---
id: TASK-6
title: SSH runner with Runner interface
status: Done
assignee:
  - '@Sergei'
created_date: '2026-08-31 15:55'
updated_date: '2026-08-31 16:34'
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
- [x] #1 Runner builds SSH invocation from Project fields and substituted command; no local agent spawn in container
- [x] #2 Remote working directory is Project.path; stdout/stderr are captured for logging
- [x] #3 Non-zero remote exit and SSH connect/auth failures return error results usable by daemon
- [x] #4 Unit tests use a fake Runner; no Cursor/AI dependency
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Define Runner interface + Result (exit code, stdout, stderr, usable error message).
2. Implement SSHRunner: build OpenSSH argv from Project (host/user/port/key) with BatchMode, IdentitiesOnly, StrictHostKeyChecking, ServerAlive*; remote cwd=Project.path; capture stdout/stderr.
3. Handle shell-string and JSON-argv substituted commands; never spawn local agent binary.
4. Map non-zero remote exit and SSH connect/auth failures into Result for the daemon.
5. Add FakeRunner for unit tests; table-driven tests for argv building and result mapping (no real SSH/AI).
6. Verify go build ./... and go test ./...
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
SSHRunner builds OpenSSH argv from Project (BatchMode/IdentitiesOnly/StrictHostKeyChecking/ServerAlive*); remote cwd via bash -lc + cd Project.path; shell and JSON argv supported. FakeRunner for unit/scheduler tests. Verified: go build ./... && go test ./... (runner stub covers stdout/stderr, exit 7, SSH 255).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added Runner interface, SSHRunner (OpenSSH-only), and FakeRunner. Unit tests cover argv construction, cwd, stdout/stderr capture, non-zero/SSH failure Result mapping, and FakeRunner — no real SSH/AI. Verified with go build ./... and go test ./...
<!-- SECTION:FINAL_SUMMARY:END -->
