package db

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"codex-mem/internal/domain/common"
)

type RuntimeDiagnostics struct {
	ForeignKeysEnabled bool
	BusyTimeout        time.Duration
	JournalMode        string
	Migrations         MigrationDiagnostics
	RequiredSchemaOK   bool
	FTSReady           bool
	Audit              AuditDiagnostics
}

type MigrationDiagnostics struct {
	Available       int
	Applied         int
	Pending         int
	LatestAvailable string
	LatestApplied   string
}

type AuditDiagnostics struct {
	NoteRecords                   int
	HandoffRecords                int
	NotesCodexExplicit            int
	NotesWatcherImport            int
	NotesRelayImport              int
	NotesRecoveryGenerated        int
	NotesInvalidSource            int
	ExcludedNotes                 int
	ExcludedHandoffs              int
	ExcludedNotesMissingReason    int
	ExcludedHandoffsMissingReason int
	RecoveryHandoffs              int
	OpenHandoffs                  int
	NoteProvenanceReady           bool
	ExclusionAuditReady           bool
}

func InspectRuntime(ctx context.Context, handle *sql.DB) (RuntimeDiagnostics, error) {
	files, err := loadMigrationFiles()
	if err != nil {
		return RuntimeDiagnostics{}, err
	}

	foreignKeysEnabled, err := pragmaBool(ctx, handle, "foreign_keys")
	if err != nil {
		return RuntimeDiagnostics{}, err
	}
	busyTimeoutMS, err := pragmaInt(ctx, handle, "busy_timeout")
	if err != nil {
		return RuntimeDiagnostics{}, err
	}
	journalMode, err := pragmaString(ctx, handle, "journal_mode")
	if err != nil {
		return RuntimeDiagnostics{}, err
	}

	applied, err := appliedMigrationRows(ctx, handle)
	if err != nil {
		return RuntimeDiagnostics{}, err
	}

	requiredSchemaOK, err := hasRequiredSchema(ctx, handle)
	if err != nil {
		return RuntimeDiagnostics{}, err
	}
	ftsReady, err := objectExists(ctx, handle, "memory_items_fts")
	if err != nil {
		return RuntimeDiagnostics{}, err
	}
	audit, err := inspectAudit(ctx, handle)
	if err != nil {
		return RuntimeDiagnostics{}, err
	}

	latestAvailable := ""
	if len(files) > 0 {
		latestAvailable = files[len(files)-1].Name
	}
	latestApplied := ""
	if len(applied) > 0 {
		latestApplied = applied[len(applied)-1].Name
	}

	return RuntimeDiagnostics{
		ForeignKeysEnabled: foreignKeysEnabled,
		BusyTimeout:        time.Duration(busyTimeoutMS) * time.Millisecond,
		JournalMode:        journalMode,
		Migrations: MigrationDiagnostics{
			Available:       len(files),
			Applied:         len(applied),
			Pending:         maxInt(0, len(files)-len(applied)),
			LatestAvailable: latestAvailable,
			LatestApplied:   latestApplied,
		},
		RequiredSchemaOK: requiredSchemaOK,
		FTSReady:         ftsReady,
		Audit:            audit,
	}, nil
}

type appliedMigration struct {
	Version int
	Name    string
}

func appliedMigrationRows(ctx context.Context, handle *sql.DB) ([]appliedMigration, error) {
	rows, err := handle.QueryContext(ctx, `
		SELECT version, name
		FROM schema_migrations
		ORDER BY version ASC
	`)
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "query applied migrations", err)
	}
	defer rows.Close()

	var applied []appliedMigration
	for rows.Next() {
		var row appliedMigration
		if err := rows.Scan(&row.Version, &row.Name); err != nil {
			return nil, common.WrapError(common.ErrReadFailed, "scan applied migration", err)
		}
		applied = append(applied, row)
	}
	if err := rows.Err(); err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "iterate applied migrations", err)
	}
	return applied, nil
}

func pragmaBool(ctx context.Context, handle *sql.DB, name string) (bool, error) {
	value, err := pragmaInt(ctx, handle, name)
	if err != nil {
		return false, err
	}
	return value != 0, nil
}

func pragmaInt(ctx context.Context, handle *sql.DB, name string) (int, error) {
	query := fmt.Sprintf("PRAGMA %s", sanitizePragmaName(name))
	var value int
	if err := handle.QueryRowContext(ctx, query).Scan(&value); err != nil {
		return 0, common.WrapError(common.ErrReadFailed, "read sqlite pragma "+name, err)
	}
	return value, nil
}

func pragmaString(ctx context.Context, handle *sql.DB, name string) (string, error) {
	query := fmt.Sprintf("PRAGMA %s", sanitizePragmaName(name))
	var value string
	if err := handle.QueryRowContext(ctx, query).Scan(&value); err != nil {
		return "", common.WrapError(common.ErrReadFailed, "read sqlite pragma "+name, err)
	}
	return strings.TrimSpace(value), nil
}

func sanitizePragmaName(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r == '_':
			return r
		default:
			return -1
		}
	}, name)
}

func hasRequiredSchema(ctx context.Context, handle *sql.DB) (bool, error) {
	required := []string{
		"systems",
		"projects",
		"workspaces",
		"sessions",
		"memory_items",
		"handoffs",
		"schema_migrations",
	}

	rows, err := handle.QueryContext(ctx, `
		SELECT name
		FROM sqlite_master
		WHERE type IN ('table', 'view')
	`)
	if err != nil {
		return false, common.WrapError(common.ErrReadFailed, "query sqlite schema objects", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return false, common.WrapError(common.ErrReadFailed, "scan sqlite schema object", err)
		}
		names = append(names, strings.TrimSpace(name))
	}
	if err := rows.Err(); err != nil {
		return false, common.WrapError(common.ErrReadFailed, "iterate sqlite schema objects", err)
	}

	for _, requiredName := range required {
		if !slices.Contains(names, requiredName) {
			return false, nil
		}
	}
	return true, nil
}

func objectExists(ctx context.Context, handle *sql.DB, name string) (bool, error) {
	var count int
	if err := handle.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM sqlite_master
		WHERE name = ?
	`, name).Scan(&count); err != nil {
		return false, common.WrapError(common.ErrReadFailed, "query sqlite object existence", err)
	}
	return count > 0, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func inspectAudit(ctx context.Context, handle *sql.DB) (AuditDiagnostics, error) {
	var audit AuditDiagnostics

	if err := handle.QueryRowContext(ctx, `
		SELECT
			COUNT(1),
			COALESCE(SUM(CASE WHEN source = 'codex_explicit' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN source = 'watcher_import' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN source = 'relay_import' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN source = 'recovery_generated' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN TRIM(COALESCE(source, '')) = '' OR source NOT IN ('codex_explicit', 'watcher_import', 'relay_import', 'recovery_generated') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN searchable = 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN searchable = 0 AND TRIM(COALESCE(exclusion_reason, '')) = '' THEN 1 ELSE 0 END), 0)
		FROM memory_items
	`).Scan(
		&audit.NoteRecords,
		&audit.NotesCodexExplicit,
		&audit.NotesWatcherImport,
		&audit.NotesRelayImport,
		&audit.NotesRecoveryGenerated,
		&audit.NotesInvalidSource,
		&audit.ExcludedNotes,
		&audit.ExcludedNotesMissingReason,
	); err != nil {
		return AuditDiagnostics{}, common.WrapError(common.ErrReadFailed, "inspect note audit diagnostics", err)
	}

	if err := handle.QueryRowContext(ctx, `
		SELECT
			COUNT(1),
			COALESCE(SUM(CASE WHEN searchable = 0 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN searchable = 0 AND TRIM(COALESCE(exclusion_reason, '')) = '' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN kind = 'recovery' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'open' THEN 1 ELSE 0 END), 0)
		FROM handoffs
	`).Scan(
		&audit.HandoffRecords,
		&audit.ExcludedHandoffs,
		&audit.ExcludedHandoffsMissingReason,
		&audit.RecoveryHandoffs,
		&audit.OpenHandoffs,
	); err != nil {
		return AuditDiagnostics{}, common.WrapError(common.ErrReadFailed, "inspect handoff audit diagnostics", err)
	}

	audit.NoteProvenanceReady = audit.NotesInvalidSource == 0
	audit.ExclusionAuditReady = audit.ExcludedNotesMissingReason == 0 && audit.ExcludedHandoffsMissingReason == 0
	return audit, nil
}
