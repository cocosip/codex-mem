package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"codex-mem/internal/domain/common"
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
