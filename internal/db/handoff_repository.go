package db

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/scope"
)

type HandoffRepository struct {
	db *sql.DB
}

func NewHandoffRepository(db *sql.DB) *HandoffRepository {
	return &HandoffRepository{db: db}
}

func (r *HandoffRepository) FindLatestOpenByTask(ref scope.Ref, task string) (*handoff.Handoff, error) {
	row := r.db.QueryRow(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, kind, task, summary,
			COALESCE(completed_json, '[]'), COALESCE(next_steps_json, '[]'),
			COALESCE(open_questions_json, '[]'), COALESCE(risks_json, '[]'),
			COALESCE(files_touched_json, '[]'), COALESCE(related_note_ids_json, '[]'),
			status, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM handoffs
		WHERE system_id = ? AND project_id = ? AND workspace_id = ? AND task = ? AND status = 'open' AND searchable = 1
		ORDER BY created_at DESC
		LIMIT 1
	`, ref.SystemID, ref.ProjectID, ref.WorkspaceID, task)

	record, err := scanHandoff(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &record, nil
	}
}

func (r *HandoffRepository) Create(record handoff.Handoff) error {
	if err := validateScopeRef(r.db, record.Scope); err != nil {
		return err
	}
	if err := validateSessionScope(r.db, record.SessionID, record.Scope); err != nil {
		return err
	}
	if !record.Searchable && strings.TrimSpace(record.ExclusionReason) == "" {
		record.Searchable = true
	}

	completedJSON, err := marshalStringSlice(record.Completed)
	if err != nil {
		return err
	}
	nextStepsJSON, err := marshalStringSlice(record.NextSteps)
	if err != nil {
		return err
	}
	openQuestionsJSON, err := marshalStringSlice(record.OpenQuestions)
	if err != nil {
		return err
	}
	risksJSON, err := marshalStringSlice(record.Risks)
	if err != nil {
		return err
	}
	filesTouchedJSON, err := marshalStringSlice(record.FilesTouched)
	if err != nil {
		return err
	}
	relatedNoteIDsJSON, err := marshalStringSlice(record.RelatedNoteIDs)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		INSERT INTO handoffs (
			id, session_id, system_id, project_id, workspace_id, kind, task, summary,
			completed_json, next_steps_json, open_questions_json, risks_json, files_touched_json,
			related_note_ids_json, status, searchable, exclusion_reason, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		record.ID,
		record.SessionID,
		record.Scope.SystemID,
		record.Scope.ProjectID,
		record.Scope.WorkspaceID,
		string(record.Kind),
		record.Task,
		record.Summary,
		completedJSON,
		nextStepsJSON,
		openQuestionsJSON,
		risksJSON,
		filesTouchedJSON,
		relatedNoteIDsJSON,
		string(record.Status),
		boolToInt(record.Searchable),
		record.ExclusionReason,
		record.CreatedAt.UTC().Format(time.RFC3339Nano),
		record.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return common.WrapError(common.ErrWriteFailed, "insert handoff", err)
	}
	return nil
}

func (r *HandoffRepository) FindLatestOpenInWorkspace(workspaceID string) (*handoff.Handoff, error) {
	row := r.db.QueryRow(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, kind, task, summary,
			COALESCE(completed_json, '[]'), COALESCE(next_steps_json, '[]'),
			COALESCE(open_questions_json, '[]'), COALESCE(risks_json, '[]'),
			COALESCE(files_touched_json, '[]'), COALESCE(related_note_ids_json, '[]'),
			status, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM handoffs
		WHERE workspace_id = ? AND status = 'open' AND searchable = 1
		ORDER BY created_at DESC
		LIMIT 1
	`, workspaceID)

	record, err := scanHandoff(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &record, nil
	}
}

func (r *HandoffRepository) FindLatestOpenInProject(projectID string, excludeWorkspaceID string) (*handoff.Handoff, error) {
	row := r.db.QueryRow(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, kind, task, summary,
			COALESCE(completed_json, '[]'), COALESCE(next_steps_json, '[]'),
			COALESCE(open_questions_json, '[]'), COALESCE(risks_json, '[]'),
			COALESCE(files_touched_json, '[]'), COALESCE(related_note_ids_json, '[]'),
			status, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM handoffs
		WHERE project_id = ? AND workspace_id <> ? AND status = 'open' AND searchable = 1
		ORDER BY created_at DESC
		LIMIT 1
	`, projectID, excludeWorkspaceID)

	record, err := scanHandoff(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &record, nil
	}
}

func (r *HandoffRepository) ListRecentByWorkspace(workspaceID string, limit int) ([]handoff.Handoff, error) {
	rows, err := r.db.Query(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, kind, task, summary,
			COALESCE(completed_json, '[]'), COALESCE(next_steps_json, '[]'),
			COALESCE(open_questions_json, '[]'), COALESCE(risks_json, '[]'),
			COALESCE(files_touched_json, '[]'), COALESCE(related_note_ids_json, '[]'),
			status, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM handoffs
		WHERE workspace_id = ? AND searchable = 1
		ORDER BY created_at DESC
		LIMIT ?
	`, workspaceID, positiveLimit(limit))
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "query workspace handoffs", err)
	}
	defer rows.Close()

	return scanHandoffs(rows)
}

func (r *HandoffRepository) ListRecentByProject(projectID string, excludeWorkspaceID string, limit int) ([]handoff.Handoff, error) {
	rows, err := r.db.Query(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, kind, task, summary,
			COALESCE(completed_json, '[]'), COALESCE(next_steps_json, '[]'),
			COALESCE(open_questions_json, '[]'), COALESCE(risks_json, '[]'),
			COALESCE(files_touched_json, '[]'), COALESCE(related_note_ids_json, '[]'),
			status, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM handoffs
		WHERE project_id = ? AND workspace_id <> ? AND searchable = 1
		ORDER BY created_at DESC
		LIMIT ?
	`, projectID, excludeWorkspaceID, positiveLimit(limit))
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "query project handoffs", err)
	}
	defer rows.Close()

	return scanHandoffs(rows)
}

func (r *HandoffRepository) GetByID(id string) (*handoff.Handoff, error) {
	row := r.db.QueryRow(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, kind, task, summary,
			COALESCE(completed_json, '[]'), COALESCE(next_steps_json, '[]'),
			COALESCE(open_questions_json, '[]'), COALESCE(risks_json, '[]'),
			COALESCE(files_touched_json, '[]'), COALESCE(related_note_ids_json, '[]'),
			status, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM handoffs
		WHERE id = ?
	`, id)

	record, err := scanHandoff(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, err
	default:
		return &record, nil
	}
}

func (r *HandoffRepository) Search(ref scope.Ref, query string, limit int, states []handoff.Status) ([]handoff.Handoff, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, common.NewError(common.ErrInvalidInput, "query is required")
	}

	where := []string{
		"project_id = ?",
		"searchable = 1",
		"(LOWER(task) LIKE ? OR LOWER(summary) LIKE ?)",
	}
	args := []any{ref.ProjectID, likeQuery(query), likeQuery(query)}

	if len(states) > 0 {
		where = append(where, "status IN ("+placeholders(len(states))+")")
		for _, state := range states {
			args = append(args, string(state))
		}
	}

	rows, err := r.db.Query(`
		SELECT
			id, session_id, system_id, project_id, workspace_id, kind, task, summary,
			COALESCE(completed_json, '[]'), COALESCE(next_steps_json, '[]'),
			COALESCE(open_questions_json, '[]'), COALESCE(risks_json, '[]'),
			COALESCE(files_touched_json, '[]'), COALESCE(related_note_ids_json, '[]'),
			status, searchable, COALESCE(exclusion_reason, ''), created_at, updated_at
		FROM handoffs
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY created_at DESC
		LIMIT ?
	`, append(args, positiveLimit(limit))...)
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "search handoffs", err)
	}
	defer rows.Close()

	return scanHandoffs(rows)
}

func scanHandoff(scanner rowScanner) (handoff.Handoff, error) {
	var (
		record             handoff.Handoff
		kind               string
		status             string
		searchable         int
		exclusionReason    string
		createdAt          string
		updatedAt          string
		completedJSON      string
		nextStepsJSON      string
		openQuestionsJSON  string
		risksJSON          string
		filesTouchedJSON   string
		relatedNoteIDsJSON string
	)

	err := scanner.Scan(
		&record.ID,
		&record.SessionID,
		&record.Scope.SystemID,
		&record.Scope.ProjectID,
		&record.Scope.WorkspaceID,
		&kind,
		&record.Task,
		&record.Summary,
		&completedJSON,
		&nextStepsJSON,
		&openQuestionsJSON,
		&risksJSON,
		&filesTouchedJSON,
		&relatedNoteIDsJSON,
		&status,
		&searchable,
		&exclusionReason,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return handoff.Handoff{}, err
	}

	record.Kind = handoff.Kind(kind)
	record.Status = handoff.Status(status)
	record.Searchable = intToBool(searchable)
	record.ExclusionReason = exclusionReason

	record.Completed, err = unmarshalStringSlice(completedJSON)
	if err != nil {
		return handoff.Handoff{}, err
	}
	record.NextSteps, err = unmarshalStringSlice(nextStepsJSON)
	if err != nil {
		return handoff.Handoff{}, err
	}
	record.OpenQuestions, err = unmarshalStringSlice(openQuestionsJSON)
	if err != nil {
		return handoff.Handoff{}, err
	}
	record.Risks, err = unmarshalStringSlice(risksJSON)
	if err != nil {
		return handoff.Handoff{}, err
	}
	record.FilesTouched, err = unmarshalStringSlice(filesTouchedJSON)
	if err != nil {
		return handoff.Handoff{}, err
	}
	record.RelatedNoteIDs, err = unmarshalStringSlice(relatedNoteIDsJSON)
	if err != nil {
		return handoff.Handoff{}, err
	}
	record.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return handoff.Handoff{}, common.WrapError(common.ErrReadFailed, "parse handoff created_at", err)
	}
	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return handoff.Handoff{}, common.WrapError(common.ErrReadFailed, "parse handoff updated_at", err)
	}
	return record, nil
}

func scanHandoffs(rows *sql.Rows) ([]handoff.Handoff, error) {
	var records []handoff.Handoff
	for rows.Next() {
		record, err := scanHandoff(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "iterate handoffs", err)
	}
	return records, nil
}

func likeQuery(query string) string {
	return "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
}
