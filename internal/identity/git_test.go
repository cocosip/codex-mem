package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverRepositoryFromNestedPath(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "internal", "domain")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[remote \"origin\"]\n\turl = git@github.com:Example/Codex-Mem.git\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatalf("write HEAD: %v", err)
	}

	info, err := DiscoverRepository(nested)
	if err != nil {
		t.Fatalf("DiscoverRepository: %v", err)
	}

	if !info.HasGit {
		t.Fatalf("expected HasGit to be true")
	}
	if got, want := info.Root, NormalizePath(root); got != want {
		t.Fatalf("root mismatch: got %q want %q", got, want)
	}
	if got, want := info.Branch, "main"; got != want {
		t.Fatalf("branch mismatch: got %q want %q", got, want)
	}
	if got, want := NormalizeRemote(info.Remote), "github.com/example/codex-mem"; got != want {
		t.Fatalf("remote mismatch: got %q want %q", got, want)
	}
}
