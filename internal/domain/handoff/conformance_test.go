package handoff

import (
	"context"
	"testing"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
)

func TestConformanceC08SaveHandoffValidity(t *testing.T) {
	service := NewService(&fakeRepository{}, Options{
		Clock:     fixedClock{now: time.Date(2026, 3, 17, 2, 20, 0, 0, time.UTC)},
		IDFactory: fixedIDFactory{value: "handoff_conformance"},
	})

	_, err := service.SaveHandoff(context.Background(), SaveInput{
		Scope: scope.Ref{
			SystemID:    "sys_1",
			ProjectID:   "proj_1",
			WorkspaceID: "ws_1",
		},
		SessionID: "sess_1",
		Kind:      KindFinal,
		Task:      "Continue import hardening",
		Summary:   "Import suppression behavior was reviewed.",
		Status:    StatusOpen,
	})
	if err == nil {
		t.Fatal("expected handoff without next_steps to be rejected")
	}
	if got, want := common.ErrorCode(err), common.ErrInvalidInput; got != want {
		t.Fatalf("error code mismatch: got %q want %q", got, want)
	}
}
