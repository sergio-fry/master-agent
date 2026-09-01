# Master Agent

**Schedule AI agents across your projects and machines — without babysitting them.**

Master Agent is a background orchestrator that runs on a schedule. It connects to your worker machines over SSH and launches CLI agents (Cursor Agent, custom scripts, etc.) with the prompts you define. You configure *what* runs, *where*, and *how often*; Master Agent handles the rest — scheduling, locking, SSH, and logging.

Master Agent does **not** decide what work the agent should do. That lives in each task's **prompt**. The orchestrator only ensures agents start on time, one at a time per project, and that you can see what happened.

## What problem does it solve?

If you manage several codebases or environments, you often want recurring agent work:

- triage a backlog every morning
- run periodic code health checks
- sync documentation or open pull requests on a timer

Doing this manually does not scale. Cron on each machine is fragile and hard to observe. Master Agent gives you **one place** to define projects, tasks, schedules, and prompts — and **one history** of every run.

## How it works

```
┌─────────────────┐     SSH      ┌──────────────────────────────┐
│  Master Agent   │ ──────────►  │  Worker machine              │
│  (Docker)       │              │  ┌────────────────────────┐  │
│                 │              │  │ Project folder         │  │
│  • scheduler    │   prompt +   │  │ Cursor Agent / CLI     │  │
│  • SQLite config│   command    │  │ MCP, skills, tools     │  │
│  • run logs     │ ◄──────────  │  └────────────────────────┘  │
└─────────────────┘   exit/log   └──────────────────────────────┘
```

1. You register **projects** — a remote folder + SSH target (host, user, key).
2. You add **tasks** — an interval, a command, and a prompt for the agent.
3. The daemon checks the schedule, SSHs into the worker, runs the command with your prompt, waits for completion, and stores the log.

Agents, MCP servers, and project tooling live on the **worker**. Master Agent stays thin: orchestration only.

## Use cases

| Scenario | Setup |
|----------|-------|
| **Multiple repositories** | One project per repo; each can point to a different worker or the same machine with different paths. |
| **Different machines** | Project A on `build-server`, Project B on `dev-laptop` — each with its own SSH key and host. |
| **Different agent jobs** | Several tasks per project: e.g. "backlog triage" every 6 h, "dependency audit" weekly — each with its own prompt and interval. |
| **Custom prompting** | Task prompt is passed to the remote command (`{{prompt}}` placeholder). You control instructions per task without redeploying Master Agent. |
| **Observability** | Every execution is a **run** with start time, exit code, and log file. Inspect via CLI, HTTP API, or Web UI. |

## Quick start

### Prerequisites

- Docker and Docker Compose
- SSH access from the Master Agent container to your worker machine(s)
- A CLI agent installed on each worker (e.g. `cursor agent`)

### 1. Clone and start

```bash
git clone <repo-url> master-agent
cd master-agent
docker compose build
docker compose up -d
```

The default command starts the **scheduler + Web UI** on port `8080`.

Open [http://127.0.0.1:8080](http://127.0.0.1:8080) to manage projects and tasks in the browser, or use the CLI below.

### 2. Prepare SSH

```bash
mkdir -p secrets/projects/my-app
# Copy your private key (mode 600)
cp ~/.ssh/id_ed25519 secrets/projects/my-app/id_ed25519
chmod 600 secrets/projects/my-app/id_ed25519

# Trust worker host keys (StrictHostKeyChecking=yes)
cp ssh/known_hosts.example ssh/known_hosts
ssh-keyscan -H worker.example.com >> ssh/known_hosts
```

Register the key using the **in-container path**: `/secrets/projects/my-app/id_ed25519`.

### 3. Register a project

```bash
docker compose run --rm master-agent project add \
  --name my-app \
  --path /home/worker/projects/my-app \
  --ssh-host worker.example.com \
  --ssh-user worker \
  --ssh-key /secrets/projects/my-app/id_ed25519
```

### 4. Add a scheduled task

```bash
docker compose run --rm master-agent task add \
  --project my-app \
  --name backlog-triage \
  --interval 21600 \
  --command 'cursor agent -p {{prompt}}' \
  --prompt "Review backlog.md and pick the highest-priority ready task."
```

The daemon picks up enabled tasks automatically. Check runs:

```bash
docker compose run --rm master-agent run list --project my-app
```

## Configuration model

| Concept | What it means |
|---------|---------------|
| **Project** | Where the work runs: remote path + SSH host, user, and key. |
| **Task** | What to run: interval, remote command, and agent prompt. |
| **Run** | One execution record — timestamps, exit code, log path. |
| **Lock** | Prevents overlapping runs for the same project. |

**Project** answers *where* and *how to connect*. **Task** answers *when* and *what to tell the agent*. SSH settings are not duplicated on tasks — they inherit from the project.

Command placeholders: `{{prompt}}`, `{{project_path}}`, `{{project_name}}`, `{{task_name}}`, `{{task_id}}`.

## Web UI & HTTP API

With the default Docker setup, the UI and JSON API are available at `http://127.0.0.1:8080`.

| Feature | Description |
|---------|-------------|
| Projects | Create and edit remote workspaces and SSH settings |
| Tasks | Configure schedules, commands, and prompts |
| SSH keys | Upload keys into the secrets volume (never shown after upload) |
| Runs | Browse history and read execution logs |
| Status | Daemon health and active locks |

Protect exposed deployments with admin login (Web UI) or an optional bearer token for API clients:

```bash
# docker-compose.yml
environment:
  ADMIN_USERNAME: "admin"
  ADMIN_PASSWORD: "change-me"
  SESSION_SECRET: "long-random-string"   # optional; derived from password if unset
  # MASTER_AGENT_TOKEN: "change-me"       # optional API automation token
```

Open the Web UI and sign in with `ADMIN_USERNAME` / `ADMIN_PASSWORD`. The session cookie persists across browser restarts (default 7 days, override with `SESSION_TTL`).

When admin login is **not** configured, the API is open unless `MASTER_AGENT_TOKEN` is set:

```bash
curl -H "Authorization: Bearer $MASTER_AGENT_TOKEN" \
  http://127.0.0.1:8080/api/v1/status
```

Standalone API (no scheduler):

```bash
master-agent serve --db /data/master-agent.db
```

## CLI reference

```bash
master-agent project add|list|enable|disable
master-agent task add|list|enable|disable
master-agent run list
master-agent daemon [--http-addr :8080]
master-agent serve [--addr 127.0.0.1:8080]
```

All commands accept `--db` (default `/data/master-agent.db` in Docker).

## Production / system install

For a pinned, always-on instance separate from dev compose in this repo, use the scripts under [`deploy/`](deploy/):

```bash
sudo ./deploy/install.sh              # builds master-agent:<git-describe>, installs to /opt/master-agent
sudo ./deploy/upgrade.sh v1.2.0       # rebuild and reload — only when you explicitly upgrade
sudo systemctl status master-agent
docker logs master-agent-prod-master-agent-1
```

| | Dev (`docker compose up`) | System install |
|--|---------------------------|----------------|
| Port | `8080` | **`9080`** (LAN: `http://<host-ip>:9080`) |
| Image | `master-agent:local` (build from checkout) | `master-agent:<VERSION>` (pinned tag) |
| Data | `./data`, `./secrets` | `/opt/master-agent/data`, `/opt/master-agent/secrets` |
| Updates | Rebuild anytime while developing | Only via `deploy/upgrade.sh` |

The systemd unit uses `docker compose up --no-build --pull never` so reboots never pull or rebuild the image. Admin credentials are generated in `/opt/master-agent/.env` on install (`ADMIN_USERNAME=admin` plus a random password). Sign in at `http://<host-ip>:9080/login.html`.

Before scheduling SSH work, populate `/opt/master-agent/ssh/known_hosts` and place keys under `/opt/master-agent/secrets/projects/<name>/`.

---

## Technical details

### Stack

Go binary in a minimal Docker image (OpenSSH client + SQLite). No Cursor, MCP, or agent tooling in the image — those run on workers.

### Docker volumes

| Host path | Container | Purpose |
|-----------|-----------|---------|
| `./data` | `/data` | SQLite database and run logs |
| `./secrets` | `/secrets` | Per-project SSH private keys |
| `./ssh/known_hosts` | `/etc/ssh/ssh_known_hosts` (ro) | Host key verification |

SSH keys and `known_hosts` are **runtime mounts**, never baked into the image.

Environment variables:

| Variable | Purpose |
|----------|---------|
| `TICK_INTERVAL` | How often the scheduler checks due tasks (default `30s`) |
| `HTTP_ADDR` / `--http-addr` | Bind address for API + UI |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | Web UI login (session cookie) |
| `SESSION_SECRET` / `SESSION_TTL` | Session signing key and lifetime |
| `MASTER_AGENT_TOKEN` | Optional Bearer token for `/api/v1` automation |

Health check: `GET /api/v1/status` → `{"ok":true}`.

### Build & test (local)

Requires Go — see [specs/05-tech-stack.md](specs/05-tech-stack.md).

```bash
go build ./...
go test ./...
```

Entrypoint: `cmd/master-agent`. Packages under `internal/` (`cli`, `store`, `scheduler`, `runner`, `api`, `webui`, …).

#### Acceptance (E2E)

Docker Compose harness (`docker-compose.test.yml`): `master-agent` + `sshd` worker, test keys under `test/fixtures/ssh/`. Remote commands are stubs only — no real Cursor or MCP.

```bash
go test -tags=acceptance ./test/acceptance/...

# Or manage compose manually:
docker compose -f docker-compose.test.yml up -d --build
ACCEPTANCE_SKIP_COMPOSE=1 go test -tags=acceptance ./test/acceptance/...
docker compose -f docker-compose.test.yml down -v
```

See [specs/06-testing.md](specs/06-testing.md).

### Design principles

- **Orchestration ≠ execution** — Master Agent schedules; agents work in their configured environment on the worker.
- **Configuration as data** — projects and tasks live in SQLite, not in code.
- **Fail quiet, log loud** — SSH or agent errors are logged; the lock is released; the next scheduled tick retries.

## Documentation

Spec-driven design docs:

| Spec | Description |
|------|-------------|
| [specs/00-overview.md](specs/00-overview.md) | Vision, goals, glossary |
| [specs/01-data-model.md](specs/01-data-model.md) | Project, Task, Lock, Run |
| [specs/02-scheduler.md](specs/02-scheduler.md) | Daemon loop, locks, errors |
| [specs/03-agent-invocation.md](specs/03-agent-invocation.md) | SSH runner, responsibilities |
| [specs/04-acceptance-criteria.md](specs/04-acceptance-criteria.md) | MVP checklist |
| [specs/05-tech-stack.md](specs/05-tech-stack.md) | Go, Docker, SQLite, SSH |
| [specs/06-testing.md](specs/06-testing.md) | Unit + acceptance E2E |
| [specs/07-web-ui.md](specs/07-web-ui.md) | HTTP API + Web UI |
| [AGENTS.md](AGENTS.md) | Guidelines for contributors and AI agents |
