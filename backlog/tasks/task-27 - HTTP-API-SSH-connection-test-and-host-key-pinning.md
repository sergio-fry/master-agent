---
id: TASK-27
title: 'HTTP API: SSH connection test and host key pinning'
status: Done
assignee: []
created_date: '2026-09-01 15:01'
updated_date: '2026-09-01 16:08'
labels: []
milestone: m-1
dependencies:
  - TASK-26
references:
  - specs/03-agent-invocation.md
  - specs/07-web-ui.md
priority: high
type: feature
ordinal: 1330
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
When creating or updating a project SSH target, operators need to verify credentials and pin the worker host key so scheduled runs do not fail with StrictHostKeyChecking errors (e.g. "No ED25519 host key is known").

Add an API action (e.g. POST /api/v1/projects/{id}/ssh/test and/or a pre-save test with draft project fields) that attempts SSH to the project host/user/port/path using the provided or stored private key, captures the server host key on first connect, stores it with the project in SQLite, and returns success/failure plus host key fingerprint. Re-test on key or host/port changes. Document how pinned keys are used at run time (per-project known host entry vs global file). Host key rotation after initial pin is out of scope except returning a clear error when the key changes.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Test endpoint connects with project SSH fields and inline private key from SQLite
- [x] #2 On successful first connect, host key (type + public key material or fingerprint) is persisted per project in SQLite
- [x] #3 SSH runner uses the pinned host key for that project; no manual known_hosts edit required for initial setup
- [x] #4 Failed auth, unreachable host, or host key mismatch returns structured JSON error without leaking private key
- [x] #5 Unit tests with fake SSH or fixture worker; acceptance covers happy path
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Schema v4: ssh_host_key on projects; update model/CRUD/migration
2. runner: known_hosts temp file, host key scan/fingerprint, TestSSHConnection
3. Update BuildSSHArgs and SSHRunner to use pinned host key
4. API: POST /projects/{id}/ssh/test and POST /ssh/test (draft)
5. Unit tests (stub ssh/ssh-keyscan); acceptance happy path
6. Update specs/01, specs/03, specs/07
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented POST /projects/{id}/ssh/test and POST /ssh/test, schema v4 ssh_host_key, runner host key pinning via temp known_hosts, NormalizeSSHPrivateKey fix for OpenSSH libcrypto. Verified: go test ./..., go test -tags=acceptance TestSSHTesterLiveWorker, TestScenarioAPISSHConnectionTest, TestScenarioAPIRunsAndLogs.

Implemented POST /projects/{id}/ssh/test and POST /ssh/test, schema v4 ssh_host_key, runner host key pinning via temp known_hosts, NormalizeSSHPrivateKey fix for OpenSSH libcrypto. Verified: go test ./..., go test -tags=acceptance TestSSHTesterLiveWorker, TestScenarioAPISSHConnectionTest, TestScenarioAPIRunsAndLogs.
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added SSH connection test API with per-project host key pinning in SQLite; SSH runner uses pinned keys via temp known_hosts. Fixed private key newline normalization for OpenSSH. Verified with unit tests and acceptance (SSH test API + runs).
<!-- SECTION:FINAL_SUMMARY:END -->
