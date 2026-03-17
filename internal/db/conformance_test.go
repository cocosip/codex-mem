package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
)

func TestConformanceC05CrossProjectIsolation(t *testing.T) {
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
	defer func() {
		_ = handle.Close()
	}()

	ref, sessionID := seedScopeAndSession(t, handle)
	otherProject := seedSameSystemProjectScope(t, handle)
	memoryRepo := NewMemoryRepository(handle)
	now := time.Date(2026, 3, 13, 18, 0, 0, 0, time.UTC)

	if err := memoryRepo.Create(memory.Note{
		ID:         "note_current",
		Scope:      ref,
		SessionID:  sessionID,
		Type:       memory.NoteTypeBugfix,
		Title:      "Current project fix",
		Content:    "Current project note should appear.",
		Importance: 4,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("Create current note: %v", err)
	}
	if err := memoryRepo.Create(memory.Note{
		ID:         "note_other",
		Scope:      otherProject,
		SessionID:  seedSessionForScope(t, handle, "sess_other_project", otherProject),
		Type:       memory.NoteTypeBugfix,
		Title:      "Other project fix",
		Content:    "Other project note should stay isolated.",
		Importance: 5,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now.Add(time.Minute),
		UpdatedAt:  now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("Create isolated note: %v", err)
	}

	results, err := memoryRepo.Search(ref, "project fix", 10, 1, nil, nil)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if got, want := len(results), 1; got != want {
		t.Fatalf("result count mismatch: got %d want %d", got, want)
	}
	if got, want := results[0].ID, "note_current"; got != want {
		t.Fatalf("unexpected search result: got %q want %q", got, want)
	}
}

func TestConformanceC07SaveNoteScopeValidation(t *testing.T) {
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
	defer func() {
		_ = handle.Close()
	}()

	_, sessionID := seedScopeAndSession(t, handle)
	otherRef := seedAlternateScope(t, handle)
	memoryRepo := NewMemoryRepository(handle)
	now := time.Date(2026, 3, 13, 18, 5, 0, 0, time.UTC)

	err = memoryRepo.Create(memory.Note{
		ID:         "note_scope_conflict",
		Scope:      otherRef,
		SessionID:  sessionID,
		Type:       memory.NoteTypeDecision,
		Title:      "Mismatched scope note",
		Content:    "This note should be rejected.",
		Importance: 3,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	if err == nil {
		t.Fatal("expected note create to fail for session scope mismatch")
	}
	if got, want := common.ErrorCode(err), common.ErrInvalidScope; got != want {
		t.Fatalf("error code mismatch: got %q want %q", got, want)
	}
}

func TestConformanceC09PrivacyExclusion(t *testing.T) {
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
	defer func() {
		_ = handle.Close()
	}()

	ref, sessionID := seedScopeAndSession(t, handle)
	memoryRepo := NewMemoryRepository(handle)
	handoffRepo := NewHandoffRepository(handle)
	now := time.Date(2026, 3, 13, 18, 10, 0, 0, time.UTC)

	if err := memoryRepo.Create(memory.Note{
		ID:              "note_private",
		Scope:           ref,
		SessionID:       sessionID,
		Type:            memory.NoteTypeBugfix,
		Title:           "Private note",
		Content:         "Should not be searchable.",
		Importance:      4,
		Status:          memory.StatusActive,
		Source:          memory.SourceCodexExplicit,
		Searchable:      false,
		ExclusionReason: "private",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("Create note: %v", err)
	}
	if err := handoffRepo.Create(handoff.Handoff{
		ID:              "handoff_private",
		Scope:           ref,
		SessionID:       sessionID,
		Kind:            handoff.KindFinal,
		Task:            "Private handoff",
		Summary:         "Should not be searchable.",
		NextSteps:       []string{"Keep hidden"},
		Status:          handoff.StatusOpen,
		Searchable:      false,
		ExclusionReason: "private",
		CreatedAt:       now.Add(time.Minute),
		UpdatedAt:       now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("Create handoff: %v", err)
	}

	recentNotes, err := memoryRepo.ListRecentByWorkspace(ref.WorkspaceID, 5, 1)
	if err != nil {
		t.Fatalf("ListRecentByWorkspace notes: %v", err)
	}
	if len(recentNotes) != 0 {
		t.Fatalf("expected private note exclusion, got %+v", recentNotes)
	}
	searchNotes, err := memoryRepo.Search(ref, "private", 5, 1, nil, nil)
	if err != nil {
		t.Fatalf("Search notes: %v", err)
	}
	if len(searchNotes) != 0 {
		t.Fatalf("expected private note search exclusion, got %+v", searchNotes)
	}
	loadedNote, err := memoryRepo.GetByID("note_private")
	if err != nil || loadedNote == nil || loadedNote.ExclusionReason != "private" {
		t.Fatalf("expected by-id access to excluded note, got note=%+v err=%v", loadedNote, err)
	}

	recentHandoffs, err := handoffRepo.ListRecentByWorkspace(ref.WorkspaceID, 5)
	if err != nil {
		t.Fatalf("ListRecentByWorkspace handoffs: %v", err)
	}
	if len(recentHandoffs) != 0 {
		t.Fatalf("expected private handoff exclusion, got %+v", recentHandoffs)
	}
	searchHandoffs, err := handoffRepo.Search(ref, "private", 5, nil)
	if err != nil {
		t.Fatalf("Search handoffs: %v", err)
	}
	if len(searchHandoffs) != 0 {
		t.Fatalf("expected private handoff search exclusion, got %+v", searchHandoffs)
	}
	loadedHandoff, err := handoffRepo.GetByID("handoff_private")
	if err != nil || loadedHandoff == nil || loadedHandoff.ExclusionReason != "private" {
		t.Fatalf("expected by-id access to excluded handoff, got handoff=%+v err=%v", loadedHandoff, err)
	}
}

func TestConformanceC12IdentityConflictHandling(t *testing.T) {
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
	defer func() {
		_ = handle.Close()
	}()

	scopeRepo := NewScopeRepository(handle, common.RealClock{})
	systemA, err := scopeRepo.EnsureSystem(scope.SystemRecord{ID: "sys_a", Name: "system-a", Slug: "system-a"})
	if err != nil {
		t.Fatalf("EnsureSystem A: %v", err)
	}
	systemB, err := scopeRepo.EnsureSystem(scope.SystemRecord{ID: "sys_b", Name: "system-b", Slug: "system-b"})
	if err != nil {
		t.Fatalf("EnsureSystem B: %v", err)
	}
	projectA, err := scopeRepo.EnsureProject(scope.ProjectRecord{
		ID:           "proj_a",
		SystemID:     systemA.ID,
		Name:         "project-a",
		Slug:         "project-a",
		CanonicalKey: "remote:github.com/example/shared",
	})
	if err != nil {
		t.Fatalf("EnsureProject A: %v", err)
	}
	_, err = scopeRepo.EnsureProject(scope.ProjectRecord{
		ID:           "proj_b",
		SystemID:     systemB.ID,
		Name:         "project-b",
		Slug:         "project-b",
		CanonicalKey: "remote:github.com/example/shared",
	})
	if common.ErrorCode(err) != common.ErrScopeConflict {
		t.Fatalf("expected project scope conflict, got %v", err)
	}

	if _, err := scopeRepo.EnsureWorkspace(scope.WorkspaceRecord{
		ID:           "ws_a",
		ProjectID:    projectA.ID,
		RootPath:     "d:/code/go/shared",
		WorkspaceKey: "d:/code/go/shared",
	}); err != nil {
		t.Fatalf("EnsureWorkspace A: %v", err)
	}
	_, err = scopeRepo.EnsureWorkspace(scope.WorkspaceRecord{
		ID:           "ws_b",
		ProjectID:    "proj_other",
		RootPath:     "d:/code/go/shared",
		WorkspaceKey: "d:/code/go/shared",
	})
	if common.ErrorCode(err) != common.ErrScopeConflict {
		t.Fatalf("expected workspace scope conflict, got %v", err)
	}
}
