package session

import (
	"context"
	"testing"
	"time"

	"codex-mem/internal/domain/scope"
)

type fakeRepository struct {
	created Session
}

func (f *fakeRepository) Create(session Session) error {
	f.created = session
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

func TestStartCreatesFreshActiveSession(t *testing.T) {
	repo := &fakeRepository{}
	startedAt := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)
	service := NewService(repo, Options{
		Clock:     fixedClock{now: startedAt},
		IDFactory: fixedIDFactory{value: "sess_fixed"},
	})

	result, err := service.Start(context.Background(), StartInput{
		Scope: scope.Scope{
			SystemID:      "sys_1",
			SystemName:    "codex-mem",
			ProjectID:     "proj_1",
			ProjectName:   "codex-mem",
			WorkspaceID:   "ws_1",
			WorkspaceRoot: "d:/code/go/codex-mem",
			BranchName:    "main",
			ResolvedBy:    "repo_remote",
		},
		Task: "Start Phase 1",
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if got, want := result.Session.ID, "sess_fixed"; got != want {
		t.Fatalf("session id mismatch: got %q want %q", got, want)
	}
	if got, want := result.Session.Status, StatusActive; got != want {
		t.Fatalf("session status mismatch: got %q want %q", got, want)
	}
	if got, want := result.Session.StartedAt, startedAt; !got.Equal(want) {
		t.Fatalf("started_at mismatch: got %s want %s", got, want)
	}
	if repo.created.BranchName != "main" {
		t.Fatalf("expected branch to fall back to scope branch, got %q", repo.created.BranchName)
	}
}
