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
)

func TestParseIngestImportsOptions(t *testing.T) {
	options, err := parseIngestImportsOptions([]string{
		"--source", "watcher_import",
		"--input", "events.jsonl",
		"--failed-output", "failed.jsonl",
		"--failed-manifest", "failed-manifest.json",
		"--cwd", "D:/Code/go/codex-mem",
		"--branch-name", "feature/imports",
		"--repo-remote", "git@github.com:example/codex-mem.git",
		"--task", "batch import",
		"--continue-on-error",
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
	if got, want := options.FailedOutputPath, "failed.jsonl"; got != want {
		t.Fatalf("failed output path mismatch: got %q want %q", got, want)
	}
	if got, want := options.FailedManifestPath, "failed-manifest.json"; got != want {
		t.Fatalf("failed manifest path mismatch: got %q want %q", got, want)
	}
	if got, want := options.CWD, "D:/Code/go/codex-mem"; got != want {
		t.Fatalf("cwd mismatch: got %q want %q", got, want)
	}
	if !options.JSON {
		t.Fatal("expected JSON mode")
	}
	if !options.ContinueOnError {
		t.Fatal("expected continue-on-error mode")
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

func TestParseIngestImportsOptionsRejectsFailedOutputWithoutContinueOnError(t *testing.T) {
	_, err := parseIngestImportsOptions([]string{
		"--source", "watcher_import",
		"--failed-output", "failed.jsonl",
	})
	if err == nil {
		t.Fatal("expected failed-output validation error")
	}
	if !strings.Contains(err.Error(), "--failed-output requires --continue-on-error") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseIngestImportsOptionsRejectsFailedManifestWithoutContinueOnError(t *testing.T) {
	_, err := parseIngestImportsOptions([]string{
		"--source", "watcher_import",
		"--failed-manifest", "failed-manifest.json",
	})
	if err == nil {
		t.Fatal("expected failed-manifest validation error")
	}
	if !strings.Contains(err.Error(), "--failed-manifest requires --continue-on-error") {
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
		"status=ok",
		"source=watcher_import",
		"attempted=2",
		"processed=2",
		"failed=0",
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
	if got, want := report.Attempted, 1; got != want {
		t.Fatalf("attempted mismatch: got %d want %d", got, want)
	}
	if got, want := report.Status, "ok"; got != want {
		t.Fatalf("status mismatch: got %q want %q", got, want)
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
	if report.Results[0].Error != nil {
		t.Fatalf("did not expect line error in result: %+v", report.Results[0].Error)
	}
}

func TestRunIngestImportsContinueOnErrorSkipsInvalidLines(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	failedOutputPath := filepath.Join(root, "tmp", "failed.jsonl")
	failedManifestPath := filepath.Join(root, "tmp", "failed-manifest.json")
	input := strings.Join([]string{
		`{"external_id":"watcher:1","type":"discovery","title":"Imported discovery","content":"Useful watcher discovery.","importance":4}`,
		`{"external_id":"watcher:bad","type":"discovery","title":"Broken"`,
	}, "\n")

	var stdout bytes.Buffer
	err := Run(context.Background(), cfg, []string{
		"ingest-imports",
		"--source", "watcher_import",
		"--cwd", root,
		"--repo-remote", "git@github.com:example/codex-mem.git",
		"--continue-on-error",
		"--failed-output", failedOutputPath,
		"--failed-manifest", failedManifestPath,
		"--json",
	}, strings.NewReader(input), &stdout)
	if err != nil {
		t.Fatalf("Run ingest-imports --continue-on-error --json: %v", err)
	}

	var report ingestImportsReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal ingest partial report: %v\n%s", err, stdout.String())
	}
	if got, want := report.Status, "partial"; got != want {
		t.Fatalf("status mismatch: got %q want %q", got, want)
	}
	if got, want := report.Attempted, 2; got != want {
		t.Fatalf("attempted mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedOutput, failedOutputPath; got != want {
		t.Fatalf("failed output mismatch: got %q want %q", got, want)
	}
	if got, want := report.FailedOutputWritten, 1; got != want {
		t.Fatalf("failed output written mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedManifest, failedManifestPath; got != want {
		t.Fatalf("failed manifest mismatch: got %q want %q", got, want)
	}
	if got, want := report.FailedManifestCount, 1; got != want {
		t.Fatalf("failed manifest count mismatch: got %d want %d", got, want)
	}
	if got, want := report.Processed, 1; got != want {
		t.Fatalf("processed mismatch: got %d want %d", got, want)
	}
	if got, want := report.Failed, 1; got != want {
		t.Fatalf("failed mismatch: got %d want %d", got, want)
	}
	if got, want := len(report.Results), 2; got != want {
		t.Fatalf("result count mismatch: got %d want %d", got, want)
	}
	if report.Results[0].ImportID == "" || report.Results[0].Error != nil {
		t.Fatalf("expected first result success, got %+v", report.Results[0])
	}
	if report.Results[1].Error == nil {
		t.Fatalf("expected second result error, got %+v", report.Results[1])
	}
	if got, want := report.Results[1].Error.Code, "ERR_INVALID_INPUT"; got != want {
		t.Fatalf("error code mismatch: got %q want %q", got, want)
	}
	failedOutputBody, err := os.ReadFile(failedOutputPath)
	if err != nil {
		t.Fatalf("ReadFile failed output: %v", err)
	}
	if got, want := strings.TrimSpace(string(failedOutputBody)), `{"external_id":"watcher:bad","type":"discovery","title":"Broken"`; got != want {
		t.Fatalf("failed output mismatch: got %q want %q", got, want)
	}

	var manifest struct {
		Status              string `json:"status"`
		Source              string `json:"source"`
		Input               string `json:"input"`
		FailedOutput        string `json:"failed_output"`
		FailedOutputWritten int    `json:"failed_output_written"`
		FailureCount        int    `json:"failure_count"`
		Failures            []struct {
			Line             int    `json:"line"`
			RawLine          string `json:"raw_line"`
			FailedOutputLine int    `json:"failed_output_line"`
			Error            struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"failures"`
	}
	manifestBody, err := os.ReadFile(failedManifestPath)
	if err != nil {
		t.Fatalf("ReadFile failed manifest: %v", err)
	}
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatalf("unmarshal failed manifest: %v\n%s", err, string(manifestBody))
	}
	if got, want := manifest.Status, "partial"; got != want {
		t.Fatalf("manifest status mismatch: got %q want %q", got, want)
	}
	if got, want := manifest.FailedOutput, failedOutputPath; got != want {
		t.Fatalf("manifest failed output mismatch: got %q want %q", got, want)
	}
	if got, want := manifest.FailedOutputWritten, 1; got != want {
		t.Fatalf("manifest failed output written mismatch: got %d want %d", got, want)
	}
	if got, want := manifest.FailureCount, 1; got != want {
		t.Fatalf("manifest failure count mismatch: got %d want %d", got, want)
	}
	if got, want := len(manifest.Failures), 1; got != want {
		t.Fatalf("manifest failures len mismatch: got %d want %d", got, want)
	}
	if got, want := manifest.Failures[0].Line, 2; got != want {
		t.Fatalf("manifest line mismatch: got %d want %d", got, want)
	}
	if got, want := manifest.Failures[0].RawLine, `{"external_id":"watcher:bad","type":"discovery","title":"Broken"`; got != want {
		t.Fatalf("manifest raw line mismatch: got %q want %q", got, want)
	}
	if got, want := manifest.Failures[0].FailedOutputLine, 1; got != want {
		t.Fatalf("manifest failed output line mismatch: got %d want %d", got, want)
	}
	if got, want := manifest.Failures[0].Error.Code, "ERR_INVALID_INPUT"; got != want {
		t.Fatalf("manifest error code mismatch: got %q want %q", got, want)
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
	if got, want := diagnostics.Audit.ImportRecords, 1; got != want {
		t.Fatalf("import count mismatch: got %d want %d", got, want)
	}
}

func TestRunIngestImportsContinueOnErrorStillFailsWhenNothingSucceeds(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	failedOutputPath := filepath.Join(root, "failed", "failed.jsonl")
	failedManifestPath := filepath.Join(root, "failed", "failed-manifest.json")
	input := `{"external_id":"watcher:bad","type":"discovery","title":"Broken"`

	var stdout bytes.Buffer
	err := Run(context.Background(), cfg, []string{
		"ingest-imports",
		"--source", "watcher_import",
		"--cwd", root,
		"--repo-remote", "git@github.com:example/codex-mem.git",
		"--continue-on-error",
		"--failed-output", failedOutputPath,
		"--failed-manifest", failedManifestPath,
	}, strings.NewReader(input), &stdout)
	if err == nil {
		t.Fatal("expected error when no events import successfully")
	}
	if !strings.Contains(err.Error(), "did not import any events successfully") {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"ingest imports failed",
		"status=failed",
		"failed_output=" + failedOutputPath,
		"failed_output_written=1",
		"failed_manifest=" + failedManifestPath,
		"failed_manifest_count=1",
		"attempted=1",
		"processed=0",
		"failed=1",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("ingest output missing %q:\n%s", fragment, output)
		}
	}

	failedOutputBody, err := os.ReadFile(failedOutputPath)
	if err != nil {
		t.Fatalf("ReadFile failed output: %v", err)
	}
	if got, want := strings.TrimSpace(string(failedOutputBody)), input; got != want {
		t.Fatalf("failed output mismatch: got %q want %q", got, want)
	}

	manifestBody, err := os.ReadFile(failedManifestPath)
	if err != nil {
		t.Fatalf("ReadFile failed manifest: %v", err)
	}
	var manifest struct {
		FailureCount int `json:"failure_count"`
		Failures     []struct {
			RawLine string `json:"raw_line"`
			Error   struct {
				Code string `json:"code"`
			} `json:"error"`
		} `json:"failures"`
	}
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatalf("unmarshal failed manifest: %v\n%s", err, string(manifestBody))
	}
	if got, want := manifest.FailureCount, 1; got != want {
		t.Fatalf("manifest failure count mismatch: got %d want %d", got, want)
	}
	if got, want := len(manifest.Failures), 1; got != want {
		t.Fatalf("manifest failures len mismatch: got %d want %d", got, want)
	}
	if got, want := manifest.Failures[0].RawLine, input; got != want {
		t.Fatalf("manifest raw line mismatch: got %q want %q", got, want)
	}
	if got, want := manifest.Failures[0].Error.Code, "ERR_INVALID_INPUT"; got != want {
		t.Fatalf("manifest error code mismatch: got %q want %q", got, want)
	}
}

func TestAppIngestImportsSupportsEmbeddedIntegration(t *testing.T) {
	root := t.TempDir()
	cfg := ingestTestConfig(root)
	instance, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = instance.Close() }()

	failedOutputPath := filepath.Join(root, "embedded", "failed.jsonl")
	failedManifestPath := filepath.Join(root, "embedded", "failed.json")
	report, err := instance.IngestImports(context.Background(), IngestImportsInput{
		Source:             "watcher_import",
		Reader:             strings.NewReader(strings.Join([]string{`{"external_id":"watcher:ok","type":"discovery","title":"Embedded path","content":"Embedded ingestion path works.","importance":4}`, `{"external_id":"watcher:bad","type":"discovery","title":"Broken"`}, "\n")),
		InputLabel:         "embedded-test",
		CWD:                root,
		RepoRemote:         "git@github.com:example/codex-mem.git",
		Task:               "embedded import test",
		ContinueOnError:    true,
		FailedOutputPath:   failedOutputPath,
		FailedManifestPath: failedManifestPath,
	})
	if err != nil {
		t.Fatalf("App.IngestImports: %v", err)
	}
	if got, want := report.Status, "partial"; got != want {
		t.Fatalf("status mismatch: got %q want %q", got, want)
	}
	if got, want := report.Input, "embedded-test"; got != want {
		t.Fatalf("input label mismatch: got %q want %q", got, want)
	}
	if got, want := report.Processed, 1; got != want {
		t.Fatalf("processed mismatch: got %d want %d", got, want)
	}
	if got, want := report.Failed, 1; got != want {
		t.Fatalf("failed mismatch: got %d want %d", got, want)
	}
	if got, want := report.FailedOutput, failedOutputPath; got != want {
		t.Fatalf("failed output mismatch: got %q want %q", got, want)
	}
	if got, want := report.FailedManifest, failedManifestPath; got != want {
		t.Fatalf("failed manifest mismatch: got %q want %q", got, want)
	}
	if report.Session.ID == "" {
		t.Fatalf("expected session id in report: %+v", report.Session)
	}

	failedOutputBody, err := os.ReadFile(failedOutputPath)
	if err != nil {
		t.Fatalf("ReadFile failed output: %v", err)
	}
	if got, want := strings.TrimSpace(string(failedOutputBody)), `{"external_id":"watcher:bad","type":"discovery","title":"Broken"`; got != want {
		t.Fatalf("failed output mismatch: got %q want %q", got, want)
	}

	manifestBody, err := os.ReadFile(failedManifestPath)
	if err != nil {
		t.Fatalf("ReadFile failed manifest: %v", err)
	}
	var manifest struct {
		Source       string `json:"source"`
		Input        string `json:"input"`
		FailureCount int    `json:"failure_count"`
	}
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		t.Fatalf("unmarshal failed manifest: %v\n%s", err, string(manifestBody))
	}
	if got, want := manifest.Source, "watcher_import"; got != want {
		t.Fatalf("manifest source mismatch: got %q want %q", got, want)
	}
	if got, want := manifest.Input, "embedded-test"; got != want {
		t.Fatalf("manifest input mismatch: got %q want %q", got, want)
	}
	if got, want := manifest.FailureCount, 1; got != want {
		t.Fatalf("manifest failure count mismatch: got %d want %d", got, want)
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
