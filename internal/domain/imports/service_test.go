package imports

import (
	"context"
	"testing"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
)

type fakeRepository struct {
	duplicate *Record
	created   Record
}

func (f *fakeRepository) FindDuplicate(_ Record) (*Record, error) {
	return f.duplicate, nil
}

func (f *fakeRepository) Create(record Record) error {
	f.created = record
	return nil
}

type fakeNoteSaver struct {
	output memory.SaveOutput
	err    error
	calls  int
	input  memory.SaveInput
}

func (f *fakeNoteSaver) SaveNote(_ context.Context, input memory.SaveInput) (memory.SaveOutput, error) {
	f.calls++
	f.input = input
	if f.err != nil {
		return memory.SaveOutput{}, f.err
	}
	return f.output, nil
}

type fakeProjectNoteFinder struct {
	note *memory.Note
	err  error
}

func (f fakeProjectNoteFinder) FindProjectDuplicate(_ scope.Ref, _ memory.NoteType, _, _ string) (*memory.Note, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.note, nil
}

type fixedClock struct {
	now time.Time
}

func (f fixedClock) Now() time.Time {
	return f.now
}

type fixedIDFactory struct {
	value string
}

func (f fixedIDFactory) New(_ string) string {
	return f.value
}

func TestSaveImportNormalizesAndStoresRecord(t *testing.T) {
	repo := &fakeRepository{}
	now := time.Date(2026, 3, 16, 3, 0, 0, 0, time.UTC)
	service := NewService(repo, Options{
		Clock:     fixedClock{now: now},
		IDFactory: fixedIDFactory{value: "import_fixed"},
	})

	result, err := service.SaveImport(context.Background(), SaveInput{
		Scope: scope.Ref{
			SystemID:    "sys_1",
			ProjectID:   "proj_1",
			WorkspaceID: "ws_1",
		},
		SessionID:       "sess_1",
		Source:          SourceWatcherImport,
		ExternalID:      " watcher:123 ",
		PayloadHash:     " ABCDEF ",
		DurableMemoryID: " note_1 ",
	})
	if err != nil {
		t.Fatalf("SaveImport: %v", err)
	}

	if got, want := result.Record.ID, "import_fixed"; got != want {
		t.Fatalf("id mismatch: got %q want %q", got, want)
	}
	if result.Suppressed {
		t.Fatal("expected non-suppressed import")
	}
	if got, want := repo.created.PayloadHash, "abcdef"; got != want {
		t.Fatalf("payload hash mismatch: got %q want %q", got, want)
	}
	if got, want := repo.created.DurableMemoryID, "note_1"; got != want {
		t.Fatalf("durable memory id mismatch: got %q want %q", got, want)
	}
}

func TestSaveImportReturnsExistingDuplicate(t *testing.T) {
	now := time.Date(2026, 3, 16, 3, 10, 0, 0, time.UTC)
	existing := &Record{
		ID:          "import_existing",
		Scope:       scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:   "sess_1",
		Source:      SourceRelayImport,
		ExternalID:  "relay:abc",
		PayloadHash: "hash123",
		ImportedAt:  now,
	}
	repo := &fakeRepository{duplicate: existing}
	service := NewService(repo, Options{
		Clock:     fixedClock{now: now.Add(time.Minute)},
		IDFactory: fixedIDFactory{value: "import_new"},
	})

	result, err := service.SaveImport(context.Background(), SaveInput{
		Scope:      existing.Scope,
		SessionID:  existing.SessionID,
		Source:     existing.Source,
		ExternalID: existing.ExternalID,
	})
	if err != nil {
		t.Fatalf("SaveImport: %v", err)
	}

	if !result.Deduplicated {
		t.Fatal("expected deduplicated duplicate import")
	}
	if !result.Suppressed {
		t.Fatal("expected duplicate import to be reported as suppressed")
	}
	if got, want := result.Record.ID, existing.ID; got != want {
		t.Fatalf("duplicate id mismatch: got %q want %q", got, want)
	}
	if repo.created.ID != "" {
		t.Fatalf("expected duplicate import to skip create, got %+v", repo.created)
	}
}

func TestSaveImportStoresPrivacySuppressionAuditRecord(t *testing.T) {
	repo := &fakeRepository{}
	now := time.Date(2026, 3, 16, 3, 20, 0, 0, time.UTC)
	service := NewService(repo, Options{
		Clock:     fixedClock{now: now},
		IDFactory: fixedIDFactory{value: "import_private"},
	})

	result, err := service.SaveImport(context.Background(), SaveInput{
		Scope:         scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:     "sess_1",
		Source:        SourceWatcherImport,
		ExternalID:    "watcher:private",
		PrivacyIntent: "private",
	})
	if err != nil {
		t.Fatalf("SaveImport: %v", err)
	}

	if !result.Suppressed {
		t.Fatal("expected privacy-blocked import to be suppressed")
	}
	if got, want := repo.created.SuppressionReason, "privacy_intent"; got != want {
		t.Fatalf("suppression reason mismatch: got %q want %q", got, want)
	}
	if repo.created.DurableMemoryID != "" {
		t.Fatalf("expected suppressed import to clear durable memory id, got %q", repo.created.DurableMemoryID)
	}
}

func TestSaveImportedNoteCreatesDurableNoteAndImportAudit(t *testing.T) {
	repo := &fakeRepository{}
	note := memory.Note{
		ID:         "note_imported",
		Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:  "sess_1",
		Type:       memory.NoteTypeDiscovery,
		Title:      "Imported discovery",
		Content:    "Watcher captured a reusable discovery.",
		Importance: 4,
		Status:     memory.StatusActive,
		Source:     memory.SourceWatcherImport,
		CreatedAt:  time.Date(2026, 3, 16, 3, 30, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 16, 3, 30, 0, 0, time.UTC),
	}
	noteSaver := &fakeNoteSaver{
		output: memory.SaveOutput{
			Note:     note,
			StoredAt: note.CreatedAt,
		},
	}
	service := NewService(repo, Options{
		Clock:             fixedClock{now: time.Date(2026, 3, 16, 3, 31, 0, 0, time.UTC)},
		IDFactory:         fixedIDFactory{value: "import_materialized"},
		NoteSaver:         noteSaver,
		ProjectNoteFinder: fakeProjectNoteFinder{},
	})

	result, err := service.SaveImportedNote(context.Background(), SaveImportedNoteInput{
		Scope:      note.Scope,
		SessionID:  note.SessionID,
		Source:     SourceWatcherImport,
		ExternalID: "watcher:note-1",
		Type:       note.Type,
		Title:      note.Title,
		Content:    note.Content,
		Importance: note.Importance,
	})
	if err != nil {
		t.Fatalf("SaveImportedNote: %v", err)
	}

	if !result.Materialized || result.Note == nil || result.Note.ID != note.ID {
		t.Fatalf("expected materialized note result, got %+v", result)
	}
	if noteSaver.calls != 1 {
		t.Fatalf("expected one note save call, got %d", noteSaver.calls)
	}
	if got, want := repo.created.DurableMemoryID, note.ID; got != want {
		t.Fatalf("expected import audit to link note %q, got %q", want, got)
	}
	if result.Suppressed {
		t.Fatal("expected non-suppressed imported note workflow")
	}
}

func TestSaveImportedNoteSuppressesWhenExplicitMemoryExists(t *testing.T) {
	repo := &fakeRepository{}
	explicit := &memory.Note{
		ID:         "note_explicit",
		Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:  "sess_existing",
		Type:       memory.NoteTypeDecision,
		Title:      "Keep explicit decision",
		Content:    "Explicit decision already exists.",
		Importance: 5,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  time.Date(2026, 3, 16, 3, 35, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 16, 3, 35, 0, 0, time.UTC),
	}
	noteSaver := &fakeNoteSaver{}
	service := NewService(repo, Options{
		Clock:             fixedClock{now: time.Date(2026, 3, 16, 3, 36, 0, 0, time.UTC)},
		IDFactory:         fixedIDFactory{value: "import_suppressed"},
		NoteSaver:         noteSaver,
		ProjectNoteFinder: fakeProjectNoteFinder{note: explicit},
	})

	result, err := service.SaveImportedNote(context.Background(), SaveImportedNoteInput{
		Scope:      explicit.Scope,
		SessionID:  "sess_1",
		Source:     SourceRelayImport,
		ExternalID: "relay:decision-1",
		Type:       explicit.Type,
		Title:      explicit.Title,
		Content:    explicit.Content,
		Importance: explicit.Importance,
	})
	if err != nil {
		t.Fatalf("SaveImportedNote: %v", err)
	}

	if result.Materialized {
		t.Fatalf("expected imported note materialization to be suppressed, got %+v", result)
	}
	if !result.Suppressed {
		t.Fatal("expected suppressed import audit")
	}
	if result.Note == nil || result.Note.ID != explicit.ID {
		t.Fatalf("expected explicit note to be returned, got %+v", result.Note)
	}
	if noteSaver.calls != 0 {
		t.Fatalf("expected note saver to be skipped, got %d calls", noteSaver.calls)
	}
	if got, want := repo.created.SuppressionReason, "explicit_memory_exists"; got != want {
		t.Fatalf("suppression reason mismatch: got %q want %q", got, want)
	}
	if got, want := result.Warnings[0].Code, common.WarnImportSuppressed; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
}

func TestSaveImportedNoteReusesExistingImportedNote(t *testing.T) {
	repo := &fakeRepository{}
	existing := &memory.Note{
		ID:         "note_existing_import",
		Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_2"},
		SessionID:  "sess_existing",
		Type:       memory.NoteTypeBugfix,
		Title:      "Imported bugfix",
		Content:    "Imported bugfix already stored.",
		Importance: 4,
		Status:     memory.StatusActive,
		Source:     memory.SourceWatcherImport,
		CreatedAt:  time.Date(2026, 3, 16, 3, 40, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 16, 3, 40, 0, 0, time.UTC),
	}
	noteSaver := &fakeNoteSaver{}
	service := NewService(repo, Options{
		Clock:             fixedClock{now: time.Date(2026, 3, 16, 3, 41, 0, 0, time.UTC)},
		IDFactory:         fixedIDFactory{value: "import_reuse"},
		NoteSaver:         noteSaver,
		ProjectNoteFinder: fakeProjectNoteFinder{note: existing},
	})

	result, err := service.SaveImportedNote(context.Background(), SaveImportedNoteInput{
		Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:  "sess_1",
		Source:     SourceWatcherImport,
		ExternalID: "watcher:bugfix-1",
		Type:       existing.Type,
		Title:      existing.Title,
		Content:    existing.Content,
		Importance: existing.Importance,
	})
	if err != nil {
		t.Fatalf("SaveImportedNote: %v", err)
	}

	if !result.Materialized || result.Note == nil || result.Note.ID != existing.ID {
		t.Fatalf("expected existing imported note to be reused, got %+v", result)
	}
	if !result.NoteDeduplicated {
		t.Fatal("expected note reuse to report deduplication")
	}
	if noteSaver.calls != 0 {
		t.Fatalf("expected note saver to be skipped, got %d calls", noteSaver.calls)
	}
	if got, want := repo.created.DurableMemoryID, existing.ID; got != want {
		t.Fatalf("durable memory link mismatch: got %q want %q", got, want)
	}
	if got, want := result.Warnings[0].Code, common.WarnDedupeApplied; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
}

func TestSaveImportedNotePrivacyIntentSkipsMaterialization(t *testing.T) {
	repo := &fakeRepository{}
	noteSaver := &fakeNoteSaver{}
	service := NewService(repo, Options{
		Clock:             fixedClock{now: time.Date(2026, 3, 16, 3, 50, 0, 0, time.UTC)},
		IDFactory:         fixedIDFactory{value: "import_private_note"},
		NoteSaver:         noteSaver,
		ProjectNoteFinder: fakeProjectNoteFinder{},
	})

	result, err := service.SaveImportedNote(context.Background(), SaveImportedNoteInput{
		Scope:         scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:     "sess_1",
		Source:        SourceWatcherImport,
		ExternalID:    "watcher:private-note",
		Type:          memory.NoteTypeDiscovery,
		Title:         "Private import",
		Content:       "Should stay audit-only.",
		Importance:    3,
		PrivacyIntent: "private",
	})
	if err != nil {
		t.Fatalf("SaveImportedNote: %v", err)
	}

	if result.Materialized || !result.Suppressed {
		t.Fatalf("expected privacy suppression without materialization, got %+v", result)
	}
	if noteSaver.calls != 0 {
		t.Fatalf("expected note saver to be skipped, got %d calls", noteSaver.calls)
	}
	if got, want := repo.created.SuppressionReason, "privacy_intent"; got != want {
		t.Fatalf("suppression reason mismatch: got %q want %q", got, want)
	}
}
