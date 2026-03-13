// Package db provides SQLite-backed storage for codex-mem.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // Register the pure-Go SQLite driver.

	"codex-mem/internal/domain/common"
)

// Options configures SQLite database opening and pragmas.
type Options struct {
	Path        string
	DriverName  string
	BusyTimeout time.Duration
	JournalMode string
}

// Open opens the SQLite database, applies pragmas, and runs migrations.
func Open(ctx context.Context, options Options) (*sql.DB, error) {
	driverName := options.DriverName
	if driverName == "" {
		driverName = "sqlite"
	}
	if options.Path == "" {
		return nil, common.NewError(common.ErrInvalidInput, "database path is required")
	}

	if options.Path != ":memory:" {
		dir := filepath.Dir(options.Path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, common.WrapError(common.ErrStorageUnavailable, "create database directory", err)
		}
	}

	handle, err := sql.Open(driverName, options.Path)
	if err != nil {
		return nil, common.WrapError(common.ErrStorageUnavailable, "open sqlite database", err)
	}

	if err := HealthCheck(ctx, handle); err != nil {
		_ = handle.Close()
		return nil, err
	}
	if err := applyPragmas(ctx, handle, options); err != nil {
		_ = handle.Close()
		return nil, err
	}
	if err := RunMigrations(ctx, handle); err != nil {
		_ = handle.Close()
		return nil, err
	}

	return handle, nil
}

// HealthCheck verifies the database connection is reachable and usable.
func HealthCheck(ctx context.Context, handle *sql.DB) error {
	if err := handle.PingContext(ctx); err != nil {
		return common.WrapError(common.ErrStorageUnavailable, "ping sqlite database", err)
	}
	if _, err := handle.ExecContext(ctx, "SELECT 1"); err != nil {
		return common.WrapError(common.ErrStorageUnavailable, "run sqlite health query", err)
	}
	return nil
}

func applyPragmas(ctx context.Context, handle *sql.DB, options Options) error {
	timeoutMS := options.BusyTimeout.Milliseconds()
	if timeoutMS <= 0 {
		timeoutMS = 5000
	}
	journalMode := options.JournalMode
	if journalMode == "" {
		journalMode = "WAL"
	}

	statements := []string{
		"PRAGMA foreign_keys = ON",
		fmt.Sprintf("PRAGMA busy_timeout = %d", timeoutMS),
		fmt.Sprintf("PRAGMA journal_mode = %s", journalMode),
		"PRAGMA synchronous = NORMAL",
	}
	for _, statement := range statements {
		if _, err := handle.ExecContext(ctx, statement); err != nil {
			return common.WrapError(common.ErrStorageUnavailable, "apply sqlite pragma", err)
		}
	}
	return nil
}
