# Agent Invocation (SSH Runner)

Master-agent — тонкая обёртка над **SSH**. Вся предметная логика — в **prompt** задачи. Агент всегда выполняется на **worker-машине проекта**, не в контейнере оркестратора.

## Responsibilities Split

| Master Agent (container) | CLI Agent (on worker via SSH) |
|--------------------------|-------------------------------|
| Выбрать due task | Прочитать файлы, API, backlog — как сказано в prompt |
| Подставить плейсхолдеры | Выполнить работу |
| SSH с Project.key на Project.host | Закрыть/обновить задачу в источнике (если нужно) |
| `cd Project.path && command` на remote | Использовать MCP/skills, настроенные на worker |
| Wait + log exit code | Commit, PR, email — по prompt |
| Release lock | |

Master-agent **не закрывает задачи** и **не знает** формат backlog/Jira/очереди.  
MCP, Cursor skills, API keys — только на worker (`~/.cursor/`, shell profile). Контейнер их не монтирует и не передаёт.

## SSH Execution

Параметры из **Project** (не из Task):

| Parameter | Source |
|-----------|--------|
| Host | `Project.ssh_host` |
| User | `Project.ssh_user` |
| Port | `Project.ssh_port` |
| Identity file | `Project.ssh_key_path` |
| Remote cwd | `Project.path` |
| Remote command | Task.command после подстановки плейсхолдеров |
| Local stdout/stderr | Capture to Run log in container volume |

**Обязательные SSH options (MVP):**

- `BatchMode=yes` — без интерактивного ввода пароля
- `IdentitiesOnly=yes`
- `StrictHostKeyChecking=yes` (known_hosts смонтирован заранее)
- ServerAliveInterval / ServerAliveCountMax — для длинных агентов

**Концептуальный вызов:**

```bash
ssh -i "$ssh_key_path" \
    -p "$ssh_port" \
    -o BatchMode=yes \
    -o IdentitiesOnly=yes \
    "${ssh_user}@${ssh_host}" \
    "cd $(printf %q "$project_path") && exec $remote_command"
```

Рекомендуется `bash -lc '…'` на remote, если нужен PATH из login profile (иначе `cursor` / `backlog` могут не находиться).

**Жизненный цикл:** локальный процесс = `ssh` client. Exit code SSH при успешной сессии = exit code remote command. Обрыв SSH → Run `error`, lock release / stale recovery.

## Prompt Assembly

Final prompt:

```text
{{task.prompt}}
```

Опциональный wrapper (global, MVP можно omit):

```text
You are running scheduled task "{{task_name}}" for project "{{project_name}}".
Project directory: {{project_path}}

{{task.prompt}}
```

Промпт передаётся как аргумент remote command (`{{prompt}}`) или через remote temp file / stdin, если упираемся в длину командной строки.

## Command Examples (выполняются на worker)

```bash
cursor agent -p {{prompt}}
```

```json
["cursor", "agent", "--workspace", "{{project_path}}", "-p", "{{prompt}}"]
```

```bash
claude -p {{prompt}}
```

Не оборачивайте плейсхолдеры в кавычки в shell-командах: подстановка сама делает POSIX single-quote escaping (`internal/placeholder`).

Пользователь задаёт command per Task; master-agent не навязывает конкретный CLI.

## Success / Failure

| Exit / event | Run.status | Lock | Next run |
|--------------|------------|------|----------|
| remote exit 0 | `success` | released | scheduled |
| remote exit ≠ 0 | `error` | released | scheduled (no retry) |
| SSH connect/auth fail | `error` | released | scheduled |
| SSH disconnect / local ssh killed | `error` | released or stale recovery | scheduled |

`error_message`: последние N KB stderr или сообщение SSH-клиента.

## What Goes in the Prompt (user responsibility)

1. **Backlog.md:** «Возьми верхнюю открытую задачу через `backlog`, реализуй, закрой, остановись.»
2. **Email / scripts:** по правилам в репозитории worker.
3. **Jira/YouTrack MCP:** MCP уже настроен в Cursor на worker.

## Non-Requirements

- Не парсить stdout агента для state machine.
- Не implement task queue CRUD.
- Не configure tools/MCP/skills в контейнере.
- Не git push / commit от имени master-agent.
- Не запускать агента локально внутри Docker-образа оркестратора.
