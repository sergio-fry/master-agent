---
id: TASK-28
title: 'Web UI: Test SSH connection on project form'
status: Done
assignee: []
created_date: '2026-09-01 15:01'
updated_date: '2026-09-01 16:17'
labels: []
milestone: m-1
dependencies:
  - TASK-27
references:
  - specs/07-web-ui.md
priority: high
type: feature
ordinal: 2330
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Project create/edit form should let operators verify SSH settings before saving. Add a "Test connection" control that calls the SSH test API with current form values (host, user, port, path, private key textarea). Show success with host key fingerprint when pinned, or an actionable error (auth failed, host unreachable, host key changed). Encourage or require test on first setup and when SSH fields or key change.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Create and edit project dialogs include Test connection button
- [x] #2 Test uses unsaved form values so operators can validate before Save
- [x] #3 UI shows pass/fail message and host key fingerprint on success
- [x] #4 No private key material shown in success/error messages beyond what user typed
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add Test connection button and result area in project dialog
2. app.js: draft test for create, project test for edit with form overrides
3. Show fingerprint on success; actionable errors from API
4. webui_test.go assertions
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added Test connection button in project dialog; draft /ssh/test for create and /projects/{id}/ssh/test for edit with unsaved overrides. Verified go test ./internal/webui/...
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Project create/edit dialog includes Test connection calling SSH test API with current form values; shows fingerprint on success or API error message. Verified with webui tests.
<!-- SECTION:FINAL_SUMMARY:END -->
