---
id: TASK-3
title: CLI for project and task management
status: Done
assignee:
  - '@Sergei'
created_date: '2026-08-31 15:55'
updated_date: '2026-08-31 16:24'
labels: []
milestone: m-0
dependencies:
  - TASK-2
references:
  - specs/01-data-model.md
  - specs/04-acceptance-criteria.md
modified_files:
  - go.mod
  - go.sum
  - internal/cli/cli.go
  - internal/cli/cli_test.go
  - internal/cli/project.go
  - internal/cli/task.go
  - internal/store/crud.go
  - internal/store/store_test.go
priority: high
type: feature
ordinal: 30
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Expose CLI to manage Projects and Tasks (add/list/enable-disable) matching the examples in specs/01-data-model.md. SSH fields live only on Project; Task has schedule + command + prompt.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 project add accepts name, path, ssh-host, ssh-user, ssh-key (and port with default 22) and persists to SQLite
- [x] #2 task add requires project, name, interval, command, prompt and does not accept SSH fields
- [x] #3 Projects and tasks can be listed and disabled so they stop participating in scheduling
- [x] #4 Multiple tasks can exist on one project with different intervals and prompts
- [x] #5 CLI behavior covered by unit or integration tests against temp DB
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add store helpers: ListProjects, GetProjectByName, ListTasks (by project optional), resolve project by name/id.
2. Implement cobra CLI: global --db; project add/list/disable; task add/list/disable matching specs/01 examples.
3. Wire cmd/master-agent to the new CLI; default DB path /data/master-agent.db.
4. Add CLI integration tests against temp SQLite covering AC #1–#5.
5. Verify go build ./... and go test ./...
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Started: marking In Progress and drafting plan.

Verified: go build ./... and go test ./... (cli + store). AC covered by cli_test.go against temp SQLite.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
CLI (cobra) for project/task add|list|enable|disable with global --db; store list/lookup helpers. SSH flags only on project. Verified with go build ./... and go test ./... (temp DB integration tests for all ACs).
<!-- SECTION:FINAL_SUMMARY:END -->
