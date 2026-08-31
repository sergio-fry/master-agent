---
id: TASK-3
title: CLI for project and task management
status: To Do
assignee: []
created_date: '2026-08-31 15:55'
labels: []
milestone: m-0
dependencies:
  - TASK-2
references:
  - specs/01-data-model.md
  - specs/04-acceptance-criteria.md
priority: high
type: feature
ordinal: 30
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Expose CLI to manage Projects and Tasks (add/list/enable-disable) matching the examples in specs/01-data-model.md. SSH fields live only on Project; Task has schedule + command + prompt.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 project add accepts name, path, ssh-host, ssh-user, ssh-key (and port with default 22) and persists to SQLite
- [ ] #2 task add requires project, name, interval, command, prompt and does not accept SSH fields
- [ ] #3 Projects and tasks can be listed and disabled so they stop participating in scheduling
- [ ] #4 Multiple tasks can exist on one project with different intervals and prompts
- [ ] #5 CLI behavior covered by unit or integration tests against temp DB
<!-- AC:END -->
