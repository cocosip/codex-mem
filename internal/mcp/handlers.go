package mcp

import (
	"context"

	"codex-mem/internal/domain/agents"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/retrieval"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

// Handlers groups the domain services exposed through MCP.
type Handlers struct {
	scopeService     *scope.Service
	sessionService   *session.Service
	memoryService    *memory.Service
	handoffService   *handoff.Service
	retrievalService *retrieval.Service
	agentsService    *agents.Service
}

// NewHandlers constructs the MCP handler facade for the supplied services.
func NewHandlers(scopeService *scope.Service, sessionService *session.Service, memoryService *memory.Service, handoffService *handoff.Service, retrievalService *retrieval.Service, agentsService *agents.Service) *Handlers {
	return &Handlers{
		scopeService:     scopeService,
		sessionService:   sessionService,
		memoryService:    memoryService,
		handoffService:   handoffService,
		retrievalService: retrievalService,
		agentsService:    agentsService,
	}
}

// MemoryResolveScope resolves system, project, and workspace scope.
func (h *Handlers) MemoryResolveScope(ctx context.Context, input scope.ResolveInput) (scope.ResolveOutput, error) {
	return h.scopeService.Resolve(ctx, input)
}

// MemoryStartSession starts a new active session.
func (h *Handlers) MemoryStartSession(ctx context.Context, input session.StartInput) (session.StartOutput, error) {
	return h.sessionService.Start(ctx, input)
}

// MemorySaveNote persists a structured memory note.
func (h *Handlers) MemorySaveNote(ctx context.Context, input memory.SaveInput) (memory.SaveOutput, error) {
	return h.memoryService.SaveNote(ctx, input)
}

// MemorySaveHandoff persists a continuation handoff record.
func (h *Handlers) MemorySaveHandoff(ctx context.Context, input handoff.SaveInput) (handoff.SaveOutput, error) {
	return h.handoffService.SaveHandoff(ctx, input)
}

// MemoryBootstrapSession starts a session and loads its bootstrap context.
func (h *Handlers) MemoryBootstrapSession(ctx context.Context, input retrieval.BootstrapInput) (retrieval.BootstrapOutput, error) {
	return h.retrievalService.BootstrapSession(ctx, input)
}

// MemoryGetRecent loads recent notes and handoffs for a scope.
func (h *Handlers) MemoryGetRecent(ctx context.Context, input retrieval.GetRecentInput) (retrieval.GetRecentOutput, error) {
	return h.retrievalService.GetRecent(ctx, input)
}

// MemoryGetNote loads a single durable record by id.
func (h *Handlers) MemoryGetNote(ctx context.Context, input retrieval.GetRecordInput) (retrieval.GetRecordOutput, error) {
	return h.retrievalService.GetRecord(ctx, input)
}

// MemorySearch searches durable notes and handoffs within a scope.
func (h *Handlers) MemorySearch(ctx context.Context, input retrieval.SearchInput) (retrieval.SearchOutput, error) {
	return h.retrievalService.Search(ctx, input)
}

// MemoryInstallAgents installs AGENTS.md guidance files.
func (h *Handlers) MemoryInstallAgents(ctx context.Context, input agents.InstallInput) (agents.InstallOutput, error) {
	return h.agentsService.Install(ctx, input)
}
