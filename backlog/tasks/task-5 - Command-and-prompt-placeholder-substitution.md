---
id: TASK-5
title: Command and prompt placeholder substitution
status: Done
assignee:
  - '@Sergei'
created_date: '2026-08-31 15:55'
updated_date: '2026-08-31 16:29'
labels: []
milestone: m-0
dependencies:
  - TASK-1
references:
  - specs/01-data-model.md
  - specs/03-agent-invocation.md
  - specs/06-testing.md
modified_files:
  - internal/placeholder/placeholder.go
  - internal/placeholder/placeholder_test.go
  - specs/01-data-model.md
  - specs/03-agent-invocation.md
  - README.md
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
- [x] #1 All documented placeholders are replaced from Project/Task fields
- [x] #2 Missing or unknown placeholders have defined, tested behavior
- [x] #3 Table-driven unit tests cover success and edge cases without SSH
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Add internal/placeholder package: Expand(command, project, task) replacing documented placeholders.
2. Shell-escape substituted values (POSIX single-quote) so remote shell/SSH stays safe.
3. Define behavior: known placeholders always replaced; unknown {{…}} return error; empty values become quoted empty.
4. Table-driven unit tests for success and edge cases (no SSH).
5. Align spec examples if needed so placeholders are used without redundant outer quotes.
6. go test ./... and verify build.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Implemented internal/placeholder.Expand with shell-quote and JSON argv modes; unknown placeholders error; specs/README examples aligned.

Validation: go test ./... and go build ./cmd/master-agent passed (PATH=/usr/local/go/bin).
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added internal/placeholder.Expand: substitutes documented placeholders with POSIX shell quoting (shell commands) or literal values (JSON argv); unknown placeholders error; empty fields substitute empty. Specs/README examples updated. Verified with table-driven unit tests via go test ./... and go build.
<!-- SECTION:FINAL_SUMMARY:END -->
