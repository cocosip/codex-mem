package app

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codex-mem/internal/config"
	"codex-mem/internal/db"
)

func TestParseIngestImportsOptions(t *testing.T) {
	options, err := parseIngestImportsOptions([]string{
		"--source", "watcher_import",
		"--input", "events.jsonl",
		"--cwd", "D:/Code/go/codex-mem",
		"--branch-name", "feature/imports",
		"--repo-remote", "git@github.com:example/codex-mem.git",
		"--task", "batch import",
		"--json",
	})
	if err != nil {
		t.Fatalf("parseIngestImportsOptions: %v", err)
	}
	if got, want := string(options.Source), "watcher_import"; got != want {
		t.Fatalf("source mismatch: got %q want %q", got, want)
	}
	if got, want := options.InputPath, "events.jsonl"; got != want {
		t.Fatalf("input path mismatch: got %q want %q", got, want)
	}
	if got, want := options.CWD, "D:/Code/go/codex-mem"; got != want {
		t.Fatalf("cwd mismatch: got %q want %q", got, want)
	}
	if !options.JSON {
		t.Fatal("expected JSON mode")
	}
}

func TestParseIngestImportsOptionsRejectsMissingSource(t *testing.T) {
	_, err := parseIngestImportsOptions(nil)
	if err == nil {
		t.Fatal("expected missing source error")
	}
	if !strings.Contains(err.Error(), "ingest-imports source is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunIngestImportsPersistsImportedNotesFromJSONL(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	input := strings.Join([]string{
		`{"external_id":"watcher:1","type":"discovery","title":"Imported discovery","content":"Useful watcher discovery.","importance":4,"tags":["watcher"]}`,
		`{"external_id":"watcher:2","type":"todo","title":"Private follow-up","content":"Should stay audit-only.","importance":3,"privacy_intent":"private"}`,
	}, "\n")

	var stdout bytes.Buffer
	err := Run(context.Background(), cfg, []string{
		"ingest-imports",
		"--source", "watcher_import",
		"--cwd", root,
		"--repo-remote", "git@github.com:example/codex-mem.git",
	}, strings.NewReader(input), &stdout)
	if err != nil {
		t.Fatalf("Run ingest-imports: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"ingest imports ok",
		"source=watcher_import",
		"processed=2",
		"materialized=1",
		"suppressed=1",
		"warnings=1",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("ingest output missing %q:\n%s", fragment, output)
		}
	}

	instance, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = instance.Close() }()

	diagnostics, err := db.InspectRuntime(context.Background(), instance.DB)
	if err != nil {
		t.Fatalf("InspectRuntime: %v", err)
	}
	if got, want := diagnostics.Audit.NoteRecords, 1; got != want {
		t.Fatalf("note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.ImportRecords, 2; got != want {
		t.Fatalf("import count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.NotesWatcherImport, 1; got != want {
		t.Fatalf("watcher note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.ImportsWatcherImport, 2; got != want {
		t.Fatalf("watcher import count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.SuppressedImports, 1; got != want {
		t.Fatalf("suppressed import count mismatch: got %d want %d", got, want)
	}
}

func TestRunIngestImportsPrintsJSONReport(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	input := `{"external_id":"relay:1","type":"bugfix","title":"Relay bugfix","content":"Imported from relay.","importance":4}`

	var stdout bytes.Buffer
	err := Run(context.Background(), cfg, []string{
		"ingest-imports",
		"--source", "relay_import",
		"--cwd", root,
		"--repo-remote", "git@github.com:example/codex-mem.git",
		"--json",
	}, strings.NewReader(input), &stdout)
	if err != nil {
		t.Fatalf("Run ingest-imports --json: %v", err)
	}

	var report ingestImportsReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal ingest report: %v\n%s", err, stdout.String())
	}
	if got, want := report.Processed, 1; got != want {
		t.Fatalf("processed mismatch: got %d want %d", got, want)
	}
	if got, want := report.Materialized, 1; got != want {
		t.Fatalf("materialized mismatch: got %d want %d", got, want)
	}
	if report.Session.ID == "" {
		t.Fatalf("expected session id in report: %+v", report.Session)
	}
	if got, want := len(report.Results), 1; got != want {
		t.Fatalf("result count mismatch: got %d want %d", got, want)
	}
	if report.Results[0].ImportID == "" {
		t.Fatalf("expected import id in result: %+v", report.Results[0])
	}
}

func ingestTestConfig(root string) config.Config {
	return config.Config{
		File: config.FileConfig{
			DatabasePath:      filepath.Join(root, "data", "codex-mem.db"),
			DefaultSystemName: "codex-mem",
			SQLiteDriver:      "sqlite",
			BusyTimeout:       5 * time.Second,
			JournalMode:       "WAL",
			LogFilePath:       filepath.Join(root, "logs", "codex-mem.log"),
			LogMaxSizeMB:      20,
			LogMaxBackups:     10,
			LogMaxAgeDays:     30,
			LogCompress:       true,
			LogAlsoStderr:     false,
		},
		Meta: config.LoadMetadata{
			ConfigDir:      filepath.Join(root, "configs"),
			ConfigFilePath: filepath.Join(root, "configs", "codex-mem.json"),
			LogDir:         filepath.Join(root, "logs"),
		},
	}
}
