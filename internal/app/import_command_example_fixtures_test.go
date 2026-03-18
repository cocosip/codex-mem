package app

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

type ingestImportsExampleFixture = commandExampleFixture[ingestImportsReport]
type followImportsCommandFixture = commandExampleFixture[any]

func ingestImportsExampleFixtures() []ingestImportsExampleFixture {
	exampleScope := scope.Scope{
		SystemID:      "sys_codex_mem",
		SystemName:    "codex-mem",
		ProjectID:     "proj_ingest",
		ProjectName:   "codex-mem",
		WorkspaceID:   "ws_ingest",
		WorkspaceRoot: "D:\\Code\\go\\codex-mem",
		BranchName:    "master",
		ResolvedBy:    "repo_remote",
	}
	exampleSession := session.Session{
		ID:         "sess_20260318_013000",
		Scope:      exampleScope.Ref(),
		Status:     session.StatusActive,
		Task:       "audit imported notes (watcher_import)",
		BranchName: "master",
		StartedAt:  time.Date(2026, 3, 18, 1, 30, 0, 0, time.UTC),
	}
	return []ingestImportsExampleFixture{
		{
			Name:         "audit-only-summary-text",
			RelativePath: "ingest-imports-audit-only-summary.txt",
			JSON:         false,
			Report: ingestImportsReport{
				Status:             "ok",
				Source:             "watcher_import",
				Input:              "D:\\Ops\\imports\\events.jsonl",
				Scope:              exampleScope,
				Session:            exampleSession,
				AuditOnly:          true,
				ContinueOnError:    false,
				Attempted:          3,
				Processed:          3,
				Failed:             0,
				Materialized:       0,
				Suppressed:         2,
				SuppressionReasons: map[string]int{"explicit_memory_exists": 1, "privacy_intent": 1},
				WouldMaterialize:   1,
				LinkedExistingNote: 0,
				NoteDeduplicated:   1,
				ImportDeduplicated: 0,
				Warnings: []common.Warning{
					{Code: common.WarnImportSuppressed, Message: "import was suppressed by privacy policy"},
					{Code: common.WarnImportSuppressed, Message: "import was suppressed because stronger explicit memory already exists"},
				},
			},
		},
		{
			Name:         "audit-only-linked-json",
			RelativePath: "ingest-imports-audit-only-linked.json",
			JSON:         true,
			Report: ingestImportsReport{
				Status:             "ok",
				Source:             "relay_import",
				Input:              "stdin",
				Scope:              exampleScope,
				Session:            exampleSession,
				AuditOnly:          true,
				ContinueOnError:    false,
				Attempted:          3,
				Processed:          3,
				Failed:             0,
				Materialized:       0,
				Suppressed:         1,
				SuppressionReasons: map[string]int{"import_policy": 1},
				WouldMaterialize:   1,
				LinkedExistingNote: 1,
				NoteDeduplicated:   1,
				ImportDeduplicated: 1,
				Warnings: []common.Warning{
					{Code: common.WarnDedupeApplied, Message: "matched an existing imported note and reused it"},
					{Code: common.WarnImportSuppressed, Message: "matched an existing import record and skipped duplicate import"},
				},
				Results: []ingestImportEventResult{
					{Line: 1, ImportID: "import_1", Materialized: false, Suppressed: false, NoteDeduplicated: false, ImportDeduplicated: false},
					{Line: 2, NoteID: "note_existing_import", ImportID: "import_2", Materialized: false, Suppressed: false, NoteDeduplicated: true, ImportDeduplicated: false},
					{Line: 3, ImportID: "import_existing", Materialized: false, Suppressed: true, NoteDeduplicated: false, ImportDeduplicated: true},
				},
			},
		},
	}
}

func followImportsCommandExampleFixtures() []followImportsCommandFixture {
	exampleEvent := followImportsEvent{
		At:                 time.Date(2026, 3, 18, 1, 35, 0, 0, time.UTC),
		Kind:               "watch_fallback",
		RequestedWatchMode: "auto",
		PreviousWatchMode:  "notify",
		ActiveWatchMode:    "poll",
		Reason:             "watcher_unavailable",
		Fallbacks:          1,
		ConsumedInputs:     1,
		ConsumedBytes:      84,
	}
	exampleScope := scope.Scope{
		SystemID:      "sys_codex_mem",
		SystemName:    "codex-mem",
		ProjectID:     "proj_follow",
		ProjectName:   "codex-mem",
		WorkspaceID:   "ws_follow",
		WorkspaceRoot: "D:\\Code\\go\\codex-mem",
		BranchName:    "master",
		ResolvedBy:    "repo_remote",
	}
	return []followImportsCommandFixture{
		{
			Name:         "audit-only-single-text",
			RelativePath: "follow-imports-audit-only-single.txt",
			JSON:         false,
			Report: followImportsReport{
				Status:             "ok",
				Source:             "watcher_import",
				Input:              `D:\Ops\follow\events.jsonl`,
				StateFile:          `D:\Ops\follow\events.offset.json`,
				AuditOnly:          true,
				RequestedWatchMode: "auto",
				ActiveWatchMode:    "poll",
				WatchFallbacks:     1,
				WatchTransitions:   2,
				LastFallbackReason: "watcher_unavailable",
				WatchEventCount:    1,
				WatchEvents:        []followImportsEvent{exampleEvent},
				WatchPollCatchups:  3,
				WatchCatchupBytes:  84,
				Warnings: []common.Warning{
					{Code: common.WarnFollowImportsPollCatchup, Message: "notify mode repeatedly relied on poll catchup; treat watcher health as degraded"},
				},
				Offset:        84,
				ConsumedBytes: 84,
				PendingBytes:  12,
				Batch: &ingestImportsReport{
					Status:             "ok",
					Source:             "watcher_import",
					Input:              `D:\Ops\follow\events.jsonl`,
					Scope:              exampleScope,
					Session:            session.Session{ID: "sess_20260318_013500", Scope: exampleScope.Ref(), Status: session.StatusActive, Task: "audit imported notes (watcher_import)", BranchName: "master", StartedAt: time.Date(2026, 3, 18, 1, 35, 0, 0, time.UTC)},
					AuditOnly:          true,
					Attempted:          3,
					Processed:          3,
					Failed:             0,
					Materialized:       0,
					Suppressed:         2,
					SuppressionReasons: map[string]int{"explicit_memory_exists": 1, "privacy_intent": 1},
					WouldMaterialize:   1,
					LinkedExistingNote: 0,
				},
			},
		},
		{
			Name:         "audit-only-multi-json",
			RelativePath: "follow-imports-audit-only-multi.json",
			JSON:         true,
			Report: followImportsAggregateReport{
				Status:             "ok",
				Source:             "relay_import",
				InputCount:         2,
				AuditOnly:          true,
				ConsumedInputs:     1,
				IdleInputs:         1,
				RequestedWatchMode: "auto",
				ActiveWatchMode:    "notify",
				WatchFallbacks:     1,
				WatchTransitions:   3,
				LastFallbackReason: "watcher_unavailable",
				WatchEventCount:    1,
				WatchEvents:        []followImportsEvent{exampleEvent},
				WatchPollCatchups:  1,
				WatchCatchupBytes:  42,
				Warnings: []common.Warning{
					{Code: common.WarnFollowImportsPollCatchup, Message: "notify mode repeatedly relied on poll catchup; treat watcher health as degraded"},
				},
				TotalConsumedBytes: 42,
				TotalPendingBytes:  7,
				Inputs: []followImportsReport{
					{
						Status:             "ok",
						Source:             "relay_import",
						Input:              `D:\Ops\follow\events-a.jsonl`,
						StateFile:          `D:\Ops\follow\events-a.offset.json`,
						AuditOnly:          true,
						RequestedWatchMode: "auto",
						ActiveWatchMode:    "notify",
						Offset:             42,
						ConsumedBytes:      42,
						PendingBytes:       0,
						Batch: &ingestImportsReport{
							Status:             "ok",
							Source:             "relay_import",
							Input:              `D:\Ops\follow\events-a.jsonl`,
							Scope:              exampleScope,
							Session:            session.Session{ID: "sess_20260318_013600", Scope: exampleScope.Ref(), Status: session.StatusActive, Task: "audit imported notes (relay_import)", BranchName: "master", StartedAt: time.Date(2026, 3, 18, 1, 36, 0, 0, time.UTC)},
							AuditOnly:          true,
							Attempted:          2,
							Processed:          2,
							Failed:             0,
							Materialized:       0,
							Suppressed:         1,
							SuppressionReasons: map[string]int{"import_policy": 1},
							WouldMaterialize:   0,
							LinkedExistingNote: 1,
							NoteDeduplicated:   1,
							ImportDeduplicated: 1,
						},
					},
					{
						Status:             "idle",
						Source:             "relay_import",
						Input:              `D:\Ops\follow\events-b.jsonl`,
						StateFile:          `D:\Ops\follow\events-b.offset.json`,
						AuditOnly:          true,
						RequestedWatchMode: "auto",
						ActiveWatchMode:    "notify",
						Offset:             0,
						ConsumedBytes:      0,
						PendingBytes:       7,
					},
				},
			},
		},
	}
}

func assertIngestImportsExampleOutput(t *testing.T, path string, jsonOutput bool, report ingestImportsReport) {
	t.Helper()

	body, err := renderIngestImportsExample(report, jsonOutput)
	if err != nil {
		t.Fatalf("render ingest-imports example: %v", err)
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if !bytes.Equal(body, expected) {
		t.Fatalf("ingest example mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", path, string(body), string(expected))
	}
}

func assertFollowImportsExampleOutput(t *testing.T, path string, jsonOutput bool, report any) {
	t.Helper()

	body, err := renderFollowImportsExample(report, jsonOutput)
	if err != nil {
		t.Fatalf("render follow-imports example: %v", err)
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if !bytes.Equal(body, expected) {
		t.Fatalf("follow-imports example mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", path, string(body), string(expected))
	}
}

func renderIngestImportsExample(report ingestImportsReport, jsonOutput bool) ([]byte, error) {
	if jsonOutput {
		body, err := marshalIndented(report)
		if err != nil {
			return nil, err
		}
		return []byte(body), nil
	}
	return []byte(formatIngestImportsReport(report)), nil
}

func renderFollowImportsExample(report any, jsonOutput bool) ([]byte, error) {
	if jsonOutput {
		body, err := marshalIndented(report)
		if err != nil {
			return nil, err
		}
		return []byte(body), nil
	}
	switch typed := report.(type) {
	case followImportsReport:
		return []byte(formatFollowImportsReport(typed)), nil
	case followImportsAggregateReport:
		return []byte(formatFollowImportsAggregateReport(typed)), nil
	default:
		return nil, fmt.Errorf("unsupported follow-imports example report type %T", report)
	}
}

func writeIngestImportsExampleFixtures(baseDir string, names []string) ([]string, error) {
	return writeCommandExampleFixtures(baseDir, names, "ingest-imports", ingestImportsExampleFixtures(), renderIngestImportsExample)
}

func writeFollowImportsCommandExampleFixtures(baseDir string, names []string) ([]string, error) {
	return writeCommandExampleFixtures(baseDir, names, "follow-imports", followImportsCommandExampleFixtures(), renderFollowImportsExample)
}
