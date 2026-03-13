package retrieval

import (
	"context"
	"testing"
	"time"

	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

type fakeScopeResolver struct {
	output scope.ResolveOutput
}

func (f *fakeScopeResolver) Resolve(ctx context.Context, input scope.ResolveInput) (scope.ResolveOutput, error) {
	return f.output, nil
}

type fakeSessionStarter struct {
	output session.StartOutput
}

func (f *fakeSessionStarter) Start(ctx context.Context, input session.StartInput) (session.StartOutput, error) {
	return f.output, nil
}

type fakeMemoryReader struct {
	workspaceNotes []memory.Note
	projectNotes   []memory.Note
}

func (f *fakeMemoryReader) ListRecentByWorkspace(workspaceID string, limit int, minImportance int) ([]memory.Note, error) {
	return takeNotes(f.workspaceNotes, limit), nil
}

func (f *fakeMemoryReader) ListRecentByProject(projectID string, excludeWorkspaceID string, limit int, minImportance int) ([]memory.Note, error) {
	return takeNotes(f.projectNotes, limit), nil
}

type fakeHandoffReader struct {
	workspaceHandoff *handoff.Handoff
	projectHandoff   *handoff.Handoff
}

func (f *fakeHandoffReader) FindLatestOpenInWorkspace(workspaceID string) (*handoff.Handoff, error) {
	return f.workspaceHandoff, nil
}

func (f *fakeHandoffReader) FindLatestOpenInProject(projectID string, excludeWorkspaceID string) (*handoff.Handoff, error) {
	return f.projectHandoff, nil
}

func TestBootstrapSessionPrefersWorkspaceHandoffAndBuildsBrief(t *testing.T) {
	currentScope := scope.Scope{
		SystemID:      "sys_1",
		SystemName:    "codex-mem",
		ProjectID:     "proj_1",
		ProjectName:   "codex-mem",
		WorkspaceID:   "ws_1",
		WorkspaceRoot: "d:/code/go/codex-mem",
		BranchName:    "main",
		ResolvedBy:    "repo_remote",
	}
	now := time.Date(2026, 3, 13, 15, 0, 0, 0, time.UTC)
	service := NewService(
		&fakeScopeResolver{output: scope.ResolveOutput{Scope: currentScope}},
		&fakeSessionStarter{output: session.StartOutput{
			Session: session.Session{
				ID:        "sess_1",
				Scope:     currentScope.Ref(),
				Status:    session.StatusActive,
				StartedAt: now,
			},
		}},
		&fakeMemoryReader{
			workspaceNotes: []memory.Note{
				{
					ID:         "note_1",
					Scope:      currentScope.Ref(),
					SessionID:  "sess_prev",
					Type:       memory.NoteTypeDecision,
					Title:      "Follow backend metadata",
					Content:    "Validation should use generated backend metadata.",
					Importance: 4,
					Status:     memory.StatusActive,
					Source:     memory.SourceCodexExplicit,
					CreatedAt:  now.Add(-time.Hour),
				},
			},
		},
		&fakeHandoffReader{
			workspaceHandoff: &handoff.Handoff{
				ID:           "handoff_ws",
				Scope:        currentScope.Ref(),
				SessionID:    "sess_prev",
				Kind:         handoff.KindFinal,
				Task:         "Continue validation work",
				Summary:      "Validation logic is updated but draft checkout still needs retest.",
				NextSteps:    []string{"Retest draft checkout"},
				FilesTouched: []string{"src/order/validation.ts"},
				Status:       handoff.StatusOpen,
				CreatedAt:    now.Add(-30 * time.Minute),
			},
			projectHandoff: &handoff.Handoff{ID: "handoff_proj"},
		},
	)

	result, err := service.BootstrapSession(context.Background(), BootstrapInput{
		CWD:      "d:/code/go/codex-mem",
		Task:     "Continue validation work",
		MaxNotes: 5,
	})
	if err != nil {
		t.Fatalf("BootstrapSession: %v", err)
	}

	if result.LatestHandoff == nil || result.LatestHandoff.ID != "handoff_ws" {
		t.Fatalf("expected workspace handoff, got %+v", result.LatestHandoff)
	}
	if got, want := result.StartupBrief.CurrentTask, "Continue validation work"; got != want {
		t.Fatalf("current task mismatch: got %q want %q", got, want)
	}
	if got, want := len(result.RecentNotes), 1; got != want {
		t.Fatalf("recent notes mismatch: got %d want %d", got, want)
	}
	if got, want := len(result.Warnings), 0; got != want {
		t.Fatalf("warnings mismatch: got %d want %d", got, want)
	}
}

func TestBootstrapSessionFallsBackToProjectHandoffAndWarnsOnEmptyMemory(t *testing.T) {
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
		&fakeHandoffReader{
			projectHandoff: &handoff.Handoff{
				ID:        "handoff_proj",
				Scope:     scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_other"},
				SessionID: "sess_prev",
				Kind:      handoff.KindRecovery,
				Task:      "Recover state",
				Summary:   "Recovered after interruption.",
				NextSteps: []string{"Reopen workspace"},
				Status:    handoff.StatusOpen,
			},
		},
	)

	result, err := service.BootstrapSession(context.Background(), BootstrapInput{
		CWD:                    "d:/code/go/codex-mem",
		IncludeRelatedProjects: true,
		MaxNotes:               2,
	})
	if err != nil {
		t.Fatalf("BootstrapSession: %v", err)
	}

	if result.LatestHandoff == nil || result.LatestHandoff.ID != "handoff_proj" {
		t.Fatalf("expected project handoff fallback, got %+v", result.LatestHandoff)
	}
	if got, want := len(result.Warnings), 3; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
}

func takeNotes(notes []memory.Note, limit int) []memory.Note {
	if limit <= 0 || len(notes) <= limit {
		return notes
	}
	return notes[:limit]
}
