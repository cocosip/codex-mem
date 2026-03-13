package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"codex-mem/internal/buildinfo"
	"codex-mem/internal/config"
	"codex-mem/internal/db"
	"codex-mem/internal/mcp"
)

func Run(ctx context.Context, cfg config.Config, args []string, stdin io.Reader, stdout io.Writer) error {
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
	case "version":
		_, err := fmt.Fprintf(stdout, "codex-mem %s\ncommit=%s\ndate=%s\n", buildinfo.Summary(), buildinfo.Commit, buildinfo.Date)
		return err
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
		options, err := parseDoctorOptions(args[1:])
		if err != nil {
			return err
		}
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer instance.Close()
		if err := db.HealthCheck(ctx, instance.DB); err != nil {
			return err
		}
		runtimeDiagnostics, err := db.InspectRuntime(ctx, instance.DB)
		if err != nil {
			return err
		}
		mcpServer := mcp.NewServer(instance.Handlers)
		logger.Info("doctor check passed",
			"database", cfg.File.DatabasePath,
			"system", cfg.File.DefaultSystemName,
			"config_file_used", doctorConfigFileUsed(cfg),
			"migrations_applied", runtimeDiagnostics.Migrations.Applied,
			"required_schema_ok", runtimeDiagnostics.RequiredSchemaOK,
			"fts_ready", runtimeDiagnostics.FTSReady,
			"json", options.JSON,
		)
		report := buildDoctorReport(cfg, runtimeDiagnostics, mcpServer.ToolCount())
		output := formatDoctorReport(report)
		if options.JSON {
			output, err = formatDoctorReportJSON(report)
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(stdout, output)
		return err
	case "serve":
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer instance.Close()
		logger.Info("starting MCP stdio server")
		return mcp.NewServer(instance.Handlers).Serve(ctx, stdin, stdout)
	case "serve-http":
		options, err := parseServeHTTPOptions(args[1:])
		if err != nil {
			return err
		}
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer instance.Close()
		logger.Info("starting MCP HTTP server",
			"listen", options.ListenAddr,
			"path", options.EndpointPath,
			"allowed_origins", options.AllowedOrigins,
		)
		return mcp.ServeHTTP(ctx, options.ListenAddr, mcp.NewHTTPHandler(
			mcp.NewServer(instance.Handlers),
			mcp.HTTPOptions{
				EndpointPath:   options.EndpointPath,
				AllowedOrigins: options.AllowedOrigins,
			},
		))
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
