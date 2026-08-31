---
id: TASK-22
title: 'Web UI: runs table and log viewer'
status: Done
assignee: []
created_date: '2026-08-31 17:27'
updated_date: '2026-08-31 20:08'
labels: []
milestone: m-1
dependencies:
  - TASK-16
  - TASK-19
references:
  - specs/07-web-ui.md
priority: high
type: feature
ordinal: 290
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
UI to browse runs with filters and open a run detail with log text (size-limited).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Runs table supports filter by project and status
- [x] #2 Run detail shows exit_code, timestamps, error_message, and log content when available
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add runs.html + runs.js with project/status filters and run detail dialog (metadata + log).
2. Update nav links on existing pages and add CSS for log viewer.
3. Extend webui tests; run go test and verify build.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented runs.html/runs.js: filterable runs table (project + status), detail dialog with metadata and log tail via API.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added runs.html/runs.js with project/status filters, run detail dialog (exit_code, timestamps, error_message, log via /api/v1/runs/{id}/log). Nav links on all pages. Verified: go test ./... && go build ./...
<!-- SECTION:FINAL_SUMMARY:END -->
