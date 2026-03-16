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

// Run executes the selected CLI subcommand with the configured application wiring.
func Run(ctx context.Context, cfg config.Config, args []string, stdin io.Reader, stdout io.Writer) error {
	logger := slog.Default().With(
		"component", "cli",
		"config_dir", cfg.Meta.ConfigDir,
		"config_file", cfg.Meta.ConfigFilePath,
		"config_file_used", cfg.Meta.ConfigFileUsed,
	)
	command := "doctor"
	commandArgs := []string{}
	if len(args) > 0 {
		command = args[0]
		commandArgs = args[1:]
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
		defer func() {
			_ = instance.Close()
		}()
		logger.Info("migrations applied", "database", cfg.File.DatabasePath)
		_, err = fmt.Fprintf(stdout, "migrations applied successfully to %s\n", cfg.File.DatabasePath)
		return err
	case "doctor":
		options, err := parseDoctorOptions(commandArgs)
		if err != nil {
			return err
		}
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer func() {
			_ = instance.Close()
		}()
		if err := db.HealthCheck(ctx, instance.DB); err != nil {
			return err
		}
		runtimeDiagnostics, err := db.InspectRuntime(ctx, instance.DB)
		if err != nil {
			return err
		}
		logger.Info("doctor check passed",
			"database", cfg.File.DatabasePath,
			"system", cfg.File.DefaultSystemName,
			"config_file_used", doctorConfigFileUsed(cfg),
			"migrations_applied", runtimeDiagnostics.Migrations.Applied,
			"required_schema_ok", runtimeDiagnostics.RequiredSchemaOK,
			"fts_ready", runtimeDiagnostics.FTSReady,
			"json", options.JSON,
		)
		report := buildDoctorReport(cfg, runtimeDiagnostics, mcp.ToolCount())
		output := formatDoctorReport(report)
		if options.JSON {
			output, err = formatDoctorReportJSON(report)
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(stdout, output)
		return err
	case "ingest-imports":
		return runIngestImports(ctx, cfg, stdin, stdout, commandArgs)
	case "serve":
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer func() {
			_ = instance.Close()
		}()
		logger.Info("starting MCP stdio server")
		return mcp.ServeStdio(ctx, mcp.NewSDKServer(instance.Handlers), stdin, stdout)
	case "serve-http":
		options, err := parseServeHTTPOptions(commandArgs)
		if err != nil {
			return err
		}
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer func() {
			_ = instance.Close()
		}()
		logger.Info("starting MCP HTTP server",
			"listen", options.ListenAddr,
			"path", options.EndpointPath,
			"allowed_origins", options.AllowedOrigins,
		)
		return mcp.ServeHTTP(ctx, options.ListenAddr, mcp.NewSDKHTTPHandler(
			mcp.NewSDKServer(instance.Handlers),
			mcp.HTTPOptions{
				EndpointPath:   options.EndpointPath,
				AllowedOrigins: options.AllowedOrigins,
				SessionTimeout: options.SessionTimeout,
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
