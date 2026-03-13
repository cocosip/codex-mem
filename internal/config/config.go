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
	LogDir            string
	LogFilePath       string
	DefaultSystemName string
	SQLiteDriver      string
	BusyTimeout       time.Duration
	JournalMode       string
	LogLevel          slog.Level
	LogMaxSizeMB      int
	LogMaxBackups     int
	LogMaxAgeDays     int
	LogCompress       bool
	LogAlsoStderr     bool
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
	logDir := filepath.Join(absCWD, "logs")
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
	logFilePath, err := resolveLogFilePath(logDir, os.Getenv("CODEX_MEM_LOG_FILE"))
	if err != nil {
		return Config{}, err
	}
	logMaxSizeMB, err := parsePositiveInt("CODEX_MEM_LOG_MAX_SIZE_MB", os.Getenv("CODEX_MEM_LOG_MAX_SIZE_MB"), 20)
	if err != nil {
		return Config{}, err
	}
	logMaxBackups, err := parsePositiveInt("CODEX_MEM_LOG_MAX_BACKUPS", os.Getenv("CODEX_MEM_LOG_MAX_BACKUPS"), 10)
	if err != nil {
		return Config{}, err
	}
	logMaxAgeDays, err := parsePositiveInt("CODEX_MEM_LOG_MAX_AGE_DAYS", os.Getenv("CODEX_MEM_LOG_MAX_AGE_DAYS"), 30)
	if err != nil {
		return Config{}, err
	}
	logCompress, err := parseBool("CODEX_MEM_LOG_COMPRESS", os.Getenv("CODEX_MEM_LOG_COMPRESS"), true)
	if err != nil {
		return Config{}, err
	}
	logAlsoStderr, err := parseBool("CODEX_MEM_LOG_STDERR", os.Getenv("CODEX_MEM_LOG_STDERR"), true)
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabasePath:      databasePath,
		ConfigDir:         configDir,
		ConfigFilePath:    configFilePath,
		LogDir:            logDir,
		LogFilePath:       logFilePath,
		DefaultSystemName: systemName,
		SQLiteDriver:      driverName,
		BusyTimeout:       5 * time.Second,
		JournalMode:       "WAL",
		LogLevel:          logLevel,
		LogMaxSizeMB:      logMaxSizeMB,
		LogMaxBackups:     logMaxBackups,
		LogMaxAgeDays:     logMaxAgeDays,
		LogCompress:       logCompress,
		LogAlsoStderr:     logAlsoStderr,
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

func resolveLogFilePath(logDir, envValue string) (string, error) {
	logFilePath := envValue
	if logFilePath == "" {
		logFilePath = filepath.Join(logDir, "codex-mem.log")
	} else if !filepath.IsAbs(logFilePath) {
		logFilePath = filepath.Join(logDir, logFilePath)
	}
	absPath, err := filepath.Abs(logFilePath)
	if err != nil {
		return "", fmt.Errorf("resolve log file path: %w", err)
	}
	return absPath, nil
}

func parsePositiveInt(name, value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid %s %q", name, value)
	}
	return parsed, nil
}

func parseBool(name, value string, fallback bool) (bool, error) {
	switch value {
	case "":
		return fallback, nil
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true, nil
	case "0", "false", "FALSE", "False", "no", "NO", "off", "OFF":
		return false, nil
	default:
		return false, fmt.Errorf("invalid %s %q", name, value)
	}
}
