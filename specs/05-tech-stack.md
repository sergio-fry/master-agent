# Tech Stack

Зафиксированный стек MVP.

## Language & Runtime

| Choice | Detail |
|--------|--------|
| Language | **Go** (один статический бинарь) |
| Module | `master-agent` (имя модуля — при инициализации репо) |
| CLI | `cobra` или `urfave/cli` |
| DB driver | SQLite без CGO предпочтительно (`modernc.org/sqlite`) или с CGO (`mattn/go-sqlite3`) |
| Migrations | embed SQL / lightweight migrator |

## Storage

- **SQLite** — единственное хранилище состояния (projects, tasks, locks, runs).
- Файл на volume, например `/data/master-agent.db`.

## Execution

- **OpenSSH client** в образе (`ssh`).
- Исполнение агентов **только** по SSH на worker из полей Project.
- Локальный spawn агента в контейнере — **не поддерживается** в MVP.

## Containerization

```
┌─────────────────────────────────────┐
│ Docker image: master-agent          │
│  - Go binary                        │
│  - openssh-client                   │
│  - no Cursor / no MCP / no backlog  │
└─────────────────────────────────────┘
         │ volumes
         ├─ /data              → SQLite, run logs
         ├─ /secrets/...       → per-project private keys (ro)
         └─ known_hosts        → host key verification
```

**Dockerfile (концепт):** multi-stage build → runtime на minimal base (distroless/alpine) + `openssh-client`.

**Compose / run:** daemon как long-running service; ключи и DB снаружи образа.

## What stays on the Worker

- Cursor CLI / другие агенты
- `backlog` (backlog.md)
- MCP servers, skills, git credentials
- сами репозитории проектов (`Project.path`)

## Testing

| Layer | Tools |
|-------|--------|
| Unit | `testing` + testify; fake Runner |
| Acceptance | Docker Compose (app + sshd) + `go test` (acceptance tag) |

Подробности и правила: [06-testing.md](./06-testing.md), также `AGENTS.md`.

## Out of stack (MVP)

- Postgres / Redis
- HTTP API / Web UI
- Kubernetes-specific controllers (plain Docker/Compose достаточно)
- Agent SDKs внутри оркестратора
- Gherkin/godog, browser E2E
