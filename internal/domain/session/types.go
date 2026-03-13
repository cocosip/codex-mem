package session

import (
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusEnded     Status = "ended"
	StatusRecovered Status = "recovered"
)

func (s Status) Validate() error {
	switch s {
	case StatusActive, StatusPaused, StatusEnded, StatusRecovered:
		return nil
	default:
		return common.NewError(common.ErrInvalidState, "invalid session status")
	}
}

type Session struct {
	ID         string     `json:"session_id"`
	Scope      scope.Ref  `json:"scope"`
	Status     Status     `json:"status"`
	Task       string     `json:"task,omitempty"`
	BranchName string     `json:"branch_name,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}

type StartInput struct {
	Scope      scope.Scope
	Task       string
	BranchName string
}

type StartOutput struct {
	Session  Session          `json:"session"`
	Warnings []common.Warning `json:"warnings"`
}

type Repository interface {
	Create(session Session) error
}
