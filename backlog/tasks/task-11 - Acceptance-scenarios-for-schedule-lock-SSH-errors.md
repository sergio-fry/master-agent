---
id: TASK-11
title: 'Acceptance scenarios for schedule, lock, SSH, errors'
status: Done
assignee:
  - root
created_date: '2026-08-31 15:56'
updated_date: '2026-08-31 17:50'
labels: []
milestone: m-0
dependencies:
  - TASK-10
references:
  - specs/04-acceptance-criteria.md
  - specs/06-testing.md
priority: high
type: feature
ordinal: 110
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement acceptance tests for the Gherkin scenarios in specs/04: idle, due run over SSH, skip when locked, agent/SSH failure, per-project SSH identity. Use stub commands and assert DB state and/or remote side effects (flag files).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Idle: no SSH when nothing is due
- [x] #2 Due task: SSH to project host/key, cwd=path, lock during run, run status and next_run_at updated
- [x] #3 Locked project: second due task on same project does not start
- [x] #4 Remote exit 1 or SSH failure: run error, lock released, next_run_at scheduled, daemon continues
- [x] #5 Two projects with different keys/hosts use the correct identity each
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Read specs/04 and specs/06 and existing acceptance harness from TASK-10
2. Map Gherkin scenarios to concrete test cases with stub SSH commands and DB/flag assertions
3. Implement idle, due-run, locked-project, failure, and multi-identity scenarios
4. Run unit + acceptance tests and fix failures
5. Finalize AC checks and commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
In progress: researching harness, CLI, scheduler; planning scenario tests with stub SSH + DB assertions. Will extend compose for second worker/key for multi-identity.

Harness extended: worker-b + key B, bind-mounted testdata DB, scenario helpers. Implementing Gherkin scenarios in scenarios_test.go; next: go build/test + acceptance.

Switched DB asserts to in-container sqlite3 (named volume); fixed bind-mount SQLITE_BUSY/IOERR. Re-running acceptance.

Validation: go build ./...; go test ./...; go test -tags=acceptance -count=1 -v ./test/acceptance/... (all PASS, ~393s).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added acceptance scenarios for idle/due/lock/error/SSH-failure/per-project identity (worker-b + key B). SQLite WAL+busy_timeout for concurrent daemon/CLI reads. Verified with unit + full acceptance suite.
<!-- SECTION:FINAL_SUMMARY:END -->
