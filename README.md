# Master Agent

Фоновый оркестратор на **Go** в **Docker**: по расписанию заходит по **SSH** на worker-машины и запускает CLI-агентов. Предметная логика — в промпте каждой задачи.

## Документация

Спецификации (Spec-Driven Development):

| Spec | Description |
|------|-------------|
| [specs/00-overview.md](specs/00-overview.md) | Vision, goals, glossary |
| [specs/01-data-model.md](specs/01-data-model.md) | Project (path+SSH), Task (schedule+prompt), Lock, Run |
| [specs/02-scheduler.md](specs/02-scheduler.md) | Daemon loop, locks, errors |
| [specs/03-agent-invocation.md](specs/03-agent-invocation.md) | SSH runner, границы ответственности |
| [specs/04-acceptance-criteria.md](specs/04-acceptance-criteria.md) | MVP checklist |
| [specs/05-tech-stack.md](specs/05-tech-stack.md) | Go, Docker, SQLite, SSH |
| [specs/06-testing.md](specs/06-testing.md) | Unit + acceptance E2E |
| [AGENTS.md](AGENTS.md) | Правила для агентов (актуальность, тесты, backlog) |

## Кратко

- **Project** — remote path + SSH host/user/key.
- **Task** — interval + command + prompt (без своих SSH-полей).
- **Lock** — в SQLite; MVP: один run глобально.
- Агент и MCP живут на worker; контейнер только оркестрирует.
- Тесты: unit + acceptance через SSH stubs в Compose; правила держать краткими и актуальными.

## Build & test (local)

Requires Go (see [specs/05-tech-stack.md](specs/05-tech-stack.md)).

```bash
go build ./...
go test ./...
```

Binary entrypoint: `cmd/master-agent`. Package layout under `internal/` (`cli`, `store`, `placeholder`, `scheduler`, `runner`, …).

### Acceptance (E2E)

Docker Compose harness (`docker-compose.test.yml`): `master-agent` + `sshd` worker, test keys under `test/fixtures/ssh/`. Remote commands are stubs only (`touch` / `echo` / `exit 1` / `sleep`) — no Cursor, backlog CLI, or MCP.

```bash
# TestMain brings compose up/down automatically:
go test -tags=acceptance ./test/acceptance/...

# Or manage compose yourself, then skip lifecycle in tests:
docker compose -f docker-compose.test.yml up -d --build
ACCEPTANCE_SKIP_COMPOSE=1 go test -tags=acceptance ./test/acceptance/...
docker compose -f docker-compose.test.yml down -v
```

See [specs/06-testing.md](specs/06-testing.md).

## Docker

Multi-stage image: Go binary + `openssh-client` only. No Cursor, MCP, or backlog in the image. SSH private keys and `known_hosts` are **runtime mounts**, never `COPY`'d.

```bash
docker compose build
docker compose up -d
```

### Volumes (`docker-compose.yml`)

| Host path | Container | Purpose |
|-----------|-----------|---------|
| `./data` | `/data` | SQLite (`--db` default `/data/master-agent.db`) and run logs |
| `./secrets` | `/secrets` (ro) | Per-project private keys |
| `./ssh/known_hosts` | `/etc/ssh/ssh_known_hosts` (ro) | Host keys for `StrictHostKeyChecking=yes` |

Prepare SSH before enabling runs:

```bash
mkdir -p secrets/projects/my-app
# install private key → secrets/projects/my-app/id_ed25519 (mode 600)
cp ssh/known_hosts.example ssh/known_hosts
ssh-keyscan -H worker.example.com >> ssh/known_hosts
```

Register a project with the **in-container** key path, e.g. `/secrets/projects/my-app/id_ed25519`.

```bash
docker compose run --rm master-agent project add \
  --name my-app \
  --path /home/worker/projects/my-app \
  --ssh-host worker.example.com \
  --ssh-user worker \
  --ssh-key /secrets/projects/my-app/id_ed25519
```

Daemon is the default container command (`master-agent daemon`). Override `TICK_INTERVAL` in compose if needed.
