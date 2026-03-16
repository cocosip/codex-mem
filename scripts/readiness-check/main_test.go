package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const readinessMCPTransportStdio = "stdio"

func successfulReadinessReport(t *testing.T) readinessReport {
	t.Helper()

	updatedAt := time.Date(2026, time.March, 16, 8, 0, 0, 0, time.UTC)
	doctor := &doctorReport{
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
	doctor.MCP.Transport = readinessMCPTransportStdio
	doctor.MCP.ToolCount = 11

	return readinessReport{
		Status:  readinessStatusOK,
		Summary: "all readiness phases passed",
		Doctor:  doctor,
		Stdio: readinessSmokeTest{
			Status:  readinessStatusOK,
			Summary: "mcp smoke test passed",
		},
		HTTP: readinessSmokeTest{
			Status:  readinessStatusOK,
			Summary: "http mcp smoke test passed",
		},
		Phases: []readinessPhaseResult{
			{Name: readinessPhaseDoctor, Status: readinessStatusOK, Summary: "doctor --json passed"},
			{Name: readinessPhaseStdio, Status: readinessStatusOK, Summary: "mcp smoke test passed"},
			{Name: readinessPhaseHTTP, Status: readinessStatusOK, Summary: "http mcp smoke test passed"},
		},
	}
}

func TestParseOptions(t *testing.T) {
	options, err := parseOptions([]string{"--json"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if !options.JSON {
		t.Fatal("expected JSON option to be enabled")
	}
}

func TestParseOptionsRejectsUnknownFlag(t *testing.T) {
	_, err := parseOptions([]string{"--yaml"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if !strings.Contains(err.Error(), `unknown readiness-check flag "--yaml"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteReadinessSummaryIncludesFollowImportsHealth(t *testing.T) {
	report := successfulReadinessReport(t)

	var buffer bytes.Buffer
	if err := writeReadinessSummary(&buffer, report); err != nil {
		t.Fatalf("writeReadinessSummary: %v", err)
	}

	output := buffer.String()
	for _, fragment := range []string{
		"readiness check passed",
		"status=ok",
		"summary=all readiness phases passed",
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
		"stdio_mcp_smoke_test_status=ok",
		"stdio_mcp_smoke_test_summary=mcp smoke test passed",
		"http_mcp_smoke_test_status=ok",
		"http_mcp_smoke_test_summary=http mcp smoke test passed",
		"phase_doctor_status=ok",
		"phase_doctor_summary=doctor --json passed",
		"phase_stdio_mcp_smoke_test_status=ok",
		"phase_http_mcp_smoke_test_status=ok",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("summary missing %q:\n%s", fragment, output)
		}
	}
}

func TestWriteReadinessJSONIncludesDoctorAndSmokeSummaries(t *testing.T) {
	report := successfulReadinessReport(t)

	var buffer bytes.Buffer
	if err := writeReadinessJSON(&buffer, report); err != nil {
		t.Fatalf("writeReadinessJSON: %v", err)
	}

	var decoded readinessReport
	if err := json.Unmarshal(buffer.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v\n%s", err, buffer.String())
	}
	if decoded.Status != "ok" {
		t.Fatalf("status mismatch: %q", decoded.Status)
	}
	if decoded.Summary != "all readiness phases passed" {
		t.Fatalf("summary mismatch: %q", decoded.Summary)
	}
	if decoded.Doctor == nil || !decoded.Doctor.Follow.HealthPresent || decoded.Doctor.Follow.Status != "partial" || decoded.Doctor.Follow.Source != "watcher_import" {
		t.Fatalf("unexpected doctor follow report: %+v", decoded.Doctor)
	}
	if len(decoded.Doctor.Follow.Warnings) != 2 || decoded.Doctor.Follow.Warnings[0].Code != "WARN_FOLLOW_IMPORTS_POLL_CATCHUP" {
		t.Fatalf("unexpected warning codes: %+v", decoded.Doctor.Follow.Warnings)
	}
	if decoded.Stdio.Status != "ok" || decoded.Stdio.Summary != "mcp smoke test passed" {
		t.Fatalf("unexpected stdio smoke summary: %+v", decoded.Stdio)
	}
	if decoded.HTTP.Status != "ok" || decoded.HTTP.Summary != "http mcp smoke test passed" {
		t.Fatalf("unexpected http smoke summary: %+v", decoded.HTTP)
	}
	if len(decoded.Phases) != 3 {
		t.Fatalf("unexpected phases: %+v", decoded.Phases)
	}
}

func TestWriteReadinessSummaryUsesNoneWhenFollowImportsHealthMissing(t *testing.T) {
	report := newReadinessReport()

	var buffer bytes.Buffer
	if err := writeReadinessSummary(&buffer, report); err != nil {
		t.Fatalf("writeReadinessSummary: %v", err)
	}

	output := buffer.String()
	for _, fragment := range []string{
		"readiness check failed",
		"status=not_run",
		"doctor_follow_imports_health_present=false",
		"doctor_follow_imports_status=none",
		"doctor_follow_imports_source=none",
		"doctor_follow_imports_last_updated_at=none",
		"doctor_follow_imports_warning_codes=none",
		"stdio_mcp_smoke_test_status=not_run",
		"stdio_mcp_smoke_test_summary=none",
		"http_mcp_smoke_test_status=not_run",
		"http_mcp_smoke_test_summary=none",
		"phase_doctor_status=not_run",
		"phase_stdio_mcp_smoke_test_status=not_run",
		"phase_http_mcp_smoke_test_status=not_run",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("summary missing %q:\n%s", fragment, output)
		}
	}
}

func TestWriteReadinessSummaryShowsFailedPhase(t *testing.T) {
	report := newReadinessReport()
	report.Status = readinessStatusFailed
	report.Summary = "stdio mcp smoke test failed: exit status 1"
	report.Stdio = readinessSmokeTest{
		Status:  readinessStatusFailed,
		Summary: "stdio mcp smoke test failed: exit status 1 (panic: boom)",
	}
	setReadinessPhase(&report, readinessPhaseDoctor, readinessStatusOK, "doctor --json passed")
	setReadinessPhase(&report, readinessPhaseStdio, readinessStatusFailed, report.Stdio.Summary)

	var buffer bytes.Buffer
	if err := writeReadinessSummary(&buffer, report); err != nil {
		t.Fatalf("writeReadinessSummary: %v", err)
	}

	output := buffer.String()
	for _, fragment := range []string{
		"readiness check failed",
		"status=failed",
		"summary=stdio mcp smoke test failed: exit status 1",
		"phase_doctor_status=ok",
		"phase_stdio_mcp_smoke_test_status=failed",
		"phase_stdio_mcp_smoke_test_summary=stdio mcp smoke test failed: exit status 1 (panic: boom)",
		"phase_http_mcp_smoke_test_status=not_run",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("summary missing %q:\n%s", fragment, output)
		}
	}
}

func TestValidateDoctorRejectsBrokenReport(t *testing.T) {
	report := doctorReport{Status: "ok"}
	report.MCP.Transport = "stdio"
	report.MCP.ToolCount = 11
	report.Runtime.ForeignKeys = true
	report.Runtime.RequiredSchemaOK = true
	report.Runtime.FTSReady = true
	report.Audit.NoteProvenanceReady = true
	report.Audit.ExclusionAuditReady = true
	report.Audit.ImportAuditReady = true
	report.Migrations.Pending = 1

	err := validateDoctor(report)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "doctor migrations pending: 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}
