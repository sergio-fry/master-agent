---
id: TASK-17
title: 'HTTP API: SSH key upload for a project'
status: To Do
assignee: []
created_date: '2026-08-31 17:26'
labels: []
milestone: m-1
dependencies:
  - TASK-14
references:
  - specs/07-web-ui.md
  - specs/03-agent-invocation.md
priority: high
type: feature
ordinal: 240
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Allow uploading/replacing a project private key into the secrets volume and updating ssh_key_path. Responses only indicate whether a key is present—never return key material.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 POST multipart upload writes key under configured secrets dir with restrictive permissions
- [ ] #2 Project ssh_key_path updated to in-container path; GET key status returns present=true/false only
- [ ] #3 Tests use temp secrets dir; uploaded file mode is verified; response bodies never include key bytes
<!-- AC:END -->
