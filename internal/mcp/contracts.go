// Package mcp exposes the codex-mem MCP transport surface.
package mcp

import (
	"context"
	"time"

	"codex-mem/internal/domain/agents"
	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/imports"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/retrieval"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

// Response is the common success-or-failure envelope returned by MCP handlers.
type Response[T any] struct {
	Ok       bool                 `json:"ok"`
	Data     *T                   `json:"data,omitempty"`
	Warnings []common.Warning     `json:"warnings,omitempty"`
	Error    *common.ErrorPayload `json:"error,omitempty"`
}

// ResolveScopeData carries scope-resolution results in MCP responses.
type ResolveScopeData struct {
	Scope      scope.Scope `json:"scope"`
	ResolvedBy string      `json:"resolved_by"`
}

// StartSessionData carries session-start results in MCP responses.
type StartSessionData struct {
	Session session.Session `json:"session"`
}

// SaveNoteData carries note-persistence results in MCP responses.
type SaveNoteData struct {
	Note         memory.Note `json:"note"`
	StoredAt     time.Time   `json:"stored_at"`
	Deduplicated bool        `json:"deduplicated"`
}

// SaveHandoffData carries handoff-persistence results in MCP responses.
type SaveHandoffData struct {
	Handoff              handoff.Handoff `json:"handoff"`
	StoredAt             time.Time       `json:"stored_at"`
	EligibleForBootstrap bool            `json:"eligible_for_bootstrap"`
}

// SaveImportData carries import-audit persistence results in MCP responses.
type SaveImportData struct {
	Import       imports.Record `json:"import"`
	StoredAt     time.Time      `json:"stored_at"`
	Suppressed   bool           `json:"suppressed"`
	Deduplicated bool           `json:"deduplicated"`
}

// BootstrapSessionData carries bootstrap-session results in MCP responses.
type BootstrapSessionData struct {
	Scope         scope.Scope            `json:"scope"`
	Session       session.Session        `json:"session"`
	LatestHandoff *handoff.Handoff       `json:"latest_handoff"`
	RecentNotes   []memory.Note          `json:"recent_notes"`
	RelatedNotes  []memory.Note          `json:"related_notes"`
	StartupBrief  retrieval.StartupBrief `json:"startup_brief"`
}

// GetRecentData carries recent note and handoff results in MCP responses.
type GetRecentData struct {
	Handoffs []handoff.Handoff `json:"handoffs"`
	Notes    []memory.Note     `json:"notes"`
}

// GetRecordData carries a single durable record in MCP responses.
type GetRecordData struct {
	Record any `json:"record"`
}

// SearchData carries ranked retrieval results in MCP responses.
type SearchData struct {
	Results []retrieval.SearchResult `json:"results"`
}

// InstallAgentsData carries AGENTS installation results in MCP responses.
type InstallAgentsData struct {
	WrittenFiles []agents.FileChange `json:"written_files"`
	SkippedFiles []agents.FileChange `json:"skipped_files"`
}

func success[T any](data T, warnings []common.Warning) Response[T] {
	payload := data
	return Response[T]{
		Ok:       true,
		Data:     &payload,
		Warnings: warnings,
	}
}

func failure[T any](err error, fallbackCode string, fallbackMessage string) Response[T] {
	details := common.ErrorDetails(err, fallbackCode, fallbackMessage)
	return Response[T]{
		Ok:    false,
		Error: &details,
	}
}

// HandleMemoryResolveScope adapts scope resolution into an MCP response envelope.
func (h *Handlers) HandleMemoryResolveScope(ctx context.Context, input scope.ResolveInput) Response[ResolveScopeData] {
	output, err := h.MemoryResolveScope(ctx, input)
	if err != nil {
		return failure[ResolveScopeData](err, common.ErrInvalidScope, "resolve scope failed")
	}
	return success(ResolveScopeData{
		Scope:      output.Scope,
		ResolvedBy: output.ResolvedBy,
	}, output.Warnings)
}

// HandleMemoryStartSession adapts session start into an MCP response envelope.
func (h *Handlers) HandleMemoryStartSession(ctx context.Context, input session.StartInput) Response[StartSessionData] {
	output, err := h.MemoryStartSession(ctx, input)
	if err != nil {
		return failure[StartSessionData](err, common.ErrWriteFailed, "start session failed")
	}
	return success(StartSessionData{
		Session: output.Session,
	}, output.Warnings)
}

// HandleMemorySaveNote adapts note persistence into an MCP response envelope.
func (h *Handlers) HandleMemorySaveNote(ctx context.Context, input memory.SaveInput) Response[SaveNoteData] {
	output, err := h.MemorySaveNote(ctx, input)
	if err != nil {
		return failure[SaveNoteData](err, common.ErrWriteFailed, "save note failed")
	}
	return success(SaveNoteData{
		Note:         output.Note,
		StoredAt:     output.StoredAt,
		Deduplicated: output.Deduplicated,
	}, output.Warnings)
}

// HandleMemorySaveHandoff adapts handoff persistence into an MCP response envelope.
func (h *Handlers) HandleMemorySaveHandoff(ctx context.Context, input handoff.SaveInput) Response[SaveHandoffData] {
	output, err := h.MemorySaveHandoff(ctx, input)
	if err != nil {
		return failure[SaveHandoffData](err, common.ErrWriteFailed, "save handoff failed")
	}
	return success(SaveHandoffData{
		Handoff:              output.Handoff,
		StoredAt:             output.StoredAt,
		EligibleForBootstrap: output.EligibleForBootstrap,
	}, output.Warnings)
}

// HandleMemorySaveImport adapts import persistence into an MCP response envelope.
func (h *Handlers) HandleMemorySaveImport(ctx context.Context, input imports.SaveInput) Response[SaveImportData] {
	output, err := h.MemorySaveImport(ctx, input)
	if err != nil {
		return failure[SaveImportData](err, common.ErrWriteFailed, "save import failed")
	}
	return success(SaveImportData{
		Import:       output.Record,
		StoredAt:     output.StoredAt,
		Suppressed:   output.Suppressed,
		Deduplicated: output.Deduplicated,
	}, output.Warnings)
}

// HandleMemoryBootstrapSession adapts bootstrap retrieval into an MCP response envelope.
func (h *Handlers) HandleMemoryBootstrapSession(ctx context.Context, input retrieval.BootstrapInput) Response[BootstrapSessionData] {
	output, err := h.MemoryBootstrapSession(ctx, input)
	if err != nil {
		return failure[BootstrapSessionData](err, common.ErrReadFailed, "bootstrap session failed")
	}
	return success(BootstrapSessionData{
		Scope:         output.Scope,
		Session:       output.Session,
		LatestHandoff: output.LatestHandoff,
		RecentNotes:   output.RecentNotes,
		RelatedNotes:  output.RelatedNotes,
		StartupBrief:  output.StartupBrief,
	}, output.Warnings)
}

// HandleMemoryGetRecent adapts recent-record retrieval into an MCP response envelope.
func (h *Handlers) HandleMemoryGetRecent(ctx context.Context, input retrieval.GetRecentInput) Response[GetRecentData] {
	output, err := h.MemoryGetRecent(ctx, input)
	if err != nil {
		return failure[GetRecentData](err, common.ErrReadFailed, "load recent memory failed")
	}
	return success(GetRecentData{
		Handoffs: output.Handoffs,
		Notes:    output.Notes,
	}, output.Warnings)
}

// HandleMemoryGetNote adapts single-record retrieval into an MCP response envelope.
func (h *Handlers) HandleMemoryGetNote(ctx context.Context, input retrieval.GetRecordInput) Response[GetRecordData] {
	output, err := h.MemoryGetNote(ctx, input)
	if err != nil {
		return failure[GetRecordData](err, common.ErrReadFailed, "load record failed")
	}
	return success(GetRecordData{
		Record: output.Record,
	}, output.Warnings)
}

// HandleMemorySearch adapts search results into an MCP response envelope.
func (h *Handlers) HandleMemorySearch(ctx context.Context, input retrieval.SearchInput) Response[SearchData] {
	output, err := h.MemorySearch(ctx, input)
	if err != nil {
		return failure[SearchData](err, common.ErrReadFailed, "search memory failed")
	}
	return success(SearchData{
		Results: output.Results,
	}, output.Warnings)
}

// HandleMemoryInstallAgents adapts AGENTS installation into an MCP response envelope.
func (h *Handlers) HandleMemoryInstallAgents(ctx context.Context, input agents.InstallInput) Response[InstallAgentsData] {
	output, err := h.MemoryInstallAgents(ctx, input)
	if err != nil {
		return failure[InstallAgentsData](err, common.ErrAgentsWriteDenied, "install AGENTS failed")
	}
	return success(InstallAgentsData{
		WrittenFiles: output.WrittenFiles,
		SkippedFiles: output.SkippedFiles,
	}, output.Warnings)
}
