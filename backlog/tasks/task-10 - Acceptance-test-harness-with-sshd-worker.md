---
id: TASK-10
title: Acceptance test harness with sshd worker
status: Done
assignee:
  - '@composer'
created_date: '2026-08-31 15:56'
updated_date: '2026-08-31 17:09'
labels: []
milestone: m-0
dependencies:
  - TASK-9
references:
  - specs/06-testing.md
  - specs/04-acceptance-criteria.md
priority: high
type: feature
ordinal: 100
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add docker-compose.test.yml (or equivalent) with master-agent + sshd worker, test SSH keys, and a Go acceptance package (build tag acceptance) that drives E2E without Cursor/AI. Stub remote commands only (touch, echo, exit 1, sleep).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Compose brings up sshd worker and master-agent (or test runner) with working key-based SSH
- [x] #2 go test -tags=acceptance ./test/acceptance/... can run against the harness
- [x] #3 Harness does not depend on Cursor, backlog CLI, or MCP
- [x] #4 Document how to run acceptance tests in README or AGENTS
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Review Dockerfile, compose, SSH runner, and CLI for test wiring.
2. Add test SSH keypair + worker sshd image/config under test/.
3. Add docker-compose.test.yml (master-agent + sshd worker + volumes/keys).
4. Add test/acceptance Go package with //go:build acceptance: bring-up helpers, SSH smoke, stub remote cmd.
5. Document how to run acceptance tests in README/AGENTS.
6. Verify: compose up, go test -tags=acceptance, unit build still green.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Started: reviewing Docker/SSH layout; next: keys + compose.test + acceptance package.

Added worker Dockerfile, fixtures, docker-compose.test.yml, acceptance package; docs in README/AGENTS. Running compose build + acceptance tests.

Validation: go build ./...; go test ./...; go test -tags=acceptance -count=1 -v ./test/acceptance/... (all PASS, ~92s including compose build/up/down).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added docker-compose.test.yml (master-agent + sshd worker), test SSH fixtures, and go test -tags=acceptance harness with stub remote commands. Documented in README/AGENTS. Verified with unit build/tests and full acceptance run.
<!-- SECTION:FINAL_SUMMARY:END -->
