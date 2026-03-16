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
)

func TestRunDoctorPrintsEffectiveConfigSummary(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
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
			LogAlsoStderr:     true,
		},
		Meta: config.LoadMetadata{
			ConfigDir:      filepath.Join(root, "configs"),
			ConfigFilePath: filepath.Join(root, "configs", "codex-mem.json"),
			ConfigFileUsed: filepath.Join(root, "configs", "codex-mem.json"),
			LogDir:         filepath.Join(root, "logs"),
		},
	}

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{"doctor"}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run doctor: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"doctor ok",
		"config_precedence=defaults<config_file<environment",
		"config_file_used=" + cfg.Meta.ConfigFileUsed,
		"database=" + cfg.File.DatabasePath,
		"sqlite_driver=" + cfg.File.SQLiteDriver,
		"journal_mode=wal",
		"foreign_keys=true",
		"required_schema_ok=true",
		"fts_ready=true",
		"migrations_pending=0",
		"latest_migration_applied=005_import_records.sql",
		"note_records=0",
		"handoff_records=0",
		"import_records=0",
		"note_source_invalid=0",
		"note_provenance_ready=true",
		"exclusion_audit_ready=true",
		"import_audit_ready=true",
		"log_file=" + cfg.File.LogFilePath,
		"log_stderr=true",
		"mcp_transport=stdio",
		"mcp_tool_count=11",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("doctor output missing %q:\n%s", fragment, output)
		}
	}
}

func TestRunDoctorReportsMissingConfigFileAsNone(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
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

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{"doctor"}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run doctor: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "config_file_used=none") {
		t.Fatalf("expected config_file_used=none in output:\n%s", output)
	}
	if !strings.Contains(output, "log_stderr=false") {
		t.Fatalf("expected log_stderr=false in output:\n%s", output)
	}
	if !strings.Contains(output, "required_schema_ok=true") {
		t.Fatalf("expected required_schema_ok=true in output:\n%s", output)
	}
	if !strings.Contains(output, "note_records=0") {
		t.Fatalf("expected note_records=0 in output:\n%s", output)
	}
	if !strings.Contains(output, "import_records=0") {
		t.Fatalf("expected import_records=0 in output:\n%s", output)
	}
}

func TestRunDefaultsToDoctorWhenNoCommandIsProvided(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
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
			LogAlsoStderr:     true,
		},
		Meta: config.LoadMetadata{
			ConfigDir:      filepath.Join(root, "configs"),
			ConfigFilePath: filepath.Join(root, "configs", "codex-mem.json"),
			LogDir:         filepath.Join(root, "logs"),
		},
	}

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, nil, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run default doctor: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "doctor ok") {
		t.Fatalf("expected doctor output when no command is provided:\n%s", output)
	}
	if !strings.Contains(output, "config_file_used=none") {
		t.Fatalf("expected default doctor path to report config_file_used=none:\n%s", output)
	}
}

func TestRunDoctorPrintsJSONDiagnostics(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
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

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{"doctor", "--json"}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run doctor --json: %v", err)
	}

	var report struct {
		Status string `json:"status"`
		Config struct {
			Precedence     string  `json:"precedence"`
			ConfigFileUsed *string `json:"config_file_used"`
			Database       string  `json:"database"`
		} `json:"config"`
		Runtime struct {
			BusyTimeoutMS    int64  `json:"busy_timeout_ms"`
			JournalMode      string `json:"journal_mode"`
			RequiredSchemaOK bool   `json:"required_schema_ok"`
			FTSReady         bool   `json:"fts_ready"`
		} `json:"runtime"`
		Migrations struct {
			Pending       int     `json:"pending"`
			LatestApplied *string `json:"latest_applied"`
		} `json:"migrations"`
		Audit struct {
			NoteRecords         int  `json:"note_records"`
			ImportRecords       int  `json:"import_records"`
			NoteProvenanceReady bool `json:"note_provenance_ready"`
			ExclusionAuditReady bool `json:"exclusion_audit_ready"`
			ImportAuditReady    bool `json:"import_audit_ready"`
		} `json:"audit"`
		Logging struct {
			LogStderr bool `json:"log_stderr"`
		} `json:"logging"`
		MCP struct {
			Transport string `json:"transport"`
			ToolCount int    `json:"tool_count"`
		} `json:"mcp"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal doctor JSON: %v\n%s", err, stdout.String())
	}

	if report.Status != "ok" {
		t.Fatalf("status mismatch: got %q", report.Status)
	}
	if report.Config.Precedence != "defaults<config_file<environment" {
		t.Fatalf("config precedence mismatch: %q", report.Config.Precedence)
	}
	if report.Config.ConfigFileUsed != nil {
		t.Fatalf("expected null config_file_used, got %q", *report.Config.ConfigFileUsed)
	}
	if report.Config.Database != cfg.File.DatabasePath {
		t.Fatalf("database mismatch: got %q want %q", report.Config.Database, cfg.File.DatabasePath)
	}
	if got, want := report.Runtime.BusyTimeoutMS, int64(5000); got != want {
		t.Fatalf("busy timeout mismatch: got %d want %d", got, want)
	}
	if report.Runtime.JournalMode != "wal" {
		t.Fatalf("journal mode mismatch: got %q", report.Runtime.JournalMode)
	}
	if !report.Runtime.RequiredSchemaOK || !report.Runtime.FTSReady {
		t.Fatalf("expected ready runtime diagnostics, got %+v", report.Runtime)
	}
	if report.Migrations.Pending != 0 {
		t.Fatalf("expected no pending migrations, got %d", report.Migrations.Pending)
	}
	if report.Migrations.LatestApplied == nil || *report.Migrations.LatestApplied != "005_import_records.sql" {
		t.Fatalf("unexpected latest applied migration: %+v", report.Migrations.LatestApplied)
	}
	if report.Audit.NoteRecords != 0 || report.Audit.ImportRecords != 0 || !report.Audit.NoteProvenanceReady || !report.Audit.ExclusionAuditReady || !report.Audit.ImportAuditReady {
		t.Fatalf("unexpected audit diagnostics: %+v", report.Audit)
	}
	if report.Logging.LogStderr {
		t.Fatal("expected log_stderr=false")
	}
	if report.MCP.Transport != "stdio" || report.MCP.ToolCount != 11 {
		t.Fatalf("unexpected mcp diagnostics: %+v", report.MCP)
	}
}

func TestRunDoctorRejectsUnknownFlag(t *testing.T) {
	cfg := config.Config{
		File: config.FileConfig{
			DatabasePath:      ":memory:",
			DefaultSystemName: "codex-mem",
			SQLiteDriver:      "sqlite",
			BusyTimeout:       5 * time.Second,
			JournalMode:       "WAL",
			LogFilePath:       filepath.Join(t.TempDir(), "codex-mem.log"),
			LogMaxSizeMB:      20,
			LogMaxBackups:     10,
			LogMaxAgeDays:     30,
			LogCompress:       true,
			LogAlsoStderr:     true,
		},
	}

	var stdout bytes.Buffer
	err := Run(context.Background(), cfg, []string{"doctor", "--yaml"}, strings.NewReader(""), &stdout)
	if err == nil {
		t.Fatal("expected unknown doctor flag error")
	}
	if !strings.Contains(err.Error(), `unknown doctor flag "--yaml"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
