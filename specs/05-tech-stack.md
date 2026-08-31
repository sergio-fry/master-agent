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
         ├─ /secrets/...       → per-project private keys (ro for daemon-only; rw when API uploads keys)
         └─ known_hosts        → host key verification
```

**Dockerfile:** multi-stage build → Alpine runtime + `openssh-client` (см. `Dockerfile` в корне).

**Compose / run:** `docker-compose.yml` — daemon как long-running service с опциональным `--http-addr`; volumes `/data`, `/secrets` (rw при HTTP upload ключей), `/etc/ssh/ssh_known_hosts` (ro). Порт `8080`, `MASTER_AGENT_TOKEN` при доступе к API извне. Ключи и DB снаружи образа.

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
| Web UI / API | `net/http` tests + acceptance HTTP client ([07-web-ui.md](./07-web-ui.md)) |

Подробности и правила: [06-testing.md](./06-testing.md), также `AGENTS.md`.

## Web UI (post-MVP milestone)

| Choice | Detail |
|--------|--------|
| API | Go `net/http` (or chi/echo — keep thin) JSON `/api/v1` |
| UI | Embedded static or server-rendered from the same binary |
| Process | `master-agent serve` and/or `daemon --http-addr` |
| Auth | Optional Bearer token (`MASTER_AGENT_TOKEN`) |

## Out of stack

- Postgres / Redis
- Kubernetes-specific controllers (plain Docker/Compose достаточно)
- Agent SDKs внутри оркестратора
- Gherkin/godog; full browser E2E suite (optional smoke only)
- OAuth / multi-tenant IAM
