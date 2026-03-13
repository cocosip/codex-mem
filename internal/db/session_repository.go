package db

import (
	"database/sql"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

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

func validateScopeRef(db *sql.DB, ref scope.Ref) error {
	var matched int
	err := db.QueryRow(`
		SELECT COUNT(1)
		FROM workspaces w
		INNER JOIN projects p ON p.id = w.project_id
		INNER JOIN systems s ON s.id = p.system_id
		WHERE w.id = ? AND p.id = ? AND s.id = ?
	`, ref.WorkspaceID, ref.ProjectID, ref.SystemID).Scan(&matched)
	if err != nil {
		return common.WrapError(common.ErrReadFailed, "validate scope chain", err)
	}
	if matched != 1 {
		return common.NewError(common.ErrInvalidScope, "scope chain does not match stored workspace/project/system hierarchy")
	}
	return nil
}

func nilTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
