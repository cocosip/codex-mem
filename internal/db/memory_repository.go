package db

import (
	"database/sql"
	"errors"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/memory"
)

type MemoryRepository struct {
	db *sql.DB
}

func NewMemoryRepository(db *sql.DB) *MemoryRepository {
	return &MemoryRepository{db: db}
}

func (r *MemoryRepository) FindDuplicate(note memory.Note) (*memory.Note, error) {
	row := r.db.QueryRow(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, COALESCE(tags_json, '[]'), COALESCE(file_paths_json, '[]'),
			COALESCE(related_project_ids_json, '[]'), status, source, created_at, updated_at
		FROM memory_items
		WHERE session_id = ? AND system_id = ? AND project_id = ? AND workspace_id = ?
			AND type = ? AND title = ? AND content = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, note.SessionID, note.Scope.SystemID, note.Scope.ProjectID, note.Scope.WorkspaceID, string(note.Type), note.Title, note.Content)

	record, err := scanNote(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &record, nil
	}
}

func (r *MemoryRepository) Create(note memory.Note) error {
	if err := validateScopeRef(r.db, note.Scope); err != nil {
		return err
	}
	if err := validateSessionScope(r.db, note.SessionID, note.Scope); err != nil {
		return err
	}

	tagsJSON, err := marshalStringSlice(note.Tags)
	if err != nil {
		return err
	}
	filePathsJSON, err := marshalStringSlice(note.FilePaths)
	if err != nil {
		return err
	}
	relatedProjectsJSON, err := marshalStringSlice(note.RelatedProjectIDs)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		INSERT INTO memory_items (
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, tags_json, file_paths_json, related_project_ids_json, status, source, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		note.ID,
		note.SessionID,
		note.Scope.SystemID,
		note.Scope.ProjectID,
		note.Scope.WorkspaceID,
		string(note.Type),
		note.Title,
		note.Content,
		note.Importance,
		tagsJSON,
		filePathsJSON,
		relatedProjectsJSON,
		string(note.Status),
		string(note.Source),
		note.CreatedAt.UTC().Format(time.RFC3339Nano),
		note.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return common.WrapError(common.ErrWriteFailed, "insert memory note", err)
	}
	return nil
}

func (r *MemoryRepository) ListRecentByWorkspace(workspaceID string, limit int, minImportance int) ([]memory.Note, error) {
	rows, err := r.db.Query(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, COALESCE(tags_json, '[]'), COALESCE(file_paths_json, '[]'),
			COALESCE(related_project_ids_json, '[]'), status, source, created_at, updated_at
		FROM memory_items
		WHERE workspace_id = ? AND importance >= ?
		ORDER BY importance DESC, created_at DESC
		LIMIT ?
	`, workspaceID, minImportance, positiveLimit(limit))
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "query workspace notes", err)
	}
	defer rows.Close()

	return scanNotes(rows)
}

func (r *MemoryRepository) ListRecentByProject(projectID string, excludeWorkspaceID string, limit int, minImportance int) ([]memory.Note, error) {
	rows, err := r.db.Query(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, COALESCE(tags_json, '[]'), COALESCE(file_paths_json, '[]'),
			COALESCE(related_project_ids_json, '[]'), status, source, created_at, updated_at
		FROM memory_items
		WHERE project_id = ? AND workspace_id <> ? AND importance >= ?
		ORDER BY importance DESC, created_at DESC
		LIMIT ?
	`, projectID, excludeWorkspaceID, minImportance, positiveLimit(limit))
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "query project notes", err)
	}
	defer rows.Close()

	return scanNotes(rows)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNote(scanner rowScanner) (memory.Note, error) {
	var (
		record              memory.Note
		noteType            string
		status              string
		source              string
		createdAt           string
		updatedAt           string
		tagsJSON            string
		filePathsJSON       string
		relatedProjectsJSON string
	)

	err := scanner.Scan(
		&record.ID,
		&record.SessionID,
		&record.Scope.SystemID,
		&record.Scope.ProjectID,
		&record.Scope.WorkspaceID,
		&noteType,
		&record.Title,
		&record.Content,
		&record.Importance,
		&tagsJSON,
		&filePathsJSON,
		&relatedProjectsJSON,
		&status,
		&source,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return memory.Note{}, err
	}

	record.Type = memory.NoteType(noteType)
	record.Status = memory.Status(status)
	record.Source = memory.Source(source)

	record.Tags, err = unmarshalStringSlice(tagsJSON)
	if err != nil {
		return memory.Note{}, err
	}
	record.FilePaths, err = unmarshalStringSlice(filePathsJSON)
	if err != nil {
		return memory.Note{}, err
	}
	record.RelatedProjectIDs, err = unmarshalStringSlice(relatedProjectsJSON)
	if err != nil {
		return memory.Note{}, err
	}
	record.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return memory.Note{}, common.WrapError(common.ErrReadFailed, "parse note created_at", err)
	}
	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return memory.Note{}, common.WrapError(common.ErrReadFailed, "parse note updated_at", err)
	}
	return record, nil
}

func scanNotes(rows *sql.Rows) ([]memory.Note, error) {
	var notes []memory.Note
	for rows.Next() {
		record, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, record)
	}
	if err := rows.Err(); err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "iterate notes", err)
	}
	return notes, nil
}

func positiveLimit(limit int) int {
	if limit <= 0 {
		return 5
	}
	return limit
}
