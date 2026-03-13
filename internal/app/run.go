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
		"config_dir", cfg.Meta.ConfigDir,
		"config_file", cfg.Meta.ConfigFilePath,
		"config_file_used", cfg.Meta.ConfigFileUsed,
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
		logger.Info("migrations applied", "database", cfg.File.DatabasePath)
		_, err = fmt.Fprintf(stdout, "migrations applied successfully to %s\n", cfg.File.DatabasePath)
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
		logger.Info("doctor check passed",
			"database", cfg.File.DatabasePath,
			"system", cfg.File.DefaultSystemName,
			"config_file_used", doctorConfigFileUsed(cfg),
		)
		_, err = fmt.Fprintf(stdout, "doctor ok: database=%s system=%s config=%s\n", cfg.File.DatabasePath, cfg.File.DefaultSystemName, doctorConfigFileUsed(cfg))
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

func doctorConfigFileUsed(cfg config.Config) string {
	if cfg.Meta.ConfigFileUsed != "" {
		return cfg.Meta.ConfigFileUsed
	}
	return "none"
}
