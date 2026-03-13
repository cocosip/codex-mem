package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	DatabasePath      string
	ConfigDir         string
	ConfigFilePath    string
	DefaultSystemName string
	SQLiteDriver      string
	BusyTimeout       time.Duration
	JournalMode       string
	LogLevel          slog.Level
}

func Load(cwd string) (Config, error) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return Config{}, fmt.Errorf("get working directory: %w", err)
		}
	}

	absCWD, err := filepath.Abs(cwd)
	if err != nil {
		return Config{}, fmt.Errorf("resolve working directory: %w", err)
	}

	configDir := filepath.Join(absCWD, "configs")
	configFilePath := os.Getenv("CODEX_MEM_CONFIG_FILE")
	if configFilePath == "" {
		configFilePath = filepath.Join(configDir, "codex-mem.json")
	} else if !filepath.IsAbs(configFilePath) {
		configFilePath = filepath.Join(configDir, configFilePath)
	}
	configFilePath, err = filepath.Abs(configFilePath)
	if err != nil {
		return Config{}, fmt.Errorf("resolve config file path: %w", err)
	}

	databasePath := os.Getenv("CODEX_MEM_DB_PATH")
	if databasePath == "" {
		databasePath = filepath.Join(absCWD, "data", "codex-mem.db")
	}
	if databasePath != ":memory:" {
		databasePath, err = filepath.Abs(databasePath)
		if err != nil {
			return Config{}, fmt.Errorf("resolve database path: %w", err)
		}
	}

	systemName := os.Getenv("CODEX_MEM_SYSTEM_NAME")
	if systemName == "" {
		systemName = "codex-mem"
	}

	driverName := os.Getenv("CODEX_MEM_SQLITE_DRIVER")
	if driverName == "" {
		driverName = "sqlite"
	}

	logLevel, err := parseLogLevel(os.Getenv("CODEX_MEM_LOG_LEVEL"))
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabasePath:      databasePath,
		ConfigDir:         configDir,
		ConfigFilePath:    configFilePath,
		DefaultSystemName: systemName,
		SQLiteDriver:      driverName,
		BusyTimeout:       5 * time.Second,
		JournalMode:       "WAL",
		LogLevel:          logLevel,
	}, nil
}

func parseLogLevel(value string) (slog.Level, error) {
	switch value {
	case "", "info", "INFO":
		return slog.LevelInfo, nil
	case "debug", "DEBUG":
		return slog.LevelDebug, nil
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn, nil
	case "error", "ERROR":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid CODEX_MEM_LOG_LEVEL %q", value)
	}
}
