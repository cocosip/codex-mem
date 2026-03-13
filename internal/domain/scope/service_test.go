package scope

import (
	"context"
	"testing"
)

type fakeRepository struct {
	system    SystemRecord
	project   ProjectRecord
	workspace WorkspaceRecord
}

func (f *fakeRepository) EnsureSystem(system SystemRecord) (SystemRecord, error) {
	f.system = system
	return system, nil
}

func (f *fakeRepository) EnsureProject(project ProjectRecord) (ProjectRecord, error) {
	f.project = project
	return project, nil
}

func (f *fakeRepository) EnsureWorkspace(workspace WorkspaceRecord) (WorkspaceRecord, error) {
	f.workspace = workspace
	return workspace, nil
}

func TestResolveUsesRepoRemoteWhenProvided(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, Options{DefaultSystemName: "codex-mem"})

	result, err := service.Resolve(context.Background(), ResolveInput{
		CWD:        t.TempDir(),
		RepoRemote: "git@github.com:Example/Codex-Mem.git",
		BranchName: "feature/foundation",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got, want := result.ResolvedBy, "repo_remote"; got != want {
		t.Fatalf("resolved_by mismatch: got %q want %q", got, want)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if repo.project.RemoteNormalized != "github.com/example/codex-mem" {
		t.Fatalf("unexpected normalized remote: %q", repo.project.RemoteNormalized)
	}
	if result.Scope.BranchName != "feature/foundation" {
		t.Fatalf("unexpected branch: %q", result.Scope.BranchName)
	}
}

func TestResolveFallsBackWithoutRepositoryMetadata(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, Options{DefaultSystemName: "codex-mem"})

	result, err := service.Resolve(context.Background(), ResolveInput{
		CWD: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got, want := result.ResolvedBy, "local_directory"; got != want {
		t.Fatalf("resolved_by mismatch: got %q want %q", got, want)
	}
	if len(result.Warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(result.Warnings))
	}
}
