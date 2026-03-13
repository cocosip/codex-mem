package memory

import (
	"strings"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
)

// NoteType identifies the durable note category.
type NoteType string

const (
	// NoteTypeDecision stores an implementation or product decision.
	NoteTypeDecision NoteType = "decision"
	// NoteTypeBugfix stores a reusable bug root-cause or fix insight.
	NoteTypeBugfix NoteType = "bugfix"
	// NoteTypeDiscovery stores a reusable technical finding.
	NoteTypeDiscovery NoteType = "discovery"
	// NoteTypeConstraint stores an ongoing rule or limitation.
	NoteTypeConstraint NoteType = "constraint"
	// NoteTypePreference stores a durable preference.
	NoteTypePreference NoteType = "preference"
	// NoteTypeTodo stores a durable open task worth resurfacing later.
	NoteTypeTodo NoteType = "todo"
)

// Validate ensures the note type is supported.
func (t NoteType) Validate() error {
	switch t {
	case NoteTypeDecision, NoteTypeBugfix, NoteTypeDiscovery, NoteTypeConstraint, NoteTypePreference, NoteTypeTodo:
		return nil
	default:
		return common.NewError(common.ErrInvalidInput, "invalid note type")
	}
}

// Status identifies the lifecycle state of a note.
type Status string

const (
	// StatusActive marks a note as currently relevant.
	StatusActive Status = "active"
	// StatusResolved marks a note as resolved but still historically useful.
	StatusResolved Status = "resolved"
	// StatusSuperseded marks a note as replaced by a newer conclusion.
	StatusSuperseded Status = "superseded"
)

// Validate ensures the note status is supported.
func (s Status) Validate() error {
	switch s {
	case StatusActive, StatusResolved, StatusSuperseded:
		return nil
	default:
		return common.NewError(common.ErrInvalidState, "invalid note status")
	}
}

// Source identifies how a note entered durable storage.
type Source string

const (
	// SourceCodexExplicit marks a note explicitly requested by the user or agent.
	SourceCodexExplicit Source = "codex_explicit"
	// SourceWatcherImport marks a note imported by a watcher integration.
	SourceWatcherImport Source = "watcher_import"
	// SourceRelayImport marks a note imported through a relay integration.
	SourceRelayImport Source = "relay_import"
	// SourceRecoveryGenerated marks a note synthesized during recovery.
	SourceRecoveryGenerated Source = "recovery_generated"
)

// Validate ensures the note source is supported.
func (s Source) Validate() error {
	switch s {
	case SourceCodexExplicit, SourceWatcherImport, SourceRelayImport, SourceRecoveryGenerated:
		return nil
	default:
		return common.NewError(common.ErrInvalidInput, "invalid note source")
	}
}

// Note is the durable record returned by note reads and searches.
type Note struct {
	ID                string    `json:"note_id"`
	Scope             scope.Ref `json:"scope"`
	SessionID         string    `json:"session_id"`
	Type              NoteType  `json:"type"`
	Title             string    `json:"title"`
	Content           string    `json:"content"`
	Importance        int       `json:"importance"`
	Tags              []string  `json:"tags,omitempty"`
	FilePaths         []string  `json:"file_paths,omitempty"`
	RelatedProjectIDs []string  `json:"related_project_ids,omitempty"`
	Status            Status    `json:"status"`
	Source            Source    `json:"source"`
	RelationType      string    `json:"relation_type,omitempty"`
	Searchable        bool      `json:"searchable"`
	ExclusionReason   string    `json:"exclusion_reason,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"-"`
}

// SaveInput is the caller-facing payload for writing a note.
type SaveInput struct {
	Scope             scope.Ref
	SessionID         string
	Type              NoteType
	Title             string
	Content           string
	Importance        int
	Tags              []string
	FilePaths         []string
	RelatedProjectIDs []string
	Status            Status
	Source            Source
	PrivacyIntent     string
}

// SaveOutput reports the stored note and any warnings.
type SaveOutput struct {
	Note         Note             `json:"note"`
	StoredAt     time.Time        `json:"stored_at"`
	Deduplicated bool             `json:"deduplicated"`
	Warnings     []common.Warning `json:"warnings"`
}

// Repository persists and queries durable note records.
type Repository interface {
	FindDuplicate(note Note) (*Note, error)
	Create(note Note) error
}

// Validate ensures the note has the required fields for durable storage.
func (n Note) Validate() error {
	if err := n.Scope.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(n.SessionID) == "" {
		return common.NewError(common.ErrInvalidInput, "session_id is required")
	}
	if err := n.Type.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(n.Title) == "" {
		return common.NewError(common.ErrInvalidInput, "title is required")
	}
	if strings.TrimSpace(n.Content) == "" {
		return common.NewError(common.ErrInvalidInput, "content is required")
	}
	if n.Importance < 1 || n.Importance > 5 {
		return common.NewError(common.ErrInvalidInput, "importance must be between 1 and 5")
	}
	if err := n.Status.Validate(); err != nil {
		return err
	}
	if err := n.Source.Validate(); err != nil {
		return err
	}
	if n.CreatedAt.IsZero() {
		return common.NewError(common.ErrInvalidInput, "created_at is required")
	}
	return nil
}
