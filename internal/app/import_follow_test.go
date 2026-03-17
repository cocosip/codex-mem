package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codex-mem/internal/config"
	"codex-mem/internal/db"
	"codex-mem/internal/domain/common"
	"github.com/fsnotify/fsnotify"
)

const (
	followImportsWatcherSource = "watcher_import"
	followImportsFirstEvent    = `{"external_id":"watcher:1","type":"discovery","title":"First","content":"First event.","importance":4}`
)

func TestParseFollowImportsOptions(t *testing.T) {
	options, err := parseFollowImportsOptions([]string{
		"--source", followImportsWatcherSource,
		"--input", "events.jsonl",
		"--state-file", "events.offset.json",
		"--failed-output", "failed.jsonl",
		"--failed-manifest", "failed.json",
		"--cwd", "D:/Code/go/codex-mem",
		"--branch-name", "feature/import-follow",
		"--repo-remote", "git@github.com:example/codex-mem.git",
		"--task", "follow imports",
		"--poll-interval", "10s",
		"--watch-mode", "notify",
		"--once",
		"--json",
	})
	if err != nil {
		t.Fatalf("parseFollowImportsOptions: %v", err)
	}
	if got, want := string(options.Source), followImportsWatcherSource; got != want {
		t.Fatalf("source mismatch: got %q want %q", got, want)
	}
	if got, want := len(options.InputPaths), 1; got != want {
		t.Fatalf("input count mismatch: got %d want %d", got, want)
	}
	if got, want := options.InputPaths[0], "events.jsonl"; got != want {
		t.Fatalf("input path mismatch: got %q want %q", got, want)
	}
	if got, want := len(options.StatePaths), 1; got != want {
		t.Fatalf("state path count mismatch: got %d want %d", got, want)
	}
	if got, want := options.StatePaths[0], "events.offset.json"; got != want {
		t.Fatalf("state path mismatch: got %q want %q", got, want)
	}
	if got, want := options.PollInterval.String(), "10s"; got != want {
		t.Fatalf("poll interval mismatch: got %q want %q", got, want)
	}
	if got, want := string(options.WatchMode), "notify"; got != want {
		t.Fatalf("watch mode mismatch: got %q want %q", got, want)
	}
	if !options.Once {
		t.Fatal("expected once mode")
	}
	if !options.JSON {
		t.Fatal("expected JSON mode")
	}
}

func TestParseFollowImportsOptionsSupportsMultipleInputs(t *testing.T) {
	options, err := parseFollowImportsOptions([]string{
		"--source", followImportsWatcherSource,
		"--input", "events-a.jsonl",
		"--input", "events-b.jsonl",
		"--state-file", "events-a.offset.json",
		"--state-file", "events-b.offset.json",
	})
	if err != nil {
		t.Fatalf("parseFollowImportsOptions: %v", err)
	}
	if got, want := len(options.InputPaths), 2; got != want {
		t.Fatalf("input count mismatch: got %d want %d", got, want)
	}
	if got, want := options.InputPaths[1], "events-b.jsonl"; got != want {
		t.Fatalf("second input mismatch: got %q want %q", got, want)
	}
	if got, want := len(options.StatePaths), 2; got != want {
		t.Fatalf("state-file count mismatch: got %d want %d", got, want)
	}
}

func TestParseFollowImportsOptionsRejectsMissingInput(t *testing.T) {
	_, err := parseFollowImportsOptions([]string{"--source", followImportsWatcherSource})
	if err == nil {
		t.Fatal("expected missing input error")
	}
	if !strings.Contains(err.Error(), "follow-imports input is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseFollowImportsOptionsRejectsInvalidWatchMode(t *testing.T) {
	_, err := parseFollowImportsOptions([]string{
		"--source", followImportsWatcherSource,
		"--input", "events.jsonl",
		"--watch-mode", "interrupts",
	})
	if err == nil {
		t.Fatal("expected invalid watch mode error")
	}
	if !strings.Contains(err.Error(), `invalid follow-imports watch mode "interrupts"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseFollowImportsOptionsRejectsMismatchedStateFileCount(t *testing.T) {
	_, err := parseFollowImportsOptions([]string{
		"--source", followImportsWatcherSource,
		"--input", "events-a.jsonl",
		"--input", "events-b.jsonl",
		"--state-file", "events.offset.json",
	})
	if err == nil {
		t.Fatal("expected mismatched state-file error")
	}
	if !strings.Contains(err.Error(), "follow-imports state-file count (1) must match input count (2)") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCleanupFollowImportsOptions(t *testing.T) {
	options, err := parseCleanupFollowImportsOptions([]string{
		"--input", "events.jsonl",
		"--state-file", "events.offset.json",
		"--failed-output", "failed.jsonl",
		"--failed-manifest", "failed.json",
		"--include", "*.offset.json,*.0-42.*",
		"--exclude", "*.43-84.*",
		"--retention-profile", "daily",
		"--cwd", "D:/Code/go/codex-mem",
		"--older-than", "2h",
		"--dry-run",
		"--fail-if-matched",
		"--prune-state",
		"--prune-failed-output",
		"--prune-failed-manifest",
		"--prune-stale-follow-health",
		"--json",
	})
	if err != nil {
		t.Fatalf("parseCleanupFollowImportsOptions: %v", err)
	}
	if !options.PruneState || !options.PruneFailedOutput || !options.PruneFailedManifest || !options.PruneStaleFollowHealth {
		t.Fatalf("expected all prune targets to be enabled, got %+v", options)
	}
	if !options.JSON {
		t.Fatal("expected JSON output")
	}
	if !options.DryRun {
		t.Fatal("expected dry-run option")
	}
	if !options.FailIfMatched {
		t.Fatal("expected fail-if-matched option")
	}
	if got, want := options.OlderThan, 2*time.Hour; got != want {
		t.Fatalf("older-than mismatch: got %s want %s", got, want)
	}
	if got, want := options.RetentionProfile, cleanupFollowImportsRetentionProfileDaily; got != want {
		t.Fatalf("retention profile mismatch: got %q want %q", got, want)
	}
	if got, want := len(options.IncludePatterns), 2; got != want {
		t.Fatalf("include pattern count mismatch: got %d want %d", got, want)
	}
	if got, want := len(options.ExcludePatterns), 1; got != want {
		t.Fatalf("exclude pattern count mismatch: got %d want %d", got, want)
	}
	if got, want := len(options.InputPaths), 1; got != want {
		t.Fatalf("input count mismatch: got %d want %d", got, want)
	}
	if got, want := options.StatePaths[0], "events.offset.json"; got != want {
		t.Fatalf("state path mismatch: got %q want %q", got, want)
	}
}

func TestParseCleanupFollowImportsOptionsRejectsMissingTargets(t *testing.T) {
	_, err := parseCleanupFollowImportsOptions(nil)
	if err == nil {
		t.Fatal("expected missing target error")
	}
	if !strings.Contains(err.Error(), "cleanup-follow-imports requires at least one prune target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCleanupFollowImportsOptionsRejectsStateFileCountMismatch(t *testing.T) {
	_, err := parseCleanupFollowImportsOptions([]string{
		"--input", "events-a.jsonl",
		"--input", "events-b.jsonl",
		"--state-file", "events.offset.json",
		"--prune-state",
	})
	if err == nil {
		t.Fatal("expected mismatched state-file error")
	}
	if !strings.Contains(err.Error(), "cleanup-follow-imports state-file count (1) must match input count (2)") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCleanupFollowImportsOptionsRejectsInvalidPattern(t *testing.T) {
	_, err := parseCleanupFollowImportsOptions([]string{
		"--failed-output", "failed.jsonl",
		"--prune-failed-output",
		"--include", "[",
	})
	if err == nil {
		t.Fatal("expected invalid include pattern error")
	}
	if !strings.Contains(err.Error(), `invalid cleanup-follow-imports --include pattern "["`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCleanupFollowImportsOptionsAppliesRetentionProfileDefaults(t *testing.T) {
	options, err := parseCleanupFollowImportsOptions([]string{
		"--failed-output", "failed.jsonl",
		"--prune-failed-output",
		"--retention-profile", "stale",
	})
	if err != nil {
		t.Fatalf("parseCleanupFollowImportsOptions: %v", err)
	}
	if got, want := options.RetentionProfile, cleanupFollowImportsRetentionProfileStale; got != want {
		t.Fatalf("retention profile mismatch: got %q want %q", got, want)
	}
	if got, want := options.OlderThan, time.Hour; got != want {
		t.Fatalf("older-than mismatch: got %s want %s", got, want)
	}
}

func TestParseCleanupFollowImportsOptionsAllowsOlderThanOverrideOnRetentionProfile(t *testing.T) {
	options, err := parseCleanupFollowImportsOptions([]string{
		"--failed-output", "failed.jsonl",
		"--prune-failed-output",
		"--retention-profile", "daily",
		"--older-than", "2h",
	})
	if err != nil {
		t.Fatalf("parseCleanupFollowImportsOptions: %v", err)
	}
	if got, want := options.RetentionProfile, cleanupFollowImportsRetentionProfileDaily; got != want {
		t.Fatalf("retention profile mismatch: got %q want %q", got, want)
	}
	if got, want := options.OlderThan, 2*time.Hour; got != want {
		t.Fatalf("older-than override mismatch: got %s want %s", got, want)
	}
}

func TestParseCleanupFollowImportsOptionsRejectsUnknownRetentionProfile(t *testing.T) {
	_, err := parseCleanupFollowImportsOptions([]string{
		"--failed-output", "failed.jsonl",
		"--prune-failed-output",
		"--retention-profile", "hourly",
	})
	if err == nil {
		t.Fatal("expected invalid retention profile error")
	}
	if !strings.Contains(err.Error(), `"--retention-profile"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAuditFollowImportsOptions(t *testing.T) {
	options, err := parseAuditFollowImportsOptions([]string{
		"--input", "events.jsonl",
		"--state-file", "events.offset.json",
		"--failed-output", "failed.jsonl",
		"--failed-manifest", "failed.json",
		"--include", "*.offset.json,*.0-42.*",
		"--exclude", "*.43-84.*",
		"--retention-profile", "daily",
		"--cwd", "D:/Code/go/codex-mem",
		"--older-than", "2h",
		"--fail-if-matched",
		"--check-state",
		"--check-failed-output",
		"--check-failed-manifest",
		"--check-follow-health",
		"--json",
	})
	if err != nil {
		t.Fatalf("parseAuditFollowImportsOptions: %v", err)
	}
	if !options.CheckState || !options.CheckFailedOutput || !options.CheckFailedManifest || !options.CheckFollowHealth {
		t.Fatalf("expected all audit targets to be enabled, got %+v", options)
	}
	if !options.JSON {
		t.Fatal("expected JSON output")
	}
	if !options.FailIfMatched {
		t.Fatal("expected fail-if-matched option")
	}
	if got, want := options.OlderThan, 2*time.Hour; got != want {
		t.Fatalf("older-than mismatch: got %s want %s", got, want)
	}
	if got, want := options.RetentionProfile, cleanupFollowImportsRetentionProfileDaily; got != want {
		t.Fatalf("retention profile mismatch: got %q want %q", got, want)
	}
	if got, want := len(options.IncludePatterns), 2; got != want {
		t.Fatalf("include pattern count mismatch: got %d want %d", got, want)
	}
	if got, want := len(options.ExcludePatterns), 1; got != want {
		t.Fatalf("exclude pattern count mismatch: got %d want %d", got, want)
	}
}

func TestParseAuditFollowImportsOptionsRejectsMissingTargets(t *testing.T) {
	_, err := parseAuditFollowImportsOptions(nil)
	if err == nil {
		t.Fatal("expected missing target error")
	}
	if !strings.Contains(err.Error(), "audit-follow-imports requires at least one check target") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAuditFollowImportsOptionsRejectsStateFileCountMismatch(t *testing.T) {
	_, err := parseAuditFollowImportsOptions([]string{
		"--input", "events-a.jsonl",
		"--input", "events-b.jsonl",
		"--state-file", "events.offset.json",
		"--check-state",
	})
	if err == nil {
		t.Fatal("expected mismatched state-file error")
	}
	if !strings.Contains(err.Error(), "audit-follow-imports state-file count (1) must match input count (2)") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteAuditFollowImportsExampleFixtures(t *testing.T) {
	tempDir := t.TempDir()

	writtenPaths, err := writeAuditFollowImportsExampleFixtures(tempDir, nil)
	if err != nil {
		t.Fatalf("writeAuditFollowImportsExampleFixtures: %v", err)
	}
	if len(writtenPaths) != len(auditFollowImportsExampleFixtures()) {
		t.Fatalf("unexpected written path count: got %d want %d", len(writtenPaths), len(auditFollowImportsExampleFixtures()))
	}

	for _, fixture := range auditFollowImportsExampleFixtures() {
		body, err := renderAuditFollowImportsExample(fixture.Report, fixture.JSON)
		if err != nil {
			t.Fatalf("renderAuditFollowImportsExample(%s): %v", fixture.Name, err)
		}
		path := filepath.Join(tempDir, fixture.RelativePath)
		written, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		if !bytes.Equal(written, body) {
			t.Fatalf("fixture mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", fixture.Name, string(written), string(body))
		}
	}
}

func TestWriteAuditFollowImportsExampleFixturesSelectsNamedSubset(t *testing.T) {
	tempDir := t.TempDir()

	writtenPaths, err := writeAuditFollowImportsExampleFixtures(tempDir, []string{"filtered-audit-json"})
	if err != nil {
		t.Fatalf("writeAuditFollowImportsExampleFixtures: %v", err)
	}
	if len(writtenPaths) != 1 {
		t.Fatalf("unexpected written path count: got %d want 1", len(writtenPaths))
	}

	selected, err := selectAuditFollowImportsExampleFixtures([]string{"filtered-audit-json"})
	if err != nil {
		t.Fatalf("selectAuditFollowImportsExampleFixtures: %v", err)
	}
	fixture := selected[0]
	body, err := renderAuditFollowImportsExample(fixture.Report, fixture.JSON)
	if err != nil {
		t.Fatalf("renderAuditFollowImportsExample(%s): %v", fixture.Name, err)
	}
	path := filepath.Join(tempDir, fixture.RelativePath)
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if !bytes.Equal(written, body) {
		t.Fatalf("fixture mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", fixture.Name, string(written), string(body))
	}
	if _, err := os.Stat(filepath.Join(tempDir, "audit-follow-imports-daily-audit.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no daily-audit fixture, got err=%v", err)
	}
}

func TestListAuditFollowImportsExamples(t *testing.T) {
	var buffer bytes.Buffer
	if err := listAuditFollowImportsExamples(&buffer); err != nil {
		t.Fatalf("listAuditFollowImportsExamples: %v", err)
	}
	output := buffer.String()
	for _, fragment := range []string{
		"example=daily-audit-text path=testdata\\audit-follow-imports-daily-audit.txt format=text",
		"example=filtered-audit-json path=testdata\\audit-follow-imports-filtered-audit.json format=json",
		"example_count=2",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("list output missing %q:\n%s", fragment, output)
		}
	}
}

func TestWriteCleanupFollowImportsExampleFixtures(t *testing.T) {
	tempDir := t.TempDir()

	writtenPaths, err := writeCleanupFollowImportsExampleFixtures(tempDir, nil)
	if err != nil {
		t.Fatalf("writeCleanupFollowImportsExampleFixtures: %v", err)
	}
	if len(writtenPaths) != len(cleanupFollowImportsExampleFixtures()) {
		t.Fatalf("unexpected written path count: got %d want %d", len(writtenPaths), len(cleanupFollowImportsExampleFixtures()))
	}

	for _, fixture := range cleanupFollowImportsExampleFixtures() {
		body, err := renderCleanupFollowImportsExample(fixture.Report, fixture.JSON)
		if err != nil {
			t.Fatalf("renderCleanupFollowImportsExample(%s): %v", fixture.Name, err)
		}
		path := filepath.Join(tempDir, fixture.RelativePath)
		written, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		if !bytes.Equal(written, body) {
			t.Fatalf("fixture mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", fixture.Name, string(written), string(body))
		}
	}
}

func TestWriteCleanupFollowImportsExampleFixturesSelectsNamedSubset(t *testing.T) {
	tempDir := t.TempDir()

	writtenPaths, err := writeCleanupFollowImportsExampleFixtures(tempDir, []string{"filtered-cleanup-json"})
	if err != nil {
		t.Fatalf("writeCleanupFollowImportsExampleFixtures: %v", err)
	}
	if len(writtenPaths) != 1 {
		t.Fatalf("unexpected written path count: got %d want 1", len(writtenPaths))
	}

	selected, err := selectCleanupFollowImportsExampleFixtures([]string{"filtered-cleanup-json"})
	if err != nil {
		t.Fatalf("selectCleanupFollowImportsExampleFixtures: %v", err)
	}
	fixture := selected[0]
	body, err := renderCleanupFollowImportsExample(fixture.Report, fixture.JSON)
	if err != nil {
		t.Fatalf("renderCleanupFollowImportsExample(%s): %v", fixture.Name, err)
	}
	path := filepath.Join(tempDir, fixture.RelativePath)
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if !bytes.Equal(written, body) {
		t.Fatalf("fixture mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", fixture.Name, string(written), string(body))
	}
	if _, err := os.Stat(filepath.Join(tempDir, "cleanup-follow-imports-daily-dry-run.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no daily-dry-run fixture, got err=%v", err)
	}
}

func TestListCleanupFollowImportsExamples(t *testing.T) {
	var buffer bytes.Buffer
	if err := listCleanupFollowImportsExamples(&buffer); err != nil {
		t.Fatalf("listCleanupFollowImportsExamples: %v", err)
	}
	output := buffer.String()
	for _, fragment := range []string{
		"example=daily-dry-run-text path=testdata\\cleanup-follow-imports-daily-dry-run.txt format=text",
		"example=filtered-cleanup-json path=testdata\\cleanup-follow-imports-filtered-cleanup.json format=json",
		"example_count=2",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("list output missing %q:\n%s", fragment, output)
		}
	}
}

func TestCleanupFollowImportsPrunesDerivedArtifacts(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Meta: config.LoadMetadata{
			LogDir: filepath.Join(root, "logs"),
		},
	}
	inputA := filepath.Join(root, "events-a.jsonl")
	inputB := filepath.Join(root, "events-b.jsonl")
	stateA := inputA + ".offset.json"
	stateB := inputB + ".offset.json"
	failedOutputBase := filepath.Join(root, "failed", "failed.jsonl")
	failedManifestBase := filepath.Join(root, "failed", "failed.json")
	failedOutputA := filepath.Join(root, "failed", "failed.events-a.0-42.jsonl")
	failedOutputB := filepath.Join(root, "failed", "failed.events-b.43-84.jsonl")
	failedManifestA := filepath.Join(root, "failed", "failed.events-a.0-42.json")
	failedManifestB := filepath.Join(root, "failed", "failed.events-b.43-84.json")
	preservedOutputBase := failedOutputBase
	preservedManifestBase := failedManifestBase

	for _, path := range []string{
		inputA,
		inputB,
		stateA,
		stateB,
		failedOutputA,
		failedOutputB,
		failedManifestA,
		failedManifestB,
		preservedOutputBase,
		preservedManifestBase,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}

	report, err := cleanupFollowImports(cfg, cleanupFollowImportsOptions{
		InputPaths:          []string{inputA, inputB},
		FailedOutputPath:    failedOutputBase,
		FailedManifestPath:  failedManifestBase,
		CWD:                 root,
		PruneState:          true,
		PruneFailedOutput:   true,
		PruneFailedManifest: true,
	})
	if err != nil {
		t.Fatalf("cleanupFollowImports: %v", err)
	}

	if got, want := report.StateFiles.Removed, 2; got != want {
		t.Fatalf("state removed mismatch: got %d want %d", got, want)
	}
	if got, want := report.StateFiles.Matched, 2; got != want {
		t.Fatalf("state matched mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedOutputs.Removed, 2; got != want {
		t.Fatalf("failed output removed mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedOutputs.Matched, 2; got != want {
		t.Fatalf("failed output matched mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedManifests.Removed, 2; got != want {
		t.Fatalf("failed manifest removed mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedManifests.Matched, 2; got != want {
		t.Fatalf("failed manifest matched mismatch: got %d want %d", got, want)
	}

	for _, removed := range []string{stateA, stateB, failedOutputA, failedOutputB, failedManifestA, failedManifestB} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, stat err=%v", removed, err)
		}
	}
	for _, preserved := range []string{preservedOutputBase, preservedManifestBase} {
		if _, err := os.Stat(preserved); err != nil {
			t.Fatalf("expected %s to remain, stat err=%v", preserved, err)
		}
	}
}

func TestCleanupFollowImportsDryRunAndOlderThanFilter(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 3, 17, 4, 0, 0, 0, time.UTC)
	cfg := config.Config{
		Meta: config.LoadMetadata{
			LogDir: filepath.Join(root, "logs"),
		},
	}
	inputPath := filepath.Join(root, "events.jsonl")
	statePath := inputPath + ".offset.json"
	failedOutputBase := filepath.Join(root, "failed", "failed.jsonl")
	failedManifestBase := filepath.Join(root, "failed", "failed.json")
	oldFailedOutput := filepath.Join(root, "failed", "failed.0-42.jsonl")
	newFailedOutput := filepath.Join(root, "failed", "failed.43-84.jsonl")
	oldFailedManifest := filepath.Join(root, "failed", "failed.0-42.json")
	newFailedManifest := filepath.Join(root, "failed", "failed.43-84.json")

	for _, path := range []string{inputPath, statePath, oldFailedOutput, newFailedOutput, oldFailedManifest, newFailedManifest} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}
	for _, path := range []string{statePath, oldFailedOutput, oldFailedManifest} {
		oldTime := now.Add(-2 * time.Hour)
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatalf("Chtimes old %s: %v", path, err)
		}
	}
	for _, path := range []string{newFailedOutput, newFailedManifest} {
		newTime := now.Add(-30 * time.Minute)
		if err := os.Chtimes(path, newTime, newTime); err != nil {
			t.Fatalf("Chtimes new %s: %v", path, err)
		}
	}

	staleSnapshot := followImportsHealthSnapshot{
		Status:              "ok",
		UpdatedAt:           now.Add(-2 * time.Minute),
		Source:              "watcher_import",
		InputCount:          1,
		Continuous:          true,
		PollIntervalSeconds: 5,
	}
	if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, staleSnapshot); err != nil {
		t.Fatalf("saveFollowImportsHealthSnapshot: %v", err)
	}

	report, err := cleanupFollowImportsAt(cfg, cleanupFollowImportsOptions{
		InputPaths:             []string{inputPath},
		FailedOutputPath:       failedOutputBase,
		FailedManifestPath:     failedManifestBase,
		CWD:                    root,
		DryRun:                 true,
		OlderThan:              time.Hour,
		PruneState:             true,
		PruneFailedOutput:      true,
		PruneFailedManifest:    true,
		PruneStaleFollowHealth: true,
	}, now)
	if err != nil {
		t.Fatalf("cleanupFollowImportsAt: %v", err)
	}

	if !report.DryRun {
		t.Fatal("expected dry-run report")
	}
	if !report.MatchFound {
		t.Fatalf("expected dry-run match detection, got %+v", report)
	}
	if got, want := report.OlderThanSeconds, int64(3600); got != want {
		t.Fatalf("older-than seconds mismatch: got %d want %d", got, want)
	}
	if got, want := report.StateFiles.Matched, 1; got != want {
		t.Fatalf("state matched mismatch: got %d want %d", got, want)
	}
	if got := report.StateFiles.Removed; got != 0 {
		t.Fatalf("expected no state removals during dry-run, got %d", got)
	}
	if got, want := report.FailedOutputs.Matched, 1; got != want {
		t.Fatalf("failed output matched mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedOutputs.SkippedByAge, 1; got != want {
		t.Fatalf("failed output skipped mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedManifests.Matched, 1; got != want {
		t.Fatalf("failed manifest matched mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedManifests.SkippedByAge, 1; got != want {
		t.Fatalf("failed manifest skipped mismatch: got %d want %d", got, want)
	}
	if !report.FollowHealth.WouldPrune || report.FollowHealth.Pruned {
		t.Fatalf("unexpected follow health dry-run state: %+v", report.FollowHealth)
	}

	for _, preserved := range []string{statePath, oldFailedOutput, newFailedOutput, oldFailedManifest, newFailedManifest, followImportsHealthPath(cfg.Meta.LogDir)} {
		if _, err := os.Stat(preserved); err != nil {
			t.Fatalf("expected %s to remain after dry-run, stat err=%v", preserved, err)
		}
	}
}

func TestCleanupFollowImportsIncludeExcludePatterns(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Meta: config.LoadMetadata{
			LogDir: filepath.Join(root, "logs"),
		},
	}
	inputA := filepath.Join(root, "events-a.jsonl")
	inputB := filepath.Join(root, "events-b.jsonl")
	stateA := inputA + ".offset.json"
	stateB := inputB + ".offset.json"
	failedOutputBase := filepath.Join(root, "failed", "failed.jsonl")
	failedManifestBase := filepath.Join(root, "failed", "failed.json")
	failedOutputA := filepath.Join(root, "failed", "failed.events-a.0-42.jsonl")
	failedOutputB := filepath.Join(root, "failed", "failed.events-b.43-84.jsonl")
	failedManifestA := filepath.Join(root, "failed", "failed.events-a.0-42.json")
	failedManifestB := filepath.Join(root, "failed", "failed.events-b.43-84.json")

	for _, path := range []string{
		inputA,
		inputB,
		stateA,
		stateB,
		failedOutputA,
		failedOutputB,
		failedManifestA,
		failedManifestB,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}

	report, err := cleanupFollowImports(cfg, cleanupFollowImportsOptions{
		InputPaths:          []string{inputA, inputB},
		FailedOutputPath:    failedOutputBase,
		FailedManifestPath:  failedManifestBase,
		IncludePatterns:     []string{"*events-a*", "*.offset.json"},
		ExcludePatterns:     []string{"*events-b*"},
		CWD:                 root,
		PruneState:          true,
		PruneFailedOutput:   true,
		PruneFailedManifest: true,
	})
	if err != nil {
		t.Fatalf("cleanupFollowImports: %v", err)
	}

	if got, want := report.StateFiles.Removed, 1; got != want {
		t.Fatalf("state removed mismatch: got %d want %d", got, want)
	}
	if got, want := report.StateFiles.SkippedByPattern, 1; got != want {
		t.Fatalf("state skipped-by-pattern mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedOutputs.Removed, 1; got != want {
		t.Fatalf("failed output removed mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedOutputs.SkippedByPattern, 1; got != want {
		t.Fatalf("failed output skipped-by-pattern mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedManifests.Removed, 1; got != want {
		t.Fatalf("failed manifest removed mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedManifests.SkippedByPattern, 1; got != want {
		t.Fatalf("failed manifest skipped-by-pattern mismatch: got %d want %d", got, want)
	}

	for _, removed := range []string{stateA, failedOutputA, failedManifestA} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, stat err=%v", removed, err)
		}
	}
	for _, preserved := range []string{stateB, failedOutputB, failedManifestB} {
		if _, err := os.Stat(preserved); err != nil {
			t.Fatalf("expected %s to remain, stat err=%v", preserved, err)
		}
	}
}

func TestAuditFollowImportsReportsPendingArtifactsWithoutDeleting(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 3, 17, 4, 0, 0, 0, time.UTC)
	cfg := config.Config{
		Meta: config.LoadMetadata{
			LogDir: filepath.Join(root, "logs"),
		},
	}
	inputPath := filepath.Join(root, "events.jsonl")
	statePath := inputPath + ".offset.json"
	failedOutputBase := filepath.Join(root, "failed", "failed.jsonl")
	failedManifestBase := filepath.Join(root, "failed", "failed.json")
	failedOutput := filepath.Join(root, "failed", "failed.0-42.jsonl")
	failedManifest := filepath.Join(root, "failed", "failed.0-42.json")
	for _, path := range []string{inputPath, statePath, failedOutput, failedManifest} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}
	oldTime := now.Add(-2 * time.Hour)
	for _, path := range []string{statePath, failedOutput, failedManifest} {
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatalf("Chtimes %s: %v", path, err)
		}
	}

	staleSnapshot := followImportsHealthSnapshot{
		Status:              "partial",
		UpdatedAt:           now.Add(-2 * time.Minute),
		Source:              "watcher_import",
		InputCount:          1,
		Continuous:          true,
		PollIntervalSeconds: 5,
		RequestedWatchMode:  "auto",
		ActiveWatchMode:     "notify",
	}
	if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, staleSnapshot); err != nil {
		t.Fatalf("saveFollowImportsHealthSnapshot: %v", err)
	}

	report, err := auditFollowImportsAt(cfg, auditFollowImportsOptions{
		InputPaths:          []string{inputPath},
		FailedOutputPath:    failedOutputBase,
		FailedManifestPath:  failedManifestBase,
		CWD:                 root,
		OlderThan:           time.Hour,
		FailIfMatched:       true,
		CheckState:          true,
		CheckFailedOutput:   true,
		CheckFailedManifest: true,
		CheckFollowHealth:   true,
	}, now)
	if err != nil {
		t.Fatalf("auditFollowImportsAt: %v", err)
	}

	if !report.MatchFound {
		t.Fatalf("expected audit report to find matches, got %+v", report)
	}
	if got, want := report.StateFiles.Matched, 1; got != want {
		t.Fatalf("state matched mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedOutputs.Matched, 1; got != want {
		t.Fatalf("failed output matched mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedManifests.Matched, 1; got != want {
		t.Fatalf("failed manifest matched mismatch: got %d want %d", got, want)
	}
	if !report.FollowHealth.Present || !report.FollowHealth.Stale {
		t.Fatalf("unexpected follow health audit view: %+v", report.FollowHealth)
	}
	if got, want := len(report.FollowHealth.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}

	for _, preserved := range []string{statePath, failedOutput, failedManifest, followImportsHealthPath(cfg.Meta.LogDir)} {
		if _, err := os.Stat(preserved); err != nil {
			t.Fatalf("expected %s to remain after audit, stat err=%v", preserved, err)
		}
	}
}

func TestAuditFollowImportsReportsHealthyFollowHealthWithoutMatch(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 3, 17, 4, 0, 0, 0, time.UTC)
	cfg := config.Config{
		Meta: config.LoadMetadata{
			LogDir: filepath.Join(root, "logs"),
		},
	}
	healthySnapshot := followImportsHealthSnapshot{
		Status:              "ok",
		UpdatedAt:           now.Add(-10 * time.Second),
		Source:              "watcher_import",
		InputCount:          2,
		Continuous:          true,
		PollIntervalSeconds: 5,
		RequestedWatchMode:  "auto",
		ActiveWatchMode:     "notify",
		Warnings: []common.Warning{{
			Code:    common.WarnFollowImportsPollCatchup,
			Message: "notify mode required poll catchup 3 times and 96 bytes so far",
		}},
	}
	if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, healthySnapshot); err != nil {
		t.Fatalf("saveFollowImportsHealthSnapshot: %v", err)
	}

	report, err := auditFollowImportsAt(cfg, auditFollowImportsOptions{
		CheckFollowHealth: true,
	}, now)
	if err != nil {
		t.Fatalf("auditFollowImportsAt: %v", err)
	}

	if report.MatchFound {
		t.Fatalf("expected no audit match for healthy follow health, got %+v", report)
	}
	if !report.FollowHealth.Present {
		t.Fatal("expected follow health to be present")
	}
	if report.FollowHealth.Stale {
		t.Fatalf("expected fresh follow health snapshot, got %+v", report.FollowHealth)
	}
	if got, want := len(report.FollowHealth.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := report.FollowHealth.Status, "ok"; got != want {
		t.Fatalf("status mismatch: got %q want %q", got, want)
	}
}

func TestShouldTriggerFollowImportsEvent(t *testing.T) {
	inputPath := filepath.Clean(filepath.Join("D:", "Code", "go", "codex-mem", "events.jsonl"))
	otherInputPath := filepath.Clean(filepath.Join("D:", "Code", "go", "codex-mem", "other.jsonl"))
	ignoredPath := filepath.Clean(filepath.Join("D:", "Code", "go", "codex-mem", "ignored.jsonl"))

	tests := []struct {
		name  string
		event fsnotify.Event
		want  bool
	}{
		{
			name:  "write to input",
			event: fsnotify.Event{Name: inputPath, Op: fsnotify.Write},
			want:  true,
		},
		{
			name:  "create input",
			event: fsnotify.Event{Name: inputPath, Op: fsnotify.Create},
			want:  true,
		},
		{
			name:  "rename input",
			event: fsnotify.Event{Name: inputPath, Op: fsnotify.Rename},
			want:  true,
		},
		{
			name:  "chmod ignored",
			event: fsnotify.Event{Name: inputPath, Op: fsnotify.Chmod},
			want:  false,
		},
		{
			name:  "sibling file ignored",
			event: fsnotify.Event{Name: ignoredPath, Op: fsnotify.Write},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldTriggerFollowImportsEvent([]string{inputPath, otherInputPath}, tt.event); got != tt.want {
				t.Fatalf("shouldTriggerFollowImportsEvent() = %t, want %t for %+v", got, tt.want, tt.event)
			}
		})
	}
}

func TestFollowImportsRuntimeStateApply(t *testing.T) {
	state := &followImportsRuntimeState{
		Requested:          followImportsWatchModeAuto,
		Active:             followImportsWatchModePoll,
		Fallbacks:          2,
		Transitions:        1,
		LastFallbackReason: "watcher_error",
		PollCatchups:       3,
		PollCatchupBytes:   96,
		PendingEvents: []followImportsEvent{
			{
				At:                 time.Date(2026, 3, 16, 6, 30, 0, 0, time.UTC),
				Kind:               "watch_fallback",
				RequestedWatchMode: "auto",
				PreviousWatchMode:  "notify",
				ActiveWatchMode:    "poll",
				Reason:             "watcher_error",
				Fallbacks:          2,
			},
		},
	}
	report := followImportsReport{}

	state.Apply(&report)

	if got, want := report.RequestedWatchMode, "auto"; got != want {
		t.Fatalf("requested watch mode mismatch: got %q want %q", got, want)
	}
	if got, want := report.ActiveWatchMode, string(followImportsWatchModePoll); got != want {
		t.Fatalf("active watch mode mismatch: got %q want %q", got, want)
	}
	if got, want := report.WatchFallbacks, 2; got != want {
		t.Fatalf("watch fallbacks mismatch: got %d want %d", got, want)
	}
	if got, want := report.WatchTransitions, 1; got != want {
		t.Fatalf("watch transitions mismatch: got %d want %d", got, want)
	}
	if got, want := report.LastFallbackReason, "watcher_error"; got != want {
		t.Fatalf("last fallback reason mismatch: got %q want %q", got, want)
	}
	if got, want := report.WatchEventCount, 1; got != want {
		t.Fatalf("watch event count mismatch: got %d want %d", got, want)
	}
	if got, want := report.WatchPollCatchups, 3; got != want {
		t.Fatalf("watch poll catchups mismatch: got %d want %d", got, want)
	}
	if got, want := report.WatchCatchupBytes, 96; got != want {
		t.Fatalf("watch poll catchup bytes mismatch: got %d want %d", got, want)
	}
	if got, want := len(report.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := report.Warnings[0].Code, common.WarnFollowImportsPollCatchup; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
	if len(report.WatchEvents) != 1 {
		t.Fatalf("watch events mismatch: %+v", report.WatchEvents)
	}
	if len(state.PendingEvents) != 0 {
		t.Fatalf("expected pending events to drain after apply, got %+v", state.PendingEvents)
	}
}

func TestFormatFollowImportsReportIncludesWatchState(t *testing.T) {
	output := formatFollowImportsReport(followImportsReport{
		Status:             "ok",
		Source:             "watcher_import",
		Input:              "events.jsonl",
		StateFile:          "events.offset.json",
		RequestedWatchMode: "auto",
		ActiveWatchMode:    "poll",
		WatchFallbacks:     1,
		WatchTransitions:   2,
		LastFallbackReason: "watcher_unavailable",
		WatchEventCount:    1,
		WatchPollCatchups:  3,
		WatchCatchupBytes:  42,
		Warnings: []common.Warning{{
			Code:    common.WarnFollowImportsPollCatchup,
			Message: "notify mode required poll catchup 3 times and 42 bytes so far",
		}},
		WatchEvents: []followImportsEvent{
			{
				At:                 time.Date(2026, 3, 16, 6, 45, 0, 0, time.UTC),
				Kind:               "watch_fallback",
				RequestedWatchMode: "auto",
				PreviousWatchMode:  "notify",
				ActiveWatchMode:    "poll",
				Reason:             "watcher_unavailable",
				Fallbacks:          1,
				ConsumedInputs:     1,
				ConsumedBytes:      42,
			},
		},
	})

	for _, fragment := range []string{
		"requested_watch_mode=auto",
		"active_watch_mode=poll",
		"watch_fallbacks=1",
		"watch_transitions=2",
		"last_fallback_reason=watcher_unavailable",
		"watch_event_count=1",
		"watch_poll_catchups=3",
		"watch_poll_catchup_bytes=42",
		"warnings=1",
		"warning_1_code=WARN_FOLLOW_IMPORTS_POLL_CATCHUP",
		"watch_event_1_kind=watch_fallback",
		"watch_event_1_previous_watch_mode=notify",
		"watch_event_1_consumed_inputs=1",
		"watch_event_1_consumed_bytes=42",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("report output missing %q:\n%s", fragment, output)
		}
	}
}

func TestSetFollowImportsActiveModeRecordsTransitionEvent(t *testing.T) {
	state := &followImportsRuntimeState{
		Requested: followImportsWatchModeAuto,
		Active:    followImportsWatchModePoll,
	}

	setFollowImportsActiveMode(state, followImportsWatchModeNotify)

	if got, want := state.Active, followImportsWatchModeNotify; got != want {
		t.Fatalf("active watch mode mismatch: got %q want %q", got, want)
	}
	if got, want := state.Transitions, 1; got != want {
		t.Fatalf("watch transitions mismatch: got %d want %d", got, want)
	}
	if len(state.PendingEvents) != 1 {
		t.Fatalf("expected one pending event, got %+v", state.PendingEvents)
	}
	if got, want := state.PendingEvents[0].Kind, "watch_mode_transition"; got != want {
		t.Fatalf("watch event kind mismatch: got %q want %q", got, want)
	}
	if got, want := state.PendingEvents[0].PreviousWatchMode, "poll"; got != want {
		t.Fatalf("previous watch mode mismatch: got %q want %q", got, want)
	}
	if got, want := state.PendingEvents[0].ActiveWatchMode, "notify"; got != want {
		t.Fatalf("active watch mode mismatch: got %q want %q", got, want)
	}
}

func TestMarkFollowImportsFallbackRecordsEventWithoutTransitionWhenAlreadyPolling(t *testing.T) {
	state := &followImportsRuntimeState{
		Requested: followImportsWatchModeAuto,
		Active:    followImportsWatchModePoll,
	}

	markFollowImportsFallback(state, "watcher_unavailable")

	if got, want := state.Fallbacks, 1; got != want {
		t.Fatalf("watch fallbacks mismatch: got %d want %d", got, want)
	}
	if got, want := state.Transitions, 0; got != want {
		t.Fatalf("watch transitions mismatch: got %d want %d", got, want)
	}
	if got, want := state.LastFallbackReason, "watcher_unavailable"; got != want {
		t.Fatalf("last fallback reason mismatch: got %q want %q", got, want)
	}
	if len(state.PendingEvents) != 1 {
		t.Fatalf("expected one pending event, got %+v", state.PendingEvents)
	}
	if got, want := state.PendingEvents[0].Kind, "watch_fallback"; got != want {
		t.Fatalf("watch event kind mismatch: got %q want %q", got, want)
	}
	if got, want := state.PendingEvents[0].ActiveWatchMode, "poll"; got != want {
		t.Fatalf("active watch mode mismatch: got %q want %q", got, want)
	}
}

func TestMarkFollowImportsRecoveryRecordsEvent(t *testing.T) {
	state := &followImportsRuntimeState{
		Requested:          followImportsWatchModeAuto,
		Active:             followImportsWatchModePoll,
		Fallbacks:          2,
		LastFallbackReason: "watcher_error",
	}

	markFollowImportsRecovery(state, "watcher_recovered")

	if got, want := state.Active, followImportsWatchModeNotify; got != want {
		t.Fatalf("active watch mode mismatch: got %q want %q", got, want)
	}
	if got, want := state.Transitions, 1; got != want {
		t.Fatalf("watch transitions mismatch: got %d want %d", got, want)
	}
	if len(state.PendingEvents) != 1 {
		t.Fatalf("expected one pending event, got %+v", state.PendingEvents)
	}
	if got, want := state.PendingEvents[0].Kind, followImportsEventRecovery; got != want {
		t.Fatalf("watch event kind mismatch: got %q want %q", got, want)
	}
	if got, want := state.PendingEvents[0].Reason, "watcher_recovered"; got != want {
		t.Fatalf("watch event reason mismatch: got %q want %q", got, want)
	}
}

func TestMarkFollowImportsPollCatchupRecordsEvent(t *testing.T) {
	state := &followImportsRuntimeState{
		Requested: followImportsWatchModeAuto,
		Active:    followImportsWatchModeNotify,
		Fallbacks: 1,
	}

	markFollowImportsPollCatchup(state, 2, 128)

	if len(state.PendingEvents) != 1 {
		t.Fatalf("expected one pending event, got %+v", state.PendingEvents)
	}
	event := state.PendingEvents[0]
	if got, want := event.Kind, followImportsEventCatchup; got != want {
		t.Fatalf("watch event kind mismatch: got %q want %q", got, want)
	}
	if got, want := event.ConsumedInputs, 2; got != want {
		t.Fatalf("consumed input count mismatch: got %d want %d", got, want)
	}
	if got, want := event.ConsumedBytes, 128; got != want {
		t.Fatalf("consumed byte count mismatch: got %d want %d", got, want)
	}
	if got, want := state.PollCatchups, 1; got != want {
		t.Fatalf("poll catchup count mismatch: got %d want %d", got, want)
	}
	if got, want := state.PollCatchupBytes, 128; got != want {
		t.Fatalf("poll catchup bytes mismatch: got %d want %d", got, want)
	}
}

func TestEnterFollowImportsNotifyModeUsesRecoveryEventAfterFallback(t *testing.T) {
	state := &followImportsRuntimeState{
		Requested: followImportsWatchModeAuto,
		Active:    followImportsWatchModePoll,
		Fallbacks: 1,
	}

	enterFollowImportsNotifyMode(state)

	if got, want := state.Active, followImportsWatchModeNotify; got != want {
		t.Fatalf("active watch mode mismatch: got %q want %q", got, want)
	}
	if len(state.PendingEvents) != 1 {
		t.Fatalf("expected one pending event, got %+v", state.PendingEvents)
	}
	if got, want := state.PendingEvents[0].Kind, followImportsEventRecovery; got != want {
		t.Fatalf("watch event kind mismatch: got %q want %q", got, want)
	}
}

func TestShouldWriteFollowImportsReport(t *testing.T) {
	tests := []struct {
		name   string
		report followImportsReport
		once   bool
		want   bool
	}{
		{
			name:   "once mode always writes",
			report: followImportsReport{Status: "idle"},
			once:   true,
			want:   true,
		},
		{
			name:   "non idle writes",
			report: followImportsReport{Status: "ok"},
			want:   true,
		},
		{
			name:   "idle watch event writes",
			report: followImportsReport{Status: "idle", WatchEventCount: 1},
			want:   true,
		},
		{
			name:   "plain idle skipped",
			report: followImportsReport{Status: "idle"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldWriteFollowImportsReport(tt.report, tt.once); got != tt.want {
				t.Fatalf("shouldWriteFollowImportsReport() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestShouldWriteFollowImportsAggregateReport(t *testing.T) {
	tests := []struct {
		name   string
		report followImportsAggregateReport
		once   bool
		want   bool
	}{
		{
			name:   "once mode always writes",
			report: followImportsAggregateReport{Status: "idle"},
			once:   true,
			want:   true,
		},
		{
			name:   "non idle writes",
			report: followImportsAggregateReport{Status: "ok"},
			want:   true,
		},
		{
			name:   "idle watch event writes",
			report: followImportsAggregateReport{Status: "idle", WatchEventCount: 1},
			want:   true,
		},
		{
			name:   "plain idle skipped",
			report: followImportsAggregateReport{Status: "idle"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldWriteFollowImportsAggregateReport(tt.report, tt.once); got != tt.want {
				t.Fatalf("shouldWriteFollowImportsAggregateReport() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestBuildFollowImportsInputsRejectsDuplicateInputs(t *testing.T) {
	root := t.TempDir()
	_, _, err := buildFollowImportsInputs(followImportsOptions{
		Source:     followImportsWatcherSource,
		InputPaths: []string{"events.jsonl", ".\\events.jsonl"},
		CWD:        root,
	})
	if err == nil {
		t.Fatal("expected duplicate input error")
	}
	if !strings.Contains(err.Error(), `events.jsonl" is duplicated`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildFollowImportsInputsDerivesPerInputFailureBases(t *testing.T) {
	root := t.TempDir()
	inputs, watchPaths, err := buildFollowImportsInputs(followImportsOptions{
		Source:             followImportsWatcherSource,
		InputPaths:         []string{"events-a.jsonl", "events-b.jsonl"},
		FailedOutputPath:   "failed.jsonl",
		FailedManifestPath: "failed.json",
		CWD:                root,
	})
	if err != nil {
		t.Fatalf("buildFollowImportsInputs: %v", err)
	}
	if got, want := len(inputs), 2; got != want {
		t.Fatalf("input count mismatch: got %d want %d", got, want)
	}
	if got, want := len(watchPaths), 2; got != want {
		t.Fatalf("watch path count mismatch: got %d want %d", got, want)
	}
	if !strings.Contains(inputs[0].FailedOutputPath, "failed.events-a.jsonl") {
		t.Fatalf("expected per-input failed output base, got %q", inputs[0].FailedOutputPath)
	}
	if !strings.Contains(inputs[1].FailedManifestPath, "failed.events-b.json") {
		t.Fatalf("expected per-input failed manifest base, got %q", inputs[1].FailedManifestPath)
	}
}

func TestSummarizeFollowImportsConsumption(t *testing.T) {
	consumedInputs, consumedBytes := summarizeFollowImportsConsumption([]followImportsReport{
		{ConsumedBytes: 0},
		{ConsumedBytes: 12},
		{ConsumedBytes: 8},
	})
	if got, want := consumedInputs, 2; got != want {
		t.Fatalf("consumed input count mismatch: got %d want %d", got, want)
	}
	if got, want := consumedBytes, 20; got != want {
		t.Fatalf("consumed byte count mismatch: got %d want %d", got, want)
	}
}

func TestFollowImportsRuntimeWarningsThreshold(t *testing.T) {
	warnings := followImportsRuntimeWarnings(&followImportsRuntimeState{
		PollCatchups:     3,
		PollCatchupBytes: 256,
	})
	if got, want := len(warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := warnings[0].Code, common.WarnFollowImportsPollCatchup; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}

	noWarnings := followImportsRuntimeWarnings(&followImportsRuntimeState{
		PollCatchups:     2,
		PollCatchupBytes: 128,
	})
	if len(noWarnings) != 0 {
		t.Fatalf("expected no warnings below threshold, got %+v", noWarnings)
	}
}

func TestRunFollowImportsPollingRecoveryLoopRecoversWatcher(t *testing.T) {
	root := t.TempDir()
	inputPath := filepath.Join(root, "watched", "events.jsonl")
	state := &followImportsRuntimeState{
		Requested:          followImportsWatchModeAuto,
		Active:             followImportsWatchModePoll,
		Fallbacks:          1,
		LastFallbackReason: "watcher_unavailable",
	}
	runCount := 0
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = os.MkdirAll(filepath.Dir(inputPath), 0o755)
	}()

	err := runFollowImportsPollingRecoveryLoop(ctx, []string{inputPath}, 20*time.Millisecond, func(_ followImportsRunTrigger) error {
		runCount++
		return nil
	}, state)
	if err != nil {
		t.Fatalf("runFollowImportsPollingRecoveryLoop: %v", err)
	}
	if runCount < 2 {
		t.Fatalf("expected recovery loop to poll at least twice, got %d", runCount)
	}
	if got, want := state.Active, followImportsWatchModeNotify; got != want {
		t.Fatalf("active watch mode mismatch: got %q want %q", got, want)
	}
	if len(state.PendingEvents) == 0 {
		t.Fatalf("expected recovery event, got none")
	}
	if got, want := state.PendingEvents[len(state.PendingEvents)-1].Kind, followImportsEventRecovery; got != want {
		t.Fatalf("watch event kind mismatch: got %q want %q", got, want)
	}
}

func TestNewFollowImportsAggregateReport(t *testing.T) {
	report := newFollowImportsAggregateReport(followImportsWatcherSource, []followImportsReport{
		{
			Status:        "ok",
			ConsumedBytes: 10,
			PendingBytes:  1,
		},
		{
			Status:        "partial",
			ConsumedBytes: 20,
			PendingBytes:  2,
		},
		{
			Status:       "idle",
			PendingBytes: 3,
		},
	})

	if got, want := report.Status, "partial"; got != want {
		t.Fatalf("aggregate status mismatch: got %q want %q", got, want)
	}
	if got, want := report.InputCount, 3; got != want {
		t.Fatalf("input count mismatch: got %d want %d", got, want)
	}
	if got, want := report.ConsumedInputs, 2; got != want {
		t.Fatalf("consumed input count mismatch: got %d want %d", got, want)
	}
	if got, want := report.IdleInputs, 1; got != want {
		t.Fatalf("idle input count mismatch: got %d want %d", got, want)
	}
	if got, want := report.PartialInputs, 1; got != want {
		t.Fatalf("partial input count mismatch: got %d want %d", got, want)
	}
	if got, want := report.TotalConsumedBytes, 30; got != want {
		t.Fatalf("consumed bytes mismatch: got %d want %d", got, want)
	}
	if got, want := report.TotalPendingBytes, 6; got != want {
		t.Fatalf("pending bytes mismatch: got %d want %d", got, want)
	}
}

func TestFormatFollowImportsAggregateReportIncludesInputSections(t *testing.T) {
	output := formatFollowImportsAggregateReport(followImportsAggregateReport{
		Status:             "ok",
		Source:             followImportsWatcherSource,
		InputCount:         2,
		ConsumedInputs:     1,
		IdleInputs:         1,
		RequestedWatchMode: "auto",
		ActiveWatchMode:    "notify",
		TotalConsumedBytes: 42,
		TotalPendingBytes:  3,
		Inputs: []followImportsReport{
			{
				Status:    "ok",
				Source:    followImportsWatcherSource,
				Input:     "a.jsonl",
				StateFile: "a.offset.json",
			},
			{
				Status:    "idle",
				Source:    followImportsWatcherSource,
				Input:     "b.jsonl",
				StateFile: "b.offset.json",
			},
		},
	})

	for _, fragment := range []string{
		"input_count=2",
		"input_1_input=a.jsonl",
		"input_2_state_file=b.offset.json",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("aggregate report missing %q:\n%s", fragment, output)
		}
	}
}

func TestAppFollowImportsMultiInputConsumesEachCheckpointSeparately(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	instance, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = instance.Close() }()

	inputA := filepath.Join(root, "events-a.jsonl")
	inputB := filepath.Join(root, "events-b.jsonl")
	stateA := filepath.Join(root, "events-a.offset.json")
	stateB := filepath.Join(root, "events-b.offset.json")
	firstA := `{"external_id":"watcher:a1","type":"discovery","title":"A1","content":"A one.","importance":4}`
	firstB := `{"external_id":"watcher:b1","type":"todo","title":"B1","content":"B one.","importance":3}`
	if err := os.WriteFile(inputA, []byte(firstA+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile inputA: %v", err)
	}
	if err := os.WriteFile(inputB, []byte(firstB+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile inputB: %v", err)
	}

	inputs := []FollowImportsInput{
		{
			Source:     followImportsWatcherSource,
			InputPath:  inputA,
			StatePath:  stateA,
			CWD:        root,
			RepoRemote: "git@github.com:example/codex-mem.git",
		},
		{
			Source:     followImportsWatcherSource,
			InputPath:  inputB,
			StatePath:  stateB,
			CWD:        root,
			RepoRemote: "git@github.com:example/codex-mem.git",
		},
	}

	reports, err := runFollowImportsInputsOnce(context.Background(), instance, inputs)
	if err != nil {
		t.Fatalf("runFollowImportsInputsOnce: %v", err)
	}
	if got, want := len(reports), 2; got != want {
		t.Fatalf("report count mismatch: got %d want %d", got, want)
	}
	for i, report := range reports {
		if got, want := report.Status, "ok"; got != want {
			t.Fatalf("report %d status mismatch: got %q want %q", i, got, want)
		}
		if report.Batch == nil || report.Batch.Processed != 1 {
			t.Fatalf("report %d batch mismatch: %+v", i, report.Batch)
		}
	}

	if got, want := readFollowImportsStateForTest(t, stateA).Offset, int64(len(firstA)+1); got != want {
		t.Fatalf("stateA offset mismatch: got %d want %d", got, want)
	}
	if got, want := readFollowImportsStateForTest(t, stateB).Offset, int64(len(firstB)+1); got != want {
		t.Fatalf("stateB offset mismatch: got %d want %d", got, want)
	}

	diagnostics, err := db.InspectRuntime(context.Background(), instance.DB)
	if err != nil {
		t.Fatalf("InspectRuntime: %v", err)
	}
	if got, want := diagnostics.Audit.NoteRecords, 2; got != want {
		t.Fatalf("note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.ImportRecords, 2; got != want {
		t.Fatalf("import count mismatch: got %d want %d", got, want)
	}
}

func TestAppFollowImportsOnceConsumesOnlyCompleteLinesAndPersistsCheckpoint(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	instance, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = instance.Close() }()

	eventsPath := filepath.Join(root, "events.jsonl")
	statePath := filepath.Join(root, "events.offset.json")
	first := followImportsFirstEvent
	secondPrefix := `{"external_id":"watcher:2","type":"todo","title":"Second"`
	if err := os.WriteFile(eventsPath, []byte(first+"\n"+secondPrefix), 0o644); err != nil {
		t.Fatalf("WriteFile events: %v", err)
	}

	report, err := instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     followImportsWatcherSource,
		InputPath:  eventsPath,
		StatePath:  statePath,
		CWD:        root,
		RepoRemote: "git@github.com:example/codex-mem.git",
		Task:       "follow imports test",
	})
	if err != nil {
		t.Fatalf("FollowImportsOnce first pass: %v", err)
	}
	if got, want := report.Status, "ok"; got != want {
		t.Fatalf("status mismatch: got %q want %q", got, want)
	}
	if got, want := report.ConsumedBytes, len(first)+1; got != want {
		t.Fatalf("consumed bytes mismatch: got %d want %d", got, want)
	}
	if got, want := report.PendingBytes, len(secondPrefix); got != want {
		t.Fatalf("pending bytes mismatch: got %d want %d", got, want)
	}
	if report.Batch == nil || report.Batch.Processed != 1 {
		t.Fatalf("expected one processed batch result, got %+v", report.Batch)
	}
	state := readFollowImportsStateForTest(t, statePath)
	if got, want := state.Offset, int64(len(first)+1); got != want {
		t.Fatalf("state offset mismatch: got %d want %d", got, want)
	}
	if state.Checkpoint == nil || state.Checkpoint.TailSHA256 == "" {
		t.Fatalf("expected checkpoint hash after first pass, got %+v", state.Checkpoint)
	}

	secondSuffix := `,"content":"Second event.","importance":3}`
	third := `{"external_id":"watcher:3","type":"bugfix","title":"Third","content":"Third event.","importance":5}`
	file, err := os.OpenFile(eventsPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile append events: %v", err)
	}
	if _, err := file.WriteString(secondSuffix + "\n" + third + "\n"); err != nil {
		_ = file.Close()
		t.Fatalf("WriteString append events: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close appended events: %v", err)
	}

	report, err = instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     followImportsWatcherSource,
		InputPath:  eventsPath,
		StatePath:  statePath,
		CWD:        root,
		RepoRemote: "git@github.com:example/codex-mem.git",
		Task:       "follow imports test",
	})
	if err != nil {
		t.Fatalf("FollowImportsOnce second pass: %v", err)
	}
	if got, want := report.Status, "ok"; got != want {
		t.Fatalf("second status mismatch: got %q want %q", got, want)
	}
	if report.Batch == nil || report.Batch.Processed != 2 {
		t.Fatalf("expected second pass to process two events, got %+v", report.Batch)
	}
	finalState := readFollowImportsStateForTest(t, statePath)
	info, err := os.Stat(eventsPath)
	if err != nil {
		t.Fatalf("Stat events: %v", err)
	}
	if got, want := finalState.Offset, info.Size(); got != want {
		t.Fatalf("final state offset mismatch: got %d want %d", got, want)
	}

	diagnostics, err := db.InspectRuntime(context.Background(), instance.DB)
	if err != nil {
		t.Fatalf("InspectRuntime: %v", err)
	}
	if got, want := diagnostics.Audit.NoteRecords, 3; got != want {
		t.Fatalf("note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.ImportRecords, 3; got != want {
		t.Fatalf("import count mismatch: got %d want %d", got, want)
	}
}

func TestAppFollowImportsOnceUsesCheckpointRecoveryWhenNoNewLinesExist(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	eventsPath := filepath.Join(root, "events.jsonl")
	statePath := filepath.Join(root, "events.offset.json")
	event := followImportsFirstEvent
	if err := os.WriteFile(eventsPath, []byte(event+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile events: %v", err)
	}

	instance, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New first instance: %v", err)
	}
	report, err := instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     followImportsWatcherSource,
		InputPath:  eventsPath,
		StatePath:  statePath,
		CWD:        root,
		RepoRemote: "git@github.com:example/codex-mem.git",
	})
	if err != nil {
		_ = instance.Close()
		t.Fatalf("FollowImportsOnce first pass: %v", err)
	}
	if report.Batch == nil || report.Batch.Processed != 1 {
		_ = instance.Close()
		t.Fatalf("expected first pass to process one event, got %+v", report.Batch)
	}
	if err := instance.Close(); err != nil {
		t.Fatalf("Close first instance: %v", err)
	}

	instance, err = New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New second instance: %v", err)
	}
	defer func() { _ = instance.Close() }()

	report, err = instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     followImportsWatcherSource,
		InputPath:  eventsPath,
		StatePath:  statePath,
		CWD:        root,
		RepoRemote: "git@github.com:example/codex-mem.git",
	})
	if err != nil {
		t.Fatalf("FollowImportsOnce recovery pass: %v", err)
	}
	if got, want := report.Status, "idle"; got != want {
		t.Fatalf("status mismatch: got %q want %q", got, want)
	}
	if report.CheckpointReset {
		t.Fatalf("did not expect checkpoint reset on idle recovery: %+v", report)
	}
	if report.Batch != nil {
		t.Fatalf("did not expect batch on idle recovery pass: %+v", report.Batch)
	}

	diagnostics, err := db.InspectRuntime(context.Background(), instance.DB)
	if err != nil {
		t.Fatalf("InspectRuntime: %v", err)
	}
	if got, want := diagnostics.Audit.NoteRecords, 1; got != want {
		t.Fatalf("note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.ImportRecords, 1; got != want {
		t.Fatalf("import count mismatch: got %d want %d", got, want)
	}
}

func TestAppFollowImportsOnceResetsOffsetAfterTruncation(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	instance, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = instance.Close() }()

	eventsPath := filepath.Join(root, "events.jsonl")
	statePath := filepath.Join(root, "events.offset.json")
	first := followImportsFirstEvent
	if err := os.WriteFile(eventsPath, []byte(first+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile first events: %v", err)
	}
	if _, err := instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     followImportsWatcherSource,
		InputPath:  eventsPath,
		StatePath:  statePath,
		CWD:        root,
		RepoRemote: "git@github.com:example/codex-mem.git",
	}); err != nil {
		t.Fatalf("FollowImportsOnce initial pass: %v", err)
	}

	second := `{"external_id":"watcher:2","type":"todo","title":"Second","content":"Second event.","importance":3}`
	if err := os.WriteFile(eventsPath, []byte(second+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile truncated events: %v", err)
	}

	report, err := instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     followImportsWatcherSource,
		InputPath:  eventsPath,
		StatePath:  statePath,
		CWD:        root,
		RepoRemote: "git@github.com:example/codex-mem.git",
	})
	if err != nil {
		t.Fatalf("FollowImportsOnce truncated pass: %v", err)
	}
	if !report.Truncated {
		t.Fatalf("expected truncation to be reported: %+v", report)
	}
	if !report.CheckpointReset || report.ResetReason != "truncated" {
		t.Fatalf("expected truncation reset metadata, got %+v", report)
	}
	if report.Batch == nil || report.Batch.Processed != 1 {
		t.Fatalf("expected one processed event after truncation, got %+v", report.Batch)
	}

	diagnostics, err := db.InspectRuntime(context.Background(), instance.DB)
	if err != nil {
		t.Fatalf("InspectRuntime: %v", err)
	}
	if got, want := diagnostics.Audit.NoteRecords, 2; got != want {
		t.Fatalf("note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.ImportRecords, 2; got != want {
		t.Fatalf("import count mismatch: got %d want %d", got, want)
	}
}

func TestAppFollowImportsOnceWritesBatchScopedFailureExports(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	instance, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = instance.Close() }()

	eventsPath := filepath.Join(root, "events.jsonl")
	statePath := filepath.Join(root, "events.offset.json")
	failedOutputBase := filepath.Join(root, "failed", "failed.jsonl")
	failedManifestBase := filepath.Join(root, "failed", "failed.json")
	badLine := `{"external_id":"watcher:bad","type":"discovery","title":"Broken"`
	if err := os.WriteFile(eventsPath, []byte(badLine+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile bad events: %v", err)
	}

	report, err := instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:             followImportsWatcherSource,
		InputPath:          eventsPath,
		StatePath:          statePath,
		CWD:                root,
		RepoRemote:         "git@github.com:example/codex-mem.git",
		FailedOutputPath:   failedOutputBase,
		FailedManifestPath: failedManifestBase,
	})
	if err != nil {
		t.Fatalf("FollowImportsOnce bad batch: %v", err)
	}
	if got, want := report.Status, "failed"; got != want {
		t.Fatalf("status mismatch: got %q want %q", got, want)
	}
	if report.Batch == nil {
		t.Fatalf("expected batch report, got %+v", report)
	}
	if report.BatchError == nil {
		t.Fatalf("expected batch error, got %+v", report)
	}
	if got, want := report.BatchError.Code, "ERR_WRITE_FAILED"; got != want {
		t.Fatalf("batch error code mismatch: got %q want %q", got, want)
	}
	if !strings.Contains(report.Batch.FailedOutput, ".0-") {
		t.Fatalf("expected derived failed output path, got %q", report.Batch.FailedOutput)
	}
	if !strings.Contains(report.Batch.FailedManifest, ".0-") {
		t.Fatalf("expected derived failed manifest path, got %q", report.Batch.FailedManifest)
	}

	failedOutputBody, err := os.ReadFile(report.Batch.FailedOutput)
	if err != nil {
		t.Fatalf("ReadFile failed output: %v", err)
	}
	if got, want := strings.TrimSpace(string(failedOutputBody)), badLine; got != want {
		t.Fatalf("failed output mismatch: got %q want %q", got, want)
	}

	manifestBody, err := os.ReadFile(report.Batch.FailedManifest)
	if err != nil {
		t.Fatalf("ReadFile failed manifest: %v", err)
	}
	var manifest struct {
		FailureCount int `json:"failure_count"`
	}
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatalf("unmarshal failed manifest: %v\n%s", err, string(manifestBody))
	}
	if got, want := manifest.FailureCount, 1; got != want {
		t.Fatalf("manifest failure count mismatch: got %d want %d", got, want)
	}

	state := readFollowImportsStateForTest(t, statePath)
	info, err := os.Stat(eventsPath)
	if err != nil {
		t.Fatalf("Stat events: %v", err)
	}
	if got, want := state.Offset, info.Size(); got != want {
		t.Fatalf("state offset mismatch: got %d want %d", got, want)
	}
}

func TestAppFollowImportsOnceResetsOffsetWhenFileIsReplacedWithoutShrinking(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	instance, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = instance.Close() }()

	eventsPath := filepath.Join(root, "events.jsonl")
	statePath := filepath.Join(root, "events.offset.json")
	first := `{"external_id":"watcher:1","type":"discovery","title":"Alpha","content":"11111","importance":4}`
	second := `{"external_id":"watcher:2","type":"discovery","title":"Bravo","content":"22222","importance":4}`
	if len(first) != len(second) {
		t.Fatalf("test data must stay same size: %d vs %d", len(first), len(second))
	}
	if err := os.WriteFile(eventsPath, []byte(first+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile first events: %v", err)
	}

	report, err := instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     followImportsWatcherSource,
		InputPath:  eventsPath,
		StatePath:  statePath,
		CWD:        root,
		RepoRemote: "git@github.com:example/codex-mem.git",
	})
	if err != nil {
		t.Fatalf("FollowImportsOnce first pass: %v", err)
	}
	if report.Batch == nil || report.Batch.Processed != 1 {
		t.Fatalf("expected first event to be processed, got %+v", report.Batch)
	}

	if err := os.WriteFile(eventsPath, []byte(second+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile replacement events: %v", err)
	}

	report, err = instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     followImportsWatcherSource,
		InputPath:  eventsPath,
		StatePath:  statePath,
		CWD:        root,
		RepoRemote: "git@github.com:example/codex-mem.git",
	})
	if err != nil {
		t.Fatalf("FollowImportsOnce replacement pass: %v", err)
	}
	if !report.CheckpointReset {
		t.Fatalf("expected checkpoint reset on replacement, got %+v", report)
	}
	if got, want := report.ResetReason, "file_replaced"; got != want {
		t.Fatalf("reset reason mismatch: got %q want %q", got, want)
	}
	if report.Batch == nil || report.Batch.Processed != 1 {
		t.Fatalf("expected replacement file to be processed from the start, got %+v", report.Batch)
	}
	if report.Truncated {
		t.Fatalf("did not expect truncation for same-size replacement: %+v", report)
	}

	diagnostics, err := db.InspectRuntime(context.Background(), instance.DB)
	if err != nil {
		t.Fatalf("InspectRuntime: %v", err)
	}
	if got, want := diagnostics.Audit.NoteRecords, 2; got != want {
		t.Fatalf("note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.ImportRecords, 2; got != want {
		t.Fatalf("import count mismatch: got %d want %d", got, want)
	}
}

func readFollowImportsStateForTest(t *testing.T, path string) followImportsState {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile state: %v", err)
	}

	var state followImportsState
	if err := json.Unmarshal(body, &state); err != nil {
		t.Fatalf("unmarshal follow-imports state: %v\n%s", err, string(body))
	}
	return state
}
