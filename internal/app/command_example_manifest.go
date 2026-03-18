package app

import (
	"fmt"
	"strings"

	_ "embed"
)

// EmbeddedCommandExampleManifest contains the checked-in import/follow example catalog.
//
//go:embed testdata/command-example-manifest.txt
var EmbeddedCommandExampleManifest string

type listCommandExamplesOptions struct {
	JSON bool
}

type commandExampleManifestEntry struct {
	Command      string `json:"command"`
	Name         string `json:"name"`
	RelativePath string `json:"path"`
	Format       string `json:"format"`
}

type commandExampleManifestReport struct {
	Version      string                        `json:"version"`
	ExampleCount int                           `json:"example_count"`
	Examples     []commandExampleManifestEntry `json:"examples"`
}

func parseListCommandExamplesOptions(args []string) (listCommandExamplesOptions, error) {
	options := listCommandExamplesOptions{}
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "":
			continue
		case "--json":
			options.JSON = true
		default:
			return listCommandExamplesOptions{}, fmt.Errorf("unknown list-command-examples flag %q", arg)
		}
	}
	return options, nil
}

func commandExampleManifestReportFromEmbedded() (commandExampleManifestReport, error) {
	return parseCommandExampleManifest(EmbeddedCommandExampleManifest)
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
	fields := strings.Fields(line)
	entry := commandExampleManifestEntry{}
	for _, field := range fields {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			return commandExampleManifestEntry{}, fmt.Errorf("invalid command example manifest field %q", field)
		}
		switch key {
		case "command":
			entry.Command = value
		case "example":
			entry.Name = value
		case "format":
			entry.Format = value
		case "path":
			entry.RelativePath = value
		default:
			return commandExampleManifestEntry{}, fmt.Errorf("unknown command example manifest field %q", key)
		}
	}
	if entry.Command == "" || entry.Name == "" || entry.Format == "" || entry.RelativePath == "" {
		return commandExampleManifestEntry{}, fmt.Errorf("incomplete command example manifest entry %q", line)
	}
	return entry, nil
}

func formatCommandExampleManifestJSON(report commandExampleManifestReport) (string, error) {
	return marshalIndented(report)
}
