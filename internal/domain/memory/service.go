package memory

import (
	"context"
	"path/filepath"
	"strings"

	"codex-mem/internal/domain/common"
)

type Options struct {
	Clock     common.Clock
	IDFactory common.IDFactory
}

type Service struct {
	repo    Repository
	options Options
}

func NewService(repo Repository, options Options) *Service {
	if options.Clock == nil {
		options.Clock = common.RealClock{}
	}
	if options.IDFactory == nil {
		options.IDFactory = common.DefaultIDFactory{Clock: options.Clock}
	}
	return &Service{repo: repo, options: options}
}

func (s *Service) SaveNote(ctx context.Context, input SaveInput) (SaveOutput, error) {
	_ = ctx

	status := input.Status
	if status == "" {
		status = StatusActive
	}
	source := input.Source
	if source == "" {
		source = SourceCodexExplicit
	}

	now := s.options.Clock.Now().UTC()
	record := Note{
		ID:                s.options.IDFactory.New("note"),
		Scope:             input.Scope,
		SessionID:         strings.TrimSpace(input.SessionID),
		Type:              input.Type,
		Title:             strings.TrimSpace(input.Title),
		Content:           strings.TrimSpace(input.Content),
		Importance:        input.Importance,
		Tags:              normalizeTags(input.Tags),
		FilePaths:         normalizePaths(input.FilePaths),
		RelatedProjectIDs: normalizeStrings(input.RelatedProjectIDs),
		Status:            status,
		Source:            source,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := record.Validate(); err != nil {
		return SaveOutput{}, err
	}

	duplicate, err := s.repo.FindDuplicate(record)
	if err != nil {
		return SaveOutput{}, err
	}
	if duplicate != nil {
		return SaveOutput{
			Note:         *duplicate,
			StoredAt:     duplicate.CreatedAt,
			Deduplicated: true,
			Warnings: []common.Warning{
				{Code: common.WarnDedupeApplied, Message: "matched an existing note and reused it"},
			},
		}, nil
	}

	if err := s.repo.Create(record); err != nil {
		return SaveOutput{}, err
	}

	return SaveOutput{
		Note:     record,
		StoredAt: record.CreatedAt,
	}, nil
}

func normalizeTags(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		tag := common.Slug(value)
		if tag == "unknown" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
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
		normalized := filepath.ToSlash(filepath.Clean(trimmed))
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
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
