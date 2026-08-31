---
id: TASK-2
title: 'SQLite schema, migrations, and store'
status: To Do
assignee: []
created_date: '2026-08-31 15:55'
labels: []
milestone: m-0
dependencies:
  - TASK-1
references:
  - specs/01-data-model.md
  - specs/04-acceptance-criteria.md
  - specs/06-testing.md
priority: high
type: feature
ordinal: 20
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement SQLite persistence for projects, tasks, locks, and runs per specs/01-data-model.md. Auto-create DB file and migrate schema on startup. Provide a store API used by CLI and daemon (no HTTP).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 On first start, DB file is created and tables projects, tasks, locks, runs exist
- [ ] #2 Store can create/read/update Project with path, ssh_host, ssh_user, ssh_port, ssh_key_path, enabled
- [ ] #3 Store can create/read/update Task bound to project with prompt, command, interval_seconds, enabled, last_run_at, next_run_at
- [ ] #4 Store supports lock insert/delete and run insert/update for lifecycle fields
- [ ] #5 Unit/integration tests cover schema init and basic CRUD against a temp SQLite file
<!-- AC:END -->
