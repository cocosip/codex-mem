package handoff

import (
	"strings"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
)

// Kind identifies the purpose of a persisted handoff.
type Kind string

// Supported handoff kinds.
const (
	KindFinal      Kind = "final"
	KindCheckpoint Kind = "checkpoint"
	KindRecovery   Kind = "recovery"
)

// Validate reports whether k is a supported handoff kind.
func (k Kind) Validate() error {
	switch k {
	case KindFinal, KindCheckpoint, KindRecovery:
		return nil
	default:
		return common.NewError(common.ErrInvalidInput, "invalid handoff kind")
	}
}

// Status identifies the lifecycle state of a handoff record.
type Status string

// Supported handoff statuses.
const (
	StatusOpen      Status = "open"
	StatusCompleted Status = "completed"
	StatusAbandoned Status = "abandoned"
)

// Validate reports whether s is a supported handoff status.
func (s Status) Validate() error {
	switch s {
	case StatusOpen, StatusCompleted, StatusAbandoned:
		return nil
	default:
		return common.NewError(common.ErrInvalidState, "invalid handoff status")
	}
}

// Handoff represents durable continuation context for a task.
type Handoff struct {
	ID              string    `json:"handoff_id"`
	Scope           scope.Ref `json:"scope"`
	SessionID       string    `json:"session_id"`
	Kind            Kind      `json:"kind"`
	Task            string    `json:"task"`
	Summary         string    `json:"summary"`
	Completed       []string  `json:"completed,omitempty"`
	NextSteps       []string  `json:"next_steps"`
	OpenQuestions   []string  `json:"open_questions,omitempty"`
	Risks           []string  `json:"risks,omitempty"`
	FilesTouched    []string  `json:"files_touched,omitempty"`
	RelatedNoteIDs  []string  `json:"related_note_ids,omitempty"`
	Status          Status    `json:"status"`
	Searchable      bool      `json:"searchable"`
	ExclusionReason string    `json:"exclusion_reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"-"`
}

// SaveInput captures the fields required to persist a handoff.
type SaveInput struct {
	Scope          scope.Ref
	SessionID      string
	Kind           Kind
	Task           string
	Summary        string
	Completed      []string
	NextSteps      []string
	OpenQuestions  []string
	Risks          []string
	FilesTouched   []string
	RelatedNoteIDs []string
	Status         Status
	PrivacyIntent  string
}

// SaveOutput returns the stored handoff and any non-fatal warnings.
type SaveOutput struct {
	Handoff              Handoff          `json:"handoff"`
	StoredAt             time.Time        `json:"stored_at"`
	EligibleForBootstrap bool             `json:"eligible_for_bootstrap"`
	Warnings             []common.Warning `json:"warnings"`
}

// Repository defines the storage operations required by the handoff service.
type Repository interface {
	FindLatestOpenByTask(scope scope.Ref, task string) (*Handoff, error)
	Create(handoff Handoff) error
}

// Validate checks that the handoff contains the required persisted fields.
func (h Handoff) Validate() error {
	if err := h.Scope.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(h.SessionID) == "" {
		return common.NewError(common.ErrInvalidInput, "session_id is required")
	}
	if err := h.Kind.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(h.Task) == "" {
		return common.NewError(common.ErrInvalidInput, "task is required")
	}
	if strings.TrimSpace(h.Summary) == "" {
		return common.NewError(common.ErrInvalidInput, "summary is required")
	}
	if len(h.NextSteps) == 0 {
		return common.NewError(common.ErrInvalidInput, "next_steps must contain at least one actionable item")
	}
	if err := h.Status.Validate(); err != nil {
		return err
	}
	if h.CreatedAt.IsZero() {
		return common.NewError(common.ErrInvalidInput, "created_at is required")
	}
	return nil
}
