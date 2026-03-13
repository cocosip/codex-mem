package app

import (
	"bytes"
	"context"
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
		"latest_migration_applied=004_searchability_controls.sql",
		"note_records=0",
		"handoff_records=0",
		"note_source_invalid=0",
		"note_provenance_ready=true",
		"exclusion_audit_ready=true",
		"log_file=" + cfg.File.LogFilePath,
		"log_stderr=true",
		"mcp_transport=stdio",
		"mcp_tool_count=9",
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
}
