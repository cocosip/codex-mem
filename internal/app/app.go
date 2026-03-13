package app

import (
	"context"
	"database/sql"
	"log/slog"

	"codex-mem/internal/config"
	"codex-mem/internal/db"
	"codex-mem/internal/domain/agents"
	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/retrieval"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
	"codex-mem/internal/mcp"
)

// App wires the Phase 1 foundation services together.
type App struct {
	Config           config.Config
	DB               *sql.DB
	Logger           *slog.Logger
	ScopeService     *scope.Service
	SessionService   *session.Service
	MemoryService    *memory.Service
	HandoffService   *handoff.Service
	RetrievalService *retrieval.Service
	AgentsService    *agents.Service
	Handlers         *mcp.Handlers
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	logger := slog.Default().With(
		"component", "app",
		"db_path", cfg.File.DatabasePath,
		"config_file", cfg.Meta.ConfigFilePath,
		"config_file_used", cfg.Meta.ConfigFileUsed,
	)
	store, err := db.Open(ctx, db.Options{
		Path:        cfg.File.DatabasePath,
		DriverName:  cfg.File.SQLiteDriver,
		BusyTimeout: cfg.File.BusyTimeout,
		JournalMode: cfg.File.JournalMode,
	})
	if err != nil {
		return nil, err
	}

	clock := common.RealClock{}
	ids := common.DefaultIDFactory{Clock: clock}
	scopeRepo := db.NewScopeRepository(store, clock)
	sessionRepo := db.NewSessionRepository(store)
	memoryRepo := db.NewMemoryRepository(store)
	handoffRepo := db.NewHandoffRepository(store)
	scopeService := scope.NewService(scopeRepo, scope.Options{
		DefaultSystemName: cfg.File.DefaultSystemName,
	})
	sessionService := session.NewService(sessionRepo, session.Options{
		Clock:     clock,
		IDFactory: ids,
	})
	memoryService := memory.NewService(memoryRepo, memory.Options{
		Clock:     clock,
		IDFactory: ids,
	})
	agentsService := agents.NewService(agents.Options{})
	handoffService := handoff.NewService(handoffRepo, handoff.Options{
		Clock:     clock,
		IDFactory: ids,
	})
	retrievalService := retrieval.NewService(scopeService, sessionService, memoryRepo, handoffRepo)

	return &App{
		Config:           cfg,
		DB:               store,
		Logger:           logger,
		ScopeService:     scopeService,
		SessionService:   sessionService,
		MemoryService:    memoryService,
		HandoffService:   handoffService,
		RetrievalService: retrievalService,
		AgentsService:    agentsService,
		Handlers:         mcp.NewHandlers(scopeService, sessionService, memoryService, handoffService, retrievalService, agentsService),
	}, nil
}

func (a *App) Close() error {
	if a == nil || a.DB == nil {
		return nil
	}
	return a.DB.Close()
}
