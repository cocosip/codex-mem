package agents

import (
	"strings"

	"codex-mem/internal/domain/common"
)

// Target identifies where AGENTS.md content should be installed.
type Target string

// Supported AGENTS installation targets.
const (
	TargetGlobal  Target = "global"
	TargetProject Target = "project"
	TargetBoth    Target = "both"
)

// Validate reports whether t is a supported installation target.
func (t Target) Validate() error {
	switch t {
	case TargetGlobal, TargetProject, TargetBoth:
		return nil
	default:
		return common.NewError(common.ErrInvalidTarget, "target must be global, project, or both")
	}
}

// Mode controls how AGENTS.md content is written.
type Mode string

// Supported AGENTS installation modes.
const (
	ModeSafe      Mode = "safe"
	ModeAppend    Mode = "append"
	ModeOverwrite Mode = "overwrite"
)

// Validate reports whether m is a supported write mode.
func (m Mode) Validate() error {
	switch m {
	case ModeSafe, ModeAppend, ModeOverwrite:
		return nil
	default:
		return common.NewError(common.ErrInvalidTarget, "mode must be safe, append, or overwrite")
	}
}

// InstallInput captures the request parameters for AGENTS installation.
type InstallInput struct {
	Target                    Target   `json:"target"`
	Mode                      Mode     `json:"mode"`
	CWD                       string   `json:"cwd,omitempty"`
	ProjectName               string   `json:"project_name,omitempty"`
	SystemName                string   `json:"system_name,omitempty"`
	RelatedRepositories       []string `json:"related_repositories,omitempty"`
	PreferredTags             []string `json:"preferred_tags,omitempty"`
	AllowRelatedProjectMemory *bool    `json:"allow_related_project_memory,omitempty"`
}

// FileChange describes an AGENTS file that was written or skipped.
type FileChange struct {
	Path   string `json:"path"`
	Target Target `json:"target"`
	Mode   Mode   `json:"mode"`
	Reason string `json:"reason,omitempty"`
}

// InstallOutput reports the file changes and warnings produced by AGENTS installation.
type InstallOutput struct {
	WrittenFiles []FileChange     `json:"written_files"`
	SkippedFiles []FileChange     `json:"skipped_files"`
	Warnings     []common.Warning `json:"warnings"`
}

func normalizeList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
