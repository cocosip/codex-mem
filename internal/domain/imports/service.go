package imports

import (
	"context"
	"strings"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/memory"
)

// Options configures time and id generation for import persistence.
type Options struct {
	Clock             common.Clock
	IDFactory         common.IDFactory
	NoteSaver         NoteSaver
	ProjectNoteFinder ProjectNoteFinder
	TransactionRunner TransactionRunner
}

// Service validates and stores import audit records.
type Service struct {
	repo              Repository
	noteSaver         NoteSaver
	projectNoteFinder ProjectNoteFinder
	transactionRunner TransactionRunner
	options           Options
}

// NewService constructs an import service with default clock and id generation.
func NewService(repo Repository, options Options) *Service {
	if options.Clock == nil {
		options.Clock = common.RealClock{}
	}
	if options.IDFactory == nil {
		options.IDFactory = common.DefaultIDFactory{Clock: options.Clock}
	}
	return &Service{
		repo:              repo,
		noteSaver:         options.NoteSaver,
		projectNoteFinder: options.ProjectNoteFinder,
		transactionRunner: options.TransactionRunner,
		options:           options,
	}
}

// SaveImport tracks one imported artifact and suppresses duplicate or privacy-blocked imports.
func (s *Service) SaveImport(ctx context.Context, input SaveInput) (SaveOutput, error) {
	return s.saveImportWithRepo(ctx, input, s.repo)
}

func (s *Service) saveImportWithRepo(ctx context.Context, input SaveInput, repo Repository) (SaveOutput, error) {
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
	if reason := normalizeSuppressionReason(input.SuppressionReason); reason != "" {
		record.Suppressed = true
		record.SuppressionReason = reason
		record.DurableMemoryID = ""
	} else if isPrivateIntent(input.PrivacyIntent) {
		record.Suppressed = true
		record.SuppressionReason = "privacy_intent"
		record.DurableMemoryID = ""
	}
	if err := record.Validate(); err != nil {
		return SaveOutput{}, err
	}

	duplicate, err := repo.FindDuplicate(record)
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

	if err := repo.Create(record); err != nil {
		return SaveOutput{}, common.EnsureCoded(err, common.ErrWriteFailed, "create import record")
	}

	output := SaveOutput{
		Record:     record,
		StoredAt:   record.ImportedAt,
		Suppressed: record.Suppressed,
	}
	if record.Suppressed {
		output.Warnings = []common.Warning{
			{Code: common.WarnImportSuppressed, Message: suppressionWarningMessage(record.SuppressionReason)},
		}
	}
	return output, nil
}

// SaveImportedNote materializes an imported artifact into durable note memory plus import audit.
func (s *Service) SaveImportedNote(ctx context.Context, input SaveImportedNoteInput) (SaveImportedNoteOutput, error) {
	if s.transactionRunner != nil {
		var output SaveImportedNoteOutput
		err := s.transactionRunner.RunSaveImportedNote(ctx, func(repo Repository, noteSaver NoteSaver, projectNoteFinder ProjectNoteFinder) error {
			var err error
			output, err = s.saveImportedNoteWithDeps(ctx, input, repo, noteSaver, projectNoteFinder)
			return err
		})
		return output, err
	}
	return s.saveImportedNoteWithDeps(ctx, input, s.repo, s.noteSaver, s.projectNoteFinder)
}

func (s *Service) saveImportedNoteWithDeps(ctx context.Context, input SaveImportedNoteInput, repo Repository, noteSaver NoteSaver, projectNoteFinder ProjectNoteFinder) (SaveImportedNoteOutput, error) {
	if repo == nil || noteSaver == nil || projectNoteFinder == nil {
		return SaveImportedNoteOutput{}, common.NewError(common.ErrWriteFailed, "imported note materialization is not configured")
	}
	if err := input.Source.Validate(); err != nil {
		return SaveImportedNoteOutput{}, err
	}
	if isPrivateIntent(input.PrivacyIntent) {
		importOutput, err := s.saveImportWithRepo(ctx, SaveInput{
			Scope:         input.Scope,
			SessionID:     input.SessionID,
			Source:        input.Source,
			ExternalID:    input.ExternalID,
			PayloadHash:   input.PayloadHash,
			PrivacyIntent: input.PrivacyIntent,
		}, repo)
		if err != nil {
			return SaveImportedNoteOutput{}, err
		}
		return SaveImportedNoteOutput{
			Import:             importOutput.Record,
			Materialized:       false,
			ImportDeduplicated: importOutput.Deduplicated,
			Suppressed:         true,
			Warnings:           importOutput.Warnings,
		}, nil
	}

	title := strings.TrimSpace(input.Title)
	content := strings.TrimSpace(input.Content)
	existing, err := projectNoteFinder.FindProjectDuplicate(input.Scope, input.Type, title, content)
	if err != nil {
		return SaveImportedNoteOutput{}, common.EnsureCoded(err, common.ErrReadFailed, "find project duplicate note")
	}

	if existing != nil && existing.Source == memory.SourceCodexExplicit {
		importOutput, err := s.saveImportWithRepo(ctx, SaveInput{
			Scope:             input.Scope,
			SessionID:         input.SessionID,
			Source:            input.Source,
			ExternalID:        input.ExternalID,
			PayloadHash:       input.PayloadHash,
			PrivacyIntent:     input.PrivacyIntent,
			SuppressionReason: "explicit_memory_exists",
		}, repo)
		if err != nil {
			return SaveImportedNoteOutput{}, err
		}
		return SaveImportedNoteOutput{
			Note:               existing,
			Import:             importOutput.Record,
			Materialized:       false,
			NoteDeduplicated:   true,
			ImportDeduplicated: importOutput.Deduplicated,
			Suppressed:         true,
			Warnings:           importOutput.Warnings,
		}, nil
	}

	var (
		note             *memory.Note
		noteDeduplicated bool
		warnings         []common.Warning
	)
	if existing != nil {
		note = existing
		noteDeduplicated = true
		warnings = common.MergeWarnings(warnings, []common.Warning{{
			Code:    common.WarnDedupeApplied,
			Message: "matched an existing imported note and reused it",
		}})
	} else {
		noteOutput, err := noteSaver.SaveNote(ctx, memory.SaveInput{
			Scope:             input.Scope,
			SessionID:         input.SessionID,
			Type:              input.Type,
			Title:             title,
			Content:           content,
			Importance:        input.Importance,
			Tags:              input.Tags,
			FilePaths:         input.FilePaths,
			RelatedProjectIDs: input.RelatedProjectIDs,
			Status:            input.Status,
			Source:            noteSourceFromImport(input.Source),
			PrivacyIntent:     input.PrivacyIntent,
		})
		if err != nil {
			return SaveImportedNoteOutput{}, common.EnsureCoded(err, common.ErrWriteFailed, "save imported note")
		}
		note = &noteOutput.Note
		noteDeduplicated = noteOutput.Deduplicated
		warnings = common.MergeWarnings(warnings, noteOutput.Warnings)
	}

	importOutput, err := s.saveImportWithRepo(ctx, SaveInput{
		Scope:           input.Scope,
		SessionID:       input.SessionID,
		Source:          input.Source,
		ExternalID:      input.ExternalID,
		PayloadHash:     input.PayloadHash,
		DurableMemoryID: note.ID,
		PrivacyIntent:   input.PrivacyIntent,
	}, repo)
	if err != nil {
		return SaveImportedNoteOutput{}, err
	}

	warnings = common.MergeWarnings(warnings, importOutput.Warnings)
	return SaveImportedNoteOutput{
		Note:               note,
		Import:             importOutput.Record,
		Materialized:       note != nil,
		NoteDeduplicated:   noteDeduplicated,
		ImportDeduplicated: importOutput.Deduplicated,
		Suppressed:         importOutput.Suppressed,
		Warnings:           warnings,
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

func normalizeHash(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeSuppressionReason(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func suppressionWarningMessage(reason string) string {
	switch normalizeSuppressionReason(reason) {
	case "privacy_intent":
		return "import was suppressed by privacy policy"
	case "explicit_memory_exists":
		return "import was suppressed because stronger explicit memory already exists"
	default:
		return "import was suppressed by import policy"
	}
}

func noteSourceFromImport(source Source) memory.Source {
	switch source {
	case SourceRelayImport:
		return memory.SourceRelayImport
	default:
		return memory.SourceWatcherImport
	}
}
