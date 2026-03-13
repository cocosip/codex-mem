package session

import (
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
)

// Status identifies the lifecycle state of a session.
type Status string

// Supported session statuses.
const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusEnded     Status = "ended"
	StatusRecovered Status = "recovered"
)

// Validate reports whether s is a supported session status.
func (s Status) Validate() error {
	switch s {
	case StatusActive, StatusPaused, StatusEnded, StatusRecovered:
		return nil
	default:
		return common.NewError(common.ErrInvalidState, "invalid session status")
	}
}

// Session represents persisted work-session context.
type Session struct {
	ID         string     `json:"session_id"`
	Scope      scope.Ref  `json:"scope"`
	Status     Status     `json:"status"`
	Task       string     `json:"task,omitempty"`
	BranchName string     `json:"branch_name,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}

// StartInput captures the fields needed to create a new session.
type StartInput struct {
	Scope      scope.Scope
	Task       string
	BranchName string
}

// StartOutput returns the created session and any warnings.
type StartOutput struct {
	Session  Session          `json:"session"`
	Warnings []common.Warning `json:"warnings"`
}

// Repository defines the storage operation required to create sessions.
type Repository interface {
	Create(session Session) error
}
