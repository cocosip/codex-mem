package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	"codex-mem/internal/domain/common"
	projectmigrations "codex-mem/migrations"
)

type migrationFile struct {
	Name    string
	Version int
	Body    string
}

// RunMigrations applies embedded schema migrations that have not yet been recorded.
func RunMigrations(ctx context.Context, handle *sql.DB) error {
	if _, err := handle.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)
	`); err != nil {
		return common.WrapError(common.ErrStorageUnavailable, "ensure schema_migrations table", err)
	}

	files, err := loadMigrationFiles()
	if err != nil {
		return err
	}

	applied, err := appliedMigrations(ctx, handle)
	if err != nil {
		return err
	}

	for _, migration := range files {
		if applied[migration.Version] {
			continue
		}
		tx, err := handle.BeginTx(ctx, nil)
		if err != nil {
			return common.WrapError(common.ErrStorageUnavailable, "begin migration transaction", err)
		}
		if _, err := tx.ExecContext(ctx, migration.Body); err != nil {
			_ = tx.Rollback()
			return common.WrapError(common.ErrWriteFailed, "apply migration "+migration.Name, err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO schema_migrations (version, name, applied_at)
			VALUES (?, ?, ?)
		`, migration.Version, migration.Name, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			_ = tx.Rollback()
			return common.WrapError(common.ErrWriteFailed, "record migration "+migration.Name, err)
		}
		if err := tx.Commit(); err != nil {
			return common.WrapError(common.ErrWriteFailed, "commit migration "+migration.Name, err)
		}
	}

	return nil
}

func loadMigrationFiles() ([]migrationFile, error) {
	entries, err := fs.ReadDir(projectmigrations.FS, ".")
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "read embedded migrations", err)
	}

	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, parseErr := parseMigrationVersion(entry.Name())
		if parseErr != nil {
			return nil, parseErr
		}
		body, readErr := fs.ReadFile(projectmigrations.FS, entry.Name())
		if readErr != nil {
			return nil, common.WrapError(common.ErrReadFailed, "read embedded migration", readErr)
		}
		files = append(files, migrationFile{
			Name:    entry.Name(),
			Version: version,
			Body:    string(body),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Version < files[j].Version
	})
	return files, nil
}

func appliedMigrations(ctx context.Context, handle *sql.DB) (_ map[int]bool, err error) {
	rows, err := handle.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "query applied migrations", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = common.WrapError(common.ErrReadFailed, "close applied migrations rows", closeErr)
		}
	}()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, common.WrapError(common.ErrReadFailed, "scan applied migration", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, common.WrapError(common.ErrReadFailed, "iterate applied migrations", err)
	}
	return applied, nil
}

func parseMigrationVersion(name string) (int, error) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return 0, common.NewError(common.ErrInvalidInput, "invalid migration filename")
	}
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, common.WrapError(common.ErrInvalidInput, fmt.Sprintf("invalid migration version in %s", name), err)
	}
	return version, nil
}
