package app

import (
	"slices"
	"strings"
	"testing"
)

func TestParseListCommandExamplesOptionsSupportsRepeatedAndCSVTags(t *testing.T) {
	options, err := parseListCommandExamplesOptions([]string{
		"--tag", "audit-only,target-profile",
		"--tag", "audit-only",
		"--format", "json",
	})
	if err != nil {
		t.Fatalf("parseListCommandExamplesOptions: %v", err)
	}

	if got, want := len(options.Tags), 2; got != want {
		t.Fatalf("tag count mismatch: got %d want %d", got, want)
	}
	if !slices.Equal(options.Tags, []string{"audit-only", "target-profile"}) {
		t.Fatalf("unexpected tags: %+v", options.Tags)
	}
	if !slices.Equal(options.Formats, []string{"json"}) {
		t.Fatalf("unexpected formats: %+v", options.Formats)
	}
}

func TestParseCommandExampleManifestEntrySupportsQuotedSummary(t *testing.T) {
	line := `command=follow-imports example=audit-only-single-text format=text tags="audit-only, single-input, audit-only" summary="Audit-only \"follow\" report at C:\\Ops\\follow." path=testdata/follow-imports-audit-only-single.txt`

	entry, err := parseCommandExampleManifestEntry(line)
	if err != nil {
		t.Fatalf("parseCommandExampleManifestEntry: %v", err)
	}

	if got, want := entry.Command, commandFollowImports; got != want {
		t.Fatalf("command mismatch: got %q want %q", got, want)
	}
	if !slices.Equal(entry.Tags, []string{"audit-only", "single-input"}) {
		t.Fatalf("unexpected tags: %+v", entry.Tags)
	}
	if got, want := entry.Summary, `Audit-only "follow" report at C:\Ops\follow.`; got != want {
		t.Fatalf("summary mismatch: got %q want %q", got, want)
	}
}

func TestParseCommandExampleManifestEntryRejectsEmptyTag(t *testing.T) {
	line := `command=follow-imports example=audit-only-single-text format=text tags="audit-only,,single-input" summary="Audit-only follow report." path=testdata/follow-imports-audit-only-single.txt`

	_, err := parseCommandExampleManifestEntry(line)
	if err == nil {
		t.Fatal("expected invalid tags error")
	}
	if got, want := err.Error(), `invalid command example manifest tags value "audit-only,,single-input"`; got != want {
		t.Fatalf("error mismatch: got %q want %q", got, want)
	}
}

func TestParseCommandExampleManifestEntryRejectsMissingTags(t *testing.T) {
	line := `command=follow-imports example=audit-only-single-text format=text summary="Audit-only follow report." path=testdata/follow-imports-audit-only-single.txt`

	_, err := parseCommandExampleManifestEntry(line)
	if err == nil {
		t.Fatal("expected missing tags error")
	}
	for _, fragment := range []string{"missing tags", "follow-imports", "audit-only-single-text"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("error %q missing fragment %q", err.Error(), fragment)
		}
	}
}

func TestParseCommandExampleManifestEntryRejectsMissingSummary(t *testing.T) {
	line := `command=follow-imports example=audit-only-single-text format=text tags="audit-only,single-input" path=testdata/follow-imports-audit-only-single.txt`

	_, err := parseCommandExampleManifestEntry(line)
	if err == nil {
		t.Fatal("expected missing summary error")
	}
	for _, fragment := range []string{"missing summary", "follow-imports", "audit-only-single-text"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("error %q missing fragment %q", err.Error(), fragment)
		}
	}
}

func TestParseCommandExampleManifestRejectsUnterminatedQuotedSummary(t *testing.T) {
	line := `command=follow-imports example=audit-only-single-text format=text tags=audit-only,single-input summary="broken path=testdata/follow-imports-audit-only-single.txt`

	_, err := parseCommandExampleManifestEntry(line)
	if err == nil {
		t.Fatal("expected unterminated quote error")
	}
	if got := err.Error(); got == "" || got == line {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFormatCommandExampleManifestRoundTripsQuotedSummary(t *testing.T) {
	report := commandExampleManifestReport{
		Version:      "v1",
		ExampleCount: 1,
		Examples: []commandExampleManifestEntry{{
			Command:      "follow-imports",
			Name:         "audit-only-single-text",
			RelativePath: "testdata/follow-imports-audit-only-single.txt",
			Format:       "text",
			Tags:         []string{"audit-only", "single-input"},
			Summary:      `Audit-only "follow" report at C:\Ops\follow.`,
		}},
	}

	formatted := formatCommandExampleManifest(report)
	parsed, err := parseCommandExampleManifest(formatted)
	if err != nil {
		t.Fatalf("parseCommandExampleManifest: %v\n%s", err, formatted)
	}

	if got, want := parsed.Version, report.Version; got != want {
		t.Fatalf("version mismatch: got %q want %q", got, want)
	}
	if got, want := parsed.ExampleCount, report.ExampleCount; got != want {
		t.Fatalf("example_count mismatch: got %d want %d", got, want)
	}
	if got, want := len(parsed.Examples), 1; got != want {
		t.Fatalf("examples len mismatch: got %d want %d", got, want)
	}
	if got, want := parsed.Examples[0].Summary, report.Examples[0].Summary; got != want {
		t.Fatalf("summary mismatch: got %q want %q", got, want)
	}
	if !slices.Equal(parsed.Examples[0].Tags, report.Examples[0].Tags) {
		t.Fatalf("tags mismatch: got %+v want %+v", parsed.Examples[0].Tags, report.Examples[0].Tags)
	}
}
