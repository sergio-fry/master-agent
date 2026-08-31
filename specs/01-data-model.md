# Data Model

Хранилище: **SQLite** (один файл в volume контейнера, по умолчанию `/data/master-agent.db`).

Master-agent **не моделирует очереди, issue, статусы задач трекера** — только проекты (куда SSH), расписания запусков и историю runs.

## Entity: Project

Проект = **удалённая рабочая папка + SSH-цель**. Все Task проекта выполняются на этой машине под этим ключом.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | TEXT (UUID) | yes | Primary key |
| `name` | TEXT | yes | Человекочитаемое имя |
| `path` | TEXT | yes | Абсолютный путь к папке проекта **на worker-машине** |
| `ssh_host` | TEXT | yes | Хост или SSH alias (`worker.example`, `host.docker.internal`) |
| `ssh_user` | TEXT | yes | Пользователь SSH |
| `ssh_port` | INTEGER | yes | Порт SSH (default: 22) |
| `ssh_key_path` | TEXT | yes | Путь к private key **внутри контейнера** (например `/secrets/projects/my-app/id_ed25519`) |
| `enabled` | INTEGER (0/1) | yes | Участвует ли в scheduling (default: 1) |
| `created_at` | TEXT (ISO8601) | yes | |
| `updated_at` | TEXT (ISO8601) | yes | |

**Constraints:**

- `path` — путь на **удалённой** ФС; локальная проверка существования в контейнере **не требуется** (опционально: `ssh … test -d`).
- У каждого проекта свой ключ и своя машина (могут совпадать между проектами, но хранятся per-project).
- `ssh_key_path` должен быть доступен для чтения процессу daemon (обычно volume mount `:ro`).
- `name` — unique среди enabled-проектов (рекомендация).

**Что принадлежит Project vs Task:**

| Project | Task |
|---------|------|
| `path`, `ssh_host`, `ssh_user`, `ssh_port`, `ssh_key_path` | `interval_seconds`, `command`, `prompt` |
| «где» и «под каким ключом» | «когда» и «что сказать агенту» |

## Entity: Task

Задача = **расписание + команда + промпт** в рамках одного Project. SSH-параметры **не** дублируются на Task — наследуются от Project.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | TEXT (UUID) | yes | Primary key |
| `project_id` | TEXT | yes | FK → Project |
| `name` | TEXT | yes | Имя задачи (для логов и подстановки в промпт) |
| `prompt` | TEXT | yes | Инструкция для CLI-агента |
| `command` | TEXT | yes | Команда запуска **на worker** (shell string или JSON argv) |
| `interval_seconds` | INTEGER | yes | Периодичность запуска |
| `enabled` | INTEGER (0/1) | yes | default: 1 |
| `last_run_at` | TEXT (ISO8601) | no | Время последнего *завершённого* run |
| `next_run_at` | TEXT (ISO8601) | no | Время следующего запуска |
| `created_at` | TEXT (ISO8601) | yes | |
| `updated_at` | TEXT (ISO8601) | yes | |

**Command format (MVP):**

- Строка, выполняемая через remote shell в `Project.path`, **или**
- JSON-массив argv (предпочтительно).

Примеры:

```text
cursor agent -p {{prompt}}
```

```json
["cursor", "agent", "-p", "{{prompt}}"]
```

| Placeholder | Value |
|-------------|-------|
| `{{prompt}}` | Task.prompt (shell-quoted in string commands; literal in JSON argv) |
| `{{project_path}}` | Project.path |
| `{{project_name}}` | Project.name |
| `{{task_name}}` | Task.name |
| `{{task_id}}` | Task.id |

**Substitution rules:** shell-string commands get POSIX single-quote escaping per value; JSON argv commands substitute literally. Unknown `{{…}}` → error. Empty fields → empty value (`''` in shell mode).

**Scheduling rule:**

- Задача due, если `enabled = 1`, project enabled, и (`next_run_at IS NULL` OR `now >= next_run_at`).
- После завершения run (успех или ошибка): `last_run_at = now`, `next_run_at = now + interval_seconds`.

## Entity: Lock

Блокировка на уровне **Project** — пока lock активен, ни одна Task этого проекта не запускается.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `project_id` | TEXT | yes | PK, FK → Project |
| `task_id` | TEXT | yes | Какая task захватила lock |
| `run_id` | TEXT | yes | FK → Run |
| `pid` | INTEGER | no | PID **локального** SSH-клиента в контейнере |
| `acquired_at` | TEXT (ISO8601) | yes | |

**Rules:**

- Acquire: INSERT в транзакции; если row для `project_id` уже есть — skip run.
- Release: DELETE при завершении SSH-сессии (в finally), даже при ошибке.
- Stale lock (MVP): если локальный процесс с `pid` не существует — снять lock при recovery.

## Entity: Run

История запусков для аудита и отладки.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | TEXT (UUID) | yes | Primary key |
| `task_id` | TEXT | yes | FK → Task |
| `project_id` | TEXT | yes | FK → Project (denormalized) |
| `started_at` | TEXT (ISO8601) | yes | |
| `finished_at` | TEXT (ISO8601) | no | |
| `exit_code` | INTEGER | no | Exit code remote command / SSH |
| `status` | TEXT | yes | `running` \| `success` \| `error` |
| `error_message` | TEXT | no | stderr tail или SSH error |
| `log_path` | TEXT | no | Путь к логу **в контейнере** (volume) |

## ER Diagram

```
Project 1 ──< Task
Project 1 ──o Lock (0..1 active)
Task    1 ──< Run
```

## Example: типичная настройка

**Project:** `my-app`

- path: `/home/dev/my-app`
- ssh_host: `dev-box`
- ssh_user: `dev`
- ssh_key_path: `/secrets/projects/my-app/id_ed25519`

**Task A** — каждые 30 мин:

```text
Посмотри backlog (backlog.md CLI). Возьми верхнюю открытую задачу,
реализуй, закоммить, закрой задачу. Остановись.
```

**Task B** — каждые 24 ч:

```text
Проверь зависимости на уязвимости, обнови lockfile если нужно, создай PR.
```

Master-agent не парсит backlog — это ответственность CLI-агента на worker по промпту.

## CLI для управления данными (MVP)

```bash
master-agent project add \
  --name my-app \
  --path /home/dev/my-app \
  --ssh-host dev-box \
  --ssh-user dev \
  --ssh-key /secrets/projects/my-app/id_ed25519

master-agent task add --project my-app --name drain --interval 1800 \
  --command 'cursor agent -p {{prompt}}' \
  --prompt '...'

master-agent run list --project my-app
master-agent daemon start
```
