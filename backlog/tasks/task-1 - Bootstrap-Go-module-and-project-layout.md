---
id: TASK-1
title: Bootstrap Go module and project layout
status: To Do
assignee: []
created_date: '2026-08-31 15:55'
labels: []
milestone: m-0
dependencies: []
references:
  - specs/05-tech-stack.md
  - specs/06-testing.md
  - AGENTS.md
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
- [ ] #1 go.mod and main package build with go build ./...
- [ ] #2 testify is available for tests; a smoke unit test passes via go test ./...
- [ ] #3 README or AGENTS mentions how to build/test locally
<!-- AC:END -->
