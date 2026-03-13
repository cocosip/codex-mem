package memory

import (
	"strings"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
)

type NoteType string

const (
	NoteTypeDecision   NoteType = "decision"
	NoteTypeBugfix     NoteType = "bugfix"
	NoteTypeDiscovery  NoteType = "discovery"
	NoteTypeConstraint NoteType = "constraint"
	NoteTypePreference NoteType = "preference"
	NoteTypeTodo       NoteType = "todo"
)

func (t NoteType) Validate() error {
	switch t {
	case NoteTypeDecision, NoteTypeBugfix, NoteTypeDiscovery, NoteTypeConstraint, NoteTypePreference, NoteTypeTodo:
		return nil
	default:
		return common.NewError(common.ErrInvalidInput, "invalid note type")
	}
}

type Status string

const (
	StatusActive     Status = "active"
	StatusResolved   Status = "resolved"
	StatusSuperseded Status = "superseded"
)

func (s Status) Validate() error {
	switch s {
	case StatusActive, StatusResolved, StatusSuperseded:
		return nil
	default:
		return common.NewError(common.ErrInvalidState, "invalid note status")
	}
}

type Source string

const (
	SourceCodexExplicit     Source = "codex_explicit"
	SourceWatcherImport     Source = "watcher_import"
	SourceRelayImport       Source = "relay_import"
	SourceRecoveryGenerated Source = "recovery_generated"
)

func (s Source) Validate() error {
	switch s {
	case SourceCodexExplicit, SourceWatcherImport, SourceRelayImport, SourceRecoveryGenerated:
		return nil
	default:
		return common.NewError(common.ErrInvalidInput, "invalid note source")
	}
}

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
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"-"`
}

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
}

type SaveOutput struct {
	Note         Note             `json:"note"`
	StoredAt     time.Time        `json:"stored_at"`
	Deduplicated bool             `json:"deduplicated"`
	Warnings     []common.Warning `json:"warnings"`
}

type Repository interface {
	FindDuplicate(note Note) (*Note, error)
	Create(note Note) error
}

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
