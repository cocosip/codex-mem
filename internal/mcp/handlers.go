package mcp

import (
	"context"

	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

type Handlers struct {
	scopeService   *scope.Service
	sessionService *session.Service
}

func NewHandlers(scopeService *scope.Service, sessionService *session.Service) *Handlers {
	return &Handlers{
		scopeService:   scopeService,
		sessionService: sessionService,
	}
}

func (h *Handlers) MemoryResolveScope(ctx context.Context, input scope.ResolveInput) (scope.ResolveOutput, error) {
	return h.scopeService.Resolve(ctx, input)
}

func (h *Handlers) MemoryStartSession(ctx context.Context, input session.StartInput) (session.StartOutput, error) {
	return h.sessionService.Start(ctx, input)
}
