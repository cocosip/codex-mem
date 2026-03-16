package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codex-mem/internal/db"
)

func TestParseFollowImportsOptions(t *testing.T) {
	options, err := parseFollowImportsOptions([]string{
		"--source", "watcher_import",
		"--input", "events.jsonl",
		"--state-file", "events.offset.json",
		"--failed-output", "failed.jsonl",
		"--failed-manifest", "failed.json",
		"--cwd", "D:/Code/go/codex-mem",
		"--branch-name", "feature/import-follow",
		"--repo-remote", "git@github.com:example/codex-mem.git",
		"--task", "follow imports",
		"--poll-interval", "10s",
		"--once",
		"--json",
	})
	if err != nil {
		t.Fatalf("parseFollowImportsOptions: %v", err)
	}
	if got, want := string(options.Source), "watcher_import"; got != want {
		t.Fatalf("source mismatch: got %q want %q", got, want)
	}
	if got, want := options.InputPath, "events.jsonl"; got != want {
		t.Fatalf("input path mismatch: got %q want %q", got, want)
	}
	if got, want := options.StatePath, "events.offset.json"; got != want {
		t.Fatalf("state path mismatch: got %q want %q", got, want)
	}
	if got, want := options.PollInterval.String(), "10s"; got != want {
		t.Fatalf("poll interval mismatch: got %q want %q", got, want)
	}
	if !options.Once {
		t.Fatal("expected once mode")
	}
	if !options.JSON {
		t.Fatal("expected JSON mode")
	}
}

func TestParseFollowImportsOptionsRejectsMissingInput(t *testing.T) {
	_, err := parseFollowImportsOptions([]string{"--source", "watcher_import"})
	if err == nil {
		t.Fatal("expected missing input error")
	}
	if !strings.Contains(err.Error(), "follow-imports input is required") {
		t.Fatalf("unexpected error: %v", err)
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
	first := `{"external_id":"watcher:1","type":"discovery","title":"First","content":"First event.","importance":4}`
	secondPrefix := `{"external_id":"watcher:2","type":"todo","title":"Second"`
	if err := os.WriteFile(eventsPath, []byte(first+"\n"+secondPrefix), 0o644); err != nil {
		t.Fatalf("WriteFile events: %v", err)
	}

	report, err := instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     "watcher_import",
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
		Source:     "watcher_import",
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
	event := `{"external_id":"watcher:1","type":"discovery","title":"First","content":"First event.","importance":4}`
	if err := os.WriteFile(eventsPath, []byte(event+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile events: %v", err)
	}

	instance, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New first instance: %v", err)
	}
	report, err := instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     "watcher_import",
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
		Source:     "watcher_import",
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
	first := `{"external_id":"watcher:1","type":"discovery","title":"First","content":"First event.","importance":4}`
	if err := os.WriteFile(eventsPath, []byte(first+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile first events: %v", err)
	}
	if _, err := instance.FollowImportsOnce(context.Background(), FollowImportsInput{
		Source:     "watcher_import",
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
		Source:     "watcher_import",
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
		Source:             "watcher_import",
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
