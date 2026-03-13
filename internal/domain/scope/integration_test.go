package scope_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"codex-mem/internal/db"
	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
)

func TestMigrationProjectRenamePreservesProjectID(t *testing.T) {
	ctx := context.Background()
	handle, err := db.Open(ctx, db.Options{
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

	scopeRepo := db.NewScopeRepository(handle, common.RealClock{})
	service := scope.NewService(scopeRepo, scope.Options{DefaultSystemName: "codex-mem"})
	cwd := t.TempDir()

	first, err := service.Resolve(context.Background(), scope.ResolveInput{
		CWD:             cwd,
		RepoRemote:      "git@github.com:example/order-ui.git",
		ProjectNameHint: "order-ui",
	})
	if err != nil {
		t.Fatalf("Resolve first: %v", err)
	}
	second, err := service.Resolve(context.Background(), scope.ResolveInput{
		CWD:             cwd,
		RepoRemote:      "git@github.com:example/order-ui.git",
		ProjectNameHint: "order-web",
	})
	if err != nil {
		t.Fatalf("Resolve second: %v", err)
	}

	if got, want := second.Scope.ProjectID, first.Scope.ProjectID; got != want {
		t.Fatalf("project id should be preserved across rename: got %q want %q", got, want)
	}
	if got, want := second.Scope.ProjectName, "order-web"; got != want {
		t.Fatalf("project name should update: got %q want %q", got, want)
	}
}

func TestMigrationRemoteURLFormChangePreservesProjectID(t *testing.T) {
	ctx := context.Background()
	handle, err := db.Open(ctx, db.Options{
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

	scopeRepo := db.NewScopeRepository(handle, common.RealClock{})
	service := scope.NewService(scopeRepo, scope.Options{DefaultSystemName: "codex-mem"})
	cwd := t.TempDir()

	first, err := service.Resolve(context.Background(), scope.ResolveInput{
		CWD:        cwd,
		RepoRemote: "git@github.com:example/order-api.git",
	})
	if err != nil {
		t.Fatalf("Resolve first: %v", err)
	}
	second, err := service.Resolve(context.Background(), scope.ResolveInput{
		CWD:        cwd,
		RepoRemote: "https://github.com/example/order-api",
	})
	if err != nil {
		t.Fatalf("Resolve second: %v", err)
	}

	if got, want := second.Scope.ProjectID, first.Scope.ProjectID; got != want {
		t.Fatalf("project id should be preserved across remote normalization: got %q want %q", got, want)
	}
}

