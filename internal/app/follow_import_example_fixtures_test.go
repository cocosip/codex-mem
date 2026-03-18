package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"codex-mem/internal/domain/common"
)

const commandExampleDirName = "testdata"
const commandExampleManifestName = "command-example-manifest.txt"

type commandExampleFixture[T any] struct {
	Name         string
	RelativePath string
	JSON         bool
	Report       T
}

type cleanupFollowImportsExampleFixture = commandExampleFixture[cleanupFollowImportsReport]
type auditFollowImportsExampleFixture = commandExampleFixture[auditFollowImportsReport]

func normalizeFollowImportsExampleName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseFollowImportsExampleNames(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New(`invalid value for "--refresh-examples": empty`)
	}
	parts := strings.Split(raw, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		name := normalizeFollowImportsExampleName(part)
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

func selectCommandExampleFixtures[T any](fixtures []commandExampleFixture[T], names []string, command string) ([]commandExampleFixture[T], error) {
	if len(names) == 0 {
		return fixtures, nil
	}

	byName := make(map[string]commandExampleFixture[T], len(fixtures))
	for _, fixture := range fixtures {
		byName[normalizeFollowImportsExampleName(fixture.Name)] = fixture
	}

	selected := make([]commandExampleFixture[T], 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		normalized := normalizeFollowImportsExampleName(name)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		fixture, ok := byName[normalized]
		if !ok {
			return nil, fmt.Errorf("unknown %s example %q", command, name)
		}
		selected = append(selected, fixture)
		seen[normalized] = struct{}{}
	}
	return selected, nil
}

func writeCommandExampleFixtures[T any](baseDir string, names []string, command string, fixtures []commandExampleFixture[T], render func(T, bool) ([]byte, error)) ([]string, error) {
	selected, err := selectCommandExampleFixtures(fixtures, names, command)
	if err != nil {
		return nil, err
	}
	writtenPaths := make([]string, 0, len(selected))
	for _, fixture := range selected {
		body, err := render(fixture.Report, fixture.JSON)
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

func listCommandExamples[T any](fixtures []commandExampleFixture[T], w io.Writer) error {
	for _, fixture := range fixtures {
		format := "text"
		if fixture.JSON {
			format = "json"
		}
		if _, err := fmt.Fprintf(w, "example=%s path=%s format=%s\n", fixture.Name, filepath.Join(commandExampleDirName, fixture.RelativePath), format); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "example_count=%d\n", len(fixtures))
	return err
}

func commandExampleManifestEntriesFor[T any](command string, fixtures []commandExampleFixture[T]) []commandExampleManifestEntry {
	entries := make([]commandExampleManifestEntry, 0, len(fixtures))
	for _, fixture := range fixtures {
		format := "text"
		if fixture.JSON {
			format = "json"
		}
		entries = append(entries, commandExampleManifestEntry{
			Command:      command,
			Name:         fixture.Name,
			RelativePath: fixture.RelativePath,
			Format:       format,
		})
	}
	return entries
}

func commandExampleManifestEntries() []commandExampleManifestEntry {
	entries := make([]commandExampleManifestEntry, 0,
		len(ingestImportsExampleFixtures())+
			len(followImportsCommandExampleFixtures())+
			len(cleanupFollowImportsExampleFixtures())+
			len(auditFollowImportsExampleFixtures()))
	entries = append(entries, commandExampleManifestEntriesFor("ingest-imports", ingestImportsExampleFixtures())...)
	entries = append(entries, commandExampleManifestEntriesFor("follow-imports", followImportsCommandExampleFixtures())...)
	entries = append(entries, commandExampleManifestEntriesFor("cleanup-follow-imports", cleanupFollowImportsExampleFixtures())...)
	entries = append(entries, commandExampleManifestEntriesFor("audit-follow-imports", auditFollowImportsExampleFixtures())...)
	return entries
}

func renderCommandExampleManifest(entries []commandExampleManifestEntry) []byte {
	lines := make([]string, 0, len(entries)+2)
	lines = append(lines, "command example manifest v1")
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf(
			"command=%s example=%s format=%s path=%s",
			entry.Command,
			entry.Name,
			entry.Format,
			path.Join(commandExampleDirName, entry.RelativePath),
		))
	}
	lines = append(lines, fmt.Sprintf("example_count=%d", len(entries)))
	return []byte(strings.Join(lines, "\n") + "\n")
}

func writeCommandExampleManifest(baseDir string) (string, error) {
	body := renderCommandExampleManifest(commandExampleManifestEntries())
	path := filepath.Join(baseDir, commandExampleManifestName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func cleanupFollowImportsExampleFixtures() []cleanupFollowImportsExampleFixture {
	return []cleanupFollowImportsExampleFixture{
		{
			Name:         "daily-dry-run-text",
			RelativePath: "cleanup-follow-imports-daily-dry-run.txt",
			JSON:         false,
			Report: cleanupFollowImportsReport{
				DryRun:           true,
				FailIfMatched:    false,
				MatchFound:       true,
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
				FailIfMatched:    false,
				MatchFound:       true,
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
		{
			Name:         "target-profile-all-text",
			RelativePath: "cleanup-follow-imports-target-profile-all.txt",
			JSON:         false,
			Report: cleanupFollowImportsReport{
				DryRun:           false,
				FailIfMatched:    true,
				MatchFound:       true,
				TargetProfile:    followImportsTargetProfileAll,
				RetentionProfile: cleanupFollowImportsRetentionProfileReset,
				OlderThanSeconds: 0,
				Status:           "ok",
				StateFiles: cleanupFollowImportsPathSummary{
					Requested:    1,
					Matched:      1,
					Removed:      1,
					Missing:      0,
					MatchedPaths: []string{`D:\Ops\follow\events.offset.json`},
					RemovedPaths: []string{`D:\Ops\follow\events.offset.json`},
				},
				FailedOutputs: cleanupFollowImportsPatternSummary{
					Bases:        1,
					Matched:      2,
					Removed:      2,
					BasePaths:    []string{`D:\Ops\follow\failed\failed.jsonl`},
					MatchedPaths: []string{`D:\Ops\follow\failed\failed.0-42.jsonl`, `D:\Ops\follow\failed\failed.43-84.jsonl`},
					RemovedPaths: []string{`D:\Ops\follow\failed\failed.0-42.jsonl`, `D:\Ops\follow\failed\failed.43-84.jsonl`},
				},
				FailedManifests: cleanupFollowImportsPatternSummary{
					Bases:        1,
					Matched:      2,
					Removed:      2,
					BasePaths:    []string{`D:\Ops\follow\failed\failed.json`},
					MatchedPaths: []string{`D:\Ops\follow\failed\failed.0-42.json`, `D:\Ops\follow\failed\failed.43-84.json`},
					RemovedPaths: []string{`D:\Ops\follow\failed\failed.0-42.json`, `D:\Ops\follow\failed\failed.43-84.json`},
				},
				FollowHealth: cleanupFollowImportsFollowHealthView{
					File:        `D:\Ops\follow\logs\follow-imports.health.json`,
					WouldPrune:  false,
					Pruned:      true,
					PruneReason: "stale",
				},
			},
		},
	}
}

func auditFollowImportsExampleTime(year int, month time.Month, day int, hour int, minute int, second int) *time.Time {
	value := time.Date(year, month, day, hour, minute, second, 0, time.UTC)
	return &value
}

func auditFollowImportsExampleFixtures() []auditFollowImportsExampleFixture {
	return []auditFollowImportsExampleFixture{
		{
			Name:         "daily-audit-text",
			RelativePath: "audit-follow-imports-daily-audit.txt",
			JSON:         false,
			Report: auditFollowImportsReport{
				FailIfMatched:    false,
				MatchFound:       true,
				RetentionProfile: cleanupFollowImportsRetentionProfileDaily,
				OlderThanSeconds: 86400,
				Status:           "ok",
				StateFiles: auditFollowImportsPathSummary{
					Requested:         2,
					Matched:           1,
					Missing:           0,
					SkippedByPattern:  0,
					SkippedByAge:      1,
					MatchedPaths:      []string{`D:\Ops\follow\events-a.offset.json`},
					SkippedByAgePaths: []string{`D:\Ops\follow\events-b.offset.json`},
				},
				FailedOutputs: auditFollowImportsPatternSummary{
					Bases:             1,
					Matched:           1,
					SkippedByPattern:  0,
					SkippedByAge:      1,
					BasePaths:         []string{`D:\Ops\follow\failed\failed.jsonl`},
					MatchedPaths:      []string{`D:\Ops\follow\failed\failed.events-a.0-42.jsonl`},
					SkippedByAgePaths: []string{`D:\Ops\follow\failed\failed.events-b.43-84.jsonl`},
				},
				FailedManifests: auditFollowImportsPatternSummary{
					Bases:             1,
					Matched:           1,
					SkippedByPattern:  0,
					SkippedByAge:      1,
					BasePaths:         []string{`D:\Ops\follow\failed\failed.json`},
					MatchedPaths:      []string{`D:\Ops\follow\failed\failed.events-a.0-42.json`},
					SkippedByAgePaths: []string{`D:\Ops\follow\failed\failed.events-b.43-84.json`},
				},
				FollowHealth: auditFollowImportsHealthView{
					File:                  `D:\Ops\follow\logs\follow-imports.health.json`,
					Present:               true,
					LastUpdatedAt:         auditFollowImportsExampleTime(2026, time.March, 16, 8, 15, 0),
					Status:                "partial",
					Source:                "watcher_import",
					InputCount:            2,
					Continuous:            true,
					PollIntervalSeconds:   5,
					SnapshotAgeSeconds:    180,
					Stale:                 true,
					RequestedWatchMode:    "auto",
					ActiveWatchMode:       "poll",
					WatchFallbacks:        1,
					WatchTransitions:      3,
					LastFallbackReason:    "watcher_error",
					WatchPollCatchups:     4,
					WatchPollCatchupBytes: 256,
					Warnings: []common.Warning{
						{
							Code:    common.WarnFollowImportsPollCatchup,
							Message: "notify mode required poll catchup 4 times and 256 bytes so far",
						},
						{
							Code:    common.WarnFollowImportsHealthStale,
							Message: "follow-imports health snapshot is stale at 3m0s",
						},
					},
				},
			},
		},
		{
			Name:         "filtered-audit-json",
			RelativePath: "audit-follow-imports-filtered-audit.json",
			JSON:         true,
			Report: auditFollowImportsReport{
				FailIfMatched:    true,
				MatchFound:       true,
				RetentionProfile: cleanupFollowImportsRetentionProfileReset,
				OlderThanSeconds: 0,
				IncludePatterns:  []string{"*events-a*", "*.offset.json"},
				ExcludePatterns:  []string{"*.43-84.*"},
				Status:           "ok",
				StateFiles: auditFollowImportsPathSummary{
					Requested:             2,
					Matched:               1,
					Missing:               0,
					SkippedByPattern:      1,
					MatchedPaths:          []string{`D:\Ops\follow\events-a.offset.json`},
					SkippedByPatternPaths: []string{`D:\Ops\follow\events-b.offset.json`},
				},
				FailedOutputs: auditFollowImportsPatternSummary{
					Bases:                 1,
					Matched:               1,
					SkippedByPattern:      1,
					BasePaths:             []string{`D:\Ops\follow\failed\failed.jsonl`},
					MatchedPaths:          []string{`D:\Ops\follow\failed\failed.events-a.0-42.jsonl`},
					SkippedByPatternPaths: []string{`D:\Ops\follow\failed\failed.events-b.43-84.jsonl`},
				},
				FailedManifests: auditFollowImportsPatternSummary{
					Bases:                 1,
					Matched:               1,
					SkippedByPattern:      1,
					BasePaths:             []string{`D:\Ops\follow\failed\failed.json`},
					MatchedPaths:          []string{`D:\Ops\follow\failed\failed.events-a.0-42.json`},
					SkippedByPatternPaths: []string{`D:\Ops\follow\failed\failed.events-b.43-84.json`},
				},
				FollowHealth: auditFollowImportsHealthView{
					File:                `D:\Ops\follow\logs\follow-imports.health.json`,
					Present:             true,
					LastUpdatedAt:       auditFollowImportsExampleTime(2026, time.March, 16, 8, 17, 30),
					Status:              "ok",
					Source:              "watcher_import",
					InputCount:          1,
					Continuous:          true,
					PollIntervalSeconds: 5,
					SnapshotAgeSeconds:  30,
					Stale:               false,
					RequestedWatchMode:  "auto",
					ActiveWatchMode:     "notify",
				},
			},
		},
		{
			Name:         "target-profile-retry-json",
			RelativePath: "audit-follow-imports-target-profile-retry.json",
			JSON:         true,
			Report: auditFollowImportsReport{
				FailIfMatched:    true,
				MatchFound:       true,
				TargetProfile:    followImportsTargetProfileRetry,
				RetentionProfile: cleanupFollowImportsRetentionProfileDaily,
				OlderThanSeconds: 86400,
				Status:           "ok",
				StateFiles:       auditFollowImportsPathSummary{},
				FailedOutputs: auditFollowImportsPatternSummary{
					Bases:             1,
					Matched:           1,
					SkippedByAge:      1,
					BasePaths:         []string{`D:\Ops\follow\failed\failed.jsonl`},
					MatchedPaths:      []string{`D:\Ops\follow\failed\failed.0-42.jsonl`},
					SkippedByAgePaths: []string{`D:\Ops\follow\failed\failed.43-84.jsonl`},
				},
				FailedManifests: auditFollowImportsPatternSummary{
					Bases:             1,
					Matched:           1,
					SkippedByAge:      1,
					BasePaths:         []string{`D:\Ops\follow\failed\failed.json`},
					MatchedPaths:      []string{`D:\Ops\follow\failed\failed.0-42.json`},
					SkippedByAgePaths: []string{`D:\Ops\follow\failed\failed.43-84.json`},
				},
				FollowHealth: auditFollowImportsHealthView{
					File:    `D:\Ops\follow\logs\follow-imports.health.json`,
					Present: false,
				},
			},
		},
	}
}

func selectCleanupFollowImportsExampleFixtures(names []string) ([]cleanupFollowImportsExampleFixture, error) {
	return selectCommandExampleFixtures(cleanupFollowImportsExampleFixtures(), names, "cleanup-follow-imports")
}

func selectAuditFollowImportsExampleFixtures(names []string) ([]auditFollowImportsExampleFixture, error) {
	return selectCommandExampleFixtures(auditFollowImportsExampleFixtures(), names, "audit-follow-imports")
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

func renderAuditFollowImportsExample(report auditFollowImportsReport, jsonOutput bool) ([]byte, error) {
	if jsonOutput {
		body, err := marshalIndented(report)
		if err != nil {
			return nil, err
		}
		return []byte(body), nil
	}
	return []byte(formatAuditFollowImportsReport(report)), nil
}

func writeCleanupFollowImportsExampleFixtures(baseDir string, names []string) ([]string, error) {
	return writeCommandExampleFixtures(baseDir, names, "cleanup-follow-imports", cleanupFollowImportsExampleFixtures(), renderCleanupFollowImportsExample)
}

func writeAuditFollowImportsExampleFixtures(baseDir string, names []string) ([]string, error) {
	return writeCommandExampleFixtures(baseDir, names, "audit-follow-imports", auditFollowImportsExampleFixtures(), renderAuditFollowImportsExample)
}

func listCleanupFollowImportsExamples(w io.Writer) error {
	return listCommandExamples(cleanupFollowImportsExampleFixtures(), w)
}

func listAuditFollowImportsExamples(w io.Writer) error {
	return listCommandExamples(auditFollowImportsExampleFixtures(), w)
}
