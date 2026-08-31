---
id: TASK-20
title: 'Web UI: SSH key upload control'
status: To Do
assignee: []
created_date: '2026-08-31 17:27'
labels: []
milestone: m-1
dependencies:
  - TASK-17
  - TASK-19
references:
  - specs/07-web-ui.md
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
- [ ] #1 User can upload a key file for a project and see present=true afterward
- [ ] #2 UI never renders private key contents
<!-- AC:END -->
