// Package main wires the codex-mem CLI entrypoint.
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
	logger := observability.NewBootstrapLogger(slog.LevelInfo)
	slog.SetDefault(logger)

	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("get working directory", "err", err)
		os.Exit(1)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}
	logger, logCloser, err := observability.NewLogger(cfg)
	if err != nil {
		slog.Default().Error("initialize logger", "err", err, "log_file", cfg.File.LogFilePath)
		os.Exit(1)
	}
	defer func() {
		_ = logCloser.Close()
	}()
	slog.SetDefault(logger)

	if err := app.Run(ctx, cfg, os.Args[1:], os.Stdin, os.Stdout); err != nil {
		logger.Error("command failed", "err", err)
		os.Exit(1)
	}
}
