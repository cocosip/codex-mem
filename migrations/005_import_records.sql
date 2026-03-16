CREATE TABLE IF NOT EXISTS imports (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE RESTRICT,
    system_id TEXT NOT NULL REFERENCES systems(id) ON DELETE RESTRICT,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE RESTRICT,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE RESTRICT,
    source TEXT NOT NULL,
    external_id TEXT NOT NULL DEFAULT '',
    payload_hash TEXT NOT NULL DEFAULT '',
    durable_memory_id TEXT NOT NULL DEFAULT '',
    suppressed INTEGER NOT NULL DEFAULT 0,
    suppression_reason TEXT NOT NULL DEFAULT '',
    imported_at TEXT NOT NULL,
    CHECK (source IN ('watcher_import', 'relay_import')),
    CHECK (suppressed IN (0, 1))
);

CREATE INDEX IF NOT EXISTS idx_imports_session_id ON imports(session_id);
CREATE INDEX IF NOT EXISTS idx_imports_project_imported_at ON imports(project_id, imported_at DESC);
CREATE INDEX IF NOT EXISTS idx_imports_project_source_external_id
    ON imports(project_id, source, external_id)
    WHERE TRIM(external_id) <> '';
CREATE INDEX IF NOT EXISTS idx_imports_project_source_payload_hash
    ON imports(project_id, source, payload_hash)
    WHERE TRIM(payload_hash) <> '';
