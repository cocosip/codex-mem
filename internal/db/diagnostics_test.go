package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
)

func TestInspectRuntimeReportsMigrationAndSchemaReadiness(t *testing.T) {
	ctx := context.Background()
	handle, err := Open(ctx, Options{
		Path:        filepath.Join(t.TempDir(), "codex-mem.db"),
		DriverName:  "sqlite",
		BusyTimeout: 2 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer handle.Close()

	diagnostics, err := InspectRuntime(ctx, handle)
	if err != nil {
		t.Fatalf("InspectRuntime: %v", err)
	}

	if !diagnostics.ForeignKeysEnabled {
		t.Fatal("expected foreign keys to be enabled")
	}
	if got, want := diagnostics.JournalMode, "wal"; got != want {
		t.Fatalf("journal mode mismatch: got %q want %q", got, want)
	}
	if diagnostics.BusyTimeout <= 0 {
		t.Fatalf("expected positive busy timeout, got %s", diagnostics.BusyTimeout)
	}
	if !diagnostics.RequiredSchemaOK {
		t.Fatal("expected required schema to be present")
	}
	if !diagnostics.FTSReady {
		t.Fatal("expected FTS table to be present")
	}
	if diagnostics.Migrations.Available == 0 {
		t.Fatal("expected embedded migrations to be discovered")
	}
	if got, want := diagnostics.Migrations.Applied, diagnostics.Migrations.Available; got != want {
		t.Fatalf("applied migration count mismatch: got %d want %d", got, want)
	}
	if diagnostics.Migrations.Pending != 0 {
		t.Fatalf("expected no pending migrations, got %d", diagnostics.Migrations.Pending)
	}
	if got, want := diagnostics.Migrations.LatestApplied, "004_searchability_controls.sql"; got != want {
		t.Fatalf("latest applied migration mismatch: got %q want %q", got, want)
	}
	if !diagnostics.Audit.NoteProvenanceReady {
		t.Fatal("expected empty store to be provenance-ready")
	}
	if !diagnostics.Audit.ExclusionAuditReady {
		t.Fatal("expected empty store to be exclusion-audit ready")
	}
}

func TestInspectRuntimeReportsAuditCounts(t *testing.T) {
	ctx := context.Background()
	handle, err := Open(ctx, Options{
		Path:        filepath.Join(t.TempDir(), "codex-mem.db"),
		DriverName:  "sqlite",
		BusyTimeout: 2 * time.Second,
		JournalMode: "WAL",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer handle.Close()

	ref, sessionID := seedScopeAndSession(t, handle)
	memoryRepo := NewMemoryRepository(handle)
	handoffRepo := NewHandoffRepository(handle)
	now := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)

	for _, note := range []struct {
		id     string
		source string
		search bool
		reason string
	}{
		{id: "note_explicit", source: "codex_explicit", search: true},
		{id: "note_import", source: "watcher_import", search: true},
		{id: "note_hidden", source: "recovery_generated", search: false, reason: "archived"},
	} {
		if err := memoryRepo.Create(memory.Note{
			ID:              note.id,
			Scope:           ref,
			SessionID:       sessionID,
			Type:            memory.NoteTypeDiscovery,
			Title:           note.id,
			Content:         "audit",
			Importance:      3,
			Status:          memory.StatusActive,
			Source:          memory.Source(note.source),
			Searchable:      note.search,
			ExclusionReason: note.reason,
			CreatedAt:       now,
			UpdatedAt:       now,
		}); err != nil {
			t.Fatalf("Create note %s: %v", note.id, err)
		}
		now = now.Add(time.Minute)
	}

	if err := handoffRepo.Create(handoff.Handoff{
		ID:              "handoff_hidden",
		Scope:           ref,
		SessionID:       sessionID,
		Kind:            handoff.KindRecovery,
		Task:            "recovery task",
		Summary:         "recovery summary",
		NextSteps:       []string{"resume"},
		Status:          handoff.StatusOpen,
		Searchable:      false,
		ExclusionReason: "private",
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("Create hidden handoff: %v", err)
	}
	now = now.Add(time.Minute)
	if err := handoffRepo.Create(handoff.Handoff{
		ID:        "handoff_open",
		Scope:     ref,
		SessionID: sessionID,
		Kind:      handoff.KindCheckpoint,
		Task:      "normal task",
		Summary:   "checkpoint",
		NextSteps: []string{"continue"},
		Status:    handoff.StatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("Create open handoff: %v", err)
	}

	diagnostics, err := InspectRuntime(ctx, handle)
	if err != nil {
		t.Fatalf("InspectRuntime: %v", err)
	}

	if got, want := diagnostics.Audit.NoteRecords, 3; got != want {
		t.Fatalf("note record count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.HandoffRecords, 2; got != want {
		t.Fatalf("handoff record count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.NotesCodexExplicit, 1; got != want {
		t.Fatalf("explicit note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.NotesWatcherImport, 1; got != want {
		t.Fatalf("watcher import note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.NotesRecoveryGenerated, 1; got != want {
		t.Fatalf("recovery note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.ExcludedNotes, 1; got != want {
		t.Fatalf("excluded note count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.ExcludedHandoffs, 1; got != want {
		t.Fatalf("excluded handoff count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.RecoveryHandoffs, 1; got != want {
		t.Fatalf("recovery handoff count mismatch: got %d want %d", got, want)
	}
	if got, want := diagnostics.Audit.OpenHandoffs, 2; got != want {
		t.Fatalf("open handoff count mismatch: got %d want %d", got, want)
	}
	if diagnostics.Audit.NotesInvalidSource != 0 {
		t.Fatalf("expected no invalid note sources, got %d", diagnostics.Audit.NotesInvalidSource)
	}
	if !diagnostics.Audit.NoteProvenanceReady {
		t.Fatal("expected note provenance to be ready")
	}
	if !diagnostics.Audit.ExclusionAuditReady {
		t.Fatal("expected exclusion audit to be ready")
	}
}
