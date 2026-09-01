CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    ssh_host TEXT NOT NULL,
    ssh_user TEXT NOT NULL,
    ssh_port INTEGER NOT NULL DEFAULT 22,
    ssh_private_key TEXT NOT NULL DEFAULT '',
    ssh_host_key TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY NOT NULL,
    project_id TEXT NOT NULL REFERENCES projects(id),
    name TEXT NOT NULL,
    prompt TEXT NOT NULL,
    command TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    last_run_at TEXT,
    next_run_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS locks (
    project_id TEXT PRIMARY KEY NOT NULL REFERENCES projects(id),
    task_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    pid INTEGER,
    acquired_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY NOT NULL,
    task_id TEXT NOT NULL REFERENCES tasks(id),
    project_id TEXT NOT NULL REFERENCES projects(id),
    started_at TEXT NOT NULL,
    finished_at TEXT,
    exit_code INTEGER,
    status TEXT NOT NULL,
    error_message TEXT,
    log_path TEXT
);
