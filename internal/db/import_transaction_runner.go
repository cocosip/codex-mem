package db

import (
	"context"
	"database/sql"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/imports"
	"codex-mem/internal/domain/memory"
)

// ImportedNoteTransactionRunner executes imported-note materialization inside one SQL transaction.
type ImportedNoteTransactionRunner struct {
	db        *sql.DB
	clock     common.Clock
	idFactory common.IDFactory
}

// NewImportedNoteTransactionRunner constructs a transaction runner for imported-note workflows.
func NewImportedNoteTransactionRunner(db *sql.DB, clock common.Clock, idFactory common.IDFactory) *ImportedNoteTransactionRunner {
	return &ImportedNoteTransactionRunner{
		db:        db,
		clock:     clock,
		idFactory: idFactory,
	}
}

// RunSaveImportedNote runs the provided imported-note workflow inside one SQL transaction.
func (r *ImportedNoteTransactionRunner) RunSaveImportedNote(ctx context.Context, fn func(repo imports.Repository, noteSaver imports.NoteSaver, projectNoteFinder imports.ProjectNoteFinder) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return common.WrapError(common.ErrStorageUnavailable, "begin imported note transaction", err)
	}

	memoryRepo := NewMemoryRepositoryTx(tx)
	importRepo := NewImportRepositoryTx(tx)
	noteSaver := memory.NewService(memoryRepo, memory.Options{
		Clock:     r.clock,
		IDFactory: r.idFactory,
	})

	if err := fn(importRepo, noteSaver, memoryRepo); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return common.WrapError(common.ErrWriteFailed, "commit imported note transaction", err)
	}
	return nil
}
