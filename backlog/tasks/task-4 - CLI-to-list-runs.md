---
id: TASK-4
title: CLI to list runs
status: Done
assignee:
  - '@root'
created_date: '2026-08-31 15:55'
updated_date: '2026-08-31 17:56'
labels: []
milestone: m-0
dependencies:
  - TASK-2
  - TASK-11
references:
  - specs/01-data-model.md
  - specs/04-acceptance-criteria.md
priority: high
type: feature
ordinal: 115
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Operators need to inspect run history (status, exit code, errors) without opening SQLite by hand.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 run list shows runs filtered by project (and optionally task)
- [x] #2 Output includes status, exit_code, started/finished times, and error_message when present
- [x] #3 Tests cover listing after sample runs exist in DB
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Review existing CLI (projects/tasks) and store APIs for runs
2. Add store method to list runs filtered by project (optional task)
3. Add `master-agent runs list` (or equivalent) CLI command with required output fields
4. Add unit tests with sample runs in DB
5. Verify `go build` / unit tests pass
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Ordered after TASK-11 (acceptance) so MVP E2E lands before run-list CLI; Web UI runs API depends on this.

Explored CLI/store patterns; implementing ListRuns + run list CLI per specs/01.

Implemented store.ListRuns + CLI run list (--project required, --task optional). Added store/CLI tests and README example.

Validation: go test ./internal/... ./cmd/... (PASS in golang:1.26-alpine); go build ./cmd/master-agent; docker compose build OK.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added store.ListRuns and CLI `run list --project` (optional `--task`) with status/exit/times/error columns. Covered by store + CLI tests with sample runs in temp SQLite. Verified: go test ./internal/... ./cmd/... and docker compose build.
<!-- SECTION:FINAL_SUMMARY:END -->
