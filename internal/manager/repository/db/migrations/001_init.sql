CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS specs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    model_options TEXT NOT NULL,
    data_options TEXT NOT NULL,
    resource_options TEXT NOT NULL,
    training_options TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    spec_id TEXT NOT NULL REFERENCES specs(id),
    idempotency_key TEXT,
    status TEXT NOT NULL,
    failure_reason TEXT,
    artifact_base_path TEXT,
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    UNIQUE(project_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_runs_project_created_at
    ON runs(project_id, created_at);

CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE(run_id, path)
);
