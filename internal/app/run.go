package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"codex-mem/internal/config"
	"codex-mem/internal/db"
)

func Run(ctx context.Context, cfg config.Config, args []string, stdout io.Writer) error {
	logger := slog.Default().With(
		"component", "cli",
		"config_dir", cfg.ConfigDir,
		"config_file", cfg.ConfigFilePath,
	)
	command := "doctor"
	if len(args) > 0 {
		command = args[0]
	}
	logger.Info("starting command", "command", command)

	switch command {
	case "migrate":
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer instance.Close()
		logger.Info("migrations applied", "database", cfg.DatabasePath)
		_, err = fmt.Fprintf(stdout, "migrations applied successfully to %s\n", cfg.DatabasePath)
		return err
	case "doctor":
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer instance.Close()
		if err := db.HealthCheck(ctx, instance.DB); err != nil {
			return err
		}
		logger.Info("doctor check passed", "database", cfg.DatabasePath, "system", cfg.DefaultSystemName)
		_, err = fmt.Fprintf(stdout, "doctor ok: database=%s system=%s\n", cfg.DatabasePath, cfg.DefaultSystemName)
		return err
	case "serve":
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer instance.Close()
		logger.Info("serve skeleton ready")
		_, err = fmt.Fprintln(stdout, "serve mode skeleton ready: MCP transport wiring is the next implementation slice")
		return err
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}
