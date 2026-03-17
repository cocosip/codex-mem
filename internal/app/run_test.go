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
	"codex-mem/internal/domain/common"
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
		"follow_imports_health_file=" + filepath.Join(cfg.Meta.LogDir, "follow-imports.health.json"),
		"follow_imports_health_present=false",
		"follow_imports_continuous=false",
		"follow_imports_poll_interval_seconds=0",
		"follow_imports_snapshot_age_seconds=0",
		"follow_imports_health_stale=false",
		"mcp_transport=stdio",
		"mcp_tool_count=11",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("doctor output missing %q:\n%s", fragment, output)
		}
	}
}

func TestRunDoctorIncludesFollowImportsHealthSnapshot(t *testing.T) {
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

	snapshot := followImportsHealthSnapshot{
		Status:              "partial",
		UpdatedAt:           time.Now().UTC(),
		Source:              "watcher_import",
		InputCount:          2,
		Continuous:          true,
		PollIntervalSeconds: 5,
		RequestedWatchMode:  "auto",
		ActiveWatchMode:     "notify",
		WatchFallbacks:      1,
		WatchTransitions:    3,
		LastFallbackReason:  "watcher_error",
		WatchPollCatchups:   4,
		WatchCatchupBytes:   256,
		Warnings: []common.Warning{{
			Code:    common.WarnFollowImportsPollCatchup,
			Message: "notify mode required poll catchup 4 times and 256 bytes so far",
		}},
	}
	if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, snapshot); err != nil {
		t.Fatalf("saveFollowImportsHealthSnapshot: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{"doctor"}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run doctor: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"follow_imports_health_present=true",
		"follow_imports_status=partial",
		"follow_imports_source=watcher_import",
		"follow_imports_input_count=2",
		"follow_imports_continuous=true",
		"follow_imports_poll_interval_seconds=5",
		"follow_imports_health_stale=false",
		"follow_imports_watch_poll_catchups=4",
		"follow_imports_watch_poll_catchup_bytes=256",
		"follow_imports_warnings=1",
		"follow_imports_warning_1_code=WARN_FOLLOW_IMPORTS_POLL_CATCHUP",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("doctor output missing %q:\n%s", fragment, output)
		}
	}
}

func TestRunDoctorFlagsStaleFollowImportsHealthSnapshot(t *testing.T) {
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

	snapshot := followImportsHealthSnapshot{
		Status:              "ok",
		UpdatedAt:           time.Now().UTC().Add(-2 * time.Minute),
		Source:              "watcher_import",
		InputCount:          1,
		Continuous:          true,
		PollIntervalSeconds: 5,
		RequestedWatchMode:  "auto",
		ActiveWatchMode:     "notify",
	}
	if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, snapshot); err != nil {
		t.Fatalf("saveFollowImportsHealthSnapshot: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{"doctor"}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run doctor: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"follow_imports_health_present=true",
		"follow_imports_health_pruned=false",
		"follow_imports_health_prune_reason=none",
		"follow_imports_health_stale=true",
		"follow_imports_warnings=1",
		"follow_imports_warning_1_code=WARN_FOLLOW_IMPORTS_HEALTH_STALE",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("doctor output missing %q:\n%s", fragment, output)
		}
	}
}

func TestRunDoctorPrunesStaleFollowImportsHealthSnapshotWhenRequested(t *testing.T) {
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

	snapshot := followImportsHealthSnapshot{
		Status:              "ok",
		UpdatedAt:           time.Now().UTC().Add(-2 * time.Minute),
		Source:              "watcher_import",
		InputCount:          1,
		Continuous:          true,
		PollIntervalSeconds: 5,
		RequestedWatchMode:  "auto",
		ActiveWatchMode:     "notify",
	}
	if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, snapshot); err != nil {
		t.Fatalf("saveFollowImportsHealthSnapshot: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{"doctor", "--prune-stale-follow-health"}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run doctor --prune-stale-follow-health: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"follow_imports_health_present=false",
		"follow_imports_health_pruned=true",
		"follow_imports_health_prune_reason=stale",
		"follow_imports_health_stale=false",
		"follow_imports_warnings=0",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("doctor output missing %q:\n%s", fragment, output)
		}
	}

	if _, err := os.Stat(followImportsHealthPath(cfg.Meta.LogDir)); !os.IsNotExist(err) {
		t.Fatalf("expected stale follow health snapshot to be removed, stat err=%v", err)
	}
}

func TestRunDoctorDoesNotPruneFreshFollowImportsHealthSnapshot(t *testing.T) {
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

	snapshot := followImportsHealthSnapshot{
		Status:              "partial",
		UpdatedAt:           time.Now().UTC(),
		Source:              "watcher_import",
		InputCount:          1,
		Continuous:          true,
		PollIntervalSeconds: 5,
		RequestedWatchMode:  "auto",
		ActiveWatchMode:     "notify",
	}
	if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, snapshot); err != nil {
		t.Fatalf("saveFollowImportsHealthSnapshot: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{"doctor", "--prune-stale-follow-health"}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run doctor --prune-stale-follow-health: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"follow_imports_health_present=true",
		"follow_imports_health_pruned=false",
		"follow_imports_health_prune_reason=none",
		"follow_imports_health_stale=false",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("doctor output missing %q:\n%s", fragment, output)
		}
	}

	if _, err := os.Stat(followImportsHealthPath(cfg.Meta.LogDir)); err != nil {
		t.Fatalf("expected fresh follow health snapshot to remain, stat err=%v", err)
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
		Follow struct {
			HealthFile          string           `json:"health_file"`
			HealthPresent       bool             `json:"health_present"`
			LastUpdatedAt       *time.Time       `json:"last_updated_at"`
			Continuous          bool             `json:"continuous"`
			PollIntervalSeconds int64            `json:"poll_interval_seconds"`
			SnapshotAgeSeconds  int64            `json:"snapshot_age_seconds"`
			HealthStale         bool             `json:"health_stale"`
			WatchPollCatchups   int              `json:"watch_poll_catchups"`
			WatchCatchupBytes   int              `json:"watch_poll_catchup_bytes"`
			Warnings            []common.Warning `json:"warnings"`
		} `json:"follow_imports"`
		MCP struct {
			Transport string `json:"transport"`
			ToolCount int    `json:"tool_count"`
		} `json:"mcp"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal doctor JSON: %v\n%s", err, stdout.String())
	}

	assertDoctorJSONDiagnostics(t, cfg, report)
}

func assertDoctorJSONDiagnostics(t *testing.T, cfg config.Config, report struct {
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
	Follow struct {
		HealthFile          string           `json:"health_file"`
		HealthPresent       bool             `json:"health_present"`
		LastUpdatedAt       *time.Time       `json:"last_updated_at"`
		Continuous          bool             `json:"continuous"`
		PollIntervalSeconds int64            `json:"poll_interval_seconds"`
		SnapshotAgeSeconds  int64            `json:"snapshot_age_seconds"`
		HealthStale         bool             `json:"health_stale"`
		WatchPollCatchups   int              `json:"watch_poll_catchups"`
		WatchCatchupBytes   int              `json:"watch_poll_catchup_bytes"`
		Warnings            []common.Warning `json:"warnings"`
	} `json:"follow_imports"`
	MCP struct {
		Transport string `json:"transport"`
		ToolCount int    `json:"tool_count"`
	} `json:"mcp"`
}) {
	t.Helper()

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
	if report.Follow.HealthFile == "" || report.Follow.HealthPresent {
		t.Fatalf("unexpected follow diagnostics: %+v", report.Follow)
	}
	if report.Follow.LastUpdatedAt != nil || report.Follow.Continuous || report.Follow.PollIntervalSeconds != 0 || report.Follow.SnapshotAgeSeconds != 0 || report.Follow.HealthStale || report.Follow.WatchPollCatchups != 0 || report.Follow.WatchCatchupBytes != 0 || len(report.Follow.Warnings) != 0 {
		t.Fatalf("expected empty follow health snapshot, got %+v", report.Follow)
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

func TestParseDoctorOptionsEnablesPruneStaleFollowHealth(t *testing.T) {
	options, err := parseDoctorOptions([]string{"--prune-stale-follow-health"})
	if err != nil {
		t.Fatalf("parseDoctorOptions: %v", err)
	}
	if !options.PruneStaleFollowHealth {
		t.Fatal("expected prune-stale-follow-health option to be enabled")
	}
}

func TestRunCleanupFollowImportsPrunesArtifactsAndStaleHealth(t *testing.T) {
	root := t.TempDir()
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

	staleSnapshot := followImportsHealthSnapshot{
		Status:              "ok",
		UpdatedAt:           time.Now().UTC().Add(-2 * time.Minute),
		Source:              "watcher_import",
		InputCount:          1,
		Continuous:          true,
		PollIntervalSeconds: 5,
	}
	if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, staleSnapshot); err != nil {
		t.Fatalf("saveFollowImportsHealthSnapshot: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{
		"cleanup-follow-imports",
		"--input", inputPath,
		"--prune-state",
		"--failed-output", failedOutputBase,
		"--prune-failed-output",
		"--failed-manifest", failedManifestBase,
		"--prune-failed-manifest",
		"--prune-stale-follow-health",
	}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run cleanup-follow-imports: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"cleanup follow-imports ok",
		"state_files_removed=1",
		"failed_output_removed=1",
		"failed_manifest_removed=1",
		"follow_health_pruned=true",
		"follow_health_prune_reason=stale",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("cleanup output missing %q:\n%s", fragment, output)
		}
	}

	for _, removed := range []string{statePath, failedOutput, failedManifest, followImportsHealthPath(cfg.Meta.LogDir)} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, stat err=%v", removed, err)
		}
	}
}

func TestRunCleanupFollowImportsDryRunReportsAgeFilteredPreview(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Meta: config.LoadMetadata{
			LogDir: filepath.Join(root, "logs"),
		},
	}

	now := time.Now().UTC()
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
	oldTime := now.Add(-2 * time.Hour)
	newTime := now.Add(-20 * time.Minute)
	for _, path := range []string{statePath, oldFailedOutput, oldFailedManifest} {
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatalf("Chtimes old %s: %v", path, err)
		}
	}
	for _, path := range []string{newFailedOutput, newFailedManifest} {
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

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{
		"cleanup-follow-imports",
		"--input", inputPath,
		"--prune-state",
		"--failed-output", failedOutputBase,
		"--prune-failed-output",
		"--failed-manifest", failedManifestBase,
		"--prune-failed-manifest",
		"--prune-stale-follow-health",
		"--older-than", "1h",
		"--dry-run",
	}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run cleanup-follow-imports dry-run: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"dry_run=true",
		"older_than_seconds=3600",
		"state_files_matched=1",
		"state_files_removed=0",
		"failed_output_matched=1",
		"failed_output_skipped_by_age=1",
		"failed_manifest_matched=1",
		"failed_manifest_skipped_by_age=1",
		"follow_health_would_prune=true",
		"follow_health_pruned=false",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("cleanup dry-run output missing %q:\n%s", fragment, output)
		}
	}
}

func TestRunCleanupFollowImportsReportsPatternFilters(t *testing.T) {
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
	for _, path := range []string{inputA, inputB, stateA, stateB, failedOutputA, failedOutputB, failedManifestA, failedManifestB} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", path, err)
		}
	}

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{
		"cleanup-follow-imports",
		"--input", inputA,
		"--input", inputB,
		"--prune-state",
		"--failed-output", failedOutputBase,
		"--prune-failed-output",
		"--failed-manifest", failedManifestBase,
		"--prune-failed-manifest",
		"--include", "*events-a*",
		"--exclude", "*.43-84.*",
	}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run cleanup-follow-imports with patterns: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"include_patterns=1",
		"exclude_patterns=1",
		"include_pattern_1=*events-a*",
		"exclude_pattern_1=*.43-84.*",
		"state_files_skipped_by_pattern=1",
		"failed_output_skipped_by_pattern=1",
		"failed_manifest_skipped_by_pattern=1",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("cleanup pattern output missing %q:\n%s", fragment, output)
		}
	}
}

func TestRunCleanupFollowImportsReportsRetentionProfile(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Meta: config.LoadMetadata{
			LogDir: filepath.Join(root, "logs"),
		},
	}

	failedOutputBase := filepath.Join(root, "failed", "failed.jsonl")
	failedOutput := filepath.Join(root, "failed", "failed.0-42.jsonl")
	if err := os.MkdirAll(filepath.Dir(failedOutput), 0o755); err != nil {
		t.Fatalf("MkdirAll failed output dir: %v", err)
	}
	if err := os.WriteFile(failedOutput, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile failed output: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{
		"cleanup-follow-imports",
		"--failed-output", failedOutputBase,
		"--prune-failed-output",
		"--retention-profile", "daily",
		"--dry-run",
	}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run cleanup-follow-imports with retention profile: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"retention_profile=daily",
		"older_than_seconds=86400",
		"dry_run=true",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("cleanup retention-profile output missing %q:\n%s", fragment, output)
		}
	}
}

func TestRunCleanupFollowImportsFailIfMatchedReturnsErrorAfterWritingReport(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Meta: config.LoadMetadata{
			LogDir: filepath.Join(root, "logs"),
		},
	}

	inputPath := filepath.Join(root, "events.jsonl")
	statePath := inputPath + ".offset.json"
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatalf("MkdirAll state dir: %v", err)
	}
	if err := os.WriteFile(statePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile state file: %v", err)
	}

	var stdout bytes.Buffer
	err := Run(context.Background(), cfg, []string{
		"cleanup-follow-imports",
		"--input", inputPath,
		"--prune-state",
		"--dry-run",
		"--fail-if-matched",
	}, strings.NewReader(""), &stdout)
	if err == nil {
		t.Fatal("expected fail-if-matched error")
	}
	if !strings.Contains(err.Error(), "found matching artifacts") {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"dry_run=true",
		"fail_if_matched=true",
		"match_found=true",
		"state_files_matched=1",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("cleanup fail-if-matched output missing %q:\n%s", fragment, output)
		}
	}
}

func TestRunCleanupFollowImportsFailIfMatchedPassesWhenNothingMatches(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Meta: config.LoadMetadata{
			LogDir: filepath.Join(root, "logs"),
		},
	}

	inputPath := filepath.Join(root, "events.jsonl")

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{
		"cleanup-follow-imports",
		"--input", inputPath,
		"--prune-state",
		"--dry-run",
		"--fail-if-matched",
	}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run cleanup-follow-imports --fail-if-matched without matches: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"fail_if_matched=true",
		"match_found=false",
		"state_files_missing=1",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("cleanup no-match output missing %q:\n%s", fragment, output)
		}
	}
}

func TestRunAuditFollowImportsReportsPendingArtifacts(t *testing.T) {
	root := t.TempDir()
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
	oldTime := time.Now().UTC().Add(-48 * time.Hour)
	for _, path := range []string{statePath, failedOutput, failedManifest} {
		if err := os.Chtimes(path, oldTime, oldTime); err != nil {
			t.Fatalf("Chtimes %s: %v", path, err)
		}
	}

	staleSnapshot := followImportsHealthSnapshot{
		Status:              "partial",
		UpdatedAt:           time.Now().UTC().Add(-2 * time.Minute),
		Source:              "watcher_import",
		InputCount:          1,
		Continuous:          true,
		PollIntervalSeconds: 5,
	}
	if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, staleSnapshot); err != nil {
		t.Fatalf("saveFollowImportsHealthSnapshot: %v", err)
	}

	var stdout bytes.Buffer
	if err := Run(context.Background(), cfg, []string{
		"audit-follow-imports",
		"--input", inputPath,
		"--check-state",
		"--failed-output", failedOutputBase,
		"--check-failed-output",
		"--failed-manifest", failedManifestBase,
		"--check-failed-manifest",
		"--check-follow-health",
		"--retention-profile", "daily",
	}, strings.NewReader(""), &stdout); err != nil {
		t.Fatalf("Run audit-follow-imports: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"audit follow-imports ok",
		"match_found=true",
		"retention_profile=daily",
		"state_files_matched=1",
		"failed_output_matched=1",
		"failed_manifest_matched=1",
		"follow_health_present=true",
		"follow_health_stale=true",
		"follow_health_warning_1_code=WARN_FOLLOW_IMPORTS_HEALTH_STALE",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("audit output missing %q:\n%s", fragment, output)
		}
	}

	for _, preserved := range []string{statePath, failedOutput, failedManifest, followImportsHealthPath(cfg.Meta.LogDir)} {
		if _, err := os.Stat(preserved); err != nil {
			t.Fatalf("expected %s to remain after audit, stat err=%v", preserved, err)
		}
	}
}

func TestRunAuditFollowImportsFailIfMatchedReturnsErrorAfterWritingReport(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		Meta: config.LoadMetadata{
			LogDir: filepath.Join(root, "logs"),
		},
	}

	inputPath := filepath.Join(root, "events.jsonl")
	statePath := inputPath + ".offset.json"
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		t.Fatalf("MkdirAll state dir: %v", err)
	}
	if err := os.WriteFile(statePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile state file: %v", err)
	}

	var stdout bytes.Buffer
	err := Run(context.Background(), cfg, []string{
		"audit-follow-imports",
		"--input", inputPath,
		"--check-state",
		"--fail-if-matched",
	}, strings.NewReader(""), &stdout)
	if err == nil {
		t.Fatal("expected fail-if-matched error")
	}
	if !strings.Contains(err.Error(), "found matching artifacts") {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	for _, fragment := range []string{
		"fail_if_matched=true",
		"match_found=true",
		"state_files_matched=1",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("audit fail-if-matched output missing %q:\n%s", fragment, output)
		}
	}
}

func TestAuditFollowImportsExampleOutputsStayInSync(t *testing.T) {
	for _, fixture := range auditFollowImportsExampleFixtures() {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			assertAuditFollowImportsExampleOutput(t, filepath.Join("testdata", fixture.RelativePath), fixture.JSON, fixture.Report)
		})
	}
}

func TestCleanupFollowImportsExampleOutputsStayInSync(t *testing.T) {
	for _, fixture := range cleanupFollowImportsExampleFixtures() {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			assertCleanupFollowImportsExampleOutput(t, filepath.Join("testdata", fixture.RelativePath), fixture.JSON, fixture.Report)
		})
	}
}

func TestRefreshCleanupFollowImportsExampleFixtures(t *testing.T) {
	names := followImportsExampleRefreshSelection(t, "CODEX_MEM_REFRESH_CLEANUP_EXAMPLES")
	writtenPaths, err := writeCleanupFollowImportsExampleFixtures("testdata", names)
	if err != nil {
		t.Fatalf("writeCleanupFollowImportsExampleFixtures: %v", err)
	}
	if len(writtenPaths) == 0 {
		t.Fatal("expected at least one cleanup fixture to be written")
	}
}

func TestRefreshAuditFollowImportsExampleFixtures(t *testing.T) {
	names := followImportsExampleRefreshSelection(t, "CODEX_MEM_REFRESH_AUDIT_EXAMPLES")
	writtenPaths, err := writeAuditFollowImportsExampleFixtures("testdata", names)
	if err != nil {
		t.Fatalf("writeAuditFollowImportsExampleFixtures: %v", err)
	}
	if len(writtenPaths) == 0 {
		t.Fatal("expected at least one audit fixture to be written")
	}
}

func followImportsExampleRefreshSelection(t *testing.T, envKey string) []string {
	t.Helper()

	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		t.Skipf("%s is not set", envKey)
		return nil
	}
	switch strings.ToLower(raw) {
	case "1", "true", "all":
		return nil
	}
	names, err := parseFollowImportsExampleNames(raw)
	if err != nil {
		t.Fatalf("parse example selection from %s: %v", envKey, err)
	}
	return names
}

func assertCleanupFollowImportsExampleOutput(t *testing.T, path string, jsonOutput bool, report cleanupFollowImportsReport) {
	t.Helper()

	body, err := renderCleanupFollowImportsExample(report, jsonOutput)
	if err != nil {
		t.Fatalf("render cleanup-follow-imports example: %v", err)
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if !bytes.Equal(body, expected) {
		t.Fatalf("cleanup example mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", path, string(body), string(expected))
	}
}

func assertAuditFollowImportsExampleOutput(t *testing.T, path string, jsonOutput bool, report auditFollowImportsReport) {
	t.Helper()

	body, err := renderAuditFollowImportsExample(report, jsonOutput)
	if err != nil {
		t.Fatalf("render audit-follow-imports example: %v", err)
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if !bytes.Equal(body, expected) {
		t.Fatalf("audit example mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", path, string(body), string(expected))
	}
}
