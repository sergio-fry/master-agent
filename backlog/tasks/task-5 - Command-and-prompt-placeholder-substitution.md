---
id: TASK-5
title: Command and prompt placeholder substitution
status: To Do
assignee: []
created_date: '2026-08-31 15:55'
labels: []
milestone: m-0
dependencies:
  - TASK-1
references:
  - specs/01-data-model.md
  - specs/03-agent-invocation.md
  - specs/06-testing.md
priority: high
type: feature
ordinal: 50
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Substitute {{prompt}}, {{project_path}}, {{project_name}}, {{task_name}}, {{task_id}} into Task.command before remote execution. Keep escaping safe for shell/SSH.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 All documented placeholders are replaced from Project/Task fields
- [ ] #2 Missing or unknown placeholders have defined, tested behavior
- [ ] #3 Table-driven unit tests cover success and edge cases without SSH
<!-- AC:END -->
