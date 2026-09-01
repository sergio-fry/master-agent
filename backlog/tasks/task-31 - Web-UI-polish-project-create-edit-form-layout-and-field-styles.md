---
id: TASK-31
title: 'Web UI: polish project create/edit form layout and field styles'
status: Done
assignee: []
created_date: '2026-09-01 15:03'
updated_date: '2026-09-01 16:47'
labels: []
milestone: m-1
dependencies:
  - TASK-19
references:
  - internal/webui/static/index.html
  - internal/webui/static/style.css
  - specs/07-web-ui.md
priority: low
type: enhancement
ordinal: 5330
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The project dialog (index.html / style.css) has inconsistent field appearance: Name, Remote path, SSH host/user look smaller and not full-width while Port (type=number) matches the intended input styling. There is no CSS framework — only shared custom styles in style.css — and many inputs omit type="text", so they miss the width/padding/border rules that apply to input[type="text"], input[type="number"], and textarea.

Audit and normalize the project form (and shared form patterns used by task form where applicable): consistent full-width inputs, label spacing, row layout for host+port, textarea and checkbox alignment, dialog width on small screens. Follow existing design tokens (--border, --primary, etc.) rather than adding a heavy UI framework unless justified. Fix root cause (missing input types and/or broaden form input selectors) and verify create + edit dialogs visually.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 All project form text fields (name, path, ssh host/user, port, key textarea) share consistent height, padding, border, and full-width layout within the dialog
- [x] #2 Host + port row aligns cleanly; port field no longer visually dominates other fields
- [x] #3 Labels, hints, checkbox, and dialog actions have uniform vertical rhythm
- [x] #4 Fix applies via CSS/HTML only or minimal markup tweaks; no new frontend build pipeline required
- [x] #5 Task form reuses the same input styles where it shares #task-form patterns (no new regressions)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Normalized form input CSS selectors and type=text on fields; path row layout; dialog width.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Project and task form inputs share full-width styling via broadened CSS selectors and explicit input types; host+port row and dialog rhythm improved.
<!-- SECTION:FINAL_SUMMARY:END -->
