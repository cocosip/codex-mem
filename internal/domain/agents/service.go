package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"codex-mem/internal/domain/common"
	projecttemplates "codex-mem/templates"
)

const (
	defaultCodexDir = ".codex"
	agentsFileName  = "AGENTS.md"
)

var unresolvedPlaceholderPattern = regexp.MustCompile(`<[^>\n]+>`)

type Options struct {
	HomeDir string
}

type Service struct {
	options         Options
	globalTemplate  string
	projectTemplate string
}

func NewService(options Options) *Service {
	return &Service{
		options:         options,
		globalTemplate:  projecttemplates.GlobalAgentsTemplate,
		projectTemplate: projecttemplates.ProjectAgentsTemplate,
	}
}

func (s *Service) Install(ctx context.Context, input InstallInput) (InstallOutput, error) {
	_ = ctx

	if err := input.Target.Validate(); err != nil {
		return InstallOutput{}, err
	}
	if err := input.Mode.Validate(); err != nil {
		return InstallOutput{}, err
	}

	plans, err := s.resolvePlans(input)
	if err != nil {
		return InstallOutput{}, err
	}

	output := InstallOutput{}
	for _, plan := range plans {
		content, renderWarnings := s.render(plan.target, input)

		change, skipped, applyWarnings, err := s.applyPlan(plan, input.Mode, content)
		if err != nil {
			return InstallOutput{}, err
		}
		output.Warnings = common.MergeWarnings(output.Warnings, applyWarnings)
		if skipped != nil {
			output.SkippedFiles = append(output.SkippedFiles, *skipped)
			continue
		}
		output.Warnings = common.MergeWarnings(output.Warnings, renderWarnings)
		output.WrittenFiles = append(output.WrittenFiles, *change)
	}

	return output, nil
}

type installPlan struct {
	target Target
	path   string
}

func (s *Service) resolvePlans(input InstallInput) ([]installPlan, error) {
	switch input.Target {
	case TargetGlobal:
		path, err := s.globalAgentsPath()
		if err != nil {
			return nil, err
		}
		return []installPlan{{target: TargetGlobal, path: path}}, nil
	case TargetProject:
		path, err := projectAgentsPath(input.CWD)
		if err != nil {
			return nil, err
		}
		return []installPlan{{target: TargetProject, path: path}}, nil
	case TargetBoth:
		globalPath, err := s.globalAgentsPath()
		if err != nil {
			return nil, err
		}
		projectPath, err := projectAgentsPath(input.CWD)
		if err != nil {
			return nil, err
		}
		return []installPlan{
			{target: TargetGlobal, path: globalPath},
			{target: TargetProject, path: projectPath},
		}, nil
	default:
		return nil, common.NewError(common.ErrInvalidTarget, "target must be global, project, or both")
	}
}

func (s *Service) globalAgentsPath() (string, error) {
	homeDir := strings.TrimSpace(s.options.HomeDir)
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return "", common.WrapError(common.ErrAgentsWriteDenied, "resolve user home directory", err)
		}
	}
	absHome, err := filepath.Abs(homeDir)
	if err != nil {
		return "", common.WrapError(common.ErrAgentsWriteDenied, "resolve global agents home directory", err)
	}
	return filepath.Join(absHome, defaultCodexDir, agentsFileName), nil
}

func projectAgentsPath(cwd string) (string, error) {
	if strings.TrimSpace(cwd) == "" {
		return "", common.NewError(common.ErrInvalidInput, "cwd is required for project AGENTS installation")
	}
	absCWD, err := filepath.Abs(cwd)
	if err != nil {
		return "", common.WrapError(common.ErrInvalidInput, "resolve project cwd", err)
	}
	return filepath.Join(absCWD, agentsFileName), nil
}

func (s *Service) render(target Target, input InstallInput) (string, []common.Warning) {
	switch target {
	case TargetGlobal:
		return strings.TrimSpace(s.globalTemplate) + "\n", nil
	default:
		return renderProjectTemplate(strings.TrimSpace(s.projectTemplate)+"\n", input)
	}
}

func renderProjectTemplate(template string, input InstallInput) (string, []common.Warning) {
	replacer := strings.NewReplacer(
		"<project-name>", strings.TrimSpace(input.ProjectName),
		"<system-name>", strings.TrimSpace(input.SystemName),
		"<tag-1>", listValue(input.PreferredTags, 0, "<tag-1>"),
		"<tag-2>", listValue(input.PreferredTags, 1, "<tag-2>"),
		"<tag-3>", listValue(input.PreferredTags, 2, "<tag-3>"),
		"<repo-a>", listValue(input.RelatedRepositories, 0, "<repo-a>"),
		"<repo-b>", listValue(input.RelatedRepositories, 1, "<repo-b>"),
		"<repo-c>", listValue(input.RelatedRepositories, 2, "<repo-c>"),
	)
	content := replacer.Replace(template)

	if strings.TrimSpace(input.ProjectName) == "" {
		content = strings.ReplaceAll(content, "- Project name: ", "- Project name: <project-name>")
	}
	if strings.TrimSpace(input.SystemName) == "" {
		content = strings.ReplaceAll(content, "- System name: ", "- System name: <system-name>")
	}

	if input.AllowRelatedProjectMemory != nil && !*input.AllowRelatedProjectMemory {
		content = strings.Replace(content,
			"- Related-project memory is allowed only when the task clearly depends on another repository in the same system.",
			"- Related-project memory is disabled by default for this repository unless explicitly needed for an integration task.",
			1,
		)
		content = strings.Replace(content,
			"- Typical examples include API contracts, schema changes, generated clients, deployment coordination, and integration debugging.",
			"- Re-enable related-project memory only for explicit cross-repository work such as API contracts or deployment coordination.",
			1,
		)
	}

	var warnings []common.Warning
	if unresolvedPlaceholderPattern.MatchString(content) {
		warnings = append(warnings, common.Warning{
			Code:    common.WarnPlaceholdersUnresolved,
			Message: "AGENTS installation completed with unresolved placeholders for manual completion",
		})
	}
	return content, warnings
}

func listValue(values []string, index int, fallback string) string {
	values = normalizeList(values)
	if index < len(values) {
		return values[index]
	}
	return fallback
}

func (s *Service) applyPlan(plan installPlan, mode Mode, content string) (*FileChange, *FileChange, []common.Warning, error) {
	existing, exists, err := readIfExists(plan.path)
	if err != nil {
		return nil, nil, nil, err
	}

	switch mode {
	case ModeSafe:
		if exists {
			return nil, &FileChange{
					Path:   plan.path,
					Target: plan.target,
					Mode:   mode,
					Reason: "existing AGENTS.md was preserved in safe mode",
				}, []common.Warning{{
					Code:    common.WarnExistingAgentsSkipped,
					Message: fmt.Sprintf("%s AGENTS.md already existed and was skipped in safe mode", plan.target),
				}}, nil
		}
		if err := writeFile(plan.path, content); err != nil {
			return nil, nil, nil, err
		}
		return &FileChange{Path: plan.path, Target: plan.target, Mode: mode}, nil, nil, nil
	case ModeAppend:
		if !exists {
			if err := writeFile(plan.path, content); err != nil {
				return nil, nil, nil, err
			}
			return &FileChange{Path: plan.path, Target: plan.target, Mode: mode}, nil, nil, nil
		}
		block := appendBlock(plan.target, content)
		if strings.Contains(existing, appendBlockStart(plan.target)) {
			return nil, &FileChange{
					Path:   plan.path,
					Target: plan.target,
					Mode:   mode,
					Reason: "append block already existed and was not duplicated",
				}, []common.Warning{{
					Code:    common.WarnExistingAgentsSkipped,
					Message: fmt.Sprintf("%s AGENTS append block already existed and was skipped", plan.target),
				}}, nil
		}
		body := strings.TrimRight(existing, "\r\n") + "\n\n" + block
		if err := writeFile(plan.path, body); err != nil {
			return nil, nil, nil, err
		}
		return &FileChange{Path: plan.path, Target: plan.target, Mode: mode}, nil, nil, nil
	case ModeOverwrite:
		if err := writeFile(plan.path, content); err != nil {
			return nil, nil, nil, err
		}
		return &FileChange{Path: plan.path, Target: plan.target, Mode: mode}, nil, nil, nil
	default:
		return nil, nil, nil, common.NewError(common.ErrInvalidTarget, "mode must be safe, append, or overwrite")
	}
}

func readIfExists(path string) (string, bool, error) {
	body, err := os.ReadFile(path)
	switch {
	case err == nil:
		return string(body), true, nil
	case os.IsNotExist(err):
		return "", false, nil
	default:
		return "", false, common.WrapError(common.ErrAgentsWriteDenied, "read AGENTS file", err)
	}
}

func writeFile(path string, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return common.WrapError(common.ErrAgentsWriteDenied, "create AGENTS directory", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return common.WrapError(common.ErrAgentsWriteDenied, "write AGENTS file", err)
	}
	return nil
}

func appendBlock(target Target, content string) string {
	start := appendBlockStart(target)
	end := appendBlockEnd(target)
	return start + "\n" + strings.TrimRight(content, "\r\n") + "\n" + end + "\n"
}

func appendBlockStart(target Target) string {
	return fmt.Sprintf("<!-- codex-mem:%s-agents:start -->", target)
}

func appendBlockEnd(target Target) string {
	return fmt.Sprintf("<!-- codex-mem:%s-agents:end -->", target)
}
