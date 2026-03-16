// Package imports tracks imported artifacts for dedupe and provenance audits.
package imports

import (
	"strings"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
)

// Source identifies the secondary source from which an artifact was imported.
type Source string

const (
	// SourceWatcherImport marks an artifact imported from a local watcher path.
	SourceWatcherImport Source = "watcher_import"
	// SourceRelayImport marks an artifact imported from a relay-side capture path.
	SourceRelayImport Source = "relay_import"
)

// Validate ensures the import source is supported.
func (s Source) Validate() error {
	switch s {
	case SourceWatcherImport, SourceRelayImport:
		return nil
	default:
		return common.NewError(common.ErrInvalidInput, "invalid import source")
	}
}

// Record captures one imported artifact audit record.
type Record struct {
	ID                string    `json:"import_id"`
	Scope             scope.Ref `json:"scope"`
	SessionID         string    `json:"session_id"`
	Source            Source    `json:"source"`
	ExternalID        string    `json:"external_id,omitempty"`
	PayloadHash       string    `json:"payload_hash,omitempty"`
	DurableMemoryID   string    `json:"durable_memory_id,omitempty"`
	Suppressed        bool      `json:"suppressed"`
	SuppressionReason string    `json:"suppression_reason,omitempty"`
	ImportedAt        time.Time `json:"imported_at"`
}

// SaveInput is the caller-facing payload for tracking an imported artifact.
type SaveInput struct {
	Scope           scope.Ref
	SessionID       string
	Source          Source
	ExternalID      string
	PayloadHash     string
	DurableMemoryID string
	PrivacyIntent   string
}

// SaveOutput reports the import audit result and any suppression warnings.
type SaveOutput struct {
	Record       Record           `json:"record"`
	StoredAt     time.Time        `json:"stored_at"`
	Suppressed   bool             `json:"suppressed"`
	Deduplicated bool             `json:"deduplicated"`
	Warnings     []common.Warning `json:"warnings"`
}

// Repository persists import audit records and detects duplicate imports.
type Repository interface {
	FindDuplicate(record Record) (*Record, error)
	Create(record Record) error
}

// Validate ensures the import record contains the minimum durable audit metadata.
func (r Record) Validate() error {
	if err := r.Scope.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.SessionID) == "" {
		return common.NewError(common.ErrInvalidInput, "session_id is required")
	}
	if err := r.Source.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.ExternalID) == "" && strings.TrimSpace(r.PayloadHash) == "" {
		return common.NewError(common.ErrInvalidInput, "external_id or payload_hash is required")
	}
	if r.Suppressed && strings.TrimSpace(r.SuppressionReason) == "" {
		return common.NewError(common.ErrInvalidInput, "suppression_reason is required when suppressed")
	}
	if r.Suppressed && strings.TrimSpace(r.DurableMemoryID) != "" {
		return common.NewError(common.ErrInvalidInput, "durable_memory_id must be empty when suppressed")
	}
	if r.ImportedAt.IsZero() {
		return common.NewError(common.ErrInvalidInput, "imported_at is required")
	}
	return nil
}
