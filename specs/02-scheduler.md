# Scheduler & Daemon Lifecycle

## Daemon

Один long-running процесс `master-agent daemon` внутри Docker-контейнера.

**Main loop (упрощённо):**

```
loop forever:
  recover stale locks
  due_tasks = find all tasks where due and project not locked
  for each due_task (order: next_run_at ASC):
    try acquire lock for project
    if acquired:
      run via SSH (blocking wait for MVP)
      on exit: release lock, update run, schedule next_run_at
  sleep TICK_INTERVAL  # e.g. 10–30 seconds
```

**TICK_INTERVAL** — глобальная настройка (env/config), не путать с `Task.interval_seconds`. Tick только опрашивает БД; задача due по своему `next_run_at`.

## Scheduling Semantics

| Rule | Behavior |
|------|----------|
| Task due | `enabled` AND project `enabled` AND `now >= next_run_at` (or `next_run_at IS NULL` on first run) |
| Project locked | Все tasks проекта пропускаются до release lock |
| Multiple due tasks, same project | Запускается **первая** по `next_run_at`; остальные ждут следующих ticks |
| Multiple due tasks, different projects | MVP: **один run глобально** (последовательно) |
| After successful run | `next_run_at = finished_at + interval_seconds` |
| After failed run | То же — **без immediate retry** |
| Agent still running | Новые runs не стартуют (lock + глобально один active run) |

### MVP: один активный run глобально

Пока любой SSH-агент работает, daemon не стартует следующий. Lock на project остаётся для корректности данных и будущего parallel-by-project.

## Lock Lifecycle

```
1. BEGIN TRANSACTION
2. SELECT lock WHERE project_id = ?
3. IF exists → ROLLBACK, skip
4. INSERT run (status=running)
5. INSERT lock (project_id, task_id, run_id, pid of local ssh client)
6. COMMIT
7. ssh … remote command
8. wait for SSH session exit
9. UPDATE run (finished_at, exit_code, status)
10. DELETE lock WHERE project_id = ?
11. UPDATE task (last_run_at, next_run_at)
```

## Stale Lock Recovery

При старте daemon и периодически (каждый N tick):

- Для каждого lock с `pid`: если локальный SSH-клиент с этим PID не существует → DELETE lock, Run → `error` (`process lost`).

MVP: жизнь remote-агента = жизнь SSH-сессии (disconnect убивает remote command). Отдельный remote process supervisor — post-MVP.

## Error Handling

| Event | Action |
|-------|--------|
| Remote command exit ≠ 0 | Run → `error`, log stderr, release lock, schedule next |
| SSH auth / connect failure | Run → `error`, log, release lock, schedule next |
| Timeout (optional MVP+) | Kill local SSH → remote dies with session; Run → `error` |
| DB error | Log fatal, daemon continues if possible |

**Нет** автоматического retry до следующего `next_run_at`.

## Logging

- Structured logs в stdout контейнера (+ опционально файл в volume).
- Каждый run: project, task, ssh_host, local ssh pid, duration, exit_code.
- ERROR при non-zero exit / SSH failure.

## Process Signals (MVP+)

- `SIGTERM` — graceful: дождаться текущего SSH run, затем выход.
- `SIGHUP` — reload (optional, not MVP).

## State Diagram

```
                    ┌──────────────┐
                    │   DAEMON     │
                    │   ticking    │
                    └──────┬───────┘
                           │
              due task & no global run
                           ▼
                    ┌──────────────┐
                    │ acquire lock │
                    └──────┬───────┘
                           │ fail → skip
                           ▼
                    ┌──────────────┐
                    │   RUNNING    │──── SSH session → remote agent
                    └──────┬───────┘
                           │ SSH exit (any code)
                           ▼
                    ┌──────────────┐
                    │ log + unlock │
                    │ schedule next│
                    └──────┬───────┘
                           │
                           ▼
                    (back to ticking)
```
