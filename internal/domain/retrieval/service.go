package retrieval

import (
	"context"
	"strings"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

const defaultBootstrapNotes = 5

type ScopeResolver interface {
	Resolve(ctx context.Context, input scope.ResolveInput) (scope.ResolveOutput, error)
}

type SessionStarter interface {
	Start(ctx context.Context, input session.StartInput) (session.StartOutput, error)
}

type MemoryReader interface {
	ListRecentByWorkspace(workspaceID string, limit int, minImportance int) ([]memory.Note, error)
	ListRecentByProject(projectID string, excludeWorkspaceID string, limit int, minImportance int) ([]memory.Note, error)
}

type HandoffReader interface {
	FindLatestOpenInWorkspace(workspaceID string) (*handoff.Handoff, error)
	FindLatestOpenInProject(projectID string, excludeWorkspaceID string) (*handoff.Handoff, error)
}

type Service struct {
	scopeResolver  ScopeResolver
	sessionStarter SessionStarter
	memoryReader   MemoryReader
	handoffReader  HandoffReader
}

func NewService(scopeResolver ScopeResolver, sessionStarter SessionStarter, memoryReader MemoryReader, handoffReader HandoffReader) *Service {
	return &Service{
		scopeResolver:  scopeResolver,
		sessionStarter: sessionStarter,
		memoryReader:   memoryReader,
		handoffReader:  handoffReader,
	}
}

type BootstrapInput struct {
	CWD                    string
	Task                   string
	BranchName             string
	RepoRemote             string
	IncludeRelatedProjects bool
	RelatedReason          string
	MaxNotes               int
	MaxHandoffs            int
}

type StartupBrief struct {
	CurrentTask        string   `json:"current_task,omitempty"`
	LastKnownState     string   `json:"last_known_state,omitempty"`
	ImportantDecisions []string `json:"important_decisions,omitempty"`
	OpenTodos          []string `json:"open_todos,omitempty"`
	Risks              []string `json:"risks,omitempty"`
	TouchedFiles       []string `json:"touched_files,omitempty"`
	RelatedContext     []string `json:"related_context,omitempty"`
}

type BootstrapOutput struct {
	Scope         scope.Scope      `json:"scope"`
	Session       session.Session  `json:"session"`
	LatestHandoff *handoff.Handoff `json:"latest_handoff"`
	RecentNotes   []memory.Note    `json:"recent_notes"`
	RelatedNotes  []memory.Note    `json:"related_notes"`
	StartupBrief  StartupBrief     `json:"startup_brief"`
	Warnings      []common.Warning `json:"warnings"`
}

func (s *Service) BootstrapSession(ctx context.Context, input BootstrapInput) (BootstrapOutput, error) {
	resolveOutput, err := s.scopeResolver.Resolve(ctx, scope.ResolveInput{
		CWD:        input.CWD,
		BranchName: input.BranchName,
		RepoRemote: input.RepoRemote,
	})
	if err != nil {
		return BootstrapOutput{}, err
	}

	startOutput, err := s.sessionStarter.Start(ctx, session.StartInput{
		Scope:      resolveOutput.Scope,
		Task:       input.Task,
		BranchName: input.BranchName,
	})
	if err != nil {
		return BootstrapOutput{}, err
	}

	warnings := append([]common.Warning{}, resolveOutput.Warnings...)
	warnings = append(warnings, startOutput.Warnings...)

	latestHandoff, err := s.loadLatestHandoff(resolveOutput.Scope)
	if err != nil {
		return BootstrapOutput{}, err
	}
	if latestHandoff == nil {
		warnings = append(warnings, common.Warning{
			Code:    common.WarnNoPriorHandoff,
			Message: "no open handoff was found for the current workspace or project",
		})
	} else if latestHandoff.Kind == handoff.KindRecovery {
		warnings = append(warnings, common.Warning{
			Code:    common.WarnRecoveryHandoffUsed,
			Message: "bootstrap fell back to a recovery-generated handoff",
		})
	}

	notes, err := s.loadRecentNotes(resolveOutput.Scope, input.MaxNotes)
	if err != nil {
		return BootstrapOutput{}, err
	}
	if len(notes) == 0 {
		warnings = append(warnings, common.Warning{
			Code:    common.WarnNoPriorNotes,
			Message: "no recent high-value notes were found for the current scope",
		})
	}
	if input.IncludeRelatedProjects {
		warnings = append(warnings, common.Warning{
			Code:    common.WarnRelatedProjectsSkipped,
			Message: "related-project retrieval is not implemented yet and was skipped",
		})
	}

	return BootstrapOutput{
		Scope:         resolveOutput.Scope,
		Session:       startOutput.Session,
		LatestHandoff: latestHandoff,
		RecentNotes:   notes,
		RelatedNotes:  nil,
		StartupBrief:  synthesizeStartupBrief(strings.TrimSpace(input.Task), latestHandoff, notes),
		Warnings:      warnings,
	}, nil
}

func (s *Service) loadLatestHandoff(currentScope scope.Scope) (*handoff.Handoff, error) {
	workspaceHandoff, err := s.handoffReader.FindLatestOpenInWorkspace(currentScope.WorkspaceID)
	if err != nil {
		return nil, err
	}
	if workspaceHandoff != nil {
		return workspaceHandoff, nil
	}
	return s.handoffReader.FindLatestOpenInProject(currentScope.ProjectID, currentScope.WorkspaceID)
}

func (s *Service) loadRecentNotes(currentScope scope.Scope, limit int) ([]memory.Note, error) {
	if limit <= 0 {
		limit = defaultBootstrapNotes
	}

	notes, err := s.memoryReader.ListRecentByWorkspace(currentScope.WorkspaceID, limit, 3)
	if err != nil {
		return nil, err
	}
	if len(notes) >= limit {
		return notes, nil
	}

	projectNotes, err := s.memoryReader.ListRecentByProject(currentScope.ProjectID, currentScope.WorkspaceID, limit-len(notes), 3)
	if err != nil {
		return nil, err
	}
	return append(notes, projectNotes...), nil
}

func synthesizeStartupBrief(task string, latestHandoff *handoff.Handoff, notes []memory.Note) StartupBrief {
	brief := StartupBrief{
		CurrentTask: task,
	}
	if latestHandoff != nil {
		if brief.CurrentTask == "" {
			brief.CurrentTask = latestHandoff.Task
		}
		brief.LastKnownState = latestHandoff.Summary
		brief.OpenTodos = appendUnique(brief.OpenTodos, latestHandoff.NextSteps...)
		brief.Risks = appendUnique(brief.Risks, latestHandoff.Risks...)
		brief.TouchedFiles = appendUnique(brief.TouchedFiles, latestHandoff.FilesTouched...)
	}

	for _, note := range notes {
		switch note.Type {
		case memory.NoteTypeDecision:
			brief.ImportantDecisions = appendUnique(brief.ImportantDecisions, note.Content)
		case memory.NoteTypeTodo:
			brief.OpenTodos = appendUnique(brief.OpenTodos, note.Title)
		}
		brief.TouchedFiles = appendUnique(brief.TouchedFiles, note.FilePaths...)
		if brief.LastKnownState == "" {
			brief.LastKnownState = note.Content
		}
	}

	return brief
}

func appendUnique(existing []string, values ...string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		existing = append(existing, trimmed)
	}
	return existing
}
