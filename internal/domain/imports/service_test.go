package imports

import (
	"context"
	"testing"
	"time"

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
