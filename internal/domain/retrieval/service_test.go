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

const workspaceHandoffID = "handoff_ws"

type fakeScopeResolver struct {
	output scope.ResolveOutput
}

func (f *fakeScopeResolver) Resolve(_ context.Context, _ scope.ResolveInput) (scope.ResolveOutput, error) {
	return f.output, nil
}

type fakeSessionStarter struct {
	output session.StartOutput
}

func (f *fakeSessionStarter) Start(_ context.Context, _ session.StartInput) (session.StartOutput, error) {
	return f.output, nil
}

type fakeMemoryReader struct {
	workspaceNotes []memory.Note
	projectNotes   []memory.Note
	notesByID      map[string]memory.Note
	searchNotes    []memory.Note
	relatedNotes   []memory.Note
	relatedIDs     []string
}

func (f *fakeMemoryReader) ListRecentByWorkspace(_ string, limit int, _ int) ([]memory.Note, error) {
	return takeNotes(f.workspaceNotes, limit), nil
}

func (f *fakeMemoryReader) ListRecentByProject(_ string, _ string, limit int, _ int) ([]memory.Note, error) {
	return takeNotes(f.projectNotes, limit), nil
}

func (f *fakeMemoryReader) GetByID(id string) (*memory.Note, error) {
	record, ok := f.notesByID[id]
	if !ok {
		return nil, nil
	}
	return &record, nil
}

func (f *fakeMemoryReader) Search(_ scope.Ref, _ string, limit int, _ int, _ []memory.NoteType, _ []memory.Status) ([]memory.Note, error) {
	return takeNotes(f.searchNotes, limit), nil
}

func (f *fakeMemoryReader) ListRecentByProjects(_ string, _ []string, limit int, _ int) ([]memory.Note, error) {
	return takeNotes(f.relatedNotes, limit), nil
}

func (f *fakeMemoryReader) ListRelatedProjectIDs(_ string, _ int) ([]string, error) {
	return f.relatedIDs, nil
}

func (f *fakeMemoryReader) SearchProjects(_ string, _ []string, _ string, limit int, _ int, _ []memory.NoteType, _ []memory.Status) ([]memory.Note, error) {
	return takeNotes(f.relatedNotes, limit), nil
}

type fakeHandoffReader struct {
	workspaceHandoff *handoff.Handoff
	projectHandoff   *handoff.Handoff
	workspaceRecent  []handoff.Handoff
	projectRecent    []handoff.Handoff
	handoffsByID     map[string]handoff.Handoff
	searchHandoffs   []handoff.Handoff
}

func (f *fakeHandoffReader) FindLatestOpenInWorkspace(_ string) (*handoff.Handoff, error) {
	return f.workspaceHandoff, nil
}

func (f *fakeHandoffReader) FindLatestOpenInProject(_ string, _ string) (*handoff.Handoff, error) {
	return f.projectHandoff, nil
}

func (f *fakeHandoffReader) ListRecentByWorkspace(_ string, limit int) ([]handoff.Handoff, error) {
	return takeHandoffs(f.workspaceRecent, limit), nil
}

func (f *fakeHandoffReader) ListRecentByProject(_ string, _ string, limit int) ([]handoff.Handoff, error) {
	return takeHandoffs(f.projectRecent, limit), nil
}

func (f *fakeHandoffReader) GetByID(id string) (*handoff.Handoff, error) {
	record, ok := f.handoffsByID[id]
	if !ok {
		return nil, nil
	}
	return &record, nil
}

func (f *fakeHandoffReader) Search(_ scope.Ref, _ string, limit int, _ []handoff.Status) ([]handoff.Handoff, error) {
	return takeHandoffs(f.searchHandoffs, limit), nil
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
				ID:           workspaceHandoffID,
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

	if result.LatestHandoff == nil || result.LatestHandoff.ID != workspaceHandoffID {
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
	if got, want := warningCodes(result.Warnings), []string{
		common.WarnRecoveryHandoffUsed,
		common.WarnNoPriorNotes,
		common.WarnRelatedProjectsSkipped,
	}; !sameStrings(got, want) {
		t.Fatalf("warning codes mismatch: got %v want %v", got, want)
	}
}

func TestBootstrapSessionLoadsRelatedNotesWhenEnabled(t *testing.T) {
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
		&fakeMemoryReader{
			relatedIDs: []string{"proj_2"},
			relatedNotes: []memory.Note{
				{ID: "note_related", Scope: scope.Ref{SystemID: "sys_1", ProjectID: "proj_2", WorkspaceID: "ws_2"}, Title: "related"},
			},
		},
		&fakeHandoffReader{},
	)

	result, err := service.BootstrapSession(context.Background(), BootstrapInput{
		CWD:                    "d:/code/go/codex-mem",
		IncludeRelatedProjects: true,
		MaxNotes:               3,
	})
	if err != nil {
		t.Fatalf("BootstrapSession: %v", err)
	}
	if got, want := len(result.RelatedNotes), 1; got != want {
		t.Fatalf("related note count mismatch: got %d want %d", got, want)
	}
	if got, want := result.RelatedNotes[0].RelationType, relatedProjectRelationType; got != want {
		t.Fatalf("relation type mismatch: got %q want %q", got, want)
	}
}

func TestGetRecentReturnsWorkspaceThenProjectRecords(t *testing.T) {
	currentRef := scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"}
	service := NewService(
		&fakeScopeResolver{},
		&fakeSessionStarter{},
		&fakeMemoryReader{
			workspaceNotes: []memory.Note{{ID: "note_ws", Scope: currentRef, Importance: 4}},
			projectNotes:   []memory.Note{{ID: "note_proj", Scope: scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_2"}, Importance: 4}},
		},
		&fakeHandoffReader{
			workspaceRecent: []handoff.Handoff{{ID: workspaceHandoffID, Scope: currentRef}},
			projectRecent:   []handoff.Handoff{{ID: "handoff_proj", Scope: scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_2"}}},
		},
	)

	result, err := service.GetRecent(context.Background(), GetRecentInput{
		Scope: currentRef,
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("GetRecent: %v", err)
	}

	if got, want := len(result.Notes), 2; got != want {
		t.Fatalf("note count mismatch: got %d want %d", got, want)
	}
	if got, want := len(result.Handoffs), 2; got != want {
		t.Fatalf("handoff count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Notes[0].ID, "note_ws"; got != want {
		t.Fatalf("workspace note ordering mismatch: got %q want %q", got, want)
	}
	if got, want := result.Handoffs[0].ID, workspaceHandoffID; got != want {
		t.Fatalf("workspace handoff ordering mismatch: got %q want %q", got, want)
	}
}

func TestGetRecentSupportsIncludeFlagsAndRelatedWarning(t *testing.T) {
	currentRef := scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"}
	service := NewService(
		&fakeScopeResolver{},
		&fakeSessionStarter{},
		&fakeMemoryReader{workspaceNotes: []memory.Note{{ID: "note_ws", Scope: currentRef, Importance: 4}}},
		&fakeHandoffReader{workspaceRecent: []handoff.Handoff{{ID: workspaceHandoffID, Scope: currentRef}}},
	)

	result, err := service.GetRecent(context.Background(), GetRecentInput{
		Scope:                  currentRef,
		Limit:                  1,
		IncludeNotes:           true,
		IncludeRelatedProjects: true,
	})
	if err != nil {
		t.Fatalf("GetRecent: %v", err)
	}

	if got, want := len(result.Notes), 1; got != want {
		t.Fatalf("note count mismatch: got %d want %d", got, want)
	}
	if len(result.Handoffs) != 0 {
		t.Fatalf("expected handoffs to be omitted, got %d", len(result.Handoffs))
	}
	if got, want := len(result.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Warnings[0].Code, common.WarnRelatedProjectsSkipped; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
}

func TestGetRecentLoadsRelatedNotesWhenEnabled(t *testing.T) {
	currentRef := scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"}
	service := NewService(
		&fakeScopeResolver{},
		&fakeSessionStarter{},
		&fakeMemoryReader{
			relatedIDs: []string{"proj_2"},
			relatedNotes: []memory.Note{
				{ID: "note_related", Scope: scope.Ref{SystemID: "sys_1", ProjectID: "proj_2", WorkspaceID: "ws_2"}},
			},
		},
		&fakeHandoffReader{},
	)

	result, err := service.GetRecent(context.Background(), GetRecentInput{
		Scope:                  currentRef,
		Limit:                  2,
		IncludeNotes:           true,
		IncludeRelatedProjects: true,
	})
	if err != nil {
		t.Fatalf("GetRecent: %v", err)
	}
	if got, want := len(result.Notes), 1; got != want {
		t.Fatalf("related note count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Notes[0].RelationType, relatedProjectRelationType; got != want {
		t.Fatalf("relation type mismatch: got %q want %q", got, want)
	}
}

func TestGetRecordLoadsNoteAndHandoffByKind(t *testing.T) {
	service := NewService(
		&fakeScopeResolver{},
		&fakeSessionStarter{},
		&fakeMemoryReader{
			notesByID: map[string]memory.Note{
				"note_1": {ID: "note_1", Title: "note"},
			},
		},
		&fakeHandoffReader{
			handoffsByID: map[string]handoff.Handoff{
				"handoff_1": {ID: "handoff_1", Task: "task"},
			},
		},
	)

	noteResult, err := service.GetRecord(context.Background(), GetRecordInput{ID: "note_1", Kind: RecordKindNote})
	if err != nil {
		t.Fatalf("GetRecord note: %v", err)
	}
	if note, ok := noteResult.Record.(memory.Note); !ok || note.ID != "note_1" {
		t.Fatalf("unexpected note record: %#v", noteResult.Record)
	}

	handoffResult, err := service.GetRecord(context.Background(), GetRecordInput{ID: "handoff_1", Kind: RecordKindHandoff})
	if err != nil {
		t.Fatalf("GetRecord handoff: %v", err)
	}
	if record, ok := handoffResult.Record.(handoff.Handoff); !ok || record.ID != "handoff_1" {
		t.Fatalf("unexpected handoff record: %#v", handoffResult.Record)
	}
}

func TestGetRecordReturnsNotFound(t *testing.T) {
	service := NewService(&fakeScopeResolver{}, &fakeSessionStarter{}, &fakeMemoryReader{}, &fakeHandoffReader{})

	_, err := service.GetRecord(context.Background(), GetRecordInput{ID: "missing", Kind: RecordKindNote})
	if err == nil {
		t.Fatal("expected missing record to fail")
	}
}

func TestSearchRanksWorkspaceResultsFirstAndSupportsHandoffs(t *testing.T) {
	currentRef := scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"}
	now := time.Now().UTC()
	service := NewService(
		&fakeScopeResolver{},
		&fakeSessionStarter{},
		&fakeMemoryReader{
			searchNotes: []memory.Note{
				{
					ID:         "note_proj",
					Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_2"},
					Type:       memory.NoteTypeBugfix,
					Title:      "Project payment validation",
					Content:    "Project-level validation fallback.",
					Importance: 5,
					Status:     memory.StatusActive,
					Source:     memory.SourceCodexExplicit,
					CreatedAt:  now.Add(-2 * time.Hour),
				},
				{
					ID:         "note_ws",
					Scope:      currentRef,
					Type:       memory.NoteTypeBugfix,
					Title:      "Workspace payment validation",
					Content:    "Workspace validation bugfix is ready.",
					Importance: 4,
					Status:     memory.StatusActive,
					Source:     memory.SourceCodexExplicit,
					CreatedAt:  now.Add(-4 * time.Hour),
				},
			},
		},
		&fakeHandoffReader{
			searchHandoffs: []handoff.Handoff{
				{
					ID:        workspaceHandoffID,
					Scope:     currentRef,
					Kind:      handoff.KindFinal,
					Task:      "Payment validation follow-up",
					Summary:   "Need to rerun checkout regression.",
					Status:    handoff.StatusOpen,
					CreatedAt: now.Add(-time.Hour),
				},
			},
		},
	)

	result, err := service.Search(context.Background(), SearchInput{
		Query:           "payment validation",
		Scope:           currentRef,
		Limit:           3,
		IncludeHandoffs: true,
		Intent:          "bugfix",
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if got, want := len(result.Results), 3; got != want {
		t.Fatalf("result count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Results[0].ID, "note_ws"; got != want {
		t.Fatalf("expected workspace note first, got %q", got)
	}
}

func TestSearchPrefersExplicitNotesOverImportedArtifacts(t *testing.T) {
	currentRef := scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"}
	now := time.Now().UTC()
	service := NewService(
		&fakeScopeResolver{},
		&fakeSessionStarter{},
		&fakeMemoryReader{
			searchNotes: []memory.Note{
				{
					ID:         "note_imported",
					Scope:      currentRef,
					Type:       memory.NoteTypeDecision,
					Title:      "Shared validation rule",
					Content:    "Use metadata-backed validation.",
					Importance: 4,
					Status:     memory.StatusActive,
					Source:     memory.SourceWatcherImport,
					CreatedAt:  now,
				},
				{
					ID:         "note_explicit",
					Scope:      currentRef,
					Type:       memory.NoteTypeDecision,
					Title:      "Shared validation rule",
					Content:    "Use metadata-backed validation.",
					Importance: 4,
					Status:     memory.StatusActive,
					Source:     memory.SourceCodexExplicit,
					CreatedAt:  now.Add(-time.Minute),
				},
			},
		},
		&fakeHandoffReader{},
	)

	result, err := service.Search(context.Background(), SearchInput{
		Query: "validation rule",
		Scope: currentRef,
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if got, want := len(result.Results), 1; got != want {
		t.Fatalf("result count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Results[0].ID, "note_explicit"; got != want {
		t.Fatalf("expected explicit note to win dedupe, got %q", got)
	}
}

func TestSearchReturnsZeroResultsWithoutError(t *testing.T) {
	currentRef := scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"}
	service := NewService(&fakeScopeResolver{}, &fakeSessionStarter{}, &fakeMemoryReader{}, &fakeHandoffReader{})

	result, err := service.Search(context.Background(), SearchInput{
		Query: "missing",
		Scope: currentRef,
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(result.Results) != 0 {
		t.Fatalf("expected zero results, got %d", len(result.Results))
	}
}

func TestSearchWarnsWhenRelatedProjectsRequested(t *testing.T) {
	currentRef := scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"}
	service := NewService(&fakeScopeResolver{}, &fakeSessionStarter{}, &fakeMemoryReader{}, &fakeHandoffReader{})

	result, err := service.Search(context.Background(), SearchInput{
		Query:                  "validation",
		Scope:                  currentRef,
		IncludeRelatedProjects: true,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if got, want := len(result.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Warnings[0].Code, common.WarnRelatedProjectsSkipped; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
}

func TestSearchWarnsWhenResultsAreDeduplicated(t *testing.T) {
	currentRef := scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"}
	now := time.Now().UTC()
	service := NewService(
		&fakeScopeResolver{},
		&fakeSessionStarter{},
		&fakeMemoryReader{
			searchNotes: []memory.Note{
				{
					ID:         "note_ws",
					Scope:      currentRef,
					Type:       memory.NoteTypeDecision,
					Title:      "Shared validation rule",
					Content:    "Workspace-specific wording.",
					Importance: 5,
					Status:     memory.StatusActive,
					Source:     memory.SourceCodexExplicit,
					CreatedAt:  now,
				},
				{
					ID:         "note_proj",
					Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_2"},
					Type:       memory.NoteTypeDecision,
					Title:      "Shared validation rule",
					Content:    "Project duplicate wording.",
					Importance: 4,
					Status:     memory.StatusActive,
					Source:     memory.SourceCodexExplicit,
					CreatedAt:  now.Add(-time.Hour),
				},
			},
		},
		&fakeHandoffReader{},
	)

	result, err := service.Search(context.Background(), SearchInput{
		Query: "validation rule",
		Scope: currentRef,
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if got, want := len(result.Results), 1; got != want {
		t.Fatalf("result count mismatch: got %d want %d", got, want)
	}
	if got, want := len(result.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Warnings[0].Code, common.WarnDedupeApplied; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
}

func TestSearchIncludesRelatedNotesWhenEnabled(t *testing.T) {
	currentRef := scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"}
	service := NewService(
		&fakeScopeResolver{},
		&fakeSessionStarter{},
		&fakeMemoryReader{
			relatedIDs: []string{"proj_2"},
			relatedNotes: []memory.Note{
				{
					ID:         "note_related",
					Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_2", WorkspaceID: "ws_2"},
					Title:      "Shared validation contract",
					Content:    "Backend payment validation contract.",
					Importance: 4,
					Status:     memory.StatusActive,
					Source:     memory.SourceCodexExplicit,
					CreatedAt:  time.Now().UTC(),
				},
			},
		},
		&fakeHandoffReader{},
	)

	result, err := service.Search(context.Background(), SearchInput{
		Query:                  "validation contract",
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
}

func takeNotes(notes []memory.Note, limit int) []memory.Note {
	if limit <= 0 || len(notes) <= limit {
		return notes
	}
	return notes[:limit]
}

func takeHandoffs(handoffs []handoff.Handoff, limit int) []handoff.Handoff {
	if limit <= 0 || len(handoffs) <= limit {
		return handoffs
	}
	return handoffs[:limit]
}

func warningCodes(warnings []common.Warning) []string {
	codes := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		codes = append(codes, warning.Code)
	}
	return codes
}

func sameStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
