package config

import (
	"path/filepath"
	"testing"
)

func TestLoadDefaultsIncludeLogFileRotationSettings(t *testing.T) {
	t.Setenv("CODEX_MEM_DB_PATH", "")
	t.Setenv("CODEX_MEM_CONFIG_FILE", "")
	t.Setenv("CODEX_MEM_LOG_FILE", "")
	t.Setenv("CODEX_MEM_LOG_MAX_SIZE_MB", "")
	t.Setenv("CODEX_MEM_LOG_MAX_BACKUPS", "")
	t.Setenv("CODEX_MEM_LOG_MAX_AGE_DAYS", "")
	t.Setenv("CODEX_MEM_LOG_COMPRESS", "")
	t.Setenv("CODEX_MEM_LOG_STDERR", "")

	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got, want := cfg.LogFilePath, filepath.Join(cfg.LogDir, "codex-mem.log"); got != want {
		t.Fatalf("log file mismatch: got %q want %q", got, want)
	}
	if cfg.LogMaxSizeMB != 20 {
		t.Fatalf("unexpected LogMaxSizeMB: %d", cfg.LogMaxSizeMB)
	}
	if cfg.LogMaxBackups != 10 {
		t.Fatalf("unexpected LogMaxBackups: %d", cfg.LogMaxBackups)
	}
	if cfg.LogMaxAgeDays != 30 {
		t.Fatalf("unexpected LogMaxAgeDays: %d", cfg.LogMaxAgeDays)
	}
	if !cfg.LogCompress {
		t.Fatal("expected log compression to default to true")
	}
	if !cfg.LogAlsoStderr {
		t.Fatal("expected stderr logging to default to true")
	}
}
