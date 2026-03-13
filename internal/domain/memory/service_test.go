package memory

import (
	"context"
	"testing"
	"time"

	"codex-mem/internal/domain/scope"
)

type fakeRepository struct {
	duplicate *Note
	created   Note
}

func (f *fakeRepository) FindDuplicate(note Note) (*Note, error) {
	return f.duplicate, nil
}

func (f *fakeRepository) Create(note Note) error {
	f.created = note
	return nil
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

func (f fixedIDFactory) New(prefix string) string {
	return f.value
}

func TestSaveNoteNormalizesDefaultsAndFields(t *testing.T) {
	repo := &fakeRepository{}
	now := time.Date(2026, 3, 13, 13, 0, 0, 0, time.UTC)
	service := NewService(repo, Options{
		Clock:     fixedClock{now: now},
		IDFactory: fixedIDFactory{value: "note_fixed"},
	})

	result, err := service.SaveNote(context.Background(), SaveInput{
		Scope: scope.Ref{
			SystemID:    "sys_1",
			ProjectID:   "proj_1",
			WorkspaceID: "ws_1",
		},
		SessionID:  "sess_1",
		Type:       NoteTypeBugfix,
		Title:      "  Fix enum drift  ",
		Content:    "  Use generated backend metadata.  ",
		Importance: 4,
		Tags:       []string{"API", "api", "frontend"},
		FilePaths:  []string{`src\order\validation.ts`, "src/order/validation.ts"},
	})
	if err != nil {
		t.Fatalf("SaveNote: %v", err)
	}

	if got, want := result.Note.Status, StatusActive; got != want {
		t.Fatalf("status mismatch: got %q want %q", got, want)
	}
	if got, want := result.Note.Source, SourceCodexExplicit; got != want {
		t.Fatalf("source mismatch: got %q want %q", got, want)
	}
	if got, want := result.Note.ID, "note_fixed"; got != want {
		t.Fatalf("id mismatch: got %q want %q", got, want)
	}
	if got, want := len(result.Note.Tags), 2; got != want {
		t.Fatalf("tag length mismatch: got %d want %d", got, want)
	}
	if got, want := len(result.Note.FilePaths), 1; got != want {
		t.Fatalf("file path length mismatch: got %d want %d", got, want)
	}
	if got, want := repo.created.Title, "Fix enum drift"; got != want {
		t.Fatalf("title mismatch: got %q want %q", got, want)
	}
	if got, want := repo.created.Content, "Use generated backend metadata."; got != want {
		t.Fatalf("content mismatch: got %q want %q", got, want)
	}
}

func TestSaveNoteReturnsExistingDuplicate(t *testing.T) {
	now := time.Date(2026, 3, 13, 13, 10, 0, 0, time.UTC)
	existing := &Note{
		ID:         "note_existing",
		Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:  "sess_1",
		Type:       NoteTypeDecision,
		Title:      "Reuse generated enums",
		Content:    "Generated metadata is the source of truth.",
		Importance: 4,
		Status:     StatusActive,
		Source:     SourceCodexExplicit,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	repo := &fakeRepository{duplicate: existing}
	service := NewService(repo, Options{
		Clock:     fixedClock{now: now},
		IDFactory: fixedIDFactory{value: "note_new"},
	})

	result, err := service.SaveNote(context.Background(), SaveInput{
		Scope:      existing.Scope,
		SessionID:  existing.SessionID,
		Type:       existing.Type,
		Title:      existing.Title,
		Content:    existing.Content,
		Importance: existing.Importance,
	})
	if err != nil {
		t.Fatalf("SaveNote: %v", err)
	}

	if !result.Deduplicated {
		t.Fatal("expected deduplicated result")
	}
	if got, want := result.Note.ID, existing.ID; got != want {
		t.Fatalf("duplicate id mismatch: got %q want %q", got, want)
	}
	if repo.created.ID != "" {
		t.Fatalf("expected duplicate note to skip create, got %+v", repo.created)
	}
}

func TestSaveNoteRejectsPrivateIntent(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, Options{
		Clock:     fixedClock{now: time.Date(2026, 3, 13, 13, 20, 0, 0, time.UTC)},
		IDFactory: fixedIDFactory{value: "note_fixed"},
	})

	_, err := service.SaveNote(context.Background(), SaveInput{
		Scope:         scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:     "sess_1",
		Type:          NoteTypeBugfix,
		Title:         "Sensitive fix",
		Content:       "Contains private data.",
		Importance:    4,
		PrivacyIntent: "private",
	})
	if err == nil {
		t.Fatal("expected private note to be rejected")
	}
}
