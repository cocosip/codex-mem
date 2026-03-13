package observability

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"codex-mem/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

func NewBootstrapLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

func NewLogger(cfg config.Config) (*slog.Logger, io.Closer, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.LogFilePath), 0o755); err != nil {
		return nil, nil, err
	}

	rotator := &lumberjack.Logger{
		Filename:   cfg.LogFilePath,
		MaxSize:    cfg.LogMaxSizeMB,
		MaxBackups: cfg.LogMaxBackups,
		MaxAge:     cfg.LogMaxAgeDays,
		Compress:   cfg.LogCompress,
		LocalTime:  true,
	}
	writers := []io.Writer{
		rotator,
	}
	if cfg.LogAlsoStderr {
		writers = append(writers, os.Stderr)
	}

	handler := slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})
	return slog.New(handler), rotator, nil
}
