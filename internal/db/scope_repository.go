package db

import (
	"database/sql"
	"errors"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
)

type ScopeRepository struct {
	db    *sql.DB
	clock common.Clock
}

func NewScopeRepository(db *sql.DB, clock common.Clock) *ScopeRepository {
	if clock == nil {
		clock = common.RealClock{}
	}
	return &ScopeRepository{db: db, clock: clock}
}

func (r *ScopeRepository) EnsureSystem(system scope.SystemRecord) (scope.SystemRecord, error) {
	now := r.clock.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.Exec(`
		INSERT INTO systems (id, slug, name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			slug = excluded.slug,
			name = excluded.name,
			updated_at = excluded.updated_at
	`, system.ID, system.Slug, system.Name, now, now)
	if err != nil {
		return scope.SystemRecord{}, common.WrapError(common.ErrWriteFailed, "ensure system", err)
	}
	return system, nil
}

func (r *ScopeRepository) EnsureProject(project scope.ProjectRecord) (scope.ProjectRecord, error) {
	var existing scope.ProjectRecord
	err := r.db.QueryRow(`
		SELECT id, system_id, name, slug, canonical_key, COALESCE(remote_normalized, '')
		FROM projects
		WHERE canonical_key = ?
	`, project.CanonicalKey).Scan(
		&existing.ID,
		&existing.SystemID,
		&existing.Name,
		&existing.Slug,
		&existing.CanonicalKey,
		&existing.RemoteNormalized,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		now := r.clock.Now().UTC().Format(time.RFC3339Nano)
		_, err = r.db.Exec(`
			INSERT INTO projects (
				id, system_id, slug, name, canonical_key, remote_normalized, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, project.ID, project.SystemID, project.Slug, project.Name, project.CanonicalKey, project.RemoteNormalized, now, now)
		if err != nil {
			return scope.ProjectRecord{}, common.WrapError(common.ErrWriteFailed, "insert project", err)
		}
		return project, nil
	case err != nil:
		return scope.ProjectRecord{}, common.WrapError(common.ErrReadFailed, "load project", err)
	case existing.SystemID != project.SystemID:
		return scope.ProjectRecord{}, common.NewError(common.ErrScopeConflict, "project canonical key already belongs to a different system")
	default:
		_, err = r.db.Exec(`
			UPDATE projects
			SET slug = ?, name = ?, remote_normalized = ?, updated_at = ?
			WHERE id = ?
		`, project.Slug, project.Name, project.RemoteNormalized, r.clock.Now().UTC().Format(time.RFC3339Nano), existing.ID)
		if err != nil {
			return scope.ProjectRecord{}, common.WrapError(common.ErrWriteFailed, "update project metadata", err)
		}
		existing.Name = project.Name
		existing.Slug = project.Slug
		existing.RemoteNormalized = project.RemoteNormalized
		return existing, nil
	}
}

func (r *ScopeRepository) EnsureWorkspace(workspace scope.WorkspaceRecord) (scope.WorkspaceRecord, error) {
	var existing scope.WorkspaceRecord
	err := r.db.QueryRow(`
		SELECT id, project_id, root_path, workspace_key, COALESCE(branch_name, '')
		FROM workspaces
		WHERE workspace_key = ?
	`, workspace.WorkspaceKey).Scan(
		&existing.ID,
		&existing.ProjectID,
		&existing.RootPath,
		&existing.WorkspaceKey,
		&existing.BranchName,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		now := r.clock.Now().UTC().Format(time.RFC3339Nano)
		_, err = r.db.Exec(`
			INSERT INTO workspaces (
				id, project_id, root_path, workspace_key, branch_name, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, workspace.ID, workspace.ProjectID, workspace.RootPath, workspace.WorkspaceKey, workspace.BranchName, now, now)
		if err != nil {
			return scope.WorkspaceRecord{}, common.WrapError(common.ErrWriteFailed, "insert workspace", err)
		}
		return workspace, nil
	case err != nil:
		return scope.WorkspaceRecord{}, common.WrapError(common.ErrReadFailed, "load workspace", err)
	case existing.ProjectID != workspace.ProjectID:
		return scope.WorkspaceRecord{}, common.NewError(common.ErrScopeConflict, "workspace key already belongs to a different project")
	default:
		_, err = r.db.Exec(`
			UPDATE workspaces
			SET root_path = ?, branch_name = ?, updated_at = ?
			WHERE id = ?
		`, workspace.RootPath, workspace.BranchName, r.clock.Now().UTC().Format(time.RFC3339Nano), existing.ID)
		if err != nil {
			return scope.WorkspaceRecord{}, common.WrapError(common.ErrWriteFailed, "update workspace metadata", err)
		}
		existing.RootPath = workspace.RootPath
		existing.BranchName = workspace.BranchName
		return existing, nil
	}
}
