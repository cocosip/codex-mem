package main

import (
	"context"
	"log/slog"
	"os"

	"codex-mem/internal/app"
	"codex-mem/internal/config"
	"codex-mem/internal/observability"
)

func main() {
	ctx := context.Background()
	logger := observability.NewLogger(slog.LevelInfo)
	slog.SetDefault(logger)

	cfg, err := config.Load("")
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}
	logger = observability.NewLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	if err := app.Run(ctx, cfg, os.Args[1:], os.Stdout); err != nil {
		logger.Error("command failed", "err", err)
		os.Exit(1)
	}
}
