package imports

import (
	"context"
	"strings"

	"codex-mem/internal/domain/common"
)

// Options configures time and id generation for import persistence.
type Options struct {
	Clock     common.Clock
	IDFactory common.IDFactory
}

// Service validates and stores import audit records.
type Service struct {
	repo    Repository
	options Options
}

// NewService constructs an import service with default clock and id generation.
func NewService(repo Repository, options Options) *Service {
	if options.Clock == nil {
		options.Clock = common.RealClock{}
	}
	if options.IDFactory == nil {
		options.IDFactory = common.DefaultIDFactory{Clock: options.Clock}
	}
	return &Service{repo: repo, options: options}
}

// SaveImport tracks one imported artifact and suppresses duplicate or privacy-blocked imports.
func (s *Service) SaveImport(ctx context.Context, input SaveInput) (SaveOutput, error) {
	_ = ctx

	now := s.options.Clock.Now().UTC()
	record := Record{
		ID:              s.options.IDFactory.New("import"),
		Scope:           input.Scope,
		SessionID:       strings.TrimSpace(input.SessionID),
		Source:          input.Source,
		ExternalID:      strings.TrimSpace(input.ExternalID),
		PayloadHash:     normalizeHash(input.PayloadHash),
		DurableMemoryID: strings.TrimSpace(input.DurableMemoryID),
		ImportedAt:      now,
	}
	if isPrivateIntent(input.PrivacyIntent) {
		record.Suppressed = true
		record.SuppressionReason = "privacy_intent"
		record.DurableMemoryID = ""
	}
	if err := record.Validate(); err != nil {
		return SaveOutput{}, err
	}

	duplicate, err := s.repo.FindDuplicate(record)
	if err != nil {
		return SaveOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "find duplicate import")
	}
	if duplicate != nil {
		return SaveOutput{
			Record:       *duplicate,
			StoredAt:     duplicate.ImportedAt,
			Suppressed:   true,
			Deduplicated: true,
			Warnings: []common.Warning{
				{Code: common.WarnImportSuppressed, Message: "matched an existing import record and skipped duplicate import"},
			},
		}, nil
	}

	if err := s.repo.Create(record); err != nil {
		return SaveOutput{}, common.EnsureCoded(err, common.ErrWriteFailed, "create import record")
	}

	output := SaveOutput{
		Record:     record,
		StoredAt:   record.ImportedAt,
		Suppressed: record.Suppressed,
	}
	if record.Suppressed {
		output.Warnings = []common.Warning{
			{Code: common.WarnImportSuppressed, Message: "import was suppressed by privacy policy"},
		}
	}
	return output, nil
}

func isPrivateIntent(privacyIntent string) bool {
	switch strings.TrimSpace(strings.ToLower(privacyIntent)) {
	case "private", "do_not_store", "ephemeral_only":
		return true
	default:
		return false
	}
}

func normalizeHash(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
