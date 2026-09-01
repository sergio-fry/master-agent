# Web UI & HTTP API

Post-MVP control plane: operators manage Projects, Tasks, SSH keys, and inspect Runs/logs via a **browser UI** backed by an **HTTP API** on the same Go binary / SQLite store.

This does **not** change SSH execution semantics. The UI configures orchestration data; agents still run on workers.

## Goals

- List / create / edit / enable-disable **Projects** (path + SSH host/user/port + key path).
- Upload / replace **SSH private keys** inline in the project (stored in SQLite); never display key material after save.
- List / create / edit / enable-disable **Tasks** (interval, command, prompt only — no SSH fields).
- List **Runs** (filter by project/task/status) and view **run logs** (`log_path` / captured output).
- Read-only **daemon status**: tick config, whether a global run / project lock is active (best-effort).

## Non-Goals

- Multi-user RBAC / OAuth (MVP UI: shared secret or localhost-only).
- Live streaming of agent tokens / Cursor transcript UI.
- Editing backlog.md / Jira from the UI.
- Replacing the CLI — CLI remains supported.

## Process model

- Command: `master-agent serve` — HTTP API + static/embedded UI against `--db`.
- Optional: `master-agent daemon --http-addr :8080` mounts the same server in-process so one container serves both scheduler and UI.
- Default bind: `127.0.0.1:8080` (override via flag/env). Docker Compose may publish the port; protect with admin login when exposed beyond localhost.

## Auth

| Mode | Behavior |
|------|----------|
| No credentials configured | Open API (bind to loopback or trusted network) |
| `ADMIN_USERNAME` + `ADMIN_PASSWORD` set | Web UI login form; session cookie on `/api/v1/*`; optional `MASTER_AGENT_TOKEN` still accepted for API clients |
| `MASTER_AGENT_TOKEN` only (legacy) | Require `Authorization: Bearer <token>` on `/api/v1/*` |

Single admin account; no per-user RBAC in this milestone. Session TTL defaults to 7 days (`SESSION_TTL` env).

## API surface (conceptual)

REST JSON under `/api/v1`:

| Area | Endpoints (sketch) |
|------|-------------------|
| Projects | `GET/POST /projects`, `GET/PATCH /projects/{id}`, `POST /projects/{id}/ssh/test` |
| SSH test (draft) | `POST /ssh/test` — проверка полей до сохранения project |
| Keys | `POST /projects/{id}/key` (multipart); `GET` returns `{present: bool}` only |
| Tasks | `GET/POST /projects/{id}/tasks`, `GET/PATCH /tasks/{id}` |
| Runs | `GET /runs?project_id=&task_id=&status=`, `GET /runs/{id}`, `GET /runs/{id}/log` |
| Status | `GET /status` — daemon/lock summary if available |

Errors: standard HTTP codes + JSON `{error: "..."}` (see `internal/api.ErrorBody`).
Requests get `X-Request-ID` (echoed from client or generated). Structured request logs include method, path, status, duration, request_id.

## UI screens

1. **Projects** — table + create/edit form (SSH fields + path + enabled).
2. **Project key** — upload control; show “key configured” / path, never the private key.
3. **Tasks** — per-project list + form (name, interval, command, prompt, enabled).
4. **Runs** — filterable table; detail drawer with metadata + log viewer (tail / full text with size limit).
5. **Status** — small header or page: DB path, optional active run/lock.

UI tech (MVP): server-rendered or embedded static assets from the Go binary (no separate Node deploy required). Keep styling simple and functional.

## Docker

- Publish port when using compose profile / `serve`.
- Secrets volume writable by the API process for key upload (today `:ro` for daemon-only — Web UI may need `:rw` on `/secrets` or a dedicated write path documented in compose).
- known_hosts volume optional when projects use pinned host keys from SSH test; legacy setups may still mount `/etc/ssh/ssh_known_hosts`.

## Testing

- Unit/integration: API handlers against temp SQLite.
- Acceptance: HTTP client exercises CRUD + key upload + run list/log; stub data, no Cursor.
- Optional single browser smoke — not required if API acceptance covers behavior.

## Relation to MVP specs

- Data model unchanged (`specs/01`).
- Scheduler/SSH unchanged (`specs/02`–`03`).
- MVP CLI acceptance remains; Web UI has its own criteria below and in backlog milestone **Web UI**.
