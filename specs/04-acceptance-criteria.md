# Acceptance Criteria (MVP)

## Storage

- [ ] SQLite файл создаётся автоматически при первом запуске (volume `/data`).
- [ ] Миграции создают таблицы: `projects`, `tasks`, `locks`, `runs`.
- [ ] CRUD для Project и Task (CLI).

## Projects

- [ ] Project создаётся с: `name`, `path`, `ssh_host`, `ssh_user`, `ssh_port`, `ssh_key_path`.
- [ ] У разных проектов могут быть разные ключи и хосты.
- [ ] Project можно disable — его tasks не schedule.

## Tasks

- [ ] Task привязана к Project: только `prompt`, `command`, `interval_seconds` (+ name/enabled).
- [ ] Task **не** хранит SSH-параметры (берёт из Project).
- [ ] Несколько tasks на один Project с разными interval и prompt.
- [ ] Task можно disable.

## Scheduler

- [ ] Daemon работает в контейнере с configurable tick interval.
- [ ] Due task запускается когда `now >= next_run_at` и нет активного run (MVP: глобально).
- [ ] Пока SSH-run выполняется — новый run не стартует.
- [ ] Lock в SQLite на время run.
- [ ] После exit — lock снят, `last_run_at` и `next_run_at` обновлены.

## SSH Invocation

- [ ] Run идёт через SSH с ключом и хостом из Project.
- [ ] Remote cwd = `Project.path`.
- [ ] Command и prompt из Task; плейсхолдеры подставляются.
- [ ] Daemon блокирующе ждёт завершения SSH-сессии.
- [ ] stdout/stderr пишутся в log (volume и/или run record).
- [ ] Агент **не** запускается локально в контейнере.

## Docker / Tech

- [ ] Образ собирается Dockerfile’ом (Go binary + openssh-client).
- [ ] SQLite и логи — в volume; SSH keys — в read-only mounts, не в образе.
- [ ] known_hosts смонтирован или иначе подготовлен для StrictHostKeyChecking.

## Error Handling

- [ ] Non-zero remote exit → run `error`, log, lock released.
- [ ] SSH failure → run `error`, log, lock released.
- [ ] Нет immediate retry — следующая попытка только по `next_run_at`.
- [ ] Stale lock (мёртвый локальный ssh pid) снимается при recovery.

## Observability

- [ ] `runs` можно просмотреть (CLI `run list` или SQL).
- [ ] Лог содержит: task, project, ssh_host, pid, duration, exit code.

## Testing (process)

- [ ] Unit-тесты на schedule/lock/placeholders/status (см. [06-testing.md](./06-testing.md)).
- [ ] Acceptance E2E через Docker Compose + sshd со stub-командами (`touch` / `exit 1`).
- [ ] Acceptance не зависит от Cursor/AI.

## Web UI (milestone — see [07-web-ui.md](./07-web-ui.md))

- [ ] HTTP API for projects, tasks, runs, log tail, key upload (no key material in responses).
- [ ] `serve` (and/or daemon `--http-addr`) serves API + UI.
- [ ] UI: manage projects/tasks, upload keys, browse runs/logs.
- [ ] API/UI acceptance tests without Cursor/AI.

## Explicitly Out of Scope

- [ ] Очередь / priority queue в master-agent.
- [ ] Конфиг MCP/skills в репозитории оркестратора.
- [ ] Multi-user OAuth/RBAC; Cursor transcript UI.
- [ ] Multi-agent parallel runs (post-MVP).
- [ ] Локальный (non-SSH) runner.

## Test Scenarios (Gherkin-style)

**Scenario: idle when no due tasks**

```gherkin
Given daemon is running
And no task has next_run_at in the past
When tick occurs
Then no SSH session is started
```

**Scenario: run due task over SSH**

```gherkin
Given task T with next_run_at in the past
And project P has ssh_host H and key K and path D
And project P is not locked
And no other run is active
When tick occurs
Then SSH connects with K to H
And remote command runs with cwd D
And lock exists for P until SSH exits
And run record status becomes success or error
And next_run_at is updated
```

**Scenario: skip when project locked**

```gherkin
Given project P is locked by running task T1
And task T2 on P is also due
When tick occurs
Then T2 is not started
```

**Scenario: agent or SSH failure**

```gherkin
Given task T runs and remote/SSH exits with code 1
Then run status is error
And lock is released
And next_run_at is scheduled per interval
And daemon continues ticking
```

**Scenario: per-project SSH identity**

```gherkin
Given project A uses key KA on host HA
And project B uses key KB on host HB
When tasks for A and B run (sequentially)
Then A uses KA→HA and B uses KB→HB
```
