CREATE TABLE IF NOT EXISTS systems (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    system_id TEXT NOT NULL REFERENCES systems(id) ON DELETE RESTRICT,
    slug TEXT NOT NULL,
    name TEXT NOT NULL,
    canonical_key TEXT NOT NULL UNIQUE,
    remote_normalized TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (system_id, slug)
);

CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
    root_path TEXT NOT NULL,
    workspace_key TEXT NOT NULL UNIQUE,
    branch_name TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (project_id, root_path)
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    system_id TEXT NOT NULL REFERENCES systems(id) ON DELETE RESTRICT,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE RESTRICT,
    task TEXT,
    branch_name TEXT,
    status TEXT NOT NULL,
    started_at TEXT NOT NULL,
    ended_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (status IN ('active', 'paused', 'ended', 'recovered'))
);

CREATE INDEX IF NOT EXISTS idx_projects_system_id ON projects(system_id);
CREATE INDEX IF NOT EXISTS idx_workspaces_project_id ON workspaces(project_id);
CREATE INDEX IF NOT EXISTS idx_sessions_workspace_status_started_at ON sessions(workspace_id, status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_project_status_started_at ON sessions(project_id, status, started_at DESC);
