package scope

import (
	"strings"

	"codex-mem/internal/domain/common"
)

type Ref struct {
	SystemID    string `json:"system_id"`
	ProjectID   string `json:"project_id"`
	WorkspaceID string `json:"workspace_id"`
}

func (r Ref) Validate() error {
	switch {
	case strings.TrimSpace(r.SystemID) == "":
		return common.NewError(common.ErrInvalidScope, "system_id is required")
	case strings.TrimSpace(r.ProjectID) == "":
		return common.NewError(common.ErrInvalidScope, "project_id is required")
	case strings.TrimSpace(r.WorkspaceID) == "":
		return common.NewError(common.ErrInvalidScope, "workspace_id is required")
	default:
		return nil
	}
}

type Scope struct {
	SystemID      string `json:"system_id"`
	SystemName    string `json:"system_name"`
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceRoot string `json:"workspace_root"`
	BranchName    string `json:"branch_name,omitempty"`
	ResolvedBy    string `json:"resolved_by"`
}

func (s Scope) Validate() error {
	if err := s.Ref().Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(s.SystemName) == "" {
		return common.NewError(common.ErrInvalidScope, "system_name is required")
	}
	if strings.TrimSpace(s.ProjectName) == "" {
		return common.NewError(common.ErrInvalidScope, "project_name is required")
	}
	if strings.TrimSpace(s.WorkspaceRoot) == "" {
		return common.NewError(common.ErrInvalidScope, "workspace_root is required")
	}
	if strings.TrimSpace(s.ResolvedBy) == "" {
		return common.NewError(common.ErrInvalidScope, "resolved_by is required")
	}
	return nil
}

func (s Scope) Ref() Ref {
	return Ref{
		SystemID:    s.SystemID,
		ProjectID:   s.ProjectID,
		WorkspaceID: s.WorkspaceID,
	}
}

type ResolveInput struct {
	CWD             string
	BranchName      string
	RepoRemote      string
	ProjectNameHint string
	SystemNameHint  string
}

type ResolveOutput struct {
	Scope      Scope            `json:"scope"`
	ResolvedBy string           `json:"resolved_by"`
	Warnings   []common.Warning `json:"warnings"`
}

type SystemRecord struct {
	ID   string
	Name string
	Slug string
}

type ProjectRecord struct {
	ID               string
	SystemID         string
	Name             string
	Slug             string
	CanonicalKey     string
	RemoteNormalized string
}

type WorkspaceRecord struct {
	ID           string
	ProjectID    string
	RootPath     string
	WorkspaceKey string
	BranchName   string
}

type Repository interface {
	EnsureSystem(system SystemRecord) (SystemRecord, error)
	EnsureProject(project ProjectRecord) (ProjectRecord, error)
	EnsureWorkspace(workspace WorkspaceRecord) (WorkspaceRecord, error)
}
