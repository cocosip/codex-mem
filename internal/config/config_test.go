package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaultsIncludeLogFileRotationSettings(t *testing.T) {
	clearConfigEnv(t)

	root := t.TempDir()
	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := cfg.Meta.ConfigFilePath, filepath.Join(root, "configs", "codex-mem.json"); got != want {
		t.Fatalf("config file path mismatch: got %q want %q", got, want)
	}
	if cfg.Meta.ConfigFileUsed != "" {
		t.Fatalf("expected no config file to be loaded, got %q", cfg.Meta.ConfigFileUsed)
	}
	if got, want := cfg.File.LogFilePath, filepath.Join(cfg.Meta.LogDir, "codex-mem.log"); got != want {
		t.Fatalf("log file mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.File.DatabasePath, filepath.Join(root, "data", "codex-mem.db"); got != want {
		t.Fatalf("database path mismatch: got %q want %q", got, want)
	}
	if cfg.File.BusyTimeout != 5*time.Second {
		t.Fatalf("unexpected BusyTimeout: %s", cfg.File.BusyTimeout)
	}
	if cfg.File.LogMaxSizeMB != 20 {
		t.Fatalf("unexpected LogMaxSizeMB: %d", cfg.File.LogMaxSizeMB)
	}
	if cfg.File.LogMaxBackups != 10 {
		t.Fatalf("unexpected LogMaxBackups: %d", cfg.File.LogMaxBackups)
	}
	if cfg.File.LogMaxAgeDays != 30 {
		t.Fatalf("unexpected LogMaxAgeDays: %d", cfg.File.LogMaxAgeDays)
	}
	if !cfg.File.LogCompress {
		t.Fatal("expected log compression to default to true")
	}
	if !cfg.File.LogAlsoStderr {
		t.Fatal("expected stderr logging to default to true")
	}
}

func TestLoadReadsConfigFromConfigsDirectory(t *testing.T) {
	clearConfigEnv(t)

	root := t.TempDir()
	configDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	configPath := filepath.Join(configDir, "codex-mem.json")
	body := `{
  "db_path": "store/app.db",
  "system_name": "order-platform",
  "sqlite_driver": "sqlite",
  "busy_timeout_ms": 9000,
  "journal_mode": "DELETE",
  "log_level": "debug",
  "log_file": "custom.log",
  "log_max_size_mb": 42,
  "log_max_backups": 7,
  "log_max_age_days": 14,
  "log_compress": false,
  "log_stderr": false
}`
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := cfg.Meta.ConfigFilePath, configPath; got != want {
		t.Fatalf("config file path mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.Meta.ConfigFileUsed, configPath; got != want {
		t.Fatalf("config file used mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.File.DatabasePath, filepath.Join(root, "store", "app.db"); got != want {
		t.Fatalf("database path mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.File.DefaultSystemName, "order-platform"; got != want {
		t.Fatalf("system name mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.File.BusyTimeout, 9*time.Second; got != want {
		t.Fatalf("busy timeout mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.File.JournalMode, "DELETE"; got != want {
		t.Fatalf("journal mode mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.File.LogFilePath, filepath.Join(root, "logs", "custom.log"); got != want {
		t.Fatalf("log file mismatch: got %q want %q", got, want)
	}
	if cfg.File.LogCompress {
		t.Fatal("expected log compression to load as false")
	}
	if cfg.File.LogAlsoStderr {
		t.Fatal("expected stderr logging to load as false")
	}
}

func TestLoadEnvironmentOverridesConfigFile(t *testing.T) {
	clearConfigEnv(t)

	root := t.TempDir()
	configDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	configPath := filepath.Join(configDir, "codex-mem.json")
	body := `{
  "db_path": "store/app.db",
  "system_name": "order-platform",
  "busy_timeout_ms": 2000,
  "journal_mode": "DELETE",
  "log_level": "warn",
  "log_file": "from-file.log",
  "log_compress": false
}`
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Setenv("CODEX_MEM_DB_PATH", "override/override.db")
	t.Setenv("CODEX_MEM_SYSTEM_NAME", "override-system")
	t.Setenv("CODEX_MEM_BUSY_TIMEOUT_MS", "12000")
	t.Setenv("CODEX_MEM_JOURNAL_MODE", "WAL")
	t.Setenv("CODEX_MEM_LOG_LEVEL", "error")
	t.Setenv("CODEX_MEM_LOG_FILE", "override.log")
	t.Setenv("CODEX_MEM_LOG_COMPRESS", "true")
	t.Setenv("CODEX_MEM_LOG_STDERR", "true")

	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := cfg.File.DatabasePath, filepath.Join(root, "override", "override.db"); got != want {
		t.Fatalf("database path mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.File.DefaultSystemName, "override-system"; got != want {
		t.Fatalf("system name mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.File.BusyTimeout, 12*time.Second; got != want {
		t.Fatalf("busy timeout mismatch: got %s want %s", got, want)
	}
	if got, want := cfg.File.JournalMode, "WAL"; got != want {
		t.Fatalf("journal mode mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.File.LogFilePath, filepath.Join(root, "logs", "override.log"); got != want {
		t.Fatalf("log file mismatch: got %q want %q", got, want)
	}
	if !cfg.File.LogCompress {
		t.Fatal("expected env to override log compression to true")
	}
	if !cfg.File.LogAlsoStderr {
		t.Fatal("expected env to override stderr logging to true")
	}
}

func TestLoadUsesExplicitConfigFileFromEnvironment(t *testing.T) {
	clearConfigEnv(t)

	root := t.TempDir()
	configDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	explicitPath := filepath.Join(configDir, "custom.toml")
	body := `system_name = "custom-system"
busy_timeout_ms = 3000`
	if err := os.WriteFile(explicitPath, []byte(body), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("CODEX_MEM_CONFIG_FILE", "custom.toml")

	cfg, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.Meta.ConfigFilePath, explicitPath; got != want {
		t.Fatalf("config file path mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.Meta.ConfigFileUsed, explicitPath; got != want {
		t.Fatalf("config file used mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.File.DefaultSystemName, "custom-system"; got != want {
		t.Fatalf("system name mismatch: got %q want %q", got, want)
	}
	if got, want := cfg.File.BusyTimeout, 3*time.Second; got != want {
		t.Fatalf("busy timeout mismatch: got %s want %s", got, want)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"CODEX_MEM_CONFIG_FILE",
		"CODEX_MEM_DB_PATH",
		"CODEX_MEM_SYSTEM_NAME",
		"CODEX_MEM_SQLITE_DRIVER",
		"CODEX_MEM_BUSY_TIMEOUT_MS",
		"CODEX_MEM_JOURNAL_MODE",
		"CODEX_MEM_LOG_LEVEL",
		"CODEX_MEM_LOG_FILE",
		"CODEX_MEM_LOG_MAX_SIZE_MB",
		"CODEX_MEM_LOG_MAX_BACKUPS",
		"CODEX_MEM_LOG_MAX_AGE_DAYS",
		"CODEX_MEM_LOG_COMPRESS",
		"CODEX_MEM_LOG_STDERR",
	} {
		t.Setenv(key, "")
	}
}
