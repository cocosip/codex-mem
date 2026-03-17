package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"codex-mem/internal/config"
	"codex-mem/internal/db"
	"codex-mem/internal/domain/common"
)

type doctorOptions struct {
	JSON                   bool
	PruneStaleFollowHealth bool
}

type doctorReport struct {
	Status     string                 `json:"status"`
	Config     doctorConfigReport     `json:"config"`
	Runtime    doctorRuntimeReport    `json:"runtime"`
	Migrations doctorMigrationsReport `json:"migrations"`
	Audit      doctorAuditReport      `json:"audit"`
	Logging    doctorLoggingReport    `json:"logging"`
	Follow     doctorFollowReport     `json:"follow_imports"`
	MCP        doctorMCPReport        `json:"mcp"`
}

type doctorConfigReport struct {
	Precedence     string  `json:"precedence"`
	ConfigDir      string  `json:"config_dir"`
	ConfigFile     string  `json:"config_file"`
	ConfigFileUsed *string `json:"config_file_used"`
	Database       string  `json:"database"`
	DefaultSystem  string  `json:"default_system"`
	SQLiteDriver   string  `json:"sqlite_driver"`
}

type doctorRuntimeReport struct {
	BusyTimeoutMS    int64  `json:"busy_timeout_ms"`
	JournalMode      string `json:"journal_mode"`
	ForeignKeys      bool   `json:"foreign_keys"`
	RequiredSchemaOK bool   `json:"required_schema_ok"`
	FTSReady         bool   `json:"fts_ready"`
}

type doctorMigrationsReport struct {
	Available       int     `json:"available"`
	Applied         int     `json:"applied"`
	Pending         int     `json:"pending"`
	LatestAvailable *string `json:"latest_available"`
	LatestApplied   *string `json:"latest_applied"`
}

type doctorAuditReport struct {
	NoteRecords                    int  `json:"note_records"`
	HandoffRecords                 int  `json:"handoff_records"`
	ImportRecords                  int  `json:"import_records"`
	NotesCodexExplicit             int  `json:"notes_codex_explicit"`
	NotesWatcherImport             int  `json:"notes_watcher_import"`
	NotesRelayImport               int  `json:"notes_relay_import"`
	NotesRecoveryGenerated         int  `json:"notes_recovery_generated"`
	NotesInvalidSource             int  `json:"notes_invalid_source"`
	ImportsWatcherImport           int  `json:"imports_watcher_import"`
	ImportsRelayImport             int  `json:"imports_relay_import"`
	SuppressedImports              int  `json:"suppressed_imports"`
	SuppressedImportsMissingReason int  `json:"suppressed_imports_missing_reason"`
	ImportsMissingDedupeKey        int  `json:"imports_missing_dedupe_key"`
	ImportsLinkedMemory            int  `json:"imports_linked_memory"`
	ExcludedNotes                  int  `json:"excluded_notes"`
	ExcludedHandoffs               int  `json:"excluded_handoffs"`
	ExcludedNotesMissingReason     int  `json:"excluded_notes_missing_reason"`
	ExcludedHandoffsMissingReason  int  `json:"excluded_handoffs_missing_reason"`
	RecoveryHandoffs               int  `json:"recovery_handoffs"`
	OpenHandoffs                   int  `json:"open_handoffs"`
	NoteProvenanceReady            bool `json:"note_provenance_ready"`
	ExclusionAuditReady            bool `json:"exclusion_audit_ready"`
	ImportAuditReady               bool `json:"import_audit_ready"`
}

type doctorLoggingReport struct {
	LogFile       string `json:"log_file"`
	LogLevel      string `json:"log_level"`
	LogMaxSizeMB  int    `json:"log_max_size_mb"`
	LogMaxBackups int    `json:"log_max_backups"`
	LogMaxAgeDays int    `json:"log_max_age_days"`
	LogCompress   bool   `json:"log_compress"`
	LogStderr     bool   `json:"log_stderr"`
}

type doctorFollowReport struct {
	HealthFile            string           `json:"health_file"`
	HealthPresent         bool             `json:"health_present"`
	HealthPruned          bool             `json:"health_pruned"`
	HealthPruneReason     string           `json:"health_prune_reason,omitempty"`
	LastUpdatedAt         *time.Time       `json:"last_updated_at,omitempty"`
	Status                string           `json:"status,omitempty"`
	Source                string           `json:"source,omitempty"`
	InputCount            int              `json:"input_count,omitempty"`
	Continuous            bool             `json:"continuous"`
	PollIntervalSeconds   int64            `json:"poll_interval_seconds,omitempty"`
	SnapshotAgeSeconds    int64            `json:"snapshot_age_seconds,omitempty"`
	HealthStale           bool             `json:"health_stale"`
	RequestedWatchMode    string           `json:"requested_watch_mode,omitempty"`
	ActiveWatchMode       string           `json:"active_watch_mode,omitempty"`
	WatchFallbacks        int              `json:"watch_fallbacks,omitempty"`
	WatchTransitions      int              `json:"watch_transitions,omitempty"`
	LastFallbackReason    string           `json:"last_fallback_reason,omitempty"`
	WatchPollCatchups     int              `json:"watch_poll_catchups,omitempty"`
	WatchPollCatchupBytes int              `json:"watch_poll_catchup_bytes,omitempty"`
	Warnings              []common.Warning `json:"warnings,omitempty"`
}

type doctorMCPReport struct {
	Transport string `json:"transport"`
	ToolCount int    `json:"tool_count"`
}

const (
	doctorJSONFlag                    = "--json"
	doctorPruneStaleFollowHealthFlag  = "--prune-stale-follow-health"
	stringNone                        = "none"
	doctorFollowHealthStaleMultiplier = 3
	doctorFollowHealthMinimumWindow   = 30 * time.Second
)

func parseDoctorOptions(args []string) (doctorOptions, error) {
	var options doctorOptions
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "":
			continue
		case doctorJSONFlag:
			options.JSON = true
		case doctorPruneStaleFollowHealthFlag:
			options.PruneStaleFollowHealth = true
		default:
			return doctorOptions{}, fmt.Errorf("unknown doctor flag %q", arg)
		}
	}
	return options, nil
}

func buildDoctorReport(cfg config.Config, runtime db.RuntimeDiagnostics, toolCount int, followHealth *followImportsHealthSnapshot, healthPruned bool, healthPruneReason string) doctorReport {
	now := time.Now().UTC()
	followReport := doctorFollowReport{
		HealthFile:        followImportsHealthPath(cfg.Meta.LogDir),
		HealthPresent:     followHealth != nil,
		HealthPruned:      healthPruned,
		HealthPruneReason: strings.TrimSpace(healthPruneReason),
	}
	if followHealth != nil {
		age, stale := evaluateFollowImportsHealthStaleness(*followHealth, now)
		followReport.LastUpdatedAt = &followHealth.UpdatedAt
		followReport.Status = followHealth.Status
		followReport.Source = followHealth.Source
		followReport.InputCount = followHealth.InputCount
		followReport.Continuous = followHealth.Continuous
		followReport.PollIntervalSeconds = followHealth.PollIntervalSeconds
		followReport.SnapshotAgeSeconds = int64(age / time.Second)
		followReport.HealthStale = stale
		followReport.RequestedWatchMode = followHealth.RequestedWatchMode
		followReport.ActiveWatchMode = followHealth.ActiveWatchMode
		followReport.WatchFallbacks = followHealth.WatchFallbacks
		followReport.WatchTransitions = followHealth.WatchTransitions
		followReport.LastFallbackReason = followHealth.LastFallbackReason
		followReport.WatchPollCatchups = followHealth.WatchPollCatchups
		followReport.WatchPollCatchupBytes = followHealth.WatchCatchupBytes
		followReport.Warnings = common.MergeWarnings(append([]common.Warning(nil), followHealth.Warnings...), followImportsHealthStaleWarnings(*followHealth, age, stale))
	}
	return doctorReport{
		Status: "ok",
		Config: doctorConfigReport{
			Precedence:     "defaults<config_file<environment",
			ConfigDir:      cfg.Meta.ConfigDir,
			ConfigFile:     cfg.Meta.ConfigFilePath,
			ConfigFileUsed: stringPointerOrNil(cfg.Meta.ConfigFileUsed),
			Database:       cfg.File.DatabasePath,
			DefaultSystem:  cfg.File.DefaultSystemName,
			SQLiteDriver:   cfg.File.SQLiteDriver,
		},
		Runtime: doctorRuntimeReport{
			BusyTimeoutMS:    runtime.BusyTimeout.Milliseconds(),
			JournalMode:      runtime.JournalMode,
			ForeignKeys:      runtime.ForeignKeysEnabled,
			RequiredSchemaOK: runtime.RequiredSchemaOK,
			FTSReady:         runtime.FTSReady,
		},
		Migrations: doctorMigrationsReport{
			Available:       runtime.Migrations.Available,
			Applied:         runtime.Migrations.Applied,
			Pending:         runtime.Migrations.Pending,
			LatestAvailable: stringPointerOrNil(runtime.Migrations.LatestAvailable),
			LatestApplied:   stringPointerOrNil(runtime.Migrations.LatestApplied),
		},
		Audit: doctorAuditReport{
			NoteRecords:                    runtime.Audit.NoteRecords,
			HandoffRecords:                 runtime.Audit.HandoffRecords,
			ImportRecords:                  runtime.Audit.ImportRecords,
			NotesCodexExplicit:             runtime.Audit.NotesCodexExplicit,
			NotesWatcherImport:             runtime.Audit.NotesWatcherImport,
			NotesRelayImport:               runtime.Audit.NotesRelayImport,
			NotesRecoveryGenerated:         runtime.Audit.NotesRecoveryGenerated,
			NotesInvalidSource:             runtime.Audit.NotesInvalidSource,
			ImportsWatcherImport:           runtime.Audit.ImportsWatcherImport,
			ImportsRelayImport:             runtime.Audit.ImportsRelayImport,
			SuppressedImports:              runtime.Audit.SuppressedImports,
			SuppressedImportsMissingReason: runtime.Audit.SuppressedImportsMissingReason,
			ImportsMissingDedupeKey:        runtime.Audit.ImportsMissingDedupeKey,
			ImportsLinkedMemory:            runtime.Audit.ImportsLinkedMemory,
			ExcludedNotes:                  runtime.Audit.ExcludedNotes,
			ExcludedHandoffs:               runtime.Audit.ExcludedHandoffs,
			ExcludedNotesMissingReason:     runtime.Audit.ExcludedNotesMissingReason,
			ExcludedHandoffsMissingReason:  runtime.Audit.ExcludedHandoffsMissingReason,
			RecoveryHandoffs:               runtime.Audit.RecoveryHandoffs,
			OpenHandoffs:                   runtime.Audit.OpenHandoffs,
			NoteProvenanceReady:            runtime.Audit.NoteProvenanceReady,
			ExclusionAuditReady:            runtime.Audit.ExclusionAuditReady,
			ImportAuditReady:               runtime.Audit.ImportAuditReady,
		},
		Logging: doctorLoggingReport{
			LogFile:       cfg.File.LogFilePath,
			LogLevel:      strings.ToLower(cfg.File.LogLevel.String()),
			LogMaxSizeMB:  cfg.File.LogMaxSizeMB,
			LogMaxBackups: cfg.File.LogMaxBackups,
			LogMaxAgeDays: cfg.File.LogMaxAgeDays,
			LogCompress:   cfg.File.LogCompress,
			LogStderr:     cfg.File.LogAlsoStderr,
		},
		Follow: followReport,
		MCP: doctorMCPReport{
			Transport: "stdio",
			ToolCount: toolCount,
		},
	}
}

func formatDoctorReport(report doctorReport) string {
	lines := []string{
		"doctor ok",
		fmt.Sprintf("config_precedence=%s", report.Config.Precedence),
		fmt.Sprintf("config_dir=%s", report.Config.ConfigDir),
		fmt.Sprintf("config_file=%s", report.Config.ConfigFile),
		fmt.Sprintf("config_file_used=%s", pointerStringOrNone(report.Config.ConfigFileUsed)),
		fmt.Sprintf("database=%s", report.Config.Database),
		fmt.Sprintf("default_system=%s", report.Config.DefaultSystem),
		fmt.Sprintf("sqlite_driver=%s", report.Config.SQLiteDriver),
		fmt.Sprintf("busy_timeout=%s", time.Duration(report.Runtime.BusyTimeoutMS)*time.Millisecond),
		fmt.Sprintf("journal_mode=%s", report.Runtime.JournalMode),
		fmt.Sprintf("foreign_keys=%t", report.Runtime.ForeignKeys),
		fmt.Sprintf("required_schema_ok=%t", report.Runtime.RequiredSchemaOK),
		fmt.Sprintf("fts_ready=%t", report.Runtime.FTSReady),
		fmt.Sprintf("migrations_available=%d", report.Migrations.Available),
		fmt.Sprintf("migrations_applied=%d", report.Migrations.Applied),
		fmt.Sprintf("migrations_pending=%d", report.Migrations.Pending),
		fmt.Sprintf("latest_migration_available=%s", pointerStringOrNone(report.Migrations.LatestAvailable)),
		fmt.Sprintf("latest_migration_applied=%s", pointerStringOrNone(report.Migrations.LatestApplied)),
		fmt.Sprintf("note_records=%d", report.Audit.NoteRecords),
		fmt.Sprintf("handoff_records=%d", report.Audit.HandoffRecords),
		fmt.Sprintf("import_records=%d", report.Audit.ImportRecords),
		fmt.Sprintf("note_source_codex_explicit=%d", report.Audit.NotesCodexExplicit),
		fmt.Sprintf("note_source_watcher_import=%d", report.Audit.NotesWatcherImport),
		fmt.Sprintf("note_source_relay_import=%d", report.Audit.NotesRelayImport),
		fmt.Sprintf("note_source_recovery_generated=%d", report.Audit.NotesRecoveryGenerated),
		fmt.Sprintf("note_source_invalid=%d", report.Audit.NotesInvalidSource),
		fmt.Sprintf("import_source_watcher_import=%d", report.Audit.ImportsWatcherImport),
		fmt.Sprintf("import_source_relay_import=%d", report.Audit.ImportsRelayImport),
		fmt.Sprintf("suppressed_imports=%d", report.Audit.SuppressedImports),
		fmt.Sprintf("suppressed_imports_missing_reason=%d", report.Audit.SuppressedImportsMissingReason),
		fmt.Sprintf("imports_missing_dedupe_key=%d", report.Audit.ImportsMissingDedupeKey),
		fmt.Sprintf("imports_linked_memory=%d", report.Audit.ImportsLinkedMemory),
		fmt.Sprintf("excluded_notes=%d", report.Audit.ExcludedNotes),
		fmt.Sprintf("excluded_handoffs=%d", report.Audit.ExcludedHandoffs),
		fmt.Sprintf("excluded_notes_missing_reason=%d", report.Audit.ExcludedNotesMissingReason),
		fmt.Sprintf("excluded_handoffs_missing_reason=%d", report.Audit.ExcludedHandoffsMissingReason),
		fmt.Sprintf("recovery_handoffs=%d", report.Audit.RecoveryHandoffs),
		fmt.Sprintf("open_handoffs=%d", report.Audit.OpenHandoffs),
		fmt.Sprintf("note_provenance_ready=%t", report.Audit.NoteProvenanceReady),
		fmt.Sprintf("exclusion_audit_ready=%t", report.Audit.ExclusionAuditReady),
		fmt.Sprintf("import_audit_ready=%t", report.Audit.ImportAuditReady),
		fmt.Sprintf("log_file=%s", report.Logging.LogFile),
		fmt.Sprintf("log_level=%s", report.Logging.LogLevel),
		fmt.Sprintf("log_max_size_mb=%d", report.Logging.LogMaxSizeMB),
		fmt.Sprintf("log_max_backups=%d", report.Logging.LogMaxBackups),
		fmt.Sprintf("log_max_age_days=%d", report.Logging.LogMaxAgeDays),
		fmt.Sprintf("log_compress=%t", report.Logging.LogCompress),
		fmt.Sprintf("log_stderr=%t", report.Logging.LogStderr),
		fmt.Sprintf("follow_imports_health_file=%s", report.Follow.HealthFile),
		fmt.Sprintf("follow_imports_health_present=%t", report.Follow.HealthPresent),
		fmt.Sprintf("follow_imports_health_pruned=%t", report.Follow.HealthPruned),
		fmt.Sprintf("follow_imports_health_prune_reason=%s", fallbackString(report.Follow.HealthPruneReason)),
		fmt.Sprintf("follow_imports_last_updated_at=%s", pointerTimeOrNone(report.Follow.LastUpdatedAt)),
		fmt.Sprintf("follow_imports_status=%s", fallbackString(report.Follow.Status)),
		fmt.Sprintf("follow_imports_source=%s", fallbackString(report.Follow.Source)),
		fmt.Sprintf("follow_imports_input_count=%d", report.Follow.InputCount),
		fmt.Sprintf("follow_imports_continuous=%t", report.Follow.Continuous),
		fmt.Sprintf("follow_imports_poll_interval_seconds=%d", report.Follow.PollIntervalSeconds),
		fmt.Sprintf("follow_imports_snapshot_age_seconds=%d", report.Follow.SnapshotAgeSeconds),
		fmt.Sprintf("follow_imports_health_stale=%t", report.Follow.HealthStale),
		fmt.Sprintf("follow_imports_requested_watch_mode=%s", fallbackString(report.Follow.RequestedWatchMode)),
		fmt.Sprintf("follow_imports_active_watch_mode=%s", fallbackString(report.Follow.ActiveWatchMode)),
		fmt.Sprintf("follow_imports_watch_fallbacks=%d", report.Follow.WatchFallbacks),
		fmt.Sprintf("follow_imports_watch_transitions=%d", report.Follow.WatchTransitions),
		fmt.Sprintf("follow_imports_last_fallback_reason=%s", fallbackString(report.Follow.LastFallbackReason)),
		fmt.Sprintf("follow_imports_watch_poll_catchups=%d", report.Follow.WatchPollCatchups),
		fmt.Sprintf("follow_imports_watch_poll_catchup_bytes=%d", report.Follow.WatchPollCatchupBytes),
		fmt.Sprintf("follow_imports_warnings=%d", len(report.Follow.Warnings)),
		fmt.Sprintf("mcp_transport=%s", report.MCP.Transport),
		fmt.Sprintf("mcp_tool_count=%d", report.MCP.ToolCount),
	}
	for i, warning := range report.Follow.Warnings {
		prefix := fmt.Sprintf("follow_imports_warning_%d", i+1)
		lines = append(lines,
			fmt.Sprintf("%s_code=%s", prefix, warning.Code),
			fmt.Sprintf("%s_message=%s", prefix, warning.Message),
		)
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatDoctorReportJSON(report doctorReport) (string, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return "", fmt.Errorf("marshal doctor report: %w", err)
	}
	return buffer.String(), nil
}

func stringPointerOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func pointerStringOrNone(value *string) string {
	if value == nil {
		return stringNone
	}
	return *value
}

func pointerTimeOrNone(value *time.Time) string {
	if value == nil {
		return stringNone
	}
	return value.UTC().Format(time.RFC3339)
}

func evaluateFollowImportsHealthStaleness(snapshot followImportsHealthSnapshot, now time.Time) (time.Duration, bool) {
	age := now.Sub(snapshot.UpdatedAt)
	if age < 0 {
		age = 0
	}
	if !snapshot.Continuous {
		return age, false
	}
	window := time.Duration(snapshot.PollIntervalSeconds) * time.Second * doctorFollowHealthStaleMultiplier
	if window < doctorFollowHealthMinimumWindow {
		window = doctorFollowHealthMinimumWindow
	}
	return age, age > window
}

func followImportsHealthStaleWarnings(_ followImportsHealthSnapshot, age time.Duration, stale bool) []common.Warning {
	if !stale {
		return nil
	}
	return []common.Warning{{
		Code:    common.WarnFollowImportsHealthStale,
		Message: fmt.Sprintf("follow-imports health snapshot is stale at %s", age.Truncate(time.Second)),
	}}
}

func loadDoctorFollowImportsHealth(logDir string, pruneStale bool, now time.Time) (*followImportsHealthSnapshot, bool, string, error) {
	followHealth, err := loadFollowImportsHealthSnapshot(logDir)
	if err != nil {
		return nil, false, "", err
	}
	if !pruneStale || followHealth == nil {
		return followHealth, false, "", nil
	}
	_, stale := evaluateFollowImportsHealthStaleness(*followHealth, now)
	if !stale {
		return followHealth, false, "", nil
	}
	if err := pruneFollowImportsHealthSnapshot(logDir); err != nil {
		return nil, false, "", err
	}
	return nil, true, "stale", nil
}
