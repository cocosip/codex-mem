package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultConfigBaseName  = "codex-mem"
	defaultConfigExtension = ".json"
)

type Config struct {
	File FileConfig
	Meta LoadMetadata
}

type FileConfig struct {
	DatabasePath      string
	DefaultSystemName string
	SQLiteDriver      string
	BusyTimeout       time.Duration
	JournalMode       string
	LogLevel          slog.Level
	LogFilePath       string
	LogMaxSizeMB      int
	LogMaxBackups     int
	LogMaxAgeDays     int
	LogCompress       bool
	LogAlsoStderr     bool
}

type LoadMetadata struct {
	ConfigDir      string
	ConfigFilePath string
	ConfigFileUsed string
	LogDir         string
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

	configFilePath, explicitConfigPath, err := resolveConfigFilePath(configDir, os.Getenv("CODEX_MEM_CONFIG_FILE"))
	if err != nil {
		return Config{}, err
	}

	settings, loadedConfigPath, err := loadSettings(absCWD, configDir, logDir, configFilePath, explicitConfigPath)
	if err != nil {
		return Config{}, err
	}

	databasePath, err := resolveDataPath(absCWD, settings.databasePath)
	if err != nil {
		return Config{}, err
	}
	logFilePath, err := resolveLogFilePath(logDir, settings.logFilePath)
	if err != nil {
		return Config{}, err
	}
	logLevel, err := parseLogLevel(settings.logLevel)
	if err != nil {
		return Config{}, err
	}
	busyTimeoutMS, err := parsePositiveInt("busy_timeout_ms", settings.busyTimeoutMS, 5000)
	if err != nil {
		return Config{}, err
	}
	logMaxSizeMB, err := parsePositiveInt("log_max_size_mb", settings.logMaxSizeMB, 20)
	if err != nil {
		return Config{}, err
	}
	logMaxBackups, err := parsePositiveInt("log_max_backups", settings.logMaxBackups, 10)
	if err != nil {
		return Config{}, err
	}
	logMaxAgeDays, err := parsePositiveInt("log_max_age_days", settings.logMaxAgeDays, 30)
	if err != nil {
		return Config{}, err
	}
	logCompress, err := parseBool("log_compress", settings.logCompress, true)
	if err != nil {
		return Config{}, err
	}
	logAlsoStderr, err := parseBool("log_stderr", settings.logAlsoStderr, true)
	if err != nil {
		return Config{}, err
	}

	return Config{
		File: FileConfig{
			DatabasePath:      databasePath,
			DefaultSystemName: firstNonEmpty(settings.systemName, "codex-mem"),
			SQLiteDriver:      firstNonEmpty(settings.sqliteDriver, "sqlite"),
			BusyTimeout:       time.Duration(busyTimeoutMS) * time.Millisecond,
			JournalMode:       firstNonEmpty(settings.journalMode, "WAL"),
			LogLevel:          logLevel,
			LogFilePath:       logFilePath,
			LogMaxSizeMB:      logMaxSizeMB,
			LogMaxBackups:     logMaxBackups,
			LogMaxAgeDays:     logMaxAgeDays,
			LogCompress:       logCompress,
			LogAlsoStderr:     logAlsoStderr,
		},
		Meta: LoadMetadata{
			ConfigDir:      configDir,
			ConfigFilePath: configFilePath,
			ConfigFileUsed: loadedConfigPath,
			LogDir:         logDir,
		},
	}, nil
}

type loadedSettings struct {
	databasePath  string
	systemName    string
	sqliteDriver  string
	busyTimeoutMS string
	journalMode   string
	logLevel      string
	logFilePath   string
	logMaxSizeMB  string
	logMaxBackups string
	logMaxAgeDays string
	logCompress   string
	logAlsoStderr string
}

func loadSettings(absCWD string, configDir string, logDir string, configFilePath string, explicitConfigPath bool) (loadedSettings, string, error) {
	v := viper.New()
	v.SetEnvPrefix("CODEX_MEM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("db_path", filepath.Join(absCWD, "data", "codex-mem.db"))
	v.SetDefault("system_name", "codex-mem")
	v.SetDefault("sqlite_driver", "sqlite")
	v.SetDefault("busy_timeout_ms", "5000")
	v.SetDefault("journal_mode", "WAL")
	v.SetDefault("log_level", "info")
	v.SetDefault("log_file", filepath.Join(logDir, "codex-mem.log"))
	v.SetDefault("log_max_size_mb", "20")
	v.SetDefault("log_max_backups", "10")
	v.SetDefault("log_max_age_days", "30")
	v.SetDefault("log_compress", "true")
	v.SetDefault("log_stderr", "true")

	if explicitConfigPath {
		v.SetConfigFile(configFilePath)
	} else {
		v.SetConfigName(defaultConfigBaseName)
		v.AddConfigPath(configDir)
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if explicitConfigPath || !errors.As(err, &notFound) {
			return loadedSettings{}, "", fmt.Errorf("read config file: %w", err)
		}
	}

	usedConfigPath := ""
	if used := strings.TrimSpace(v.ConfigFileUsed()); used != "" {
		absUsed, err := filepath.Abs(used)
		if err != nil {
			return loadedSettings{}, "", fmt.Errorf("resolve used config file path: %w", err)
		}
		usedConfigPath = absUsed
	}

	return loadedSettings{
		databasePath:  strings.TrimSpace(v.GetString("db_path")),
		systemName:    strings.TrimSpace(v.GetString("system_name")),
		sqliteDriver:  strings.TrimSpace(v.GetString("sqlite_driver")),
		busyTimeoutMS: strings.TrimSpace(v.GetString("busy_timeout_ms")),
		journalMode:   strings.TrimSpace(v.GetString("journal_mode")),
		logLevel:      strings.TrimSpace(v.GetString("log_level")),
		logFilePath:   strings.TrimSpace(v.GetString("log_file")),
		logMaxSizeMB:  strings.TrimSpace(v.GetString("log_max_size_mb")),
		logMaxBackups: strings.TrimSpace(v.GetString("log_max_backups")),
		logMaxAgeDays: strings.TrimSpace(v.GetString("log_max_age_days")),
		logCompress:   strings.TrimSpace(v.GetString("log_compress")),
		logAlsoStderr: strings.TrimSpace(v.GetString("log_stderr")),
	}, usedConfigPath, nil
}

func resolveConfigFilePath(configDir string, envValue string) (string, bool, error) {
	if strings.TrimSpace(envValue) == "" {
		return filepath.Join(configDir, defaultConfigBaseName+defaultConfigExtension), false, nil
	}

	configFilePath := strings.TrimSpace(envValue)
	if !filepath.IsAbs(configFilePath) {
		configFilePath = filepath.Join(configDir, configFilePath)
	}
	absPath, err := filepath.Abs(configFilePath)
	if err != nil {
		return "", false, fmt.Errorf("resolve config file path: %w", err)
	}
	return absPath, true, nil
}

func resolveDataPath(baseDir string, value string) (string, error) {
	if value == "" {
		value = filepath.Join(baseDir, "data", "codex-mem.db")
	}
	if value == ":memory:" {
		return value, nil
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(baseDir, value)
	}
	absPath, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve database path: %w", err)
	}
	return absPath, nil
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
		return 0, fmt.Errorf("invalid log_level %q", value)
	}
}

func resolveLogFilePath(logDir, value string) (string, error) {
	logFilePath := strings.TrimSpace(value)
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
	value = strings.TrimSpace(value)
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
	switch strings.TrimSpace(value) {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
