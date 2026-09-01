---
id: TASK-29
title: 'Web UI and API: show configured SSH private key on project edit'
status: Done
assignee: []
created_date: '2026-09-01 15:01'
updated_date: '2026-09-01 16:31'
labels: []
milestone: m-1
dependencies:
  - TASK-26
references:
  - specs/07-web-ui.md
  - specs/01-data-model.md
priority: medium
type: enhancement
ordinal: 3330
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Editing a project currently shows an empty SSH private key textarea with a hint to leave blank to keep the existing key. Operators cannot tell which key is configured.

Extend GET project (edit flow) and the Web UI so edit mode displays the stored private key from SQLite (read-only or editable textarea pre-filled). Private keys remain stored inline in SQLite (TASK-26). Update specs/07 if the prior never-show-key-after-save rule is revised. CLI project show may mirror the same behavior optionally.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 GET /api/v1/projects/{id} returns ssh_private_key for authorized callers (or a dedicated edit payload) so the UI can pre-fill the field
- [x] #2 Project edit form shows the currently stored key, not an empty textarea
- [x] #3 Saving with unchanged key keeps existing material; saving with new textarea content replaces the key
- [x] #4 List endpoint and logs still never expose key material
- [x] #5 specs/07-web-ui.md updated to match new visibility rule on single-project GET/edit only
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. GET /projects/{id} includes ssh_private_key; list omits it
2. Pre-fill edit form; PATCH unchanged key keeps material
3. Update specs/07 and tests
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
GET /projects/{id} returns ssh_private_key; list/create/patch omit it. Edit form pre-fills key. Verified go test ./internal/api/... ./internal/webui/...
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Single-project GET returns ssh_private_key for edit pre-fill; list and mutating responses still hide key material. Web UI edit dialog shows stored key. specs/07 updated. Verified with API and webui tests.
<!-- SECTION:FINAL_SUMMARY:END -->
