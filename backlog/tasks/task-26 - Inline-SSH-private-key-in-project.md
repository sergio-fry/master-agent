---
id: TASK-26
title: 'Inline SSH private key on project (replace key path + secrets volume)'
status: Done
assignee: []
created_date: '2026-08-31 21:02'
updated_date: '2026-09-01 09:55'
labels: []
milestone: m-1
dependencies:
  - TASK-6
  - TASK-17
  - TASK-19
  - TASK-20
references:
  - specs/01-data-model.md
  - specs/03-agent-invocation.md
  - specs/07-web-ui.md
  - specs/05-tech-stack.md
priority: high
type: feature
ordinal: 330
modified_files:
  - internal/store/schema.sql
  - internal/store/migrate_v2.go
  - internal/store/models.go
  - internal/store/crud.go
  - internal/store/sshkey.go
  - internal/api/projects.go
  - internal/runner/ssh.go
  - internal/runner/sshkey.go
  - internal/cli/project.go
  - internal/webui/static/index.html
  - internal/webui/static/app.js
  - docker-compose.yml
  - deploy/docker-compose.prod.yml
  - README.md
  - specs/01-data-model.md
  - specs/05-tech-stack.md
  - specs/07-web-ui.md
  - test/acceptance/api_test.go
  - test/acceptance/scenarios_test.go
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Store each project's SSH private key **inline** (PEM/OpenSSH text body) instead of a filesystem path under `/secrets`. The project create/edit form and API accept a textarea / JSON field with the key material; the runner materializes it at SSH time (temp file with mode 0600 or equivalent). Remove the requirement to configure `ssh_key_path` and the separate key-upload-to-secrets flow.

Update data model, migrations, API, Web UI, SSH runner, compose/deploy docs, and tests. API must never return private key bytes after save (only `key_configured: true/false`). Consider encryption at rest in SQLite if straightforward.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Project schema stores inline private key (or encrypted blob); `ssh_key_path` removed or deprecated with migration
- [x] #2 Create/update project accepts private key text; validation rejects empty/invalid key when SSH is required
- [x] #3 GET project and list responses never include key material; status endpoint indicates key present/absent only
- [x] #4 SSH runner uses stored key for connections; acceptance E2E still passes with fixture keys migrated to inline storage
- [x] #5 Web UI: textarea for pasting key on project form; file-upload dialog removed or replaced; no path field
- [x] #6 `/secrets` volume no longer required for keys in docker-compose and deploy templates (document any remaining uses)
- [x] #7 specs/01-data-model.md and related docs updated
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Schema migration: add key column, migrate existing file-based keys if present, drop `ssh_key_path`.
2. Update store CRUD and API project DTOs; remove POST/GET `/projects/{id}/key` file upload or repurpose for inline update only.
3. Runner: write key to temp file per run (or use ssh agent API) with secure permissions and cleanup.
4. Web UI: replace path input and upload dialog with textarea + configured/not-configured indicator.
5. Adjust compose/deploy (drop secrets mount where unused); update CLI `project add` if it still takes `--ssh-key` path.
6. Tests: unit + acceptance; verify responses never leak key bytes.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Schema v2: `ssh_private_key` column, migration reads legacy file paths when present. API returns `key_configured` only. Runner writes temp key file per SSH run. Removed `/secrets` volume from compose/deploy.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Inline SSH private keys in SQLite with schema migration v2, API/UI/CLI updates, temp-file materialization in runner, and removal of secrets volume + key upload endpoints. Verified with go test ./...
<!-- SECTION:FINAL_SUMMARY:END -->
