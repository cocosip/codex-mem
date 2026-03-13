package agents

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"codex-mem/internal/domain/common"
)

func TestConformanceC11AgentsSafeInstall(t *testing.T) {
	root := t.TempDir()
	projectFile := filepath.Join(root, agentsFileName)
	if err := os.WriteFile(projectFile, []byte("preexisting\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	service := NewService(Options{HomeDir: root})
	result, err := service.Install(context.Background(), InstallInput{
		Target: TargetProject,
		Mode:   ModeSafe,
		CWD:    root,
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(result.WrittenFiles) != 0 {
		t.Fatalf("expected no writes in safe mode, got %+v", result.WrittenFiles)
	}
	if got, want := len(result.SkippedFiles), 1; got != want {
		t.Fatalf("skipped file count mismatch: got %d want %d", got, want)
	}
	if got, want := len(result.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Warnings[0].Code, common.WarnExistingAgentsSkipped; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
	body, err := os.ReadFile(projectFile)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got, want := string(body), "preexisting\n"; got != want {
		t.Fatalf("safe mode should preserve existing file: got %q want %q", got, want)
	}
}
