package mcp

import (
	"context"
	"time"

	"codex-mem/internal/domain/agents"
	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/retrieval"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

type Response[T any] struct {
	Ok       bool                 `json:"ok"`
	Data     *T                   `json:"data,omitempty"`
	Warnings []common.Warning     `json:"warnings,omitempty"`
	Error    *common.ErrorPayload `json:"error,omitempty"`
}

type ResolveScopeData struct {
	Scope      scope.Scope `json:"scope"`
	ResolvedBy string      `json:"resolved_by"`
}

type StartSessionData struct {
	Session session.Session `json:"session"`
}

type SaveNoteData struct {
	Note         memory.Note `json:"note"`
	StoredAt     time.Time   `json:"stored_at"`
	Deduplicated bool        `json:"deduplicated"`
}

type SaveHandoffData struct {
	Handoff              handoff.Handoff `json:"handoff"`
	StoredAt             time.Time       `json:"stored_at"`
	EligibleForBootstrap bool            `json:"eligible_for_bootstrap"`
}

type BootstrapSessionData struct {
	Scope         scope.Scope            `json:"scope"`
	Session       session.Session        `json:"session"`
	LatestHandoff *handoff.Handoff       `json:"latest_handoff"`
	RecentNotes   []memory.Note          `json:"recent_notes"`
	RelatedNotes  []memory.Note          `json:"related_notes"`
	StartupBrief  retrieval.StartupBrief `json:"startup_brief"`
}

type GetRecentData struct {
	Handoffs []handoff.Handoff `json:"handoffs"`
	Notes    []memory.Note     `json:"notes"`
}

type GetRecordData struct {
	Record any `json:"record"`
}

type SearchData struct {
	Results []retrieval.SearchResult `json:"results"`
}

type InstallAgentsData struct {
	WrittenFiles []agents.FileChange `json:"written_files"`
	SkippedFiles []agents.FileChange `json:"skipped_files"`
}

func success[T any](data T, warnings []common.Warning) Response[T] {
	copy := data
	return Response[T]{
		Ok:       true,
		Data:     &copy,
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

func (h *Handlers) HandleMemoryStartSession(ctx context.Context, input session.StartInput) Response[StartSessionData] {
	output, err := h.MemoryStartSession(ctx, input)
	if err != nil {
		return failure[StartSessionData](err, common.ErrWriteFailed, "start session failed")
	}
	return success(StartSessionData{
		Session: output.Session,
	}, output.Warnings)
}

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

func (h *Handlers) HandleMemoryGetNote(ctx context.Context, input retrieval.GetRecordInput) Response[GetRecordData] {
	output, err := h.MemoryGetNote(ctx, input)
	if err != nil {
		return failure[GetRecordData](err, common.ErrReadFailed, "load record failed")
	}
	return success(GetRecordData{
		Record: output.Record,
	}, output.Warnings)
}

func (h *Handlers) HandleMemorySearch(ctx context.Context, input retrieval.SearchInput) Response[SearchData] {
	output, err := h.MemorySearch(ctx, input)
	if err != nil {
		return failure[SearchData](err, common.ErrReadFailed, "search memory failed")
	}
	return success(SearchData{
		Results: output.Results,
	}, output.Warnings)
}

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
