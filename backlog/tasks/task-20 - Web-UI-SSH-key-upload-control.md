---
id: TASK-20
title: 'Web UI: SSH key upload control'
status: Done
assignee: []
created_date: '2026-08-31 17:27'
updated_date: '2026-08-31 20:03'
labels: []
milestone: m-1
dependencies:
  - TASK-17
  - TASK-19
references:
  - specs/07-web-ui.md
modified_files:
  - internal/webui/static/index.html
  - internal/webui/static/app.js
  - internal/webui/static/style.css
  - internal/webui/webui_test.go
priority: high
type: feature
ordinal: 270
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
On the project screen, upload/replace private key; show whether a key is configured without displaying secret material.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 User can upload a key file for a project and see present=true afterward
- [x] #2 UI never renders private key contents
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add key dialog to project screen (status + file upload)\n2. Wire POST/GET /projects/{id}/key in app.js\n3. Add styles and webui tests\n4. Run go test and verify build
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added SSH key column, Key dialog with multipart upload via POST /projects/{id}/key, status from GET /projects/{id}/key. UI shows path and present flag only.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Web UI project screen: SSH key column, Key dialog with file upload and configured/not-configured status; never displays key material. Verified: go test ./... and go build ./cmd/master-agent/.
<!-- SECTION:FINAL_SUMMARY:END -->
