package db

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

func TestOpenRunsFoundationMigrationsAndPersistsScopeChain(t *testing.T) {
	ctx := context.Background()
	handle, err := Open(ctx, Options{
		Path:        filepath.Join(t.TempDir(), "codex-mem.db"),
		DriverName:  "sqlite",
		BusyTimeout: 2 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer handle.Close()

	scopeRepo := NewScopeRepository(handle, common.RealClock{})
	sessionRepo := NewSessionRepository(handle)
	systemRecord, err := scopeRepo.EnsureSystem(scope.SystemRecord{
		ID:   "sys_test",
		Name: "codex-mem",
		Slug: "codex-mem",
	})
	if err != nil {
		t.Fatalf("EnsureSystem: %v", err)
	}
	projectRecord, err := scopeRepo.EnsureProject(scope.ProjectRecord{
		ID:               "proj_test",
		SystemID:         systemRecord.ID,
		Name:             "codex-mem",
		Slug:             "codex-mem",
		CanonicalKey:     "remote:github.com/example/codex-mem",
		RemoteNormalized: "github.com/example/codex-mem",
	})
	if err != nil {
		t.Fatalf("EnsureProject: %v", err)
	}
	workspaceRecord, err := scopeRepo.EnsureWorkspace(scope.WorkspaceRecord{
		ID:           "ws_test",
		ProjectID:    projectRecord.ID,
		RootPath:     "d:/code/go/codex-mem",
		WorkspaceKey: "d:/code/go/codex-mem",
		BranchName:   "main",
	})
	if err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}

	err = sessionRepo.Create(session.Session{
		ID: "sess_test",
		Scope: scope.Ref{
			SystemID:    systemRecord.ID,
			ProjectID:   projectRecord.ID,
			WorkspaceID: workspaceRecord.ID,
		},
		Status:     session.StatusActive,
		Task:       "foundation",
		BranchName: "main",
		StartedAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}
}

func TestSessionCreateRejectsInconsistentScopeChain(t *testing.T) {
	ctx := context.Background()
	handle, err := Open(ctx, Options{
		Path:        filepath.Join(t.TempDir(), "codex-mem.db"),
		DriverName:  "sqlite",
		BusyTimeout: 2 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer handle.Close()

	scopeRepo := NewScopeRepository(handle, common.RealClock{})
	sessionRepo := NewSessionRepository(handle)
	systemA, err := scopeRepo.EnsureSystem(scope.SystemRecord{ID: "sys_a", Name: "a", Slug: "a"})
	if err != nil {
		t.Fatalf("EnsureSystem A: %v", err)
	}
	systemB, err := scopeRepo.EnsureSystem(scope.SystemRecord{ID: "sys_b", Name: "b", Slug: "b"})
	if err != nil {
		t.Fatalf("EnsureSystem B: %v", err)
	}
	projectA, err := scopeRepo.EnsureProject(scope.ProjectRecord{
		ID:           "proj_a",
		SystemID:     systemA.ID,
		Name:         "a",
		Slug:         "a",
		CanonicalKey: "remote:a",
	})
	if err != nil {
		t.Fatalf("EnsureProject A: %v", err)
	}
	projectB, err := scopeRepo.EnsureProject(scope.ProjectRecord{
		ID:           "proj_b",
		SystemID:     systemB.ID,
		Name:         "b",
		Slug:         "b",
		CanonicalKey: "remote:b",
	})
	if err != nil {
		t.Fatalf("EnsureProject B: %v", err)
	}
	workspaceA, err := scopeRepo.EnsureWorkspace(scope.WorkspaceRecord{
		ID:           "ws_a",
		ProjectID:    projectA.ID,
		RootPath:     "d:/code/go/a",
		WorkspaceKey: "d:/code/go/a",
	})
	if err != nil {
		t.Fatalf("EnsureWorkspace A: %v", err)
	}

	err = sessionRepo.Create(session.Session{
		ID: "sess_bad",
		Scope: scope.Ref{
			SystemID:    systemB.ID,
			ProjectID:   projectB.ID,
			WorkspaceID: workspaceA.ID,
		},
		Status:    session.StatusActive,
		StartedAt: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected inconsistent scope chain to fail")
	}
}

func TestMemoryAndHandoffRepositoriesPersistStructuredRecords(t *testing.T) {
	ctx := context.Background()
	handle, err := Open(ctx, Options{
		Path:        filepath.Join(t.TempDir(), "codex-mem.db"),
		DriverName:  "sqlite",
		BusyTimeout: 2 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer handle.Close()

	ref, sessionID := seedScopeAndSession(t, handle)
	memoryRepo := NewMemoryRepository(handle)
	handoffRepo := NewHandoffRepository(handle)
	now := time.Date(2026, 3, 13, 14, 0, 0, 0, time.UTC)

	note := memory.Note{
		ID:         "note_test",
		Scope:      ref,
		SessionID:  sessionID,
		Type:       memory.NoteTypeBugfix,
		Title:      "Validation now uses generated metadata",
		Content:    "Client validation no longer hard-codes enum aliases.",
		Importance: 4,
		Tags:       []string{"validation", "frontend"},
		FilePaths:  []string{"src/order/validation.ts"},
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := memoryRepo.Create(note); err != nil {
		t.Fatalf("Create note: %v", err)
	}

	duplicate, err := memoryRepo.FindDuplicate(note)
	if err != nil {
		t.Fatalf("FindDuplicate: %v", err)
	}
	if duplicate == nil || duplicate.ID != note.ID {
		t.Fatalf("expected duplicate note %q, got %+v", note.ID, duplicate)
	}

	record := handoff.Handoff{
		ID:           "handoff_test",
		Scope:        ref,
		SessionID:    sessionID,
		Kind:         handoff.KindFinal,
		Task:         "Finish validation cleanup",
		Summary:      "Main validation flow is aligned with backend metadata.",
		NextSteps:    []string{"Retest draft checkout"},
		FilesTouched: []string{"src/order/validation.ts"},
		Status:       handoff.StatusOpen,
		CreatedAt:    now.Add(time.Minute),
		UpdatedAt:    now.Add(time.Minute),
	}
	if err := handoffRepo.Create(record); err != nil {
		t.Fatalf("Create handoff: %v", err)
	}

	latest, err := handoffRepo.FindLatestOpenByTask(ref, record.Task)
	if err != nil {
		t.Fatalf("FindLatestOpenByTask: %v", err)
	}
	if latest == nil || latest.ID != record.ID {
		t.Fatalf("expected handoff %q, got %+v", record.ID, latest)
	}
}

func TestMemoryAndHandoffRepositoriesRejectSessionScopeMismatch(t *testing.T) {
	ctx := context.Background()
	handle, err := Open(ctx, Options{
		Path:        filepath.Join(t.TempDir(), "codex-mem.db"),
		DriverName:  "sqlite",
		BusyTimeout: 2 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer handle.Close()

	_, sessionID := seedScopeAndSession(t, handle)
	otherRef := seedAlternateScope(t, handle)
	memoryRepo := NewMemoryRepository(handle)
	handoffRepo := NewHandoffRepository(handle)
	now := time.Date(2026, 3, 13, 14, 10, 0, 0, time.UTC)

	err = memoryRepo.Create(memory.Note{
		ID:         "note_bad",
		Scope:      otherRef,
		SessionID:  sessionID,
		Type:       memory.NoteTypeDecision,
		Title:      "Mismatched scope",
		Content:    "This should fail.",
		Importance: 3,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err == nil {
		t.Fatal("expected note create to fail for session scope mismatch")
	}

	err = handoffRepo.Create(handoff.Handoff{
		ID:        "handoff_bad",
		Scope:     otherRef,
		SessionID: sessionID,
		Kind:      handoff.KindFinal,
		Task:      "Mismatched scope",
		Summary:   "This should fail.",
		NextSteps: []string{"Do not store"},
		Status:    handoff.StatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err == nil {
		t.Fatal("expected handoff create to fail for session scope mismatch")
	}
}

func TestMemoryAndHandoffRepositoriesSupportRecentAndByIDReads(t *testing.T) {
	ctx := context.Background()
	handle, err := Open(ctx, Options{
		Path:        filepath.Join(t.TempDir(), "codex-mem.db"),
		DriverName:  "sqlite",
		BusyTimeout: 2 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer handle.Close()

	ref, sessionID := seedScopeAndSession(t, handle)
	otherRef := seedSameProjectScope(t, handle)
	memoryRepo := NewMemoryRepository(handle)
	handoffRepo := NewHandoffRepository(handle)
	now := time.Date(2026, 3, 13, 15, 0, 0, 0, time.UTC)

	workspaceNote := memory.Note{
		ID:         "note_ws",
		Scope:      ref,
		SessionID:  sessionID,
		Type:       memory.NoteTypeDecision,
		Title:      "Workspace note",
		Content:    "Current workspace context.",
		Importance: 4,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	projectNote := memory.Note{
		ID:         "note_proj",
		Scope:      otherRef,
		SessionID:  seedSessionForScope(t, handle, "sess_other", otherRef),
		Type:       memory.NoteTypeBugfix,
		Title:      "Project note",
		Content:    "Sibling workspace context.",
		Importance: 3,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now.Add(time.Minute),
		UpdatedAt:  now.Add(time.Minute),
	}
	if err := memoryRepo.Create(workspaceNote); err != nil {
		t.Fatalf("Create workspace note: %v", err)
	}
	if err := memoryRepo.Create(projectNote); err != nil {
		t.Fatalf("Create project note: %v", err)
	}

	workspaceHandoff := handoff.Handoff{
		ID:        "handoff_ws",
		Scope:     ref,
		SessionID: sessionID,
		Kind:      handoff.KindCheckpoint,
		Task:      "Workspace task",
		Summary:   "workspace handoff",
		NextSteps: []string{"keep going"},
		Status:    handoff.StatusOpen,
		CreatedAt: now.Add(2 * time.Minute),
		UpdatedAt: now.Add(2 * time.Minute),
	}
	projectHandoff := handoff.Handoff{
		ID:        "handoff_proj",
		Scope:     otherRef,
		SessionID: seedSessionForScope(t, handle, "sess_other_2", otherRef),
		Kind:      handoff.KindFinal,
		Task:      "Project task",
		Summary:   "project handoff",
		NextSteps: []string{"review"},
		Status:    handoff.StatusCompleted,
		CreatedAt: now.Add(3 * time.Minute),
		UpdatedAt: now.Add(3 * time.Minute),
	}
	if err := handoffRepo.Create(workspaceHandoff); err != nil {
		t.Fatalf("Create workspace handoff: %v", err)
	}
	if err := handoffRepo.Create(projectHandoff); err != nil {
		t.Fatalf("Create project handoff: %v", err)
	}

	notes, err := memoryRepo.ListRecentByProject(ref.ProjectID, ref.WorkspaceID, 5, 1)
	if err != nil {
		t.Fatalf("ListRecentByProject notes: %v", err)
	}
	if got, want := len(notes), 1; got != want || notes[0].ID != "note_proj" {
		t.Fatalf("unexpected project notes: %+v", notes)
	}

	handoffs, err := handoffRepo.ListRecentByProject(ref.ProjectID, ref.WorkspaceID, 5)
	if err != nil {
		t.Fatalf("ListRecentByProject handoffs: %v", err)
	}
	if got, want := len(handoffs), 1; got != want || handoffs[0].ID != "handoff_proj" {
		t.Fatalf("unexpected project handoffs: %+v", handoffs)
	}

	loadedNote, err := memoryRepo.GetByID("note_ws")
	if err != nil || loadedNote == nil || loadedNote.ID != "note_ws" {
		t.Fatalf("GetByID note failed: note=%+v err=%v", loadedNote, err)
	}
	loadedHandoff, err := handoffRepo.GetByID("handoff_ws")
	if err != nil || loadedHandoff == nil || loadedHandoff.ID != "handoff_ws" {
		t.Fatalf("GetByID handoff failed: handoff=%+v err=%v", loadedHandoff, err)
	}
}

func TestMemorySearchUsesFTSAndProjectIsolation(t *testing.T) {
	ctx := context.Background()
	handle, err := Open(ctx, Options{
		Path:        filepath.Join(t.TempDir(), "codex-mem.db"),
		DriverName:  "sqlite",
		BusyTimeout: 2 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer handle.Close()

	ref, sessionID := seedScopeAndSession(t, handle)
	otherProject := seedSameSystemProjectScope(t, handle)
	memoryRepo := NewMemoryRepository(handle)
	handoffRepo := NewHandoffRepository(handle)
	now := time.Date(2026, 3, 13, 16, 0, 0, 0, time.UTC)

	if err := memoryRepo.Create(memory.Note{
		ID:         "note_search_hit",
		Scope:      ref,
		SessionID:  sessionID,
		Type:       memory.NoteTypeBugfix,
		Title:      "Payment validation fix",
		Content:    "Validation now uses canonical payment metadata.",
		Importance: 4,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("Create search note: %v", err)
	}
	if err := memoryRepo.Create(memory.Note{
		ID:         "note_search_miss",
		Scope:      otherProject,
		SessionID:  seedSessionForScope(t, handle, "sess_alt", otherProject),
		Type:       memory.NoteTypeBugfix,
		Title:      "Payment validation external",
		Content:    "Other project should not appear.",
		Importance: 5,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now.Add(time.Minute),
		UpdatedAt:  now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("Create isolated note: %v", err)
	}

	if err := handoffRepo.Create(handoff.Handoff{
		ID:        "handoff_search_hit",
		Scope:     ref,
		SessionID: sessionID,
		Kind:      handoff.KindFinal,
		Task:      "Payment validation follow-up",
		Summary:   "Need to verify checkout payment flow.",
		NextSteps: []string{"Run checkout regression"},
		Status:    handoff.StatusOpen,
		CreatedAt: now.Add(2 * time.Minute),
		UpdatedAt: now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("Create search handoff: %v", err)
	}

	notes, err := memoryRepo.Search(ref, "payment validation", 5, 1, nil, nil)
	if err != nil {
		t.Fatalf("Search notes: %v", err)
	}
	if got, want := len(notes), 1; got != want {
		t.Fatalf("note search count mismatch: got %d want %d", got, want)
	}
	if got, want := notes[0].ID, "note_search_hit"; got != want {
		t.Fatalf("note search mismatch: got %q want %q", got, want)
	}

	handoffs, err := handoffRepo.Search(ref, "payment validation", 5, nil)
	if err != nil {
		t.Fatalf("Search handoffs: %v", err)
	}
	if got, want := len(handoffs), 1; got != want {
		t.Fatalf("handoff search count mismatch: got %d want %d", got, want)
	}
	if got, want := handoffs[0].ID, "handoff_search_hit"; got != want {
		t.Fatalf("handoff search mismatch: got %q want %q", got, want)
	}
}

func TestMemoryRepositorySupportsRelatedProjectQueries(t *testing.T) {
	ctx := context.Background()
	handle, err := Open(ctx, Options{
		Path:        filepath.Join(t.TempDir(), "codex-mem.db"),
		DriverName:  "sqlite",
		BusyTimeout: 2 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer handle.Close()

	ref, sessionID := seedScopeAndSession(t, handle)
	otherProject := seedSameSystemProjectScope(t, handle)
	memoryRepo := NewMemoryRepository(handle)
	now := time.Date(2026, 3, 13, 16, 20, 0, 0, time.UTC)

	if err := memoryRepo.Create(memory.Note{
		ID:                "note_source",
		Scope:             ref,
		SessionID:         sessionID,
		Type:              memory.NoteTypeDecision,
		Title:             "Current project references other project",
		Content:           "Use related project context.",
		Importance:        4,
		RelatedProjectIDs: []string{otherProject.ProjectID},
		Status:            memory.StatusActive,
		Source:            memory.SourceCodexExplicit,
		CreatedAt:         now,
		UpdatedAt:         now,
	}); err != nil {
		t.Fatalf("Create source note: %v", err)
	}
	if err := memoryRepo.Create(memory.Note{
		ID:         "note_related",
		Scope:      otherProject,
		SessionID:  seedSessionForScope(t, handle, "sess_related", otherProject),
		Type:       memory.NoteTypeBugfix,
		Title:      "Related project payment change",
		Content:    "Payment validation contract changed.",
		Importance: 5,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now.Add(time.Minute),
		UpdatedAt:  now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("Create related note: %v", err)
	}

	projectIDs, err := memoryRepo.ListRelatedProjectIDs(ref.ProjectID, 5)
	if err != nil {
		t.Fatalf("ListRelatedProjectIDs: %v", err)
	}
	if got, want := len(projectIDs), 1; got != want || projectIDs[0] != otherProject.ProjectID {
		t.Fatalf("unexpected related project ids: %+v", projectIDs)
	}

	relatedNotes, err := memoryRepo.ListRecentByProjects(ref.SystemID, projectIDs, 5, 1)
	if err != nil {
		t.Fatalf("ListRecentByProjects: %v", err)
	}
	if got, want := len(relatedNotes), 1; got != want || relatedNotes[0].ID != "note_related" {
		t.Fatalf("unexpected related notes: %+v", relatedNotes)
	}

	searchNotes, err := memoryRepo.SearchProjects(ref.SystemID, projectIDs, "payment change", 5, 1, nil, nil)
	if err != nil {
		t.Fatalf("SearchProjects: %v", err)
	}
	if got, want := len(searchNotes), 1; got != want || searchNotes[0].ID != "note_related" {
		t.Fatalf("unexpected related search notes: %+v", searchNotes)
	}
}

func TestSearchabilityControlsExcludeRecordsFromRecentAndSearchButNotGetByID(t *testing.T) {
	ctx := context.Background()
	handle, err := Open(ctx, Options{
		Path:        filepath.Join(t.TempDir(), "codex-mem.db"),
		DriverName:  "sqlite",
		BusyTimeout: 2 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer handle.Close()

	ref, sessionID := seedScopeAndSession(t, handle)
	memoryRepo := NewMemoryRepository(handle)
	handoffRepo := NewHandoffRepository(handle)
	now := time.Date(2026, 3, 13, 16, 40, 0, 0, time.UTC)

	if err := memoryRepo.Create(memory.Note{
		ID:              "note_hidden",
		Scope:           ref,
		SessionID:       sessionID,
		Type:            memory.NoteTypeBugfix,
		Title:           "Hidden validation note",
		Content:         "This should stay out of search.",
		Importance:      4,
		Status:          memory.StatusActive,
		Source:          memory.SourceCodexExplicit,
		Searchable:      false,
		ExclusionReason: "archived",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("Create hidden note: %v", err)
	}
	if err := handoffRepo.Create(handoff.Handoff{
		ID:              "handoff_hidden",
		Scope:           ref,
		SessionID:       sessionID,
		Kind:            handoff.KindFinal,
		Task:            "Hidden task",
		Summary:         "This should stay out of search.",
		NextSteps:       []string{"Do not surface"},
		Status:          handoff.StatusOpen,
		Searchable:      false,
		ExclusionReason: "archived",
		CreatedAt:       now.Add(time.Minute),
		UpdatedAt:       now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("Create hidden handoff: %v", err)
	}

	notes, err := memoryRepo.ListRecentByWorkspace(ref.WorkspaceID, 5, 1)
	if err != nil {
		t.Fatalf("ListRecentByWorkspace notes: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("expected hidden note to be excluded from recent list, got %+v", notes)
	}

	handoffs, err := handoffRepo.ListRecentByWorkspace(ref.WorkspaceID, 5)
	if err != nil {
		t.Fatalf("ListRecentByWorkspace handoffs: %v", err)
	}
	if len(handoffs) != 0 {
		t.Fatalf("expected hidden handoff to be excluded from recent list, got %+v", handoffs)
	}

	noteSearch, err := memoryRepo.Search(ref, "hidden validation", 5, 1, nil, nil)
	if err != nil {
		t.Fatalf("Search notes: %v", err)
	}
	if len(noteSearch) != 0 {
		t.Fatalf("expected hidden note to be excluded from search, got %+v", noteSearch)
	}

	handoffSearch, err := handoffRepo.Search(ref, "hidden task", 5, nil)
	if err != nil {
		t.Fatalf("Search handoffs: %v", err)
	}
	if len(handoffSearch) != 0 {
		t.Fatalf("expected hidden handoff to be excluded from search, got %+v", handoffSearch)
	}

	hiddenNote, err := memoryRepo.GetByID("note_hidden")
	if err != nil || hiddenNote == nil || hiddenNote.ID != "note_hidden" || hiddenNote.Searchable {
		t.Fatalf("expected GetByID to return hidden note, got note=%+v err=%v", hiddenNote, err)
	}
	hiddenHandoff, err := handoffRepo.GetByID("handoff_hidden")
	if err != nil || hiddenHandoff == nil || hiddenHandoff.ID != "handoff_hidden" || hiddenHandoff.Searchable {
		t.Fatalf("expected GetByID to return hidden handoff, got handoff=%+v err=%v", hiddenHandoff, err)
	}
}

func seedScopeAndSession(t *testing.T, handle *sql.DB) (scope.Ref, string) {
	t.Helper()

	scopeRepo := NewScopeRepository(handle, common.RealClock{})
	sessionRepo := NewSessionRepository(handle)
	systemRecord, err := scopeRepo.EnsureSystem(scope.SystemRecord{
		ID:   "sys_test",
		Name: "codex-mem",
		Slug: "codex-mem",
	})
	if err != nil {
		t.Fatalf("EnsureSystem: %v", err)
	}
	projectRecord, err := scopeRepo.EnsureProject(scope.ProjectRecord{
		ID:               "proj_test",
		SystemID:         systemRecord.ID,
		Name:             "codex-mem",
		Slug:             "codex-mem",
		CanonicalKey:     "remote:github.com/example/codex-mem",
		RemoteNormalized: "github.com/example/codex-mem",
	})
	if err != nil {
		t.Fatalf("EnsureProject: %v", err)
	}
	workspaceRecord, err := scopeRepo.EnsureWorkspace(scope.WorkspaceRecord{
		ID:           "ws_test",
		ProjectID:    projectRecord.ID,
		RootPath:     "d:/code/go/codex-mem",
		WorkspaceKey: "d:/code/go/codex-mem",
		BranchName:   "main",
	})
	if err != nil {
		t.Fatalf("EnsureWorkspace: %v", err)
	}

	ref := scope.Ref{
		SystemID:    systemRecord.ID,
		ProjectID:   projectRecord.ID,
		WorkspaceID: workspaceRecord.ID,
	}
	sessionID := "sess_test"
	err = sessionRepo.Create(session.Session{
		ID:         sessionID,
		Scope:      ref,
		Status:     session.StatusActive,
		Task:       "foundation",
		BranchName: "main",
		StartedAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}

	return ref, sessionID
}

func seedAlternateScope(t *testing.T, handle *sql.DB) scope.Ref {
	t.Helper()

	scopeRepo := NewScopeRepository(handle, common.RealClock{})
	systemRecord, err := scopeRepo.EnsureSystem(scope.SystemRecord{
		ID:   "sys_other",
		Name: "other",
		Slug: "other",
	})
	if err != nil {
		t.Fatalf("EnsureSystem other: %v", err)
	}
	projectRecord, err := scopeRepo.EnsureProject(scope.ProjectRecord{
		ID:           "proj_other",
		SystemID:     systemRecord.ID,
		Name:         "other",
		Slug:         "other",
		CanonicalKey: "remote:github.com/example/other",
	})
	if err != nil {
		t.Fatalf("EnsureProject other: %v", err)
	}
	workspaceRecord, err := scopeRepo.EnsureWorkspace(scope.WorkspaceRecord{
		ID:           "ws_other",
		ProjectID:    projectRecord.ID,
		RootPath:     "d:/code/go/other",
		WorkspaceKey: "d:/code/go/other",
	})
	if err != nil {
		t.Fatalf("EnsureWorkspace other: %v", err)
	}

	return scope.Ref{
		SystemID:    systemRecord.ID,
		ProjectID:   projectRecord.ID,
		WorkspaceID: workspaceRecord.ID,
	}
}

func seedSameProjectScope(t *testing.T, handle *sql.DB) scope.Ref {
	t.Helper()

	scopeRepo := NewScopeRepository(handle, common.RealClock{})
	workspaceRecord, err := scopeRepo.EnsureWorkspace(scope.WorkspaceRecord{
		ID:           "ws_test_other",
		ProjectID:    "proj_test",
		RootPath:     "d:/code/go/codex-mem-other",
		WorkspaceKey: "d:/code/go/codex-mem-other",
		BranchName:   "feature",
	})
	if err != nil {
		t.Fatalf("EnsureWorkspace same-project: %v", err)
	}

	return scope.Ref{
		SystemID:    "sys_test",
		ProjectID:   "proj_test",
		WorkspaceID: workspaceRecord.ID,
	}
}

func seedSameSystemProjectScope(t *testing.T, handle *sql.DB) scope.Ref {
	t.Helper()

	scopeRepo := NewScopeRepository(handle, common.RealClock{})
	projectRecord, err := scopeRepo.EnsureProject(scope.ProjectRecord{
		ID:           "proj_related",
		SystemID:     "sys_test",
		Name:         "codex-mem-related",
		Slug:         "codex-mem-related",
		CanonicalKey: "remote:github.com/example/codex-mem-related",
	})
	if err != nil {
		t.Fatalf("EnsureProject same-system: %v", err)
	}
	workspaceRecord, err := scopeRepo.EnsureWorkspace(scope.WorkspaceRecord{
		ID:           "ws_related",
		ProjectID:    projectRecord.ID,
		RootPath:     "d:/code/go/codex-mem-related",
		WorkspaceKey: "d:/code/go/codex-mem-related",
		BranchName:   "main",
	})
	if err != nil {
		t.Fatalf("EnsureWorkspace same-system: %v", err)
	}

	return scope.Ref{
		SystemID:    "sys_test",
		ProjectID:   projectRecord.ID,
		WorkspaceID: workspaceRecord.ID,
	}
}

func seedSessionForScope(t *testing.T, handle *sql.DB, sessionID string, ref scope.Ref) string {
	t.Helper()

	sessionRepo := NewSessionRepository(handle)
	err := sessionRepo.Create(session.Session{
		ID:        sessionID,
		Scope:     ref,
		Status:    session.StatusActive,
		StartedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("Create session for scope: %v", err)
	}
	return sessionID
}
