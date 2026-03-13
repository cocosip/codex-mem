// Package session tracks active and historical Codex work sessions.
package session

import (
	"context"
	"strings"

	"codex-mem/internal/domain/common"
)

// Options configures session creation dependencies.
type Options struct {
	Clock     common.Clock
	IDFactory common.IDFactory
}

// Service creates session records.
type Service struct {
	repo    Repository
	options Options
}

// NewService constructs a Service with default clock and ID generation behavior.
func NewService(repo Repository, options Options) *Service {
	if options.Clock == nil {
		options.Clock = common.RealClock{}
	}
	if options.IDFactory == nil {
		options.IDFactory = common.DefaultIDFactory{Clock: options.Clock}
	}
	return &Service{repo: repo, options: options}
}

// Start creates a new active session from the resolved scope input.
func (s *Service) Start(ctx context.Context, input StartInput) (StartOutput, error) {
	_ = ctx

	if err := input.Scope.Validate(); err != nil {
		return StartOutput{}, err
	}

	branchName := strings.TrimSpace(input.BranchName)
	if branchName == "" {
		branchName = input.Scope.BranchName
	}

	record := Session{
		ID:         s.options.IDFactory.New("sess"),
		Scope:      input.Scope.Ref(),
		Status:     StatusActive,
		Task:       strings.TrimSpace(input.Task),
		BranchName: branchName,
		StartedAt:  s.options.Clock.Now().UTC(),
	}
	if err := record.Status.Validate(); err != nil {
		return StartOutput{}, err
	}
	if err := record.Scope.Validate(); err != nil {
		return StartOutput{}, err
	}

	if err := s.repo.Create(record); err != nil {
		return StartOutput{}, common.EnsureCoded(err, common.ErrWriteFailed, "create session")
	}

	return StartOutput{Session: record}, nil
}
