// Package retrieval assembles bootstrap, recent-history, and search views across durable records.
package retrieval

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

const defaultBootstrapNotes = 5
const relatedProjectRelationType = "referenced_by_note"

// ScopeResolver resolves the current workspace and project scope for retrieval operations.
type ScopeResolver interface {
	Resolve(ctx context.Context, input scope.ResolveInput) (scope.ResolveOutput, error)
}

// SessionStarter starts a new session tied to a resolved scope.
type SessionStarter interface {
	Start(ctx context.Context, input session.StartInput) (session.StartOutput, error)
}

// MemoryReader defines the note-reading operations used by retrieval workflows.
type MemoryReader interface {
	ListRecentByWorkspace(workspaceID string, limit int, minImportance int) ([]memory.Note, error)
	ListRecentByProject(projectID string, excludeWorkspaceID string, limit int, minImportance int) ([]memory.Note, error)
	ListRecentByProjects(systemID string, projectIDs []string, limit int, minImportance int) ([]memory.Note, error)
	GetByID(id string) (*memory.Note, error)
	ListRelatedProjectIDs(projectID string, limit int) ([]string, error)
	Search(scope scope.Ref, query string, limit int, minImportance int, types []memory.NoteType, states []memory.Status) ([]memory.Note, error)
	SearchProjects(systemID string, projectIDs []string, query string, limit int, minImportance int, types []memory.NoteType, states []memory.Status) ([]memory.Note, error)
}

// HandoffReader defines the handoff-reading operations used by retrieval workflows.
type HandoffReader interface {
	FindLatestOpenInWorkspace(workspaceID string) (*handoff.Handoff, error)
	FindLatestOpenInProject(projectID string, excludeWorkspaceID string) (*handoff.Handoff, error)
	ListRecentByWorkspace(workspaceID string, limit int) ([]handoff.Handoff, error)
	ListRecentByProject(projectID string, excludeWorkspaceID string, limit int) ([]handoff.Handoff, error)
	GetByID(id string) (*handoff.Handoff, error)
	Search(scope scope.Ref, query string, limit int, states []handoff.Status) ([]handoff.Handoff, error)
}

// Service coordinates retrieval operations across scope, session, notes, and handoffs.
type Service struct {
	scopeResolver  ScopeResolver
	sessionStarter SessionStarter
	memoryReader   MemoryReader
	handoffReader  HandoffReader
}

// NewService constructs a retrieval Service from its collaborating dependencies.
func NewService(scopeResolver ScopeResolver, sessionStarter SessionStarter, memoryReader MemoryReader, handoffReader HandoffReader) *Service {
	return &Service{
		scopeResolver:  scopeResolver,
		sessionStarter: sessionStarter,
		memoryReader:   memoryReader,
		handoffReader:  handoffReader,
	}
}

// BootstrapInput controls how a new session bootstrap should resolve and hydrate context.
type BootstrapInput struct {
	CWD                    string
	Task                   string
	BranchName             string
	RepoRemote             string
	IncludeRelatedProjects bool
	RelatedReason          string
	MaxNotes               int
	MaxHandoffs            int
}

// StartupBrief summarizes the most relevant context to resume a task.
type StartupBrief struct {
	CurrentTask        string   `json:"current_task,omitempty"`
	LastKnownState     string   `json:"last_known_state,omitempty"`
	ImportantDecisions []string `json:"important_decisions,omitempty"`
	OpenTodos          []string `json:"open_todos,omitempty"`
	Risks              []string `json:"risks,omitempty"`
	TouchedFiles       []string `json:"touched_files,omitempty"`
	RelatedContext     []string `json:"related_context,omitempty"`
}

// BootstrapOutput returns the resolved scope, created session, and bootstrap context.
type BootstrapOutput struct {
	Scope         scope.Scope      `json:"scope"`
	Session       session.Session  `json:"session"`
	LatestHandoff *handoff.Handoff `json:"latest_handoff"`
	RecentNotes   []memory.Note    `json:"recent_notes"`
	RelatedNotes  []memory.Note    `json:"related_notes"`
	StartupBrief  StartupBrief     `json:"startup_brief"`
	Warnings      []common.Warning `json:"warnings"`
}

// GetRecentInput controls retrieval of recent notes and handoffs.
type GetRecentInput struct {
	Scope                  scope.Ref
	Limit                  int
	IncludeHandoffs        bool
	IncludeNotes           bool
	IncludeRelatedProjects bool
}

// GetRecentOutput returns recent handoffs, notes, and related warnings.
type GetRecentOutput struct {
	Handoffs []handoff.Handoff `json:"handoffs"`
	Notes    []memory.Note     `json:"notes"`
	Warnings []common.Warning  `json:"warnings"`
}

// RecordKind identifies the kind of durable record being requested.
type RecordKind string

// Supported durable record kinds.
const (
	RecordKindNote    RecordKind = "note"
	RecordKindHandoff RecordKind = "handoff"
)

// GetRecordInput identifies a single durable record to load.
type GetRecordInput struct {
	ID   string
	Kind RecordKind
}

// GetRecordOutput returns a single durable record and any warnings.
type GetRecordOutput struct {
	Record   any              `json:"record"`
	Warnings []common.Warning `json:"warnings"`
}

// SearchInput controls note and handoff search behavior.
type SearchInput struct {
	Query                  string
	Scope                  scope.Ref
	Types                  []string
	States                 []string
	MinImportance          int
	Limit                  int
	IncludeHandoffs        bool
	IncludeRelatedProjects bool
	Intent                 string
}

// SearchResult is a ranked search hit across notes and handoffs.
type SearchResult struct {
	Kind         RecordKind `json:"kind"`
	ID           string     `json:"id"`
	Scope        scope.Ref  `json:"scope"`
	State        string     `json:"state"`
	Title        string     `json:"title"`
	Summary      string     `json:"summary"`
	Importance   int        `json:"importance"`
	CreatedAt    time.Time  `json:"created_at"`
	Source       string     `json:"source,omitempty"`
	RelationType string     `json:"relation_type,omitempty"`
}

// SearchOutput returns ranked search results and related warnings.
type SearchOutput struct {
	Results  []SearchResult   `json:"results"`
	Warnings []common.Warning `json:"warnings"`
}

// BootstrapSession resolves scope, starts a session, and loads bootstrap context.
func (s *Service) BootstrapSession(ctx context.Context, input BootstrapInput) (BootstrapOutput, error) {
	resolveOutput, err := s.scopeResolver.Resolve(ctx, scope.ResolveInput{
		CWD:        input.CWD,
		BranchName: input.BranchName,
		RepoRemote: input.RepoRemote,
	})
	if err != nil {
		return BootstrapOutput{}, common.EnsureCoded(err, common.ErrInvalidScope, "resolve bootstrap scope")
	}

	startOutput, err := s.sessionStarter.Start(ctx, session.StartInput{
		Scope:      resolveOutput.Scope,
		Task:       input.Task,
		BranchName: input.BranchName,
	})
	if err != nil {
		return BootstrapOutput{}, common.EnsureCoded(err, common.ErrWriteFailed, "start bootstrap session")
	}

	warnings := common.MergeWarnings(resolveOutput.Warnings, startOutput.Warnings)

	latestHandoff, err := s.loadLatestHandoff(resolveOutput.Scope)
	if err != nil {
		return BootstrapOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "load bootstrap handoff")
	}
	if latestHandoff == nil {
		warnings = common.MergeWarnings(warnings, []common.Warning{{
			Code:    common.WarnNoPriorHandoff,
			Message: "no open handoff was found for the current workspace or project",
		}})
	} else if latestHandoff.Kind == handoff.KindRecovery {
		warnings = common.MergeWarnings(warnings, []common.Warning{{
			Code:    common.WarnRecoveryHandoffUsed,
			Message: "bootstrap fell back to a recovery-generated handoff",
		}})
	}

	notes, err := s.loadRecentNotes(resolveOutput.Scope, input.MaxNotes)
	if err != nil {
		return BootstrapOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "load bootstrap notes")
	}
	if len(notes) == 0 {
		warnings = common.MergeWarnings(warnings, []common.Warning{{
			Code:    common.WarnNoPriorNotes,
			Message: "no recent high-value notes were found for the current scope",
		}})
	}
	var relatedNotes []memory.Note
	if input.IncludeRelatedProjects {
		var relatedWarnings []common.Warning
		relatedNotes, relatedWarnings, err = s.loadRelatedNotes(resolveOutput.Scope.Ref(), input.MaxNotes)
		if err != nil {
			return BootstrapOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "load related bootstrap notes")
		}
		warnings = common.MergeWarnings(warnings, relatedWarnings)
	}

	return BootstrapOutput{
		Scope:         resolveOutput.Scope,
		Session:       startOutput.Session,
		LatestHandoff: latestHandoff,
		RecentNotes:   notes,
		RelatedNotes:  relatedNotes,
		StartupBrief:  synthesizeStartupBrief(strings.TrimSpace(input.Task), latestHandoff, notes),
		Warnings:      warnings,
	}, nil
}

// GetRecent loads recent notes and handoffs for the current scope and optional related projects.
func (s *Service) GetRecent(ctx context.Context, input GetRecentInput) (GetRecentOutput, error) {
	_ = ctx

	if err := input.Scope.Validate(); err != nil {
		return GetRecentOutput{}, err
	}

	includeNotes := input.IncludeNotes
	includeHandoffs := input.IncludeHandoffs
	if !includeNotes && !includeHandoffs {
		includeNotes = true
		includeHandoffs = true
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultBootstrapNotes
	}

	output := GetRecentOutput{}
	if includeHandoffs {
		handoffs, err := s.loadRecentHandoffs(input.Scope, limit)
		if err != nil {
			return GetRecentOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "load recent handoffs")
		}
		output.Handoffs = handoffs
	}
	if includeNotes {
		notes, err := s.loadRecentNotesFromRef(input.Scope, limit)
		if err != nil {
			return GetRecentOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "load recent notes")
		}
		output.Notes = notes
	}
	if input.IncludeRelatedProjects {
		relatedNotes, warnings, err := s.loadRelatedNotes(input.Scope, limit)
		if err != nil {
			return GetRecentOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "load related recent notes")
		}
		output.Notes = append(output.Notes, relatedNotes...)
		output.Warnings = common.MergeWarnings(output.Warnings, warnings)
	}

	return output, nil
}

// GetRecord loads a single note or handoff by kind and identifier.
func (s *Service) GetRecord(ctx context.Context, input GetRecordInput) (GetRecordOutput, error) {
	_ = ctx

	id := strings.TrimSpace(input.ID)
	if id == "" {
		return GetRecordOutput{}, common.NewError(common.ErrInvalidInput, "id is required")
	}

	switch input.Kind {
	case RecordKindNote:
		record, err := s.memoryReader.GetByID(id)
		if err != nil {
			return GetRecordOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "load note record")
		}
		if record == nil {
			return GetRecordOutput{}, common.NewError(common.ErrRecordNotFound, "note record does not exist")
		}
		return GetRecordOutput{Record: *record}, nil
	case RecordKindHandoff:
		record, err := s.handoffReader.GetByID(id)
		if err != nil {
			return GetRecordOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "load handoff record")
		}
		if record == nil {
			return GetRecordOutput{}, common.NewError(common.ErrRecordNotFound, "handoff record does not exist")
		}
		return GetRecordOutput{Record: *record}, nil
	default:
		return GetRecordOutput{}, common.NewError(common.ErrInvalidInput, "kind must be note or handoff")
	}
}

// Search returns ranked note and handoff results for the given scope and filters.
func (s *Service) Search(ctx context.Context, input SearchInput) (SearchOutput, error) {
	_ = ctx

	if err := input.Scope.Validate(); err != nil {
		return SearchOutput{}, err
	}

	query := strings.TrimSpace(input.Query)
	if query == "" {
		return SearchOutput{}, common.NewError(common.ErrInvalidInput, "query is required")
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultBootstrapNotes
	}
	minImportance := input.MinImportance
	if minImportance <= 0 {
		minImportance = 1
	}

	noteTypes, err := parseNoteTypes(input.Types)
	if err != nil {
		return SearchOutput{}, err
	}
	noteStates, handoffStates, err := parseStates(input.States)
	if err != nil {
		return SearchOutput{}, err
	}

	candidateLimit := limit * 3
	if candidateLimit < 10 {
		candidateLimit = 10
	}

	noteResults, err := s.memoryReader.Search(input.Scope, query, candidateLimit, minImportance, noteTypes, noteStates)
	if err != nil {
		return SearchOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "search notes")
	}

	var handoffResults []handoff.Handoff
	if input.IncludeHandoffs {
		handoffResults, err = s.handoffReader.Search(input.Scope, query, candidateLimit, handoffStates)
		if err != nil {
			return SearchOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "search handoffs")
		}
	}

	var warnings []common.Warning
	if input.IncludeRelatedProjects {
		relatedNotes, relatedWarnings, err := s.loadRelatedSearchResults(input.Scope, query, candidateLimit, minImportance, noteTypes, noteStates)
		if err != nil {
			return SearchOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "search related notes")
		}
		noteResults = append(noteResults, relatedNotes...)
		warnings = common.MergeWarnings(warnings, relatedWarnings)
	}

	results, deduped := rankResults(input.Scope, query, input.Intent, noteResults, handoffResults)
	if len(results) > limit {
		results = results[:limit]
	}
	if deduped > 0 {
		warnings = common.MergeWarnings(warnings, []common.Warning{{
			Code:    common.WarnDedupeApplied,
			Message: "search results were deduplicated before returning",
		}})
	}

	output := SearchOutput{Results: results, Warnings: warnings}
	return output, nil
}

func (s *Service) loadLatestHandoff(currentScope scope.Scope) (*handoff.Handoff, error) {
	workspaceHandoff, err := s.handoffReader.FindLatestOpenInWorkspace(currentScope.WorkspaceID)
	if err != nil {
		return nil, common.EnsureCoded(err, common.ErrReadFailed, "load workspace handoff")
	}
	if workspaceHandoff != nil {
		return workspaceHandoff, nil
	}
	projectHandoff, err := s.handoffReader.FindLatestOpenInProject(currentScope.ProjectID, currentScope.WorkspaceID)
	if err != nil {
		return nil, common.EnsureCoded(err, common.ErrReadFailed, "load project handoff")
	}
	return projectHandoff, nil
}

func (s *Service) loadRecentNotes(currentScope scope.Scope, limit int) ([]memory.Note, error) {
	return s.loadRecentNotesFromRef(currentScope.Ref(), limit)
}

func (s *Service) loadRecentNotesFromRef(currentScope scope.Ref, limit int) ([]memory.Note, error) {
	if limit <= 0 {
		limit = defaultBootstrapNotes
	}

	notes, err := s.memoryReader.ListRecentByWorkspace(currentScope.WorkspaceID, limit, 3)
	if err != nil {
		return nil, common.EnsureCoded(err, common.ErrReadFailed, "list workspace notes")
	}
	if len(notes) >= limit {
		return notes, nil
	}

	projectNotes, err := s.memoryReader.ListRecentByProject(currentScope.ProjectID, currentScope.WorkspaceID, limit-len(notes), 3)
	if err != nil {
		return nil, common.EnsureCoded(err, common.ErrReadFailed, "list project notes")
	}
	return append(notes, projectNotes...), nil
}

func (s *Service) loadRecentHandoffs(currentScope scope.Ref, limit int) ([]handoff.Handoff, error) {
	if limit <= 0 {
		limit = defaultBootstrapNotes
	}

	handoffs, err := s.handoffReader.ListRecentByWorkspace(currentScope.WorkspaceID, limit)
	if err != nil {
		return nil, common.EnsureCoded(err, common.ErrReadFailed, "list workspace handoffs")
	}
	if len(handoffs) >= limit {
		return handoffs, nil
	}

	projectHandoffs, err := s.handoffReader.ListRecentByProject(currentScope.ProjectID, currentScope.WorkspaceID, limit-len(handoffs))
	if err != nil {
		return nil, common.EnsureCoded(err, common.ErrReadFailed, "list project handoffs")
	}
	return append(handoffs, projectHandoffs...), nil
}

func (s *Service) loadRelatedNotes(currentScope scope.Ref, limit int) ([]memory.Note, []common.Warning, error) {
	projectIDs, warnings, err := s.resolveRelatedProjectIDs(currentScope.ProjectID, limit)
	if err != nil {
		return nil, nil, err
	}
	if len(projectIDs) == 0 {
		return nil, warnings, nil
	}

	notes, err := s.memoryReader.ListRecentByProjects(currentScope.SystemID, projectIDs, limit, 3)
	if err != nil {
		return nil, nil, common.EnsureCoded(err, common.ErrReadFailed, "list related project notes")
	}
	for i := range notes {
		notes[i].RelationType = relatedProjectRelationType
	}
	if len(notes) == 0 {
		return nil, common.MergeWarnings(warnings, []common.Warning{{
			Code:    common.WarnRelatedProjectsEmpty,
			Message: "related-project retrieval was attempted but returned no results",
		}}), nil
	}
	return notes, warnings, nil
}

func (s *Service) loadRelatedSearchResults(currentScope scope.Ref, query string, limit int, minImportance int, types []memory.NoteType, states []memory.Status) ([]memory.Note, []common.Warning, error) {
	projectIDs, warnings, err := s.resolveRelatedProjectIDs(currentScope.ProjectID, limit)
	if err != nil {
		return nil, nil, err
	}
	if len(projectIDs) == 0 {
		return nil, warnings, nil
	}

	notes, err := s.memoryReader.SearchProjects(currentScope.SystemID, projectIDs, query, limit, minImportance, types, states)
	if err != nil {
		return nil, nil, common.EnsureCoded(err, common.ErrReadFailed, "search related project notes")
	}
	for i := range notes {
		notes[i].RelationType = relatedProjectRelationType
	}
	if len(notes) == 0 {
		return nil, common.MergeWarnings(warnings, []common.Warning{{
			Code:    common.WarnRelatedProjectsEmpty,
			Message: "related-project retrieval was attempted but returned no results",
		}}), nil
	}
	return notes, warnings, nil
}

func (s *Service) resolveRelatedProjectIDs(projectID string, limit int) ([]string, []common.Warning, error) {
	projectIDs, err := s.memoryReader.ListRelatedProjectIDs(projectID, limit)
	if err != nil {
		return nil, nil, common.EnsureCoded(err, common.ErrReadFailed, "resolve related project ids")
	}
	if len(projectIDs) == 0 {
		return nil, []common.Warning{{
			Code:    common.WarnRelatedProjectsSkipped,
			Message: "related-project retrieval was requested but no related project references were available",
		}}, nil
	}
	return projectIDs, nil, nil
}

func synthesizeStartupBrief(task string, latestHandoff *handoff.Handoff, notes []memory.Note) StartupBrief {
	brief := StartupBrief{
		CurrentTask: task,
	}
	if latestHandoff != nil {
		if brief.CurrentTask == "" {
			brief.CurrentTask = latestHandoff.Task
		}
		brief.LastKnownState = latestHandoff.Summary
		brief.OpenTodos = appendUnique(brief.OpenTodos, latestHandoff.NextSteps...)
		brief.Risks = appendUnique(brief.Risks, latestHandoff.Risks...)
		brief.TouchedFiles = appendUnique(brief.TouchedFiles, latestHandoff.FilesTouched...)
	}

	for _, note := range notes {
		switch note.Type {
		case memory.NoteTypeDecision:
			brief.ImportantDecisions = appendUnique(brief.ImportantDecisions, note.Content)
		case memory.NoteTypeTodo:
			brief.OpenTodos = appendUnique(brief.OpenTodos, note.Title)
		}
		brief.TouchedFiles = appendUnique(brief.TouchedFiles, note.FilePaths...)
		if brief.LastKnownState == "" {
			brief.LastKnownState = note.Content
		}
	}

	return brief
}

func appendUnique(existing []string, values ...string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		existing = append(existing, trimmed)
	}
	return existing
}

type scoredResult struct {
	result SearchResult
	score  float64
}

func rankResults(current scope.Ref, query string, intent string, notes []memory.Note, handoffs []handoff.Handoff) ([]SearchResult, int) {
	terms := queryTerms(query)
	intent = strings.TrimSpace(strings.ToLower(intent))

	candidates := make([]scoredResult, 0, len(notes)+len(handoffs))
	for _, note := range notes {
		candidates = append(candidates, scoredResult{
			result: SearchResult{
				Kind:         RecordKindNote,
				ID:           note.ID,
				Scope:        note.Scope,
				State:        string(note.Status),
				Title:        note.Title,
				Summary:      note.Content,
				Importance:   note.Importance,
				CreatedAt:    note.CreatedAt,
				Source:       string(note.Source),
				RelationType: note.RelationType,
			},
			score: noteScore(current, note, terms, intent),
		})
	}
	for _, record := range handoffs {
		candidates = append(candidates, scoredResult{
			result: SearchResult{
				Kind:       RecordKindHandoff,
				ID:         record.ID,
				Scope:      record.Scope,
				State:      string(record.Status),
				Title:      record.Task,
				Summary:    record.Summary,
				Importance: inferHandoffImportance(record),
				CreatedAt:  record.CreatedAt,
			},
			score: handoffScore(current, record, terms, intent),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].result.CreatedAt.After(candidates[j].result.CreatedAt)
		}
		return candidates[i].score > candidates[j].score
	})

	results := make([]SearchResult, 0, len(candidates))
	for _, candidate := range candidates {
		results = append(results, candidate.result)
	}
	return dedupeResults(results)
}

func noteScore(current scope.Ref, note memory.Note, terms []string, intent string) float64 {
	score := scopeScore(current, note.Scope) + noteStateScore(note.Status) + float64(note.Importance)*10
	score += noteSourceScore(note.Source)
	score += textOverlapScore(terms, note.Title+" "+note.Content)
	score += recencyScore(note.CreatedAt)
	switch intent {
	case "decision":
		if note.Type == memory.NoteTypeDecision {
			score += 12
		}
	case "bugfix":
		if note.Type == memory.NoteTypeBugfix {
			score += 12
		}
	case "discovery":
		if note.Type == memory.NoteTypeDiscovery {
			score += 12
		}
	}
	return score
}

func noteSourceScore(source memory.Source) float64 {
	switch source {
	case memory.SourceCodexExplicit:
		return 6
	case memory.SourceRecoveryGenerated:
		return -2
	default:
		return 0
	}
}

func handoffScore(current scope.Ref, record handoff.Handoff, terms []string, intent string) float64 {
	score := scopeScore(current, record.Scope) + handoffStateScore(record.Status) + float64(inferHandoffImportance(record))*10
	score += textOverlapScore(terms, record.Task+" "+record.Summary)
	score += recencyScore(record.CreatedAt)
	if intent == "continuation" {
		score += 15
	}
	return score
}

func scopeScore(current scope.Ref, target scope.Ref) float64 {
	switch {
	case current.WorkspaceID == target.WorkspaceID:
		return 100
	case current.ProjectID == target.ProjectID:
		return 70
	case current.SystemID == target.SystemID:
		return 20
	default:
		return 0
	}
}

func noteStateScore(status memory.Status) float64 {
	switch status {
	case memory.StatusActive:
		return 12
	case memory.StatusResolved:
		return 6
	default:
		return 0
	}
}

func handoffStateScore(status handoff.Status) float64 {
	switch status {
	case handoff.StatusOpen:
		return 12
	case handoff.StatusCompleted:
		return 6
	default:
		return 0
	}
}

func inferHandoffImportance(record handoff.Handoff) int {
	switch record.Status {
	case handoff.StatusOpen:
		return 4
	case handoff.StatusCompleted:
		return 3
	default:
		return 2
	}
}

func textOverlapScore(terms []string, haystack string) float64 {
	if len(terms) == 0 {
		return 0
	}
	lowered := strings.ToLower(haystack)
	var score float64
	for _, term := range terms {
		if strings.Contains(lowered, term) {
			score += 8
		}
	}
	return score
}

func recencyScore(createdAt time.Time) float64 {
	age := time.Since(createdAt)
	if age < 0 {
		age = 0
	}
	switch {
	case age < 24*time.Hour:
		return 10
	case age < 7*24*time.Hour:
		return 6
	case age < 30*24*time.Hour:
		return 3
	default:
		return 1
	}
}

func queryTerms(query string) []string {
	parts := strings.Fields(strings.ToLower(query))
	seen := make(map[string]struct{}, len(parts))
	terms := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		terms = append(terms, part)
	}
	return terms
}

func parseNoteTypes(values []string) ([]memory.NoteType, error) {
	if len(values) == 0 {
		return nil, nil
	}
	types := make([]memory.NoteType, 0, len(values))
	seen := make(map[memory.NoteType]struct{}, len(values))
	for _, value := range values {
		noteType := memory.NoteType(strings.TrimSpace(value))
		if err := noteType.Validate(); err != nil {
			return nil, err
		}
		if _, ok := seen[noteType]; ok {
			continue
		}
		seen[noteType] = struct{}{}
		types = append(types, noteType)
	}
	return types, nil
}

func parseStates(values []string) ([]memory.Status, []handoff.Status, error) {
	if len(values) == 0 {
		return nil, nil, nil
	}

	var noteStates []memory.Status
	var handoffStates []handoff.Status
	noteSeen := map[memory.Status]struct{}{}
	handoffSeen := map[handoff.Status]struct{}{}

	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if noteState := memory.Status(normalized); noteState.Validate() == nil {
			if _, ok := noteSeen[noteState]; !ok {
				noteSeen[noteState] = struct{}{}
				noteStates = append(noteStates, noteState)
			}
			continue
		}
		if handoffState := handoff.Status(normalized); handoffState.Validate() == nil {
			if _, ok := handoffSeen[handoffState]; !ok {
				handoffSeen[handoffState] = struct{}{}
				handoffStates = append(handoffStates, handoffState)
			}
			continue
		}
		return nil, nil, common.NewError(common.ErrInvalidInput, fmt.Sprintf("invalid state %q", normalized))
	}
	return noteStates, handoffStates, nil
}

func dedupeResults(results []SearchResult) ([]SearchResult, int) {
	seen := make(map[string]struct{}, len(results))
	deduped := make([]SearchResult, 0, len(results))
	suppressed := 0
	for _, result := range results {
		key := dedupeKey(result)
		if _, ok := seen[key]; ok {
			suppressed++
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, result)
	}
	return deduped, suppressed
}

func dedupeKey(result SearchResult) string {
	switch result.Kind {
	case RecordKindHandoff:
		return string(result.Kind) + ":" + strings.ToLower(strings.TrimSpace(result.Title))
	default:
		return string(result.Kind) + ":" + strings.ToLower(strings.TrimSpace(result.Title))
	}
}
