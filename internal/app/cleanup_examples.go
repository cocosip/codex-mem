package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const cleanupFollowImportsExampleDirName = "testdata"

type cleanupFollowImportsExampleFixture struct {
	Name         string
	RelativePath string
	JSON         bool
	Report       cleanupFollowImportsReport
}

func cleanupFollowImportsExampleFixtures() []cleanupFollowImportsExampleFixture {
	return []cleanupFollowImportsExampleFixture{
		{
			Name:         "daily-dry-run-text",
			RelativePath: "cleanup-follow-imports-daily-dry-run.txt",
			JSON:         false,
			Report: cleanupFollowImportsReport{
				DryRun:           true,
				RetentionProfile: cleanupFollowImportsRetentionProfileDaily,
				OlderThanSeconds: 86400,
				Status:           "ok",
				StateFiles: cleanupFollowImportsPathSummary{
					Requested:         2,
					Matched:           1,
					Removed:           0,
					Missing:           0,
					SkippedByPattern:  0,
					SkippedByAge:      1,
					MatchedPaths:      []string{`D:\Ops\follow\events-a.offset.json`},
					SkippedByAgePaths: []string{`D:\Ops\follow\events-b.offset.json`},
				},
				FailedOutputs: cleanupFollowImportsPatternSummary{
					Bases:             1,
					Matched:           1,
					Removed:           0,
					SkippedByPattern:  0,
					SkippedByAge:      1,
					BasePaths:         []string{`D:\Ops\follow\failed\failed.jsonl`},
					MatchedPaths:      []string{`D:\Ops\follow\failed\failed.events-a.0-42.jsonl`},
					SkippedByAgePaths: []string{`D:\Ops\follow\failed\failed.events-b.43-84.jsonl`},
				},
				FailedManifests: cleanupFollowImportsPatternSummary{
					Bases:             1,
					Matched:           1,
					Removed:           0,
					SkippedByPattern:  0,
					SkippedByAge:      1,
					BasePaths:         []string{`D:\Ops\follow\failed\failed.json`},
					MatchedPaths:      []string{`D:\Ops\follow\failed\failed.events-a.0-42.json`},
					SkippedByAgePaths: []string{`D:\Ops\follow\failed\failed.events-b.43-84.json`},
				},
				FollowHealth: cleanupFollowImportsFollowHealthView{
					File:        `D:\Ops\follow\logs\follow-imports.health.json`,
					WouldPrune:  true,
					Pruned:      false,
					PruneReason: "stale",
				},
			},
		},
		{
			Name:         "filtered-cleanup-json",
			RelativePath: "cleanup-follow-imports-filtered-cleanup.json",
			JSON:         true,
			Report: cleanupFollowImportsReport{
				DryRun:           false,
				RetentionProfile: cleanupFollowImportsRetentionProfileReset,
				OlderThanSeconds: 0,
				IncludePatterns:  []string{"*events-a*", "*.offset.json"},
				ExcludePatterns:  []string{"*.43-84.*"},
				Status:           "ok",
				StateFiles: cleanupFollowImportsPathSummary{
					Requested:             2,
					Matched:               1,
					Removed:               1,
					Missing:               0,
					SkippedByPattern:      1,
					MatchedPaths:          []string{`D:\Ops\follow\events-a.offset.json`},
					RemovedPaths:          []string{`D:\Ops\follow\events-a.offset.json`},
					SkippedByPatternPaths: []string{`D:\Ops\follow\events-b.offset.json`},
				},
				FailedOutputs: cleanupFollowImportsPatternSummary{
					Bases:                 1,
					Matched:               1,
					Removed:               1,
					SkippedByPattern:      1,
					BasePaths:             []string{`D:\Ops\follow\failed\failed.jsonl`},
					MatchedPaths:          []string{`D:\Ops\follow\failed\failed.events-a.0-42.jsonl`},
					RemovedPaths:          []string{`D:\Ops\follow\failed\failed.events-a.0-42.jsonl`},
					SkippedByPatternPaths: []string{`D:\Ops\follow\failed\failed.events-b.43-84.jsonl`},
				},
				FailedManifests: cleanupFollowImportsPatternSummary{
					Bases:                 1,
					Matched:               1,
					Removed:               1,
					SkippedByPattern:      1,
					BasePaths:             []string{`D:\Ops\follow\failed\failed.json`},
					MatchedPaths:          []string{`D:\Ops\follow\failed\failed.events-a.0-42.json`},
					RemovedPaths:          []string{`D:\Ops\follow\failed\failed.events-a.0-42.json`},
					SkippedByPatternPaths: []string{`D:\Ops\follow\failed\failed.events-b.43-84.json`},
				},
				FollowHealth: cleanupFollowImportsFollowHealthView{
					File:       `D:\Ops\follow\logs\follow-imports.health.json`,
					WouldPrune: false,
					Pruned:     false,
				},
			},
		},
	}
}

func cleanupFollowImportsExampleBaseDir(cwd string) (string, error) {
	base := strings.TrimSpace(cwd)
	if base == "" {
		var err error
		base, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve working directory: %w", err)
		}
	}
	resolved, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	return filepath.Join(resolved, "internal", "app", cleanupFollowImportsExampleDirName), nil
}

func selectCleanupFollowImportsExampleFixtures(names []string) ([]cleanupFollowImportsExampleFixture, error) {
	all := cleanupFollowImportsExampleFixtures()
	if len(names) == 0 {
		return all, nil
	}

	byName := make(map[string]cleanupFollowImportsExampleFixture, len(all))
	for _, fixture := range all {
		byName[normalizeCleanupFollowImportsExampleName(fixture.Name)] = fixture
	}

	selected := make([]cleanupFollowImportsExampleFixture, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		normalized := normalizeCleanupFollowImportsExampleName(name)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		fixture, ok := byName[normalized]
		if !ok {
			return nil, fmt.Errorf("unknown cleanup-follow-imports example %q", name)
		}
		selected = append(selected, fixture)
		seen[normalized] = struct{}{}
	}
	return selected, nil
}

func normalizeCleanupFollowImportsExampleName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseCleanupFollowImportsExampleNames(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New(`invalid value for "--refresh-examples": empty`)
	}
	parts := strings.Split(raw, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		name := normalizeCleanupFollowImportsExampleName(part)
		if name == "" {
			return nil, fmt.Errorf(`invalid value for "--refresh-examples": %q`, raw)
		}
		if slices.Contains(names, name) {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

func renderCleanupFollowImportsExample(report cleanupFollowImportsReport, jsonOutput bool) ([]byte, error) {
	if jsonOutput {
		body, err := marshalIndented(report)
		if err != nil {
			return nil, err
		}
		return []byte(body), nil
	}
	return []byte(formatCleanupFollowImportsReport(report)), nil
}

func writeCleanupFollowImportsExampleFixtures(baseDir string, names []string) ([]string, error) {
	fixtures, err := selectCleanupFollowImportsExampleFixtures(names)
	if err != nil {
		return nil, err
	}
	writtenPaths := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		body, err := renderCleanupFollowImportsExample(fixture.Report, fixture.JSON)
		if err != nil {
			return nil, err
		}
		path := filepath.Join(baseDir, fixture.RelativePath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return nil, err
		}
		writtenPaths = append(writtenPaths, path)
	}
	return writtenPaths, nil
}

func refreshCleanupFollowImportsExamples(baseDir string, names []string, w io.Writer) error {
	writtenPaths, err := writeCleanupFollowImportsExampleFixtures(baseDir, names)
	if err != nil {
		return err
	}
	for _, path := range writtenPaths {
		if _, err := fmt.Fprintf(w, "refreshed_example=%s\n", path); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(w, "refreshed_examples=%d\n", len(writtenPaths))
	return err
}

func listCleanupFollowImportsExamples(w io.Writer) error {
	fixtures := cleanupFollowImportsExampleFixtures()
	for _, fixture := range fixtures {
		format := "text"
		if fixture.JSON {
			format = "json"
		}
		if _, err := fmt.Fprintf(w, "example=%s path=%s format=%s\n", fixture.Name, filepath.Join(cleanupFollowImportsExampleDirName, fixture.RelativePath), format); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "example_count=%d\n", len(fixtures))
	return err
}
