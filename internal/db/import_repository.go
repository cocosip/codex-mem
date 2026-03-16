package db

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/imports"
)

// ImportRepository provides SQLite-backed persistence for import audit records.
type ImportRepository struct {
	db *sql.DB
}

// NewImportRepository constructs an ImportRepository for the provided database handle.
func NewImportRepository(db *sql.DB) *ImportRepository {
	return &ImportRepository{db: db}
}

// FindDuplicate returns the latest project-level import matching the same source and dedupe key.
func (r *ImportRepository) FindDuplicate(record imports.Record) (*imports.Record, error) {
	conditions := make([]string, 0, 2)
	args := []any{record.Scope.ProjectID, string(record.Source)}
	if strings.TrimSpace(record.ExternalID) != "" {
		conditions = append(conditions, "external_id = ?")
		args = append(args, record.ExternalID)
	}
	if strings.TrimSpace(record.PayloadHash) != "" {
		conditions = append(conditions, "payload_hash = ?")
		args = append(args, record.PayloadHash)
	}
	if len(conditions) == 0 {
		return nil, nil
	}

	row := r.db.QueryRow(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, source, external_id,
			payload_hash, durable_memory_id, suppressed, suppression_reason, imported_at
		FROM imports
		WHERE project_id = ? AND source = ? AND (`+strings.Join(conditions, " OR ")+`)
		ORDER BY imported_at DESC
		LIMIT 1
	`, args...)

	stored, err := scanImport(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &stored, nil
	}
}

// Create stores an import audit record after validating scope and session relationships.
func (r *ImportRepository) Create(record imports.Record) error {
	if err := validateScopeRef(r.db, record.Scope); err != nil {
		return err
	}
	if err := validateSessionScope(r.db, record.SessionID, record.Scope); err != nil {
		return err
	}

	_, err := r.db.Exec(`
		INSERT INTO imports (
			id, session_id, system_id, project_id, workspace_id, source, external_id,
			payload_hash, durable_memory_id, suppressed, suppression_reason, imported_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID,
		record.SessionID,
		record.Scope.SystemID,
		record.Scope.ProjectID,
		record.Scope.WorkspaceID,
		string(record.Source),
		record.ExternalID,
		record.PayloadHash,
		record.DurableMemoryID,
		boolToInt(record.Suppressed),
		record.SuppressionReason,
		record.ImportedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return common.WrapError(common.ErrWriteFailed, "insert import record", err)
	}
	return nil
}

type importRowScanner interface {
	Scan(dest ...any) error
}

func scanImport(scanner importRowScanner) (imports.Record, error) {
	var (
		record            imports.Record
		source            string
		suppressed        int
		importedAt        string
		suppressionReason string
	)

	err := scanner.Scan(
		&record.ID,
		&record.SessionID,
		&record.Scope.SystemID,
		&record.Scope.ProjectID,
		&record.Scope.WorkspaceID,
		&source,
		&record.ExternalID,
		&record.PayloadHash,
		&record.DurableMemoryID,
		&suppressed,
		&suppressionReason,
		&importedAt,
	)
	if err != nil {
		return imports.Record{}, err
	}

	record.Source = imports.Source(source)
	record.Suppressed = intToBool(suppressed)
	record.SuppressionReason = suppressionReason
	record.ImportedAt, err = time.Parse(time.RFC3339Nano, importedAt)
	if err != nil {
		return imports.Record{}, common.WrapError(common.ErrReadFailed, "parse import imported_at", err)
	}
	return record, nil
}
