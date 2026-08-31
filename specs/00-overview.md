# Master Agent — Overview

## Vision

Master Agent — фоновый оркестратор в **Docker**, написанный на **Go**. По расписанию он заходит по **SSH** на рабочие машины и запускает там CLI-агентов (Cursor Agent и т.п.). Он не знает, *что именно* делает агент: backlog, коммиты, письма, Jira — всё описывается в **промпте задачи**.

Master Agent отвечает только за:

- хранение конфигурации проектов и задач (SQLite);
- планирование запусков по расписанию;
- блокировку «один агент на проект за раз» через SQLite;
- SSH-подключение к машине проекта (ключ + host из Project);
- удалённый запуск команды с промптом в папке проекта;
- ожидание завершения и логирование результата;
- (post-MVP) HTTP API + Web UI для управления конфигурацией и просмотра runs/логов.

## Goals (MVP)

- Long-running daemon в Docker-контейнере.
- Стек: **Go + SQLite + OpenSSH client + Docker image**.
- Несколько **проектов**: у каждого — remote path, SSH host/user, SSH key.
- Несколько **задач** на проект: своя периодичность, команда, промпт.
- Исполнение **всегда через SSH** (даже если «удалённая» машина — localhost / Docker host).
- CLI-агент сам выполняет работу и сам «закрывает» задачу — master-agent в это не вмешивается.

## Goals (Web UI milestone)

- Browser UI + JSON API: projects, tasks, SSH key upload, runs, logs — see [07-web-ui.md](./07-web-ui.md).

## Non-Goals (MVP)

- Очереди задач, приоритеты, таск-трекеры — **не часть master-agent**.
- Настройка MCP, skills, тулов агента — **на стороне worker-машины / Cursor**.
- Локальный spawn агента внутри контейнера master-agent.
- Параллельный запуск нескольких агентов **в одном проекте**.
- Retry при ошибке агента — только лог + ожидание следующей итерации расписания.

## Non-Goals (still out of product)

- Multi-user OAuth/RBAC, Cursor transcript UI, replacing the CLI.

## Принципы

1. **Максимальная простота** — минимум абстракций, SQLite, один бинарь в контейнере.
2. **Оркестрация ≠ исполнение** — контейнер планирует; агент работает в настроенном окружении на SSH-хосте.
3. **Конфигурация через данные** — Project/Task в БД, не хардкод.
4. **Fail quiet, log loud** — ошибка агента/SSH: залогировать, снять lock, ждать следующего tick.

## Glossary

| Term | Meaning |
|------|---------|
| **Project** | Remote workspace: path + SSH host/user/key |
| **Task** | Расписание + команда + промпт внутри Project |
| **Run** | Один факт запуска (лог: start, end, exit code, stderr) |
| **Lock** | Запись в БД: проект занят выполняющимся агентом |
| **Worker** | Машина, куда SSH: там Cursor, backlog.md, MCP |
| **CLI Agent** | Внешний процесс на worker, которому передаётся промпт |

## Spec Index

| Spec | Topic |
|------|-------|
| [01-data-model.md](./01-data-model.md) | Project, Task, Lock, Run |
| [02-scheduler.md](./02-scheduler.md) | Daemon, расписание, lifecycle |
| [03-agent-invocation.md](./03-agent-invocation.md) | SSH runner, промпт, ошибки |
| [04-acceptance-criteria.md](./04-acceptance-criteria.md) | Критерии готовности MVP |
| [05-tech-stack.md](./05-tech-stack.md) | Go, Docker, SQLite, SSH |
| [06-testing.md](./06-testing.md) | Unit + acceptance (Docker Compose / SSH stubs) |
| [07-web-ui.md](./07-web-ui.md) | HTTP API + Web UI (projects, tasks, keys, runs) |
