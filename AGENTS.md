# AGENTS.md — Master Agent

## Keep rules current

Project rules (`AGENTS.md`) and specs (`specs/`) must stay **accurate and short**.

- When behavior, stack, or process **changes** — update the matching rule/spec in the same change.
- When you find a **mismatch** between code and docs/rules — fix the docs/rules (or the code, if the docs are the source of truth for intent).
- Prefer **brief** bullets over essays; link to `specs/` for detail.
- Do not leave stale guidance; delete or rewrite obsolete rules instead of stacking exceptions.

## Spec-Driven Development

- Source of truth for product behavior: `specs/`.
- Implement against specs; if implementation needs a different approach, update the spec first (or in the same PR).
- Index: [specs/00-overview.md](specs/00-overview.md).

## Product shape (summary)

- Go daemon in Docker; SQLite; execution **only via SSH** to a worker.
- **Project** = remote path + SSH host/user/key.
- **Task** = schedule + command + prompt (no SSH fields).
- Orchestrator does not close backlog/Jira tasks; the remote CLI agent does, per prompt.
- Details: `specs/01`–`05`.

## Definition of Done / commits

- Before committing: relevant **unit tests pass**; for orchestration/SSH changes, **acceptance tests pass** too (or are run in CI for that change).
- Do not commit knowingly broken tests or untested behavior that the specs require.

## Testing (required)

Write automated tests for new and changed behavior:

| Level | What | How |
|-------|------|-----|
| **Unit** | schedule, locks, placeholders, status mapping | `go test`, std `testing` + testify; fake `Runner` |
| **Acceptance / E2E** | due → SSH → lock → unlock → next_run / errors | Docker Compose (master-agent + sshd worker); `go test` with acceptance tag |

Rules:

- Remote commands in tests are **stubs** (`touch`, `echo`, `exit 1`, `sleep`) — never depend on Cursor/AI agents.
- Acceptance must exercise **real SSH**, same path as production (no “local spawn only in tests”).
- Cover externally visible behavior from [specs/04-acceptance-criteria.md](specs/04-acceptance-criteria.md) and [specs/06-testing.md](specs/06-testing.md).
- Unit tests run always in CI; acceptance may be separate job/tag but must exist for orchestration features.

## Stack pointers

- Go, SQLite, OpenSSH client, Docker — [specs/05-tech-stack.md](specs/05-tech-stack.md).
- Testing detail — [specs/06-testing.md](specs/06-testing.md).

<!-- BACKLOG.MD GUIDELINES START -->
<!-- backlog.md-instructions-version: 1.50.1 -->
<CRITICAL_INSTRUCTION>

## Backlog.md Workflow

This project uses Backlog.md for task and project management.

**For every user request in this project, run `backlog instructions overview` before answering or taking action.**

Use the overview to decide whether to search, read, create, or update Backlog tasks.

Before task lifecycle actions, read the matching detailed guide:
- `backlog instructions task-creation` before creating or splitting tasks
- `backlog instructions task-execution` before planning, changing status or assignee, adding a plan or implementation notes, or implementing task work
- `backlog instructions task-finalization` before checking acceptance criteria, writing final summaries, or moving tasks to terminal statuses

Use `backlog <command> --help` before running unfamiliar commands. Help shows options, fields, and examples.

Do not edit Backlog task, draft, document, decision, or milestone markdown files directly. Use the `backlog` CLI so metadata, relationships, and history stay consistent.

</CRITICAL_INSTRUCTION>
<!-- BACKLOG.MD GUIDELINES END -->
