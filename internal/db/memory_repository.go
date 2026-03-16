package db

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
)

type sqlExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// MemoryRepository provides SQLite-backed persistence for memory notes.
type MemoryRepository struct {
	exec sqlExecutor
}

// NewMemoryRepository constructs a MemoryRepository for the provided database handle.
func NewMemoryRepository(db *sql.DB) *MemoryRepository {
	return &MemoryRepository{exec: db}
}

// NewMemoryRepositoryTx constructs a MemoryRepository bound to an existing SQL transaction.
func NewMemoryRepositoryTx(tx *sql.Tx) *MemoryRepository {
	return &MemoryRepository{exec: tx}
}

// FindDuplicate returns the latest note with the same scope, type, title, and content.
func (r *MemoryRepository) FindDuplicate(note memory.Note) (*memory.Note, error) {
	row := r.exec.QueryRow(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, COALESCE(tags_json, '[]'), COALESCE(file_paths_json, '[]'),
			COALESCE(related_project_ids_json, '[]'), status, source, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
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

// FindProjectDuplicate returns the strongest matching searchable note within the same project.
func (r *MemoryRepository) FindProjectDuplicate(ref scope.Ref, noteType memory.NoteType, title string, content string) (*memory.Note, error) {
	row := r.exec.QueryRow(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, COALESCE(tags_json, '[]'), COALESCE(file_paths_json, '[]'),
			COALESCE(related_project_ids_json, '[]'), status, source, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM memory_items
		WHERE system_id = ? AND project_id = ? AND type = ? AND title = ? AND content = ?
			AND searchable = 1 AND status <> ?
		ORDER BY CASE WHEN source = ? THEN 0 ELSE 1 END, created_at DESC
		LIMIT 1
	`, ref.SystemID, ref.ProjectID, string(noteType), strings.TrimSpace(title), strings.TrimSpace(content), string(memory.StatusSuperseded), string(memory.SourceCodexExplicit))

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

// Create stores a memory note after validating scope and session relationships.
func (r *MemoryRepository) Create(note memory.Note) error {
	if err := validateScopeRef(r.exec, note.Scope); err != nil {
		return err
	}
	if err := validateSessionScope(r.exec, note.SessionID, note.Scope); err != nil {
		return err
	}
	if !note.Searchable && strings.TrimSpace(note.ExclusionReason) == "" {
		note.Searchable = true
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

	_, err = r.exec.Exec(`
		INSERT INTO memory_items (
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, tags_json, file_paths_json, related_project_ids_json, status, source, searchable, exclusion_reason, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		boolToInt(note.Searchable),
		note.ExclusionReason,
		note.CreatedAt.UTC().Format(time.RFC3339Nano),
		note.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return common.WrapError(common.ErrWriteFailed, "insert memory note", err)
	}
	return nil
}

// ListRecentByWorkspace lists recent searchable notes for a workspace.
func (r *MemoryRepository) ListRecentByWorkspace(workspaceID string, limit int, minImportance int) ([]memory.Note, error) {
	rows, err := r.exec.Query(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, COALESCE(tags_json, '[]'), COALESCE(file_paths_json, '[]'),
			COALESCE(related_project_ids_json, '[]'), status, source, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM memory_items
		WHERE workspace_id = ? AND importance >= ? AND searchable = 1
		ORDER BY importance DESC, created_at DESC
		LIMIT ?
	`, workspaceID, minImportance, positiveLimit(limit))
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "query workspace notes", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanNotes(rows)
}

// ListRecentByProject lists recent searchable notes from sibling workspaces in a project.
func (r *MemoryRepository) ListRecentByProject(projectID string, excludeWorkspaceID string, limit int, minImportance int) ([]memory.Note, error) {
	rows, err := r.exec.Query(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, COALESCE(tags_json, '[]'), COALESCE(file_paths_json, '[]'),
			COALESCE(related_project_ids_json, '[]'), status, source, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM memory_items
		WHERE project_id = ? AND workspace_id <> ? AND importance >= ? AND searchable = 1
		ORDER BY importance DESC, created_at DESC
		LIMIT ?
	`, projectID, excludeWorkspaceID, minImportance, positiveLimit(limit))
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "query project notes", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanNotes(rows)
}

// ListRecentByProjects lists recent searchable notes across related projects in the same system.
func (r *MemoryRepository) ListRecentByProjects(systemID string, projectIDs []string, limit int, minImportance int) ([]memory.Note, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}

	args := append([]any{systemID}, stringsToAny(projectIDs)...)
	args = append(args, minImportance, positiveLimit(limit))
	rows, err := r.exec.Query(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, COALESCE(tags_json, '[]'), COALESCE(file_paths_json, '[]'),
			COALESCE(related_project_ids_json, '[]'), status, source, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM memory_items
		WHERE system_id = ? AND project_id IN (`+placeholders(len(projectIDs))+`) AND importance >= ? AND searchable = 1
		ORDER BY importance DESC, created_at DESC
		LIMIT ?
	`, args...)
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "query related project notes", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanNotes(rows)
}

// GetByID loads a note record by its stable identifier.
func (r *MemoryRepository) GetByID(id string) (*memory.Note, error) {
	row := r.exec.QueryRow(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, type, title, content,
			importance, COALESCE(tags_json, '[]'), COALESCE(file_paths_json, '[]'),
			COALESCE(related_project_ids_json, '[]'), status, source, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM memory_items
		WHERE id = ?
	`, id)

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

// ListRelatedProjectIDs returns recent distinct related project identifiers referenced by notes.
func (r *MemoryRepository) ListRelatedProjectIDs(projectID string, limit int) ([]string, error) {
	rows, err := r.exec.Query(`
		SELECT COALESCE(related_project_ids_json, '[]')
		FROM memory_items
		WHERE project_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, projectID, positiveLimit(limit))
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "query related project ids", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	seen := make(map[string]struct{})
	var ids []string
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, common.WrapError(common.ErrReadFailed, "scan related project ids", err)
		}
		values, err := unmarshalStringSlice(body)
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" || value == projectID {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			ids = append(ids, value)
			if len(ids) >= positiveLimit(limit) {
				return ids, nil
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "iterate related project ids", err)
	}
	return ids, nil
}

// Search finds searchable notes within a project that match the query and filters.
func (r *MemoryRepository) Search(ref scope.Ref, query string, limit int, minImportance int, types []memory.NoteType, states []memory.Status) ([]memory.Note, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, common.NewError(common.ErrInvalidInput, "query is required")
	}

	matchQuery := ftsQuery(query)
	where := []string{
		"m.project_id = ?",
		"m.importance >= ?",
		"m.searchable = 1",
		"memory_items_fts MATCH ?",
	}
	args := []any{ref.ProjectID, minImportance, matchQuery}

	if len(types) > 0 {
		where = append(where, "m.type IN ("+placeholders(len(types))+")")
		for _, noteType := range types {
			args = append(args, string(noteType))
		}
	}
	if len(states) > 0 {
		where = append(where, "m.status IN ("+placeholders(len(states))+")")
		for _, state := range states {
			args = append(args, string(state))
		}
	}

	rows, err := r.exec.Query(`
		SELECT
			m.id, m.session_id, m.system_id, m.project_id, m.workspace_id, m.type, m.title, m.content,
			m.importance, COALESCE(m.tags_json, '[]'), COALESCE(m.file_paths_json, '[]'),
			COALESCE(m.related_project_ids_json, '[]'), m.status, m.source, m.searchable, COALESCE(m.exclusion_reason, ''), m.created_at, m.updated_at
		FROM memory_items_fts
		INNER JOIN memory_items m ON m.rowid = memory_items_fts.rowid
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY bm25(memory_items_fts), m.importance DESC, m.created_at DESC
		LIMIT ?
	`, append(args, positiveLimit(limit))...)
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "search memory notes", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	return scanNotes(rows)
}

// SearchProjects finds searchable notes across related projects that match the query and filters.
func (r *MemoryRepository) SearchProjects(systemID string, projectIDs []string, query string, limit int, minImportance int, types []memory.NoteType, states []memory.Status) ([]memory.Note, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, common.NewError(common.ErrInvalidInput, "query is required")
	}
	if len(projectIDs) == 0 {
		return nil, nil
	}

	matchQuery := ftsQuery(query)
	where := []string{
		"m.system_id = ?",
		"m.project_id IN (" + placeholders(len(projectIDs)) + ")",
		"m.importance >= ?",
		"m.searchable = 1",
		"memory_items_fts MATCH ?",
	}
	args := append([]any{systemID}, stringsToAny(projectIDs)...)
	args = append(args, minImportance, matchQuery)

	if len(types) > 0 {
		where = append(where, "m.type IN ("+placeholders(len(types))+")")
		for _, noteType := range types {
			args = append(args, string(noteType))
		}
	}
	if len(states) > 0 {
		where = append(where, "m.status IN ("+placeholders(len(states))+")")
		for _, state := range states {
			args = append(args, string(state))
		}
	}

	rows, err := r.exec.Query(`
		SELECT
			m.id, m.session_id, m.system_id, m.project_id, m.workspace_id, m.type, m.title, m.content,
			m.importance, COALESCE(m.tags_json, '[]'), COALESCE(m.file_paths_json, '[]'),
			COALESCE(m.related_project_ids_json, '[]'), m.status, m.source, m.searchable, COALESCE(m.exclusion_reason, ''), m.created_at, m.updated_at
		FROM memory_items_fts
		INNER JOIN memory_items m ON m.rowid = memory_items_fts.rowid
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY bm25(memory_items_fts), m.importance DESC, m.created_at DESC
		LIMIT ?
	`, append(args, positiveLimit(limit))...)
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "search related project notes", err)
	}
	defer func() {
		_ = rows.Close()
	}()

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
		searchable          int
		exclusionReason     string
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
		&searchable,
		&exclusionReason,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return memory.Note{}, err
	}

	record.Type = memory.NoteType(noteType)
	record.Status = memory.Status(status)
	record.Source = memory.Source(source)
	record.Searchable = intToBool(searchable)
	record.ExclusionReason = exclusionReason

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

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	parts := make([]string, count)
	for i := range parts {
		parts[i] = "?"
	}
	return strings.Join(parts, ", ")
}

func ftsQuery(query string) string {
	terms := strings.Fields(strings.TrimSpace(query))
	if len(terms) == 0 {
		return ""
	}
	for i, term := range terms {
		terms[i] = `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
	}
	return strings.Join(terms, " ")
}

func stringsToAny(values []string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}
