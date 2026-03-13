package agents

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codex-mem/internal/domain/common"
)

func TestInstallSafeCreatesMissingProjectAgents(t *testing.T) {
	root := t.TempDir()
	service := NewService(Options{HomeDir: root})

	result, err := service.Install(context.Background(), InstallInput{
		Target:      TargetProject,
		Mode:        ModeSafe,
		CWD:         root,
		ProjectName: "codex-mem",
		SystemName:  "codex-mem",
		PreferredTags: []string{
			"go", "sqlite", "mcp",
		},
		RelatedRepositories: []string{"repo-a", "repo-b", "repo-c"},
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if got, want := len(result.WrittenFiles), 1; got != want {
		t.Fatalf("written file count mismatch: got %d want %d", got, want)
	}
	if len(result.SkippedFiles) != 0 {
		t.Fatalf("expected no skipped files, got %+v", result.SkippedFiles)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %+v", result.Warnings)
	}

	body, err := os.ReadFile(filepath.Join(root, agentsFileName))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(body)
	if strings.Contains(content, "<project-name>") {
		t.Fatalf("expected project placeholder to be resolved, got %q", content)
	}
	if !strings.Contains(content, "Project name: codex-mem") {
		t.Fatalf("expected rendered project name, got %q", content)
	}
}

func TestInstallSafeSkipsExistingFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, agentsFileName)
	if err := os.WriteFile(path, []byte("existing\n"), 0o644); err != nil {
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
	if got, want := len(result.SkippedFiles), 1; got != want {
		t.Fatalf("skipped file count mismatch: got %d want %d", got, want)
	}
	if got, want := len(result.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Warnings[0].Code, common.WarnExistingAgentsSkipped; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got, want := string(body), "existing\n"; got != want {
		t.Fatalf("safe mode should preserve file: got %q want %q", got, want)
	}
}

func TestInstallAppendAddsManagedBlockOnce(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, agentsFileName)
	if err := os.WriteFile(path, []byte("# Existing\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	service := NewService(Options{HomeDir: root})

	result, err := service.Install(context.Background(), InstallInput{
		Target: TargetProject,
		Mode:   ModeAppend,
		CWD:    root,
	})
	if err != nil {
		t.Fatalf("Install append: %v", err)
	}
	if got, want := len(result.WrittenFiles), 1; got != want {
		t.Fatalf("written file count mismatch: got %d want %d", got, want)
	}

	result, err = service.Install(context.Background(), InstallInput{
		Target: TargetProject,
		Mode:   ModeAppend,
		CWD:    root,
	})
	if err != nil {
		t.Fatalf("Install append repeat: %v", err)
	}
	if got, want := len(result.SkippedFiles), 1; got != want {
		t.Fatalf("skipped file count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Warnings[0].Code, common.WarnExistingAgentsSkipped; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(body)
	if got, want := strings.Count(content, appendBlockStart(TargetProject)), 1; got != want {
		t.Fatalf("append block count mismatch: got %d want %d", got, want)
	}
}

func TestInstallOverwriteReplacesContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, agentsFileName)
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	service := NewService(Options{HomeDir: root})

	allowRelated := false
	result, err := service.Install(context.Background(), InstallInput{
		Target:                    TargetProject,
		Mode:                      ModeOverwrite,
		CWD:                       root,
		ProjectName:               "codex-mem",
		SystemName:                "codex-mem",
		AllowRelatedProjectMemory: &allowRelated,
	})
	if err != nil {
		t.Fatalf("Install overwrite: %v", err)
	}
	if got, want := len(result.WrittenFiles), 1; got != want {
		t.Fatalf("written file count mismatch: got %d want %d", got, want)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(body)
	if strings.Contains(content, "old\n") {
		t.Fatalf("expected overwrite to replace old content, got %q", content)
	}
	if !strings.Contains(content, "Related-project memory is disabled by default") {
		t.Fatalf("expected related-project policy override, got %q", content)
	}
}

func TestInstallBothUsesGlobalAndProjectTargets(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "repo")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	service := NewService(Options{HomeDir: root})

	result, err := service.Install(context.Background(), InstallInput{
		Target: TargetBoth,
		Mode:   ModeSafe,
		CWD:    projectDir,
	})
	if err != nil {
		t.Fatalf("Install both: %v", err)
	}
	if got, want := len(result.WrittenFiles), 2; got != want {
		t.Fatalf("written file count mismatch: got %d want %d", got, want)
	}
	if got, want := len(result.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := result.Warnings[0].Code, common.WarnPlaceholdersUnresolved; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(root, defaultCodexDir, agentsFileName)); err != nil {
		t.Fatalf("expected global agents file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, agentsFileName)); err != nil {
		t.Fatalf("expected project agents file: %v", err)
	}
}
