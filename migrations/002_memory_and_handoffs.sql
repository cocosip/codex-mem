CREATE TABLE IF NOT EXISTS memory_items (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE RESTRICT,
    system_id TEXT NOT NULL REFERENCES systems(id) ON DELETE RESTRICT,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE RESTRICT,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    importance INTEGER NOT NULL,
    tags_json TEXT NOT NULL DEFAULT '[]',
    file_paths_json TEXT NOT NULL DEFAULT '[]',
    related_project_ids_json TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL,
    source TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (importance BETWEEN 1 AND 5),
    CHECK (status IN ('active', 'resolved', 'superseded')),
    CHECK (source IN ('codex_explicit', 'watcher_import', 'relay_import', 'recovery_generated')),
    CHECK (type IN ('decision', 'bugfix', 'discovery', 'constraint', 'preference', 'todo'))
);

CREATE TABLE IF NOT EXISTS handoffs (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE RESTRICT,
    system_id TEXT NOT NULL REFERENCES systems(id) ON DELETE RESTRICT,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE RESTRICT,
    kind TEXT NOT NULL,
    task TEXT NOT NULL,
    summary TEXT NOT NULL,
    completed_json TEXT NOT NULL DEFAULT '[]',
    next_steps_json TEXT NOT NULL DEFAULT '[]',
    open_questions_json TEXT NOT NULL DEFAULT '[]',
    risks_json TEXT NOT NULL DEFAULT '[]',
    files_touched_json TEXT NOT NULL DEFAULT '[]',
    related_note_ids_json TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (kind IN ('final', 'checkpoint', 'recovery')),
    CHECK (status IN ('open', 'completed', 'abandoned'))
);

CREATE INDEX IF NOT EXISTS idx_memory_items_workspace_status_importance_created_at
    ON memory_items(workspace_id, status, importance DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_items_project_status_importance_created_at
    ON memory_items(project_id, status, importance DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_items_session_id ON memory_items(session_id);
CREATE INDEX IF NOT EXISTS idx_handoffs_workspace_status_created_at
    ON handoffs(workspace_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_handoffs_project_status_created_at
    ON handoffs(project_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_handoffs_session_id ON handoffs(session_id);
