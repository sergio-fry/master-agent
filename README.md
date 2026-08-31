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
