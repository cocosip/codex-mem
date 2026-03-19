package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"codex-mem/internal/buildinfo"
	"codex-mem/internal/config"
	"codex-mem/internal/db"
	"codex-mem/internal/mcp"
)

const (
	commandDoctor              = "doctor"
	commandListCommandExamples = "list-command-examples"
	commandIngestImports       = "ingest-imports"
	commandFollowImports       = "follow-imports"
	commandAuditFollowImports  = "audit-follow-imports"
	commandCleanupFollowImport = "cleanup-follow-imports"
)

// Run executes the selected CLI subcommand with the configured application wiring.
func Run(ctx context.Context, cfg config.Config, args []string, stdin io.Reader, stdout io.Writer) error {
	logger := slog.Default().With(
		"component", "cli",
		"config_dir", cfg.Meta.ConfigDir,
		"config_file", cfg.Meta.ConfigFilePath,
		"config_file_used", cfg.Meta.ConfigFileUsed,
	)
	command := commandDoctor
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
	case commandListCommandExamples:
		options, err := parseListCommandExamplesOptions(commandArgs)
		if err != nil {
			return err
		}
		report, err := commandExampleManifestReportFromEmbedded()
		if err != nil {
			return err
		}
		report, err = filterCommandExampleManifestReport(report, options.Commands, options.Examples, options.Formats, options.Tags)
		if err != nil {
			return err
		}
		if !options.JSON {
			_, err = io.WriteString(stdout, formatCommandExampleManifest(report))
			return err
		}
		output, err := formatCommandExampleManifestJSON(report)
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, output)
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
	case commandDoctor:
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
		followHealth, healthPruned, healthPruneReason, err := loadDoctorFollowImportsHealth(cfg.Meta.LogDir, options.PruneStaleFollowHealth, time.Now().UTC())
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
			"prune_stale_follow_health", options.PruneStaleFollowHealth,
			"follow_health_pruned", healthPruned,
		)
		report := buildDoctorReport(cfg, runtimeDiagnostics, mcp.ToolCount(), followHealth, healthPruned, healthPruneReason)
		output := formatDoctorReport(report)
		if options.JSON {
			output, err = formatDoctorReportJSON(report)
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(stdout, output)
		return err
	case commandIngestImports:
		return runIngestImports(ctx, cfg, stdin, stdout, commandArgs)
	case commandFollowImports:
		return runFollowImports(ctx, cfg, stdout, commandArgs)
	case commandAuditFollowImports:
		return runAuditFollowImports(cfg, stdout, commandArgs)
	case commandCleanupFollowImport:
		return runCleanupFollowImports(cfg, stdout, commandArgs)
	case "serve":
		instance, err := New(ctx, cfg)
		if err != nil {
			return err
		}
		defer func() {
			_ = instance.Close()
		}()
		logger.Info("starting MCP stdio server",
			"pid", os.Getpid(),
			"log_file", cfg.File.LogFilePath,
			"log_level", cfg.File.LogLevel.String(),
		)
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
			"pid", os.Getpid(),
			"listen", options.ListenAddr,
			"path", options.EndpointPath,
			"allowed_origins", options.AllowedOrigins,
			"log_file", cfg.File.LogFilePath,
			"log_level", cfg.File.LogLevel.String(),
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
