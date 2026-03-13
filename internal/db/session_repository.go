package db

import (
	"database/sql"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/session"
)

// SessionRepository provides SQLite-backed persistence for session records.
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository constructs a SessionRepository for the provided database handle.
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create stores a session record after validating its scope relationship.
func (r *SessionRepository) Create(record session.Session) error {
	if err := validateScopeRef(r.db, record.Scope); err != nil {
		return err
	}

	_, err := r.db.Exec(`
		INSERT INTO sessions (
			id, system_id, project_id, workspace_id, task, branch_name, status, started_at, ended_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID,
		record.Scope.SystemID,
		record.Scope.ProjectID,
		record.Scope.WorkspaceID,
		record.Task,
		record.BranchName,
		string(record.Status),
		record.StartedAt.UTC().Format(time.RFC3339Nano),
		nilTime(record.EndedAt),
		record.StartedAt.UTC().Format(time.RFC3339Nano),
		record.StartedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return common.WrapError(common.ErrWriteFailed, "insert session", err)
	}
	return nil
}

func nilTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
