// Package observability configures bootstrap and rotating runtime loggers.
package observability

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"codex-mem/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewBootstrapLogger creates the stderr logger used before full config has loaded.
func NewBootstrapLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

// NewLogger creates the configured rotating runtime logger and returns its closer.
func NewLogger(cfg config.Config) (*slog.Logger, io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.File.LogFilePath), 0o755); err != nil {
		return nil, nil, err
	}

	rotator := &lumberjack.Logger{
		Filename:   cfg.File.LogFilePath,
		MaxSize:    cfg.File.LogMaxSizeMB,
		MaxBackups: cfg.File.LogMaxBackups,
		MaxAge:     cfg.File.LogMaxAgeDays,
		Compress:   cfg.File.LogCompress,
		LocalTime:  true,
	}
	writers := []io.Writer{
		rotator,
	}
	if cfg.File.LogAlsoStderr {
		writers = append(writers, os.Stderr)
	}

	handler := slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
		Level: cfg.File.LogLevel,
	})
	return slog.New(handler), rotator, nil
}
