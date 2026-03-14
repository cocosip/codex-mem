// Package handoff stores durable continuation records for future sessions.
package handoff

import (
	"context"
	"path"
	"strings"

	"codex-mem/internal/domain/common"
)

// Options configures handoff persistence dependencies.
type Options struct {
	Clock     common.Clock
	IDFactory common.IDFactory
}

// Service validates and persists handoff records.
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

// SaveHandoff validates input, stores a handoff, and returns persistence warnings.
func (s *Service) SaveHandoff(ctx context.Context, input SaveInput) (SaveOutput, error) {
	_ = ctx
	if isPrivateIntent(input.PrivacyIntent) {
		return SaveOutput{}, common.NewError(common.ErrInvalidInput, "private/do_not_store content must not be written to durable memory")
	}

	now := s.options.Clock.Now().UTC()
	record := Handoff{
		ID:             s.options.IDFactory.New("handoff"),
		Scope:          input.Scope,
		SessionID:      strings.TrimSpace(input.SessionID),
		Kind:           input.Kind,
		Task:           strings.TrimSpace(input.Task),
		Summary:        strings.TrimSpace(input.Summary),
		Completed:      normalizeStrings(input.Completed),
		NextSteps:      normalizeStrings(input.NextSteps),
		OpenQuestions:  normalizeStrings(input.OpenQuestions),
		Risks:          normalizeStrings(input.Risks),
		FilesTouched:   normalizePaths(input.FilesTouched),
		RelatedNoteIDs: normalizeStrings(input.RelatedNoteIDs),
		Status:         input.Status,
		Searchable:     true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := record.Validate(); err != nil {
		return SaveOutput{}, err
	}

	var warnings []common.Warning
	existing, err := s.repo.FindLatestOpenByTask(record.Scope, record.Task)
	if err != nil {
		return SaveOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "find latest open handoff by task")
	}
	if existing != nil {
		warnings = append(warnings, common.Warning{
			Code:    common.WarnDedupeApplied,
			Message: "an open handoff for the same task already exists; storing a new handoff without overwriting it",
		})
	}
	if len(record.Completed) == 0 && len(record.OpenQuestions) == 0 && len(record.Risks) == 0 && len(record.FilesTouched) == 0 {
		warnings = append(warnings, common.Warning{
			Code:    common.WarnHandoffSparse,
			Message: "handoff stored with minimal continuation detail",
		})
	}

	if err := s.repo.Create(record); err != nil {
		return SaveOutput{}, common.EnsureCoded(err, common.ErrWriteFailed, "create handoff")
	}

	return SaveOutput{
		Handoff:              record,
		StoredAt:             record.CreatedAt,
		EligibleForBootstrap: record.Status == StatusOpen,
		Warnings:             warnings,
	}, nil
}

func isPrivateIntent(privacyIntent string) bool {
	switch strings.TrimSpace(strings.ToLower(privacyIntent)) {
	case "private", "do_not_store", "ephemeral_only":
		return true
	default:
		return false
	}
}

func normalizeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func normalizePaths(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized := path.Clean(strings.ReplaceAll(trimmed, "\\", "/"))
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}
