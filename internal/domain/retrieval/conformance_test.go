package retrieval

import (
	"context"
	"testing"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

func TestConformanceC01EmptyStoreBootstrap(t *testing.T) {
	currentScope := scope.Scope{
		SystemID:      "sys_1",
		SystemName:    "codex-mem",
		ProjectID:     "proj_1",
		ProjectName:   "codex-mem",
		WorkspaceID:   "ws_1",
		WorkspaceRoot: "d:/code/go/codex-mem",
		ResolvedBy:    "repo_remote",
	}
	service := NewService(
		&fakeScopeResolver{output: scope.ResolveOutput{Scope: currentScope}},
		&fakeSessionStarter{output: session.StartOutput{
			Session: session.Session{
				ID:     "sess_1",
				Scope:  currentScope.Ref(),
				Status: session.StatusActive,
			},
		}},
		&fakeMemoryReader{},
		&fakeHandoffReader{},
	)

	result, err := service.BootstrapSession(context.Background(), BootstrapInput{
		CWD:      currentScope.WorkspaceRoot,
		MaxNotes: 5,
	})
	if err != nil {
		t.Fatalf("BootstrapSession: %v", err)
	}
	if result.Session.ID == "" {
		t.Fatal("expected bootstrap to create a session")
	}
	if result.LatestHandoff != nil {
		t.Fatalf("expected no handoff in empty store, got %+v", result.LatestHandoff)
	}
	if len(result.RecentNotes) != 0 || len(result.RelatedNotes) != 0 {
		t.Fatalf("expected empty note sets, got notes=%d related=%d", len(result.RecentNotes), len(result.RelatedNotes))
	}
	if got, want := warningCodes(result.Warnings), []string{
		common.WarnNoPriorHandoff,
		common.WarnNoPriorNotes,
	}; !sameStrings(got, want) {
		t.Fatalf("warning codes mismatch: got %v want %v", got, want)
	}
	if result.StartupBrief.LastKnownState != "" || len(result.StartupBrief.OpenTodos) != 0 {
		t.Fatalf("expected minimal startup brief, got %+v", result.StartupBrief)
	}
}

func TestConformanceC02SameWorkspaceRecovery(t *testing.T) {
	currentScope := scope.Scope{
		SystemID:      "sys_1",
		SystemName:    "codex-mem",
		ProjectID:     "proj_1",
		ProjectName:   "codex-mem",
		WorkspaceID:   "ws_1",
		WorkspaceRoot: "d:/code/go/codex-mem",
		ResolvedBy:    "repo_remote",
	}
	now := time.Now().UTC()
	service := NewService(
		&fakeScopeResolver{output: scope.ResolveOutput{Scope: currentScope}},
		&fakeSessionStarter{output: session.StartOutput{
			Session: session.Session{ID: "sess_1", Scope: currentScope.Ref(), Status: session.StatusActive},
		}},
		&fakeMemoryReader{},
		&fakeHandoffReader{
			workspaceHandoff: &handoff.Handoff{
				ID:        "handoff_ws",
				Scope:     currentScope.Ref(),
				SessionID: "sess_prev",
				Kind:      handoff.KindFinal,
				Task:      "Current task",
				Summary:   "Workspace handoff should win.",
				NextSteps: []string{"Continue"},
				Status:    handoff.StatusOpen,
				CreatedAt: now,
			},
			projectHandoff: &handoff.Handoff{
				ID:        "handoff_proj",
				Scope:     scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_other"},
				SessionID: "sess_prev_2",
				Kind:      handoff.KindFinal,
				Task:      "Project task",
				Summary:   "Project fallback.",
				NextSteps: []string{"Continue"},
				Status:    handoff.StatusOpen,
				CreatedAt: now.Add(-time.Minute),
			},
		},
	)

	result, err := service.BootstrapSession(context.Background(), BootstrapInput{CWD: currentScope.WorkspaceRoot})
	if err != nil {
		t.Fatalf("BootstrapSession: %v", err)
	}
	if result.LatestHandoff == nil || result.LatestHandoff.ID != "handoff_ws" {
		t.Fatalf("expected workspace handoff preference, got %+v", result.LatestHandoff)
	}
}

func TestConformanceC03ProjectFallbackRecovery(t *testing.T) {
	currentScope := scope.Scope{
		SystemID:      "sys_1",
		SystemName:    "codex-mem",
		ProjectID:     "proj_1",
		ProjectName:   "codex-mem",
		WorkspaceID:   "ws_1",
		WorkspaceRoot: "d:/code/go/codex-mem",
		ResolvedBy:    "repo_remote",
	}
	service := NewService(
		&fakeScopeResolver{output: scope.ResolveOutput{Scope: currentScope}},
		&fakeSessionStarter{output: session.StartOutput{
			Session: session.Session{ID: "sess_1", Scope: currentScope.Ref(), Status: session.StatusActive},
		}},
		&fakeMemoryReader{},
		&fakeHandoffReader{
			projectHandoff: &handoff.Handoff{
				ID:        "handoff_proj",
				Scope:     scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_other"},
				SessionID: "sess_prev",
				Kind:      handoff.KindFinal,
				Task:      "Project fallback task",
				Summary:   "Project handoff should be used.",
				NextSteps: []string{"Continue"},
				Status:    handoff.StatusOpen,
			},
		},
	)

	result, err := service.BootstrapSession(context.Background(), BootstrapInput{CWD: currentScope.WorkspaceRoot})
	if err != nil {
		t.Fatalf("BootstrapSession: %v", err)
	}
	if result.LatestHandoff == nil || result.LatestHandoff.ID != "handoff_proj" {
		t.Fatalf("expected project fallback handoff, got %+v", result.LatestHandoff)
	}
}

func TestConformanceC06RelatedProjectExpansionAndProvenance(t *testing.T) {
	currentRef := scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"}
	now := time.Now().UTC()
	service := NewService(
		&fakeScopeResolver{},
		&fakeSessionStarter{},
		&fakeMemoryReader{
			relatedIDs: []string{"proj_2"},
			relatedNotes: []memory.Note{
				{
					ID:         "note_related",
					Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_2", WorkspaceID: "ws_2"},
					Title:      "Shared API contract",
					Content:    "Backend contract changed.",
					Importance: 5,
					Status:     memory.StatusActive,
					Source:     memory.SourceRelayImport,
					CreatedAt:  now,
				},
			},
		},
		&fakeHandoffReader{},
	)

	result, err := service.Search(context.Background(), SearchInput{
		Query:                  "API contract",
		Scope:                  currentRef,
		IncludeRelatedProjects: true,
		Limit:                  5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if got, want := len(result.Results), 1; got != want {
		t.Fatalf("result count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Results[0].RelationType, relatedProjectRelationType; got != want {
		t.Fatalf("relation type mismatch: got %q want %q", got, want)
	}
	if got, want := result.Results[0].Source, string(memory.SourceRelayImport); got != want {
		t.Fatalf("source mismatch: got %q want %q", got, want)
	}
}

func TestConformanceWarningVisibilityOnBootstrap(t *testing.T) {
	currentScope := scope.Scope{
		SystemID:      "sys_1",
		SystemName:    "codex-mem",
		ProjectID:     "proj_1",
		ProjectName:   "codex-mem",
		WorkspaceID:   "ws_1",
		WorkspaceRoot: "d:/code/go/codex-mem",
		ResolvedBy:    "local_directory",
	}
	service := NewService(
		&fakeScopeResolver{output: scope.ResolveOutput{
			Scope: currentScope,
			Warnings: []common.Warning{
				{Code: common.WarnScopeFallback, Message: "scope fallback used"},
				{Code: common.WarnScopeAmbiguous, Message: "scope ambiguous"},
			},
		}},
		&fakeSessionStarter{output: session.StartOutput{
			Session: session.Session{ID: "sess_1", Scope: currentScope.Ref(), Status: session.StatusActive},
		}},
		&fakeMemoryReader{},
		&fakeHandoffReader{},
	)

	result, err := service.BootstrapSession(context.Background(), BootstrapInput{CWD: currentScope.WorkspaceRoot})
	if err != nil {
		t.Fatalf("BootstrapSession: %v", err)
	}
	if got, want := warningCodes(result.Warnings), []string{
		common.WarnScopeFallback,
		common.WarnScopeAmbiguous,
		common.WarnNoPriorHandoff,
		common.WarnNoPriorNotes,
	}; !sameStrings(got, want) {
		t.Fatalf("warning codes mismatch: got %v want %v", got, want)
	}
}
