package imports

import (
	"context"
	"testing"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
)

func TestConformanceC10ImportSuppression(t *testing.T) {
	baseScope := scope.Ref{
		SystemID:    "sys_1",
		ProjectID:   "proj_1",
		WorkspaceID: "ws_1",
	}

	t.Run("privacy-blocked artifact is suppressed with audit visibility", func(t *testing.T) {
		repo := &fakeRepository{}
		now := time.Date(2026, 3, 17, 2, 0, 0, 0, time.UTC)
		service := NewService(repo, Options{
			Clock:             fixedClock{now: now},
			IDFactory:         fixedIDFactory{value: "import_private_conformance"},
			NoteSaver:         &fakeNoteSaver{},
			ProjectNoteFinder: fakeProjectNoteFinder{},
		})

		result, err := service.SaveImportedNote(context.Background(), SaveImportedNoteInput{
			Scope:         baseScope,
			SessionID:     "sess_1",
			Source:        SourceWatcherImport,
			ExternalID:    "watcher:private-1",
			Type:          memory.NoteTypeDiscovery,
			Title:         "Private watcher artifact",
			Content:       "Should remain audit-only.",
			Importance:    3,
			PrivacyIntent: "private",
		})
		if err != nil {
			t.Fatalf("SaveImportedNote: %v", err)
		}

		if result.Materialized {
			t.Fatalf("expected privacy-blocked artifact to avoid note materialization, got %+v", result)
		}
		if !result.Suppressed {
			t.Fatal("expected privacy-blocked artifact to be suppressed")
		}
		if result.Note != nil {
			t.Fatalf("expected no durable note for suppressed privacy import, got %+v", result.Note)
		}
		if got, want := result.Import.ID, "import_private_conformance"; got != want {
			t.Fatalf("import audit id mismatch: got %q want %q", got, want)
		}
		if !result.Import.Suppressed {
			t.Fatalf("expected import audit record to be marked suppressed, got %+v", result.Import)
		}
		if got, want := result.Import.SuppressionReason, suppressionReasonPrivacyIntent; got != want {
			t.Fatalf("suppression reason mismatch: got %q want %q", got, want)
		}
		if got, want := len(result.Warnings), 1; got != want {
			t.Fatalf("warning count mismatch: got %d want %d", got, want)
		}
		if got, want := result.Warnings[0].Code, common.WarnImportSuppressed; got != want {
			t.Fatalf("warning code mismatch: got %q want %q", got, want)
		}
	})

	t.Run("explicit duplicate is suppressed while preserving stronger note", func(t *testing.T) {
		repo := &fakeRepository{}
		explicit := &memory.Note{
			ID:         "note_explicit_conformance",
			Scope:      baseScope,
			SessionID:  "sess_explicit",
			Type:       memory.NoteTypeDecision,
			Title:      "Use the explicit decision",
			Content:    "Explicit note should win over imported duplicates.",
			Importance: 5,
			Status:     memory.StatusActive,
			Source:     memory.SourceCodexExplicit,
			CreatedAt:  time.Date(2026, 3, 17, 2, 5, 0, 0, time.UTC),
			UpdatedAt:  time.Date(2026, 3, 17, 2, 5, 0, 0, time.UTC),
		}
		noteSaver := &fakeNoteSaver{}
		service := NewService(repo, Options{
			Clock:             fixedClock{now: time.Date(2026, 3, 17, 2, 6, 0, 0, time.UTC)},
			IDFactory:         fixedIDFactory{value: "import_explicit_conformance"},
			NoteSaver:         noteSaver,
			ProjectNoteFinder: fakeProjectNoteFinder{note: explicit},
		})

		result, err := service.SaveImportedNote(context.Background(), SaveImportedNoteInput{
			Scope:      baseScope,
			SessionID:  "sess_1",
			Source:     SourceRelayImport,
			ExternalID: "relay:explicit-1",
			Type:       explicit.Type,
			Title:      explicit.Title,
			Content:    explicit.Content,
			Importance: explicit.Importance,
		})
		if err != nil {
			t.Fatalf("SaveImportedNote: %v", err)
		}

		if result.Materialized {
			t.Fatalf("expected stronger explicit note to suppress imported materialization, got %+v", result)
		}
		if !result.Suppressed {
			t.Fatal("expected imported duplicate to be suppressed")
		}
		if !result.NoteDeduplicated {
			t.Fatal("expected suppression against explicit memory to report note deduplication")
		}
		if result.Note == nil || result.Note.ID != explicit.ID {
			t.Fatalf("expected stronger explicit note to be returned, got %+v", result.Note)
		}
		if noteSaver.calls != 0 {
			t.Fatalf("expected note saver to be skipped, got %d calls", noteSaver.calls)
		}
		if got, want := result.Import.DurableMemoryID, ""; got != want {
			t.Fatalf("suppressed import audit should not link a durable memory id: got %q want %q", got, want)
		}
		if !result.Import.Suppressed {
			t.Fatalf("expected import audit record to be marked suppressed, got %+v", result.Import)
		}
		if got, want := result.Import.SuppressionReason, "explicit_memory_exists"; got != want {
			t.Fatalf("suppression reason mismatch: got %q want %q", got, want)
		}
		if got, want := len(result.Warnings), 1; got != want {
			t.Fatalf("warning count mismatch: got %d want %d", got, want)
		}
		if got, want := result.Warnings[0].Code, common.WarnImportSuppressed; got != want {
			t.Fatalf("warning code mismatch: got %q want %q", got, want)
		}
	})
}
