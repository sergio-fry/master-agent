---
id: TASK-4
title: CLI to list runs
status: To Do
assignee: []
created_date: '2026-08-31 15:55'
labels: []
milestone: m-0
dependencies:
  - TASK-2
references:
  - specs/01-data-model.md
  - specs/04-acceptance-criteria.md
priority: medium
type: feature
ordinal: 40
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Operators need to inspect run history (status, exit code, errors) without opening SQLite by hand.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 run list shows runs filtered by project (and optionally task)
- [ ] #2 Output includes status, exit_code, started/finished times, and error_message when present
- [ ] #3 Tests cover listing after sample runs exist in DB
<!-- AC:END -->
