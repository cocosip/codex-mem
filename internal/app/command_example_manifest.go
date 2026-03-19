package app

import (
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	_ "embed"
)

const commandExampleDirName = "testdata"
const commandExampleManifestName = "command-example-manifest.txt"

// EmbeddedCommandExampleManifest contains the checked-in import/follow example catalog.
//
//go:embed testdata/command-example-manifest.txt
var EmbeddedCommandExampleManifest string

type listCommandExamplesOptions struct {
	JSON     bool
	Commands []string
	Examples []string
	Formats  []string
	Tags     []string
}

type commandExampleManifestEntry struct {
	Command      string   `json:"command"`
	Name         string   `json:"name"`
	RelativePath string   `json:"path"`
	Format       string   `json:"format"`
	Tags         []string `json:"tags,omitempty"`
	Summary      string   `json:"summary,omitempty"`
}

type commandExampleManifestReport struct {
	Version      string                        `json:"version"`
	ExampleCount int                           `json:"example_count"`
	Examples     []commandExampleManifestEntry `json:"examples"`
}

func parseListCommandExamplesOptions(args []string) (listCommandExamplesOptions, error) {
	options := listCommandExamplesOptions{}
	for index := 0; index < len(args); index++ {
		arg := strings.TrimSpace(args[index])
		switch arg {
		case "":
			continue
		case "--json":
			options.JSON = true
		case "--command":
			index++
			if index >= len(args) {
				return listCommandExamplesOptions{}, fmt.Errorf("list-command-examples --command requires a value")
			}
			values, err := parseListCommandExampleCommands(args[index])
			if err != nil {
				return listCommandExamplesOptions{}, err
			}
			options.Commands = appendUniqueStrings(options.Commands, values...)
		case "--example":
			index++
			if index >= len(args) {
				return listCommandExamplesOptions{}, fmt.Errorf("list-command-examples --example requires a value")
			}
			values, err := parseListCommandExampleNames(args[index])
			if err != nil {
				return listCommandExamplesOptions{}, err
			}
			options.Examples = appendUniqueStrings(options.Examples, values...)
		case "--format":
			index++
			if index >= len(args) {
				return listCommandExamplesOptions{}, fmt.Errorf("list-command-examples --format requires a value")
			}
			values, err := parseListCommandExampleFormats(args[index])
			if err != nil {
				return listCommandExamplesOptions{}, err
			}
			options.Formats = appendUniqueStrings(options.Formats, values...)
		case "--tag":
			index++
			if index >= len(args) {
				return listCommandExamplesOptions{}, fmt.Errorf("list-command-examples --tag requires a value")
			}
			values, err := parseListCommandExampleTags(args[index])
			if err != nil {
				return listCommandExamplesOptions{}, err
			}
			options.Tags = appendUniqueStrings(options.Tags, values...)
		default:
			return listCommandExamplesOptions{}, fmt.Errorf("unknown list-command-examples flag %q", arg)
		}
	}
	return options, nil
}

func commandExampleManifestReportFromEmbedded() (commandExampleManifestReport, error) {
	return parseCommandExampleManifest(EmbeddedCommandExampleManifest)
}

func parseListCommandExampleCommands(raw string) ([]string, error) {
	return parseListCommandExamplesCSVFlag(raw, "--command", "command")
}

func parseListCommandExampleNames(raw string) ([]string, error) {
	return parseListCommandExamplesCSVFlag(raw, "--example", "example")
}

func parseListCommandExampleFormats(raw string) ([]string, error) {
	return parseListCommandExamplesCSVFlag(raw, "--format", "format")
}

func parseListCommandExampleTags(raw string) ([]string, error) {
	return parseListCommandExamplesCSVFlag(raw, "--tag", "tag")
}

func parseListCommandExamplesCSVFlag(raw string, flag string, label string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("list-command-examples %s requires a non-empty value", flag)
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			return nil, fmt.Errorf("list-command-examples %s contains an empty %s in %q", flag, label, raw)
		}
		values = appendUniqueStrings(values, value)
	}
	return values, nil
}

func parseCommandExampleManifest(raw string) (commandExampleManifestReport, error) {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		trimmed = append(trimmed, line)
	}
	if len(trimmed) < 2 {
		return commandExampleManifestReport{}, fmt.Errorf("command example manifest is incomplete")
	}
	const prefix = "command example manifest "
	if !strings.HasPrefix(trimmed[0], prefix) {
		return commandExampleManifestReport{}, fmt.Errorf("invalid command example manifest header %q", trimmed[0])
	}

	report := commandExampleManifestReport{
		Version: strings.TrimSpace(strings.TrimPrefix(trimmed[0], prefix)),
	}
	for _, line := range trimmed[1:] {
		if strings.HasPrefix(line, "example_count=") {
			var count int
			if _, err := fmt.Sscanf(line, "example_count=%d", &count); err != nil {
				return commandExampleManifestReport{}, fmt.Errorf("parse command example manifest count: %w", err)
			}
			report.ExampleCount = count
			continue
		}
		entry, err := parseCommandExampleManifestEntry(line)
		if err != nil {
			return commandExampleManifestReport{}, err
		}
		report.Examples = append(report.Examples, entry)
	}
	if report.ExampleCount != len(report.Examples) {
		return commandExampleManifestReport{}, fmt.Errorf("command example manifest count mismatch: declared %d actual %d", report.ExampleCount, len(report.Examples))
	}
	return report, nil
}

func parseCommandExampleManifestEntry(line string) (commandExampleManifestEntry, error) {
	fields, err := splitCommandExampleManifestFields(line)
	if err != nil {
		return commandExampleManifestEntry{}, err
	}
	entry := commandExampleManifestEntry{}
	seenKeys := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			return commandExampleManifestEntry{}, fmt.Errorf("invalid command example manifest field %q", field)
		}
		if _, exists := seenKeys[key]; exists {
			return commandExampleManifestEntry{}, fmt.Errorf("duplicate command example manifest field %q in %q", key, line)
		}
		seenKeys[key] = struct{}{}
		if strings.HasPrefix(value, `"`) {
			unquoted, err := strconv.Unquote(value)
			if err != nil {
				return commandExampleManifestEntry{}, fmt.Errorf("invalid quoted command example manifest value %q: %w", value, err)
			}
			value = unquoted
		}
		switch key {
		case "command":
			entry.Command = value
		case "example":
			entry.Name = value
		case "format":
			entry.Format = value
		case "tags":
			if value != "" {
				tags, err := parseCommandExampleManifestTags(value)
				if err != nil {
					return commandExampleManifestEntry{}, err
				}
				entry.Tags = appendUniqueStrings(entry.Tags, tags...)
			}
		case "summary":
			entry.Summary = value
		case "path":
			entry.RelativePath = value
		default:
			return commandExampleManifestEntry{}, fmt.Errorf("unknown command example manifest field %q", key)
		}
	}
	if entry.Command == "" || entry.Name == "" || entry.Format == "" || entry.RelativePath == "" {
		return commandExampleManifestEntry{}, fmt.Errorf("incomplete command example manifest entry %q", line)
	}
	if len(entry.Tags) == 0 {
		return commandExampleManifestEntry{}, fmt.Errorf("command example manifest entry %q is missing tags", line)
	}
	if strings.TrimSpace(entry.Summary) == "" {
		return commandExampleManifestEntry{}, fmt.Errorf("command example manifest entry %q is missing summary", line)
	}
	return entry, nil
}

func parseCommandExampleManifestTags(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" {
			return nil, fmt.Errorf("invalid command example manifest tags value %q", raw)
		}
		tags = appendUniqueStrings(tags, tag)
	}
	return tags, nil
}

func splitCommandExampleManifestFields(line string) ([]string, error) {
	fields := make([]string, 0, 8)
	var current strings.Builder
	inQuotes := false
	escaped := false
	for _, r := range line {
		switch {
		case inQuotes:
			current.WriteRune(r)
			switch {
			case escaped:
				escaped = false
			case r == '\\':
				escaped = true
			case r == '"':
				inQuotes = false
			}
		case r == '"':
			inQuotes = true
			current.WriteRune(r)
		case r == ' ' || r == '\t':
			if current.Len() == 0 {
				continue
			}
			fields = append(fields, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if inQuotes {
		return nil, fmt.Errorf("unterminated quoted command example manifest field in %q", line)
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	return fields, nil
}

func formatCommandExampleManifestJSON(report commandExampleManifestReport) (string, error) {
	return marshalIndented(report)
}

func formatCommandExampleManifest(report commandExampleManifestReport) string {
	lines := make([]string, 0, len(report.Examples)+2)
	lines = append(lines, "command example manifest "+report.Version)
	for _, entry := range report.Examples {
		lines = append(lines, fmt.Sprintf(
			"command=%s example=%s format=%s tags=%s summary=%s path=%s",
			entry.Command,
			entry.Name,
			entry.Format,
			strings.Join(entry.Tags, ","),
			strconv.Quote(entry.Summary),
			entry.RelativePath,
		))
	}
	lines = append(lines, fmt.Sprintf("example_count=%d", report.ExampleCount))
	return strings.Join(lines, "\n") + "\n"
}

func buildCommandExampleManifestReport(entries []commandExampleManifestEntry) commandExampleManifestReport {
	return commandExampleManifestReport{
		Version:      "v1",
		ExampleCount: len(entries),
		Examples:     entries,
	}
}

func commandExampleManifestEntriesForReport(entries []commandExampleManifestEntry) []commandExampleManifestEntry {
	reportEntries := make([]commandExampleManifestEntry, 0, len(entries))
	for _, entry := range entries {
		reportEntries = append(reportEntries, commandExampleManifestEntry{
			Command:      entry.Command,
			Name:         entry.Name,
			RelativePath: path.Join(commandExampleDirName, entry.RelativePath),
			Format:       entry.Format,
			Tags:         slices.Clone(entry.Tags),
			Summary:      entry.Summary,
		})
	}
	return reportEntries
}

type commandExampleFilterSet struct {
	commands map[string]struct{}
	examples map[string]struct{}
	formats  map[string]struct{}
	tags     map[string]struct{}
}

type commandExampleSeenValues struct {
	commands map[string]struct{}
	examples map[string]struct{}
	formats  map[string]struct{}
	tags     map[string]struct{}
}

func filterCommandExampleManifestReport(report commandExampleManifestReport, commands []string, examples []string, formats []string, tags []string) (commandExampleManifestReport, error) {
	if len(commands) == 0 && len(examples) == 0 && len(formats) == 0 && len(tags) == 0 {
		return report, nil
	}

	filters := newCommandExampleFilterSet(commands, examples, formats, tags)
	seen := newCommandExampleSeenValues(len(report.Examples))
	filtered := make([]commandExampleManifestEntry, 0, len(report.Examples))
	for _, entry := range report.Examples {
		seen.record(entry)
		if !filters.matches(entry) {
			continue
		}
		filtered = append(filtered, entry)
	}

	if err := seen.validateRequested("command", commands, seen.commands); err != nil {
		return commandExampleManifestReport{}, err
	}
	if err := seen.validateRequested("example", examples, seen.examples); err != nil {
		return commandExampleManifestReport{}, err
	}
	if err := seen.validateRequested("format", formats, seen.formats); err != nil {
		return commandExampleManifestReport{}, err
	}
	if err := seen.validateRequested("tag", tags, seen.tags); err != nil {
		return commandExampleManifestReport{}, err
	}

	return commandExampleManifestReport{
		Version:      report.Version,
		ExampleCount: len(filtered),
		Examples:     filtered,
	}, nil
}

func newCommandExampleFilterSet(commands []string, examples []string, formats []string, tags []string) commandExampleFilterSet {
	return commandExampleFilterSet{
		commands: buildCommandExampleAllowedSet(commands),
		examples: buildCommandExampleAllowedSet(examples),
		formats:  buildCommandExampleAllowedSet(formats),
		tags:     buildCommandExampleAllowedSet(tags),
	}
}

func buildCommandExampleAllowedSet(values []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(values))
	for _, value := range values {
		allowed[strings.TrimSpace(value)] = struct{}{}
	}
	return allowed
}

func newCommandExampleSeenValues(capacity int) commandExampleSeenValues {
	return commandExampleSeenValues{
		commands: make(map[string]struct{}, capacity),
		examples: make(map[string]struct{}, capacity),
		formats:  make(map[string]struct{}, capacity),
		tags:     make(map[string]struct{}, capacity),
	}
}

func (f commandExampleFilterSet) matches(entry commandExampleManifestEntry) bool {
	return matchesAllowedValue(f.commands, entry.Command) &&
		matchesAllowedValue(f.examples, entry.Name) &&
		matchesAllowedValue(f.formats, entry.Format) &&
		matchesAllowedTag(f.tags, entry.Tags)
}

func matchesAllowedValue(allowed map[string]struct{}, value string) bool {
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[value]
	return ok
}

func matchesAllowedTag(allowed map[string]struct{}, tags []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, tag := range tags {
		if _, ok := allowed[tag]; ok {
			return true
		}
	}
	return false
}

func (s commandExampleSeenValues) record(entry commandExampleManifestEntry) {
	s.commands[entry.Command] = struct{}{}
	s.examples[entry.Name] = struct{}{}
	s.formats[entry.Format] = struct{}{}
	for _, tag := range entry.Tags {
		s.tags[tag] = struct{}{}
	}
}

func (s commandExampleSeenValues) validateRequested(label string, requested []string, seen map[string]struct{}) error {
	unknown := unknownRequestedCommandExampleValues(requested, seen)
	if len(unknown) == 0 {
		return nil
	}
	return fmt.Errorf("unknown list-command-examples %s filter %q", label, strings.Join(unknown, ","))
}

func unknownRequestedCommandExampleValues(requested []string, seen map[string]struct{}) []string {
	unknown := make([]string, 0)
	for _, value := range requested {
		if _, ok := seen[value]; ok || slices.Contains(unknown, value) {
			continue
		}
		unknown = append(unknown, value)
	}
	return unknown
}

func appendUniqueStrings(existing []string, values ...string) []string {
	result := existing
	for _, value := range values {
		if !slices.Contains(result, value) {
			result = append(result, value)
		}
	}
	return result
}
