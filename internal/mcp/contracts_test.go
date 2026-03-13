package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"codex-mem/internal/domain/agents"
	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/scope"
)

type memoryRepoStub struct {
	duplicate *memory.Note
	findErr   error
}

func (s *memoryRepoStub) FindDuplicate(note memory.Note) (*memory.Note, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	return s.duplicate, nil
}

func (s *memoryRepoStub) Create(note memory.Note) error {
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

func TestHandleMemorySaveNotePromotesWarningsToEnvelope(t *testing.T) {
	now := time.Date(2026, 3, 13, 16, 0, 0, 0, time.UTC)
	existing := &memory.Note{
		ID:         "note_existing",
		Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:  "sess_1",
		Type:       memory.NoteTypeDecision,
		Title:      "Reuse generated enums",
		Content:    "Generated metadata is the source of truth.",
		Importance: 4,
		Status:     memory.StatusActive,
		Source:     memory.SourceCodexExplicit,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	service := memory.NewService(&memoryRepoStub{duplicate: existing}, memory.Options{
		Clock:     fixedClock{now: now},
		IDFactory: fixedIDFactory{value: "note_new"},
	})
	handlers := &Handlers{memoryService: service}

	response := handlers.HandleMemorySaveNote(context.Background(), memory.SaveInput{
		Scope:      existing.Scope,
		SessionID:  existing.SessionID,
		Type:       existing.Type,
		Title:      existing.Title,
		Content:    existing.Content,
		Importance: existing.Importance,
	})

	if !response.Ok {
		t.Fatalf("expected ok response, got error %+v", response.Error)
	}
	if response.Data == nil {
		t.Fatal("expected response data")
	}
	if !response.Data.Deduplicated {
		t.Fatal("expected deduplicated envelope data")
	}
	if got, want := len(response.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := response.Warnings[0].Code, common.WarnDedupeApplied; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
}

func TestHandleMemorySaveNoteMapsUncodedErrors(t *testing.T) {
	service := memory.NewService(&memoryRepoStub{findErr: errors.New("boom")}, memory.Options{
		Clock:     fixedClock{now: time.Date(2026, 3, 13, 16, 5, 0, 0, time.UTC)},
		IDFactory: fixedIDFactory{value: "note_new"},
	})
	handlers := &Handlers{memoryService: service}

	response := handlers.HandleMemorySaveNote(context.Background(), memory.SaveInput{
		Scope:      scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:  "sess_1",
		Type:       memory.NoteTypeBugfix,
		Title:      "Fix drift",
		Content:    "Normalize generated metadata.",
		Importance: 4,
	})

	if response.Ok {
		t.Fatal("expected error response")
	}
	if response.Error == nil {
		t.Fatal("expected error payload")
	}
	if got, want := response.Error.Code, common.ErrReadFailed; got != want {
		t.Fatalf("error code mismatch: got %q want %q", got, want)
	}
	if response.Data != nil {
		t.Fatalf("expected no data on error, got %+v", response.Data)
	}
}

func TestHandleMemoryInstallAgentsPromotesWarningsToEnvelope(t *testing.T) {
	root := t.TempDir()
	service := agents.NewService(agents.Options{HomeDir: root})
	handlers := &Handlers{agentsService: service}

	response := handlers.HandleMemoryInstallAgents(context.Background(), agents.InstallInput{
		Target: agents.TargetProject,
		Mode:   agents.ModeSafe,
		CWD:    root,
	})

	if !response.Ok {
		t.Fatalf("expected ok response, got error %+v", response.Error)
	}
	if response.Data == nil {
		t.Fatal("expected response data")
	}
	if got, want := len(response.Data.WrittenFiles), 1; got != want {
		t.Fatalf("written file count mismatch: got %d want %d", got, want)
	}
	if got, want := len(response.Warnings), 1; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := response.Warnings[0].Code, common.WarnPlaceholdersUnresolved; got != want {
		t.Fatalf("warning code mismatch: got %q want %q", got, want)
	}
}
