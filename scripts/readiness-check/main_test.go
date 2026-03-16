package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestWriteReadinessSummaryIncludesFollowImportsHealth(t *testing.T) {
	updatedAt := time.Date(2026, time.March, 16, 8, 0, 0, 0, time.UTC)
	doctor := doctorReport{
		Status: "ok",
	}
	doctor.Runtime.RequiredSchemaOK = true
	doctor.Runtime.FTSReady = true
	doctor.Migrations.Pending = 0
	doctor.Audit.NoteProvenanceReady = true
	doctor.Audit.ExclusionAuditReady = true
	doctor.Audit.ImportAuditReady = true
	doctor.Follow.HealthPresent = true
	doctor.Follow.LastUpdatedAt = &updatedAt
	doctor.Follow.Status = "partial"
	doctor.Follow.Source = "watcher_import"
	doctor.Follow.InputCount = 2
	doctor.Follow.Continuous = true
	doctor.Follow.PollIntervalSeconds = 5
	doctor.Follow.SnapshotAgeSeconds = 12
	doctor.Follow.HealthStale = false
	doctor.Follow.RequestedWatchMode = "auto"
	doctor.Follow.ActiveWatchMode = "notify"
	doctor.Follow.WatchFallbacks = 1
	doctor.Follow.WatchTransitions = 3
	doctor.Follow.WatchPollCatchups = 4
	doctor.Follow.WatchPollCatchupBytes = 256
	doctor.Follow.Warnings = []doctorWarning{
		{Code: "WARN_FOLLOW_IMPORTS_POLL_CATCHUP"},
		{Code: "WARN_FOLLOW_IMPORTS_HEALTH_STALE"},
	}
	doctor.MCP.Transport = "stdio"
	doctor.MCP.ToolCount = 11

	var buffer bytes.Buffer
	if err := writeReadinessSummary(&buffer, doctor, "mcp smoke test passed\n", "http mcp smoke test passed\n"); err != nil {
		t.Fatalf("writeReadinessSummary: %v", err)
	}

	output := buffer.String()
	for _, fragment := range []string{
		"readiness check passed",
		"doctor_follow_imports_health_present=true",
		"doctor_follow_imports_status=partial",
		"doctor_follow_imports_source=watcher_import",
		"doctor_follow_imports_input_count=2",
		"doctor_follow_imports_last_updated_at=2026-03-16T08:00:00Z",
		"doctor_follow_imports_continuous=true",
		"doctor_follow_imports_poll_interval_seconds=5",
		"doctor_follow_imports_snapshot_age_seconds=12",
		"doctor_follow_imports_health_stale=false",
		"doctor_follow_imports_requested_watch_mode=auto",
		"doctor_follow_imports_active_watch_mode=notify",
		"doctor_follow_imports_watch_fallbacks=1",
		"doctor_follow_imports_watch_transitions=3",
		"doctor_follow_imports_watch_poll_catchups=4",
		"doctor_follow_imports_watch_poll_catchup_bytes=256",
		"doctor_follow_imports_warning_codes=WARN_FOLLOW_IMPORTS_POLL_CATCHUP,WARN_FOLLOW_IMPORTS_HEALTH_STALE",
		"stdio_mcp_smoke_test=mcp smoke test passed",
		"http_mcp_smoke_test=http mcp smoke test passed",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("summary missing %q:\n%s", fragment, output)
		}
	}
}

func TestWriteReadinessSummaryUsesNoneWhenFollowImportsHealthMissing(t *testing.T) {
	doctor := doctorReport{
		Status: "ok",
	}
	doctor.Runtime.RequiredSchemaOK = true
	doctor.Runtime.FTSReady = true
	doctor.Audit.NoteProvenanceReady = true
	doctor.Audit.ExclusionAuditReady = true
	doctor.Audit.ImportAuditReady = true
	doctor.MCP.Transport = "stdio"
	doctor.MCP.ToolCount = 11

	var buffer bytes.Buffer
	if err := writeReadinessSummary(&buffer, doctor, "\n", "\n"); err != nil {
		t.Fatalf("writeReadinessSummary: %v", err)
	}

	output := buffer.String()
	for _, fragment := range []string{
		"doctor_follow_imports_health_present=false",
		"doctor_follow_imports_status=none",
		"doctor_follow_imports_source=none",
		"doctor_follow_imports_last_updated_at=none",
		"doctor_follow_imports_warning_codes=none",
		"stdio_mcp_smoke_test=none",
		"http_mcp_smoke_test=none",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("summary missing %q:\n%s", fragment, output)
		}
	}
}
