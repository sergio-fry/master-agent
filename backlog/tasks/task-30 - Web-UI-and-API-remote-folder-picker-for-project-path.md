---
id: TASK-30
title: 'Web UI and API: remote folder picker for project path'
status: Done
assignee: []
created_date: '2026-09-01 15:02'
updated_date: '2026-09-01 16:47'
labels: []
milestone: m-1
dependencies:
  - TASK-26
  - TASK-27
references:
  - specs/07-web-ui.md
  - specs/03-agent-invocation.md
priority: medium
type: feature
ordinal: 4330
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Project path is currently a free-text field for the remote workspace directory. Operators should pick the folder by browsing the real directory tree on the worker over SSH instead of typing the path manually.

Add an API that lists directories on the remote host using the project SSH settings (inline private key from SQLite, pinned host key when available). Support navigation from a sensible root (e.g. home directory or /) with parent/child listing and only directories selectable. In the Web UI project create/edit form, add Browse/open control next to Path that opens a modal or panel with an expandable folder tree fed by the API; selecting a folder fills the path field. Handle auth errors, timeouts, and permission denied with clear messages. Read-only listing only — no remote file upload or mkdir in this task.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 API lists remote directories for a project (or draft SSH fields) over SSH and returns entries safe for UI (name, path, has_children or type=directory)
- [x] #2 Listing rejects or skips non-directory paths; does not expose file contents or private keys
- [x] #3 Project create/edit form offers Browse control that loads and navigates a remote folder tree
- [x] #4 Selecting a folder in the tree sets the Path field to the absolute remote path
- [x] #5 Errors (SSH failure, auth, host key, timeout) surface in the picker without breaking the form
- [x] #6 Unit/integration tests with fixture worker or fake SSH; optional UI smoke via API acceptance
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
Done in implementation
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
POST list-dirs API, SSHBrowser, Browse UI in project form. Verified go test ./internal/runner/... ./internal/api/... ./internal/webui/...
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Remote directory listing API and Browse picker in project form; errors surface in picker. Verified with unit tests.
<!-- SECTION:FINAL_SUMMARY:END -->
