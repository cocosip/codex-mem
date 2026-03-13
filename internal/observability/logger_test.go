package observability

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"codex-mem/internal/config"
)

func TestNewLoggerWritesToFile(t *testing.T) {
	logDir := t.TempDir()
	logFilePath := logDir + string(os.PathSeparator) + "codex-mem.log"
	logger, closer, err := NewLogger(config.Config{
		File: config.FileConfig{
			LogFilePath:   logFilePath,
			LogLevel:      slog.LevelInfo,
			LogMaxSizeMB:  1,
			LogMaxBackups: 2,
			LogMaxAgeDays: 1,
			LogCompress:   true,
			LogAlsoStderr: false,
		},
		Meta: config.LoadMetadata{
			LogDir: logDir,
		},
	})
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	defer func() {
		if err := closer.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	logger.Info("logger file write test", "component", "test")

	content, err := os.ReadFile(logFilePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(content), "logger file write test") {
		t.Fatalf("expected log file to contain log message, got %q", string(content))
	}
}
