package db

import (
	"database/sql"
	"errors"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
)

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

func validateSessionScope(db *sql.DB, sessionID string, ref scope.Ref) error {
	var stored scope.Ref
	err := db.QueryRow(`
		SELECT system_id, project_id, workspace_id
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(&stored.SystemID, &stored.ProjectID, &stored.WorkspaceID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return common.NewError(common.ErrSessionNotFound, "session_id does not exist")
	case err != nil:
		return common.WrapError(common.ErrReadFailed, "load session scope", err)
	}

	if stored != ref {
		return common.NewError(common.ErrInvalidScope, "session scope does not match the provided scope")
	}
	return nil
}
