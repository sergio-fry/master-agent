---
id: TASK-17
title: 'HTTP API: SSH key upload for a project'
status: Done
assignee: []
created_date: '2026-08-31 17:26'
updated_date: '2026-08-31 19:51'
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
- [x] #1 POST multipart upload writes key under configured secrets dir with restrictive permissions
- [x] #2 Project ssh_key_path updated to in-container path; GET key status returns present=true/false only
- [x] #3 Tests use temp secrets dir; uploaded file mode is verified; response bodies never include key bytes
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add SecretsDir to api.Config and key upload/status handlers\n2. POST multipart writes key with 0600, updates project ssh_key_path\n3. GET returns {present: bool} only\n4. Unit tests with temp secrets dir; verify permissions and no key bytes in responses
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Added GET/POST /api/v1/projects/{id}/key with SecretsDir config. Multipart field 'key' writes id_ed25519 at mode 0600 under {SecretsDir}/projects/{id}/. GET returns {present: bool} only.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
HTTP API SSH key upload: POST multipart writes private key with 0600 permissions and updates project ssh_key_path; GET returns present status only (no key material). Verified with go test ./internal/api/... and go build ./cmd/master-agent.
<!-- SECTION:FINAL_SUMMARY:END -->
