CREATE TABLE projects (
    id          TEXT PRIMARY KEY, -- uuid
    name        TEXT NOT NULL,
    path        TEXT NOT NULL UNIQUE,
    trusted     BOOLEAN DEFAULT FALSE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_opened DATETIME
);

CREATE TABLE project_config (
    project_id  TEXT PRIMARY KEY,
    models      TEXT, -- JSON blob of ModelConfig map
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE task_runs (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL,
    session_id  TEXT NOT NULL,
    task_id     TEXT NOT NULL,
    agent_role  TEXT,
    status      TEXT,
    summary     TEXT,
    error       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    project_id  TEXT NOT NULL,
    goal        TEXT,
    dag         TEXT, -- JSON blob of full DAG
    status      TEXT, -- "running" | "done" | "failed"
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id)
);