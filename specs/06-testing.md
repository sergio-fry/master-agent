# Testing Strategy

Автотесты обязательны для новой и изменяемой функциональности. Цель — проверить **оркестратор** (расписание, lock, SSH, логи, next_run), а не поведение AI-агентов.

## Levels

### Unit

- Пакеты Go, стандартный `testing` + **testify** (`assert` / `require`).
- Table-driven tests — норма.
- Зависимость исполнения вынести в интерфейс (например `Runner`); в unit — **fake**, без сети и Docker.
- Покрывать: due/`next_run_at`, lock acquire/skip/release, placeholders, mapping exit → `success`/`error`, отсутствие immediate retry, stale lock (с fake process/clock).

Запуск: `go test ./...`

### Integration (optional middle)

- Реальный SQLite (temp file) + store/scheduler wiring; SSH — fake.
- Не заменяет acceptance для SSH-пути.

### Acceptance / E2E

- Максимально end-to-end по пути продакшена: **Docker Compose** с сервисами `master-agent` и **sshd worker**.
- Тот же `go test`, отдельный пакет (например `test/acceptance`) и/или build tag `acceptance`.
- Task.command — простые stub-команды на worker:
  - успех: `touch …/flag` или `echo ok > file`
  - ошибка: `exit 1`
  - удержание lock: `sleep N`
- Cursor, backlog.md CLI, MCP в acceptance **не используются**.

Пример топологии:

```
master-agent  --SSH-->  sshd worker (touch/echo/exit)
     │
   SQLite volume + mounted test keys
```

Проверки: flag/файл на worker, строки в `runs`/`locks`/`tasks`, сценарии из [04-acceptance-criteria.md](./04-acceptance-criteria.md).

Запуск (концепт): `docker compose -f docker-compose.test.yml up -d` затем `go test -tags=acceptance ./test/acceptance/...`

## Framework choices (fixed for MVP)

| Need | Choice |
|------|--------|
| Unit / most tests | `testing` + testify |
| Mocks | interfaces + fakes; mock codegen only if needed |
| E2E harness | Docker Compose test file + Go acceptance tests |
| BDD / Gherkin | **not required**; scenarios live in specs, code is Go tests |
| Browser / UI tests | N/A |

## CI expectations

- Unit (and light integration): every PR.
- Acceptance: every PR or dedicated CI job; must stay green for orchestration changes.

## Non-goals of the test suite

- Не тестировать качество ответов LLM / Cursor.
- Не требовать реальный backlog/Jira в CI.
- Не вводить local-only runner «ради тестов», если прод — только SSH.
