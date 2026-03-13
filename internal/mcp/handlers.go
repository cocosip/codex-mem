package mcp

import (
	"context"

	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/retrieval"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

type Handlers struct {
	scopeService     *scope.Service
	sessionService   *session.Service
	memoryService    *memory.Service
	handoffService   *handoff.Service
	retrievalService *retrieval.Service
}

func NewHandlers(scopeService *scope.Service, sessionService *session.Service, memoryService *memory.Service, handoffService *handoff.Service, retrievalService *retrieval.Service) *Handlers {
	return &Handlers{
		scopeService:     scopeService,
		sessionService:   sessionService,
		memoryService:    memoryService,
		handoffService:   handoffService,
		retrievalService: retrievalService,
	}
}

func (h *Handlers) MemoryResolveScope(ctx context.Context, input scope.ResolveInput) (scope.ResolveOutput, error) {
	return h.scopeService.Resolve(ctx, input)
}

func (h *Handlers) MemoryStartSession(ctx context.Context, input session.StartInput) (session.StartOutput, error) {
	return h.sessionService.Start(ctx, input)
}

func (h *Handlers) MemorySaveNote(ctx context.Context, input memory.SaveInput) (memory.SaveOutput, error) {
	return h.memoryService.SaveNote(ctx, input)
}

func (h *Handlers) MemorySaveHandoff(ctx context.Context, input handoff.SaveInput) (handoff.SaveOutput, error) {
	return h.handoffService.SaveHandoff(ctx, input)
}

func (h *Handlers) MemoryBootstrapSession(ctx context.Context, input retrieval.BootstrapInput) (retrieval.BootstrapOutput, error) {
	return h.retrievalService.BootstrapSession(ctx, input)
}

func (h *Handlers) MemoryGetRecent(ctx context.Context, input retrieval.GetRecentInput) (retrieval.GetRecentOutput, error) {
	return h.retrievalService.GetRecent(ctx, input)
}

func (h *Handlers) MemoryGetNote(ctx context.Context, input retrieval.GetRecordInput) (retrieval.GetRecordOutput, error) {
	return h.retrievalService.GetRecord(ctx, input)
}

func (h *Handlers) MemorySearch(ctx context.Context, input retrieval.SearchInput) (retrieval.SearchOutput, error) {
	return h.retrievalService.Search(ctx, input)
}
