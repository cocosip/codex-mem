package handoff

import (
	"context"
	"testing"
	"time"

	"codex-mem/internal/domain/scope"
)

type fakeRepository struct {
	existing *Handoff
	created  Handoff
}

func (f *fakeRepository) FindLatestOpenByTask(_ scope.Ref, _ string) (*Handoff, error) {
	return f.existing, nil
}

func (f *fakeRepository) Create(record Handoff) error {
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

func TestSaveHandoffWarnsOnExistingTaskAndSparsePayload(t *testing.T) {
	now := time.Date(2026, 3, 13, 13, 30, 0, 0, time.UTC)
	repo := &fakeRepository{
		existing: &Handoff{
			ID:        "handoff_old",
			Scope:     scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
			SessionID: "sess_old",
			Kind:      KindCheckpoint,
			Task:      "Continue validation work",
			Summary:   "Earlier checkpoint.",
			NextSteps: []string{"Retest API responses"},
			Status:    StatusOpen,
			CreatedAt: now.Add(-time.Hour),
			UpdatedAt: now.Add(-time.Hour),
		},
	}
	service := NewService(repo, Options{
		Clock:     fixedClock{now: now},
		IDFactory: fixedIDFactory{value: "handoff_new"},
	})

	result, err := service.SaveHandoff(context.Background(), SaveInput{
		Scope: scope.Ref{
			SystemID:    "sys_1",
			ProjectID:   "proj_1",
			WorkspaceID: "ws_1",
		},
		SessionID: "sess_1",
		Kind:      KindFinal,
		Task:      "Continue validation work",
		Summary:   "Validation logic is updated.",
		NextSteps: []string{"Run checkout regression"},
		Status:    StatusOpen,
	})
	if err != nil {
		t.Fatalf("SaveHandoff: %v", err)
	}

	if !result.EligibleForBootstrap {
		t.Fatal("expected open handoff to be bootstrap eligible")
	}
	if got, want := len(result.Warnings), 2; got != want {
		t.Fatalf("warning count mismatch: got %d want %d", got, want)
	}
	if got, want := repo.created.ID, "handoff_new"; got != want {
		t.Fatalf("id mismatch: got %q want %q", got, want)
	}
}

func TestSaveHandoffRejectsMissingNextSteps(t *testing.T) {
	service := NewService(&fakeRepository{}, Options{
		Clock:     fixedClock{now: time.Date(2026, 3, 13, 13, 40, 0, 0, time.UTC)},
		IDFactory: fixedIDFactory{value: "handoff_new"},
	})

	_, err := service.SaveHandoff(context.Background(), SaveInput{
		Scope: scope.Ref{
			SystemID:    "sys_1",
			ProjectID:   "proj_1",
			WorkspaceID: "ws_1",
		},
		SessionID: "sess_1",
		Kind:      KindFinal,
		Task:      "Continue validation work",
		Summary:   "Validation logic is updated.",
		Status:    StatusOpen,
	})
	if err == nil {
		t.Fatal("expected missing next_steps to fail")
	}
}

func TestSaveHandoffRejectsPrivateIntent(t *testing.T) {
	service := NewService(&fakeRepository{}, Options{
		Clock:     fixedClock{now: time.Date(2026, 3, 13, 13, 50, 0, 0, time.UTC)},
		IDFactory: fixedIDFactory{value: "handoff_new"},
	})

	_, err := service.SaveHandoff(context.Background(), SaveInput{
		Scope:         scope.Ref{SystemID: "sys_1", ProjectID: "proj_1", WorkspaceID: "ws_1"},
		SessionID:     "sess_1",
		Kind:          KindFinal,
		Task:          "Sensitive task",
		Summary:       "Contains sensitive content.",
		NextSteps:     []string{"Do not store"},
		Status:        StatusOpen,
		PrivacyIntent: "do_not_store",
	})
	if err == nil {
		t.Fatal("expected private handoff to be rejected")
	}
}


