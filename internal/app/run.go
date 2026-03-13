package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"codex-mem/internal/config"
	"codex-mem/internal/db"
	"codex-mem/internal/mcp"
)

func Run(ctx context.Context, cfg config.Config, args []string, stdin io.Reader, stdout io.Writer) error {
	logger := slog.Default().With(
		"component", "cli",
		"config_dir", cfg.Meta.ConfigDir,
		"config_file", cfg.Meta.ConfigFilePath,
		"config_file_used", cfg.Meta.ConfigFileUsed,
	)
	command := "doctor"
	if len(args) > 0 {
		command = args[0]
	}
	logger.Info("starting command", "command", command)

	switch command {
	case "migrate":
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer instance.Close()
		logger.Info("migrations applied", "database", cfg.File.DatabasePath)
		_, err = fmt.Fprintf(stdout, "migrations applied successfully to %s\n", cfg.File.DatabasePath)
		return err
	case "doctor":
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer instance.Close()
		if err := db.HealthCheck(ctx, instance.DB); err != nil {
			return err
		}
		runtimeDiagnostics, err := db.InspectRuntime(ctx, instance.DB)
		if err != nil {
			return err
		}
		mcpServer := mcp.NewServer(instance.Handlers)
		logger.Info("doctor check passed",
			"database", cfg.File.DatabasePath,
			"system", cfg.File.DefaultSystemName,
			"config_file_used", doctorConfigFileUsed(cfg),
			"migrations_applied", runtimeDiagnostics.Migrations.Applied,
			"required_schema_ok", runtimeDiagnostics.RequiredSchemaOK,
			"fts_ready", runtimeDiagnostics.FTSReady,
		)
		_, err = io.WriteString(stdout, formatDoctorReport(cfg, runtimeDiagnostics, mcpServer.ToolCount()))
		return err
	case "serve":
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer instance.Close()
		logger.Info("starting MCP stdio server")
		return mcp.NewServer(instance.Handlers).Serve(ctx, stdin, stdout)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func doctorConfigFileUsed(cfg config.Config) string {
	if cfg.Meta.ConfigFileUsed != "" {
		return cfg.Meta.ConfigFileUsed
	}
	return "none"
}

func formatDoctorReport(cfg config.Config, runtime db.RuntimeDiagnostics, toolCount int) string {
	lines := []string{
		"doctor ok",
		"config_precedence=defaults<config_file<environment",
		fmt.Sprintf("config_dir=%s", cfg.Meta.ConfigDir),
		fmt.Sprintf("config_file=%s", cfg.Meta.ConfigFilePath),
		fmt.Sprintf("config_file_used=%s", doctorConfigFileUsed(cfg)),
		fmt.Sprintf("database=%s", cfg.File.DatabasePath),
		fmt.Sprintf("default_system=%s", cfg.File.DefaultSystemName),
		fmt.Sprintf("sqlite_driver=%s", cfg.File.SQLiteDriver),
		fmt.Sprintf("busy_timeout=%s", runtime.BusyTimeout),
		fmt.Sprintf("journal_mode=%s", runtime.JournalMode),
		fmt.Sprintf("foreign_keys=%t", runtime.ForeignKeysEnabled),
		fmt.Sprintf("required_schema_ok=%t", runtime.RequiredSchemaOK),
		fmt.Sprintf("fts_ready=%t", runtime.FTSReady),
		fmt.Sprintf("migrations_available=%d", runtime.Migrations.Available),
		fmt.Sprintf("migrations_applied=%d", runtime.Migrations.Applied),
		fmt.Sprintf("migrations_pending=%d", runtime.Migrations.Pending),
		fmt.Sprintf("latest_migration_available=%s", emptyAsNone(runtime.Migrations.LatestAvailable)),
		fmt.Sprintf("latest_migration_applied=%s", emptyAsNone(runtime.Migrations.LatestApplied)),
		fmt.Sprintf("note_records=%d", runtime.Audit.NoteRecords),
		fmt.Sprintf("handoff_records=%d", runtime.Audit.HandoffRecords),
		fmt.Sprintf("note_source_codex_explicit=%d", runtime.Audit.NotesCodexExplicit),
		fmt.Sprintf("note_source_watcher_import=%d", runtime.Audit.NotesWatcherImport),
		fmt.Sprintf("note_source_relay_import=%d", runtime.Audit.NotesRelayImport),
		fmt.Sprintf("note_source_recovery_generated=%d", runtime.Audit.NotesRecoveryGenerated),
		fmt.Sprintf("note_source_invalid=%d", runtime.Audit.NotesInvalidSource),
		fmt.Sprintf("excluded_notes=%d", runtime.Audit.ExcludedNotes),
		fmt.Sprintf("excluded_handoffs=%d", runtime.Audit.ExcludedHandoffs),
		fmt.Sprintf("excluded_notes_missing_reason=%d", runtime.Audit.ExcludedNotesMissingReason),
		fmt.Sprintf("excluded_handoffs_missing_reason=%d", runtime.Audit.ExcludedHandoffsMissingReason),
		fmt.Sprintf("recovery_handoffs=%d", runtime.Audit.RecoveryHandoffs),
		fmt.Sprintf("open_handoffs=%d", runtime.Audit.OpenHandoffs),
		fmt.Sprintf("note_provenance_ready=%t", runtime.Audit.NoteProvenanceReady),
		fmt.Sprintf("exclusion_audit_ready=%t", runtime.Audit.ExclusionAuditReady),
		fmt.Sprintf("log_file=%s", cfg.File.LogFilePath),
		fmt.Sprintf("log_level=%s", strings.ToLower(cfg.File.LogLevel.String())),
		fmt.Sprintf("log_max_size_mb=%d", cfg.File.LogMaxSizeMB),
		fmt.Sprintf("log_max_backups=%d", cfg.File.LogMaxBackups),
		fmt.Sprintf("log_max_age_days=%d", cfg.File.LogMaxAgeDays),
		fmt.Sprintf("log_compress=%t", cfg.File.LogCompress),
		fmt.Sprintf("log_stderr=%t", cfg.File.LogAlsoStderr),
		"mcp_transport=stdio",
		fmt.Sprintf("mcp_tool_count=%d", toolCount),
	}
	return strings.Join(lines, "\n") + "\n"
}

func emptyAsNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "none"
	}
	return value
}
