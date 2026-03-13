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
