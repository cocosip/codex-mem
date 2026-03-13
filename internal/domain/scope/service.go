package scope

import (
	"context"
	"path/filepath"
	"strings"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/identity"
)

type Options struct {
	DefaultSystemName string
}

type Service struct {
	repo    Repository
	options Options
}

func NewService(repo Repository, options Options) *Service {
	return &Service{repo: repo, options: options}
}

func (s *Service) Resolve(ctx context.Context, input ResolveInput) (ResolveOutput, error) {
	_ = ctx

	if strings.TrimSpace(input.CWD) == "" {
		return ResolveOutput{}, common.NewError(common.ErrInvalidInput, "cwd is required")
	}

	absCWD, err := filepath.Abs(input.CWD)
	if err != nil {
		return ResolveOutput{}, common.WrapError(common.ErrInvalidInput, "resolve cwd", err)
	}

	repoInfo, err := identity.DiscoverRepository(absCWD)
	if err != nil {
		return ResolveOutput{}, common.WrapError(common.ErrInvalidScope, "discover repository", err)
	}

	workspaceRoot := absCWD
	if repoInfo.Root != "" {
		workspaceRoot = repoInfo.Root
	}
	workspaceRoot = identity.NormalizePath(workspaceRoot)

	branchName := firstNonEmpty(input.BranchName, repoInfo.Branch)
	remote := firstNonEmpty(input.RepoRemote, repoInfo.Remote)
	systemName := firstNonEmpty(s.options.DefaultSystemName, input.SystemNameHint, "codex-mem")
	systemSlug := common.Slug(systemName)

	var (
		projectName      string
		projectKey       string
		remoteNormalized string
		resolvedBy       string
		warnings         []common.Warning
	)

	switch {
	case strings.TrimSpace(remote) != "":
		remoteNormalized = identity.NormalizeRemote(remote)
		projectKey = "remote:" + remoteNormalized
		projectName = firstNonEmpty(input.ProjectNameHint, identity.NameFromRemote(remoteNormalized), filepath.Base(workspaceRoot))
		resolvedBy = "repo_remote"
	case repoInfo.HasGit && repoInfo.Root != "":
		projectKey = "root:" + workspaceRoot
		projectName = firstNonEmpty(input.ProjectNameHint, filepath.Base(workspaceRoot))
		resolvedBy = "canonical_root"
		warnings = append(warnings,
			common.Warning{Code: common.WarnScopeFallback, Message: "repository remote unavailable; used repository root fallback"},
			common.Warning{Code: common.WarnScopeAmbiguous, Message: "scope identity resolved without a canonical remote"},
		)
	default:
		projectKey = "local:" + workspaceRoot
		projectName = firstNonEmpty(input.ProjectNameHint, filepath.Base(workspaceRoot))
		resolvedBy = "local_directory"
		warnings = append(warnings,
			common.Warning{Code: common.WarnScopeFallback, Message: "local directory fallback was used for project identity"},
			common.Warning{Code: common.WarnScopeAmbiguous, Message: "scope identity is weak because no repository metadata was found"},
		)
	}

	systemRecord, err := s.repo.EnsureSystem(SystemRecord{
		ID:   common.StableID("sys", systemSlug),
		Name: systemName,
		Slug: systemSlug,
	})
	if err != nil {
		return ResolveOutput{}, common.EnsureCoded(err, common.ErrWriteFailed, "ensure system scope")
	}

	projectRecord, err := s.repo.EnsureProject(ProjectRecord{
		ID:               common.StableID("proj", systemRecord.ID+":"+projectKey),
		SystemID:         systemRecord.ID,
		Name:             projectName,
		Slug:             common.Slug(projectName),
		CanonicalKey:     projectKey,
		RemoteNormalized: remoteNormalized,
	})
	if err != nil {
		return ResolveOutput{}, common.EnsureCoded(err, common.ErrWriteFailed, "ensure project scope")
	}

	workspaceRecord, err := s.repo.EnsureWorkspace(WorkspaceRecord{
		ID:           common.StableID("ws", projectRecord.ID+":"+workspaceRoot),
		ProjectID:    projectRecord.ID,
		RootPath:     workspaceRoot,
		WorkspaceKey: workspaceRoot,
		BranchName:   branchName,
	})
	if err != nil {
		return ResolveOutput{}, common.EnsureCoded(err, common.ErrWriteFailed, "ensure workspace scope")
	}

	scope := Scope{
		SystemID:      systemRecord.ID,
		SystemName:    systemRecord.Name,
		ProjectID:     projectRecord.ID,
		ProjectName:   projectRecord.Name,
		WorkspaceID:   workspaceRecord.ID,
		WorkspaceRoot: workspaceRecord.RootPath,
		BranchName:    workspaceRecord.BranchName,
		ResolvedBy:    resolvedBy,
	}
	if err := scope.Validate(); err != nil {
		return ResolveOutput{}, err
	}

	return ResolveOutput{
		Scope:      scope,
		ResolvedBy: resolvedBy,
		Warnings:   warnings,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
