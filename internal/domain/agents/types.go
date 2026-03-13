package agents

import (
	"strings"

	"codex-mem/internal/domain/common"
)

type Target string

const (
	TargetGlobal  Target = "global"
	TargetProject Target = "project"
	TargetBoth    Target = "both"
)

func (t Target) Validate() error {
	switch t {
	case TargetGlobal, TargetProject, TargetBoth:
		return nil
	default:
		return common.NewError(common.ErrInvalidTarget, "target must be global, project, or both")
	}
}

type Mode string

const (
	ModeSafe      Mode = "safe"
	ModeAppend    Mode = "append"
	ModeOverwrite Mode = "overwrite"
)

func (m Mode) Validate() error {
	switch m {
	case ModeSafe, ModeAppend, ModeOverwrite:
		return nil
	default:
		return common.NewError(common.ErrInvalidTarget, "mode must be safe, append, or overwrite")
	}
}

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

type FileChange struct {
	Path   string `json:"path"`
	Target Target `json:"target"`
	Mode   Mode   `json:"mode"`
	Reason string `json:"reason,omitempty"`
}

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
