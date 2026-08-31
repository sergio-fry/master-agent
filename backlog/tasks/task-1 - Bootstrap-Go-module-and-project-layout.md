---
id: TASK-1
title: Bootstrap Go module and project layout
status: Done
assignee:
  - '@Sergei'
created_date: '2026-08-31 15:55'
updated_date: '2026-08-31 16:05'
labels: []
milestone: m-0
dependencies: []
references:
  - specs/05-tech-stack.md
  - specs/06-testing.md
  - AGENTS.md
modified_files:
  - go.mod
  - go.sum
  - cmd/master-agent/main.go
  - internal/cli/cli.go
  - internal/store/store.go
  - internal/scheduler/scheduler.go
  - internal/runner/runner.go
  - internal/version/version.go
  - internal/version/version_test.go
  - README.md
  - .gitignore
priority: high
type: chore
ordinal: 10
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Initialize the Go module, cmd/master-agent entrypoint skeleton, internal package layout, and testify so later work has a place to land. Align with specs/05-tech-stack.md (Go, SQLite without CGO preferred, cobra or urfave/cli).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 go.mod and main package build with go build ./...
- [x] #2 testify is available for tests; a smoke unit test passes via go test ./...
- [x] #3 README or AGENTS mentions how to build/test locally
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Init Go module master-agent with go mod init.
2. Create cmd/master-agent main skeleton and internal package stubs (cli, store, scheduler, runner).
3. Add testify dependency and a smoke unit test.
4. Document local build/test in README.
5. Verify go build ./... and go test ./...
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Created module layout: cmd/master-agent, internal/{cli,store,scheduler,runner,version}. Adding testify and verifying build.

Verified: go build ./... and go test ./... pass. README has Build & test section.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Bootstrapped Go module master-agent with cmd/master-agent entrypoint and internal stubs (cli, store, scheduler, runner, version). Added testify and a smoke unit test. Documented local build/test in README. Verified with: go build ./... (ok) and go test ./... (version smoke test ok).
<!-- SECTION:FINAL_SUMMARY:END -->
