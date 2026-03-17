package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	readinessMCPTransportStdio          = "stdio"
	readinessWarnFollowHealthStale      = "WARN_FOLLOW_IMPORTS_HEALTH_STALE"
	readinessWarnFollowPollCatchup      = "WARN_FOLLOW_IMPORTS_POLL_CATCHUP"
	readinessRunnerDoctorJSON           = "run ./cmd/codex-mem doctor --json"
	readinessRunnerStdioSmoke           = "run ./scripts/mcp-smoke-test"
	readinessRunnerHTTPSmoke            = "run ./scripts/http-mcp-smoke-test"
	readinessHTTPSmokePassedLine        = "http mcp smoke test passed\n"
	readinessSummaryAllPhasesPassed     = "all readiness phases passed"
	readinessSummaryWarningPolicyFailed = "warning policy failed: WARN_FOLLOW_IMPORTS_HEALTH_STALE"
)

func successfulReadinessReport(t *testing.T) readinessReport {
	t.Helper()

	updatedAt := time.Date(2026, time.March, 16, 8, 0, 0, 0, time.UTC)
	runStartedAt := time.Date(2026, time.March, 16, 7, 59, 0, 0, time.UTC)
	runCompletedAt := runStartedAt.Add(9 * time.Second)
	phaseStartedAt := time.Date(2026, time.March, 16, 8, 1, 0, 0, time.UTC)
	phaseCompletedAt := phaseStartedAt.Add(1500 * time.Millisecond)
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
		{Code: readinessWarnFollowPollCatchup},
		{Code: readinessWarnFollowHealthStale},
	}
	doctor.MCP.Transport = readinessMCPTransportStdio
	doctor.MCP.ToolCount = 11

	report := readinessReport{
		Status:               readinessStatusOK,
		Summary:              readinessSummaryAllPhasesPassed,
		KeepGoing:            true,
		SlowRunThresholdMS:   8000,
		SlowPhaseThresholdMS: 1000,
		StartedAt:            &runStartedAt,
		CompletedAt:          &runCompletedAt,
		DurationMS:           9000,
		Doctor:               doctor,
		Stdio: readinessSmokeTest{
			Status:  readinessStatusOK,
			Summary: "mcp smoke test passed",
		},
		HTTP: readinessSmokeTest{
			Status:  readinessStatusOK,
			Summary: "http mcp smoke test passed",
		},
		Phases: []readinessPhaseResult{
			{Name: readinessPhaseDoctor, Status: readinessStatusOK, Summary: "doctor --json passed", StartedAt: &phaseStartedAt, CompletedAt: &phaseCompletedAt, DurationMS: 1500},
			{Name: readinessPhaseStdio, Status: readinessStatusOK, Summary: "mcp smoke test passed", StartedAt: &phaseStartedAt, CompletedAt: &phaseCompletedAt, DurationMS: 1500},
			{Name: readinessPhaseHTTP, Status: readinessStatusOK, Summary: "http mcp smoke test passed", StartedAt: &phaseStartedAt, CompletedAt: &phaseCompletedAt, DurationMS: 1500},
		},
	}
	refreshWarningState(&report)
	return report
}

func TestParseOptions(t *testing.T) {
	options, err := parseOptions([]string{
		"--json",
		"--keep-going",
		"--slow-run-ms=8000",
		"--slow-phase-ms", "1000",
		"--fail-on-warning-code", "warn_follow_imports_health_stale,WARN_READINESS_RUN_SLOW",
	})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if !options.JSON {
		t.Fatal("expected JSON option to be enabled")
	}
	if !options.KeepGoing {
		t.Fatal("expected keep-going option to be enabled")
	}
	if options.SlowRunThresholdMS != 8000 {
		t.Fatalf("unexpected slow run threshold: %d", options.SlowRunThresholdMS)
	}
	if options.SlowPhaseThresholdMS != 1000 {
		t.Fatalf("unexpected slow phase threshold: %d", options.SlowPhaseThresholdMS)
	}
	if len(options.FailOnWarningCodes) != 2 || options.FailOnWarningCodes[0] != readinessWarnFollowHealthStale || options.FailOnWarningCodes[1] != "WARN_READINESS_RUN_SLOW" {
		t.Fatalf("unexpected fail-on-warning codes: %+v", options.FailOnWarningCodes)
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

func TestParseOptionsRejectsInvalidThreshold(t *testing.T) {
	_, err := parseOptions([]string{"--slow-run-ms=0"})
	if err == nil {
		t.Fatal("expected error for invalid threshold")
	}
	if !strings.Contains(err.Error(), `"--slow-run-ms"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseOptionsRejectsInvalidWarningCodeFlag(t *testing.T) {
	_, err := parseOptions([]string{"--fail-on-warning-code="})
	if err == nil {
		t.Fatal("expected error for invalid warning code flag")
	}
	if !strings.Contains(err.Error(), `"--fail-on-warning-code"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseOptionsAppliesCIProfileDefaults(t *testing.T) {
	options, err := parseOptions([]string{"--policy-profile", "ci"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if options.PolicyProfile != readinessPolicyProfileCI {
		t.Fatalf("unexpected policy profile: %q", options.PolicyProfile)
	}
	if options.SlowRunThresholdMS != 8000 || options.SlowPhaseThresholdMS != 1000 {
		t.Fatalf("unexpected profile thresholds: run=%d phase=%d", options.SlowRunThresholdMS, options.SlowPhaseThresholdMS)
	}
	if len(options.FailOnWarningCodes) != 0 {
		t.Fatalf("expected no fail-on-warning codes for ci profile, got %+v", options.FailOnWarningCodes)
	}
}

func TestParseOptionsAppliesSlowCIProfileDefaults(t *testing.T) {
	options, err := parseOptions([]string{"--policy-profile", "slow-ci"})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if options.PolicyProfile != readinessPolicyProfileSlowCI {
		t.Fatalf("unexpected policy profile: %q", options.PolicyProfile)
	}
	if options.SlowRunThresholdMS != 20000 || options.SlowPhaseThresholdMS != 4000 {
		t.Fatalf("unexpected profile thresholds: run=%d phase=%d", options.SlowRunThresholdMS, options.SlowPhaseThresholdMS)
	}
	if len(options.FailOnWarningCodes) != 0 {
		t.Fatalf("expected no fail-on-warning codes for slow-ci profile, got %+v", options.FailOnWarningCodes)
	}
}

func TestParseOptionsAllowsExplicitOverridesOnProfile(t *testing.T) {
	options, err := parseOptions([]string{
		"--policy-profile=release",
		"--slow-run-ms=9000",
		"--fail-on-warning-code", "WARN_READINESS_RUN_SLOW",
	})
	if err != nil {
		t.Fatalf("parseOptions: %v", err)
	}
	if options.PolicyProfile != readinessPolicyProfileRelease {
		t.Fatalf("unexpected policy profile: %q", options.PolicyProfile)
	}
	if options.SlowRunThresholdMS != 9000 || options.SlowPhaseThresholdMS != 1000 {
		t.Fatalf("unexpected profile thresholds after override: run=%d phase=%d", options.SlowRunThresholdMS, options.SlowPhaseThresholdMS)
	}
	if !slices.Equal(options.FailOnWarningCodes, []string{readinessWarnFollowHealthStale, "WARN_READINESS_RUN_SLOW"}) {
		t.Fatalf("unexpected merged fail-on-warning codes: %+v", options.FailOnWarningCodes)
	}
}

func TestWriteReadinessSummaryIncludesNamedPolicyProfile(t *testing.T) {
	report := newReadinessReport(readinessOptions{
		PolicyProfile:        readinessPolicyProfileSlowCI,
		SlowRunThresholdMS:   20000,
		SlowPhaseThresholdMS: 4000,
	})

	var buffer bytes.Buffer
	if err := writeReadinessSummary(&buffer, report); err != nil {
		t.Fatalf("writeReadinessSummary: %v", err)
	}

	output := buffer.String()
	for _, fragment := range []string{
		"policy_profile=slow-ci",
		"slow_run_threshold_ms=20000",
		"slow_phase_threshold_ms=4000",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("summary missing %q:\n%s", fragment, output)
		}
	}
}

func TestParseOptionsRejectsUnknownPolicyProfile(t *testing.T) {
	_, err := parseOptions([]string{"--policy-profile=staging"})
	if err == nil {
		t.Fatal("expected error for unknown policy profile")
	}
	if !strings.Contains(err.Error(), `"--policy-profile"`) {
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
		"keep_going=true",
		"policy_profile=none",
		"slow_run_threshold_ms=8000",
		"slow_phase_threshold_ms=1000",
		"fail_on_warning_codes=none",
		"started_at=2026-03-16T07:59:00Z",
		"completed_at=2026-03-16T07:59:09Z",
		"duration_ms=9000",
		"warning_codes=WARN_READINESS_RUN_SLOW,WARN_READINESS_PHASE_SLOW,WARN_READINESS_PHASE_SLOW,WARN_READINESS_PHASE_SLOW",
		"warning_count=4",
		"all_warning_codes=WARN_READINESS_RUN_SLOW,WARN_READINESS_PHASE_SLOW,WARN_FOLLOW_IMPORTS_POLL_CATCHUP,WARN_FOLLOW_IMPORTS_HEALTH_STALE",
		"matched_warning_codes=none",
		"warning_policy_failed=false",
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
		"phase_doctor_started_at=2026-03-16T08:01:00Z",
		"phase_doctor_completed_at=2026-03-16T08:01:01Z",
		"phase_doctor_duration_ms=1500",
		"phase_doctor_warning_codes=WARN_READINESS_PHASE_SLOW",
		"phase_stdio_mcp_smoke_test_status=ok",
		"phase_stdio_mcp_smoke_test_duration_ms=1500",
		"phase_stdio_mcp_smoke_test_warning_codes=WARN_READINESS_PHASE_SLOW",
		"phase_http_mcp_smoke_test_status=ok",
		"phase_http_mcp_smoke_test_warning_codes=WARN_READINESS_PHASE_SLOW",
		"warning_0_code=WARN_READINESS_RUN_SLOW",
		"warning_0_scope=run",
		"warning_0_threshold_ms=8000",
		"warning_1_code=WARN_READINESS_PHASE_SLOW",
		"warning_1_phase=doctor",
		"warning_2_phase=stdio_mcp_smoke_test",
		"warning_3_phase=http_mcp_smoke_test",
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
	assertSuccessfulReadinessJSONReport(t, decoded)
}

func TestWriteReadinessSummaryUsesNoneWhenFollowImportsHealthMissing(t *testing.T) {
	report := newReadinessReport(readinessOptions{})

	var buffer bytes.Buffer
	if err := writeReadinessSummary(&buffer, report); err != nil {
		t.Fatalf("writeReadinessSummary: %v", err)
	}

	output := buffer.String()
	for _, fragment := range []string{
		"readiness check failed",
		"status=not_run",
		"keep_going=false",
		"policy_profile=none",
		"slow_run_threshold_ms=0",
		"slow_phase_threshold_ms=0",
		"fail_on_warning_codes=none",
		"started_at=none",
		"completed_at=none",
		"duration_ms=0",
		"warning_codes=none",
		"warning_count=0",
		"all_warning_codes=none",
		"matched_warning_codes=none",
		"warning_policy_failed=false",
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
		"phase_doctor_started_at=none",
		"phase_doctor_completed_at=none",
		"phase_doctor_duration_ms=0",
		"phase_doctor_warning_codes=none",
		"phase_stdio_mcp_smoke_test_status=not_run",
		"phase_stdio_mcp_smoke_test_warning_codes=none",
		"phase_http_mcp_smoke_test_status=not_run",
		"phase_http_mcp_smoke_test_warning_codes=none",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("summary missing %q:\n%s", fragment, output)
		}
	}
}

func TestWriteReadinessSummaryShowsFailedPhase(t *testing.T) {
	report := newReadinessReport(readinessOptions{})
	report.Status = readinessStatusFailed
	report.Summary = "readiness phases failed: stdio_mcp_smoke_test"
	report.SlowRunThresholdMS = 4000
	report.SlowPhaseThresholdMS = 2000
	startedAt := time.Date(2026, time.March, 16, 8, 2, 0, 0, time.UTC)
	completedAt := startedAt.Add(5 * time.Second)
	report.StartedAt = &startedAt
	report.CompletedAt = &completedAt
	report.DurationMS = 5000
	report.Stdio = readinessSmokeTest{
		Status:  readinessStatusFailed,
		Summary: "stdio mcp smoke test failed: exit status 1 (panic: boom)",
	}
	phaseStartedAt := time.Date(2026, time.March, 16, 8, 2, 1, 0, time.UTC)
	phaseCompletedAt := phaseStartedAt.Add(2200 * time.Millisecond)
	completeReadinessPhase(&report, readinessPhaseDoctor, readinessStatusOK, "doctor --json passed", phaseStartedAt, phaseCompletedAt)
	completeReadinessPhase(&report, readinessPhaseStdio, readinessStatusFailed, report.Stdio.Summary, phaseStartedAt, phaseCompletedAt)
	refreshWarningState(&report)

	var buffer bytes.Buffer
	if err := writeReadinessSummary(&buffer, report); err != nil {
		t.Fatalf("writeReadinessSummary: %v", err)
	}

	output := buffer.String()
	for _, fragment := range []string{
		"readiness check failed",
		"status=failed",
		"summary=readiness phases failed: stdio_mcp_smoke_test",
		"duration_ms=5000",
		"policy_profile=none",
		"fail_on_warning_codes=none",
		"warning_codes=WARN_READINESS_RUN_SLOW,WARN_READINESS_PHASE_SLOW,WARN_READINESS_PHASE_SLOW",
		"warning_count=3",
		"all_warning_codes=WARN_READINESS_RUN_SLOW,WARN_READINESS_PHASE_SLOW",
		"matched_warning_codes=none",
		"warning_policy_failed=false",
		"phase_doctor_status=ok",
		"phase_stdio_mcp_smoke_test_status=failed",
		"phase_stdio_mcp_smoke_test_summary=stdio mcp smoke test failed: exit status 1 (panic: boom)",
		"phase_stdio_mcp_smoke_test_duration_ms=2200",
		"phase_stdio_mcp_smoke_test_warning_codes=WARN_READINESS_PHASE_SLOW",
		"phase_http_mcp_smoke_test_status=not_run",
		"phase_http_mcp_smoke_test_warning_codes=none",
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

func TestRunReadinessCheckKeepGoingContinuesAfterDoctorFailure(t *testing.T) {
	options := readinessOptions{KeepGoing: true, SlowPhaseThresholdMS: 1}
	runner := func(_ context.Context, _ string, args ...string) (string, string, error) {
		switch strings.Join(args, " ") {
		case readinessRunnerDoctorJSON:
			return "", "doctor exploded\n", errors.New("exit status 1")
		case readinessRunnerStdioSmoke:
			return "mcp smoke test passed\n", "", nil
		case readinessRunnerHTTPSmoke:
			return readinessHTTPSmokePassedLine, "", nil
		default:
			t.Fatalf("unexpected command: %v", args)
			return "", "", nil
		}
	}

	report, err := runReadinessCheckWithRunner(context.Background(), ".", options, runner)
	if err == nil {
		t.Fatal("expected readiness failure")
	}
	if report.Status != readinessStatusFailed {
		t.Fatalf("status mismatch: %q", report.Status)
	}
	if report.Summary != "readiness phases failed: doctor" {
		t.Fatalf("summary mismatch: %q", report.Summary)
	}
	if report.StartedAt == nil || report.CompletedAt == nil || report.DurationMS < 0 {
		t.Fatalf("expected run timing, got started=%v completed=%v duration=%d", report.StartedAt, report.CompletedAt, report.DurationMS)
	}
	if !report.KeepGoing {
		t.Fatal("expected keep-going report")
	}
	if report.Phases[0].Status != readinessStatusFailed || report.Phases[1].Status != readinessStatusOK || report.Phases[2].Status != readinessStatusOK {
		t.Fatalf("unexpected phase statuses: %+v", report.Phases)
	}
	if report.Phases[0].StartedAt == nil || report.Phases[0].CompletedAt == nil || report.Phases[0].DurationMS < 0 {
		t.Fatalf("expected doctor phase timing, got %+v", report.Phases[0])
	}
	if report.SlowPhaseThresholdMS != 1 {
		t.Fatalf("expected slow phase threshold to be retained, got %+v", report)
	}
	if report.Stdio.Status != readinessStatusOK || report.HTTP.Status != readinessStatusOK {
		t.Fatalf("expected later smoke tests to still run: stdio=%+v http=%+v", report.Stdio, report.HTTP)
	}
}

func TestRunReadinessCheckKeepGoingContinuesAfterStdioFailure(t *testing.T) {
	options := readinessOptions{KeepGoing: true}
	runner := func(_ context.Context, _ string, args ...string) (string, string, error) {
		switch strings.Join(args, " ") {
		case readinessRunnerDoctorJSON:
			return `{"status":"ok","runtime":{"foreign_keys":true,"required_schema_ok":true,"fts_ready":true},"migrations":{"pending":0},"audit":{"note_provenance_ready":true,"exclusion_audit_ready":true,"import_audit_ready":true},"follow_imports":{"health_present":false,"last_updated_at":null,"status":"","source":"","input_count":0,"continuous":false,"poll_interval_seconds":0,"snapshot_age_seconds":0,"health_stale":false,"requested_watch_mode":"","active_watch_mode":"","watch_fallbacks":0,"watch_transitions":0,"watch_poll_catchups":0,"watch_poll_catchup_bytes":0,"warnings":null},"mcp":{"transport":"stdio","tool_count":11}}`, "", nil
		case readinessRunnerStdioSmoke:
			return "panic: boom\n", "", errors.New("exit status 1")
		case readinessRunnerHTTPSmoke:
			return readinessHTTPSmokePassedLine, "", nil
		default:
			t.Fatalf("unexpected command: %v", args)
			return "", "", nil
		}
	}

	report, err := runReadinessCheckWithRunner(context.Background(), ".", options, runner)
	if err == nil {
		t.Fatal("expected readiness failure")
	}
	if report.Status != readinessStatusFailed {
		t.Fatalf("status mismatch: %q", report.Status)
	}
	if report.Summary != "readiness phases failed: stdio_mcp_smoke_test" {
		t.Fatalf("summary mismatch: %q", report.Summary)
	}
	if report.StartedAt == nil || report.CompletedAt == nil || report.DurationMS < 0 {
		t.Fatalf("expected run timing, got started=%v completed=%v duration=%d", report.StartedAt, report.CompletedAt, report.DurationMS)
	}
	if report.Phases[0].Status != readinessStatusOK || report.Phases[1].Status != readinessStatusFailed || report.Phases[2].Status != readinessStatusOK {
		t.Fatalf("unexpected phase statuses: %+v", report.Phases)
	}
	if report.Phases[1].StartedAt == nil || report.Phases[1].CompletedAt == nil || report.Phases[1].DurationMS < 0 {
		t.Fatalf("expected stdio phase timing, got %+v", report.Phases[1])
	}
	if report.HTTP.Status != readinessStatusOK {
		t.Fatalf("expected http phase to still run, got %+v", report.HTTP)
	}
}

func TestRunReadinessCheckFailsOnMatchedWarningCode(t *testing.T) {
	options := readinessOptions{FailOnWarningCodes: []string{readinessWarnFollowHealthStale}}
	runner := func(_ context.Context, _ string, args ...string) (string, string, error) {
		switch strings.Join(args, " ") {
		case readinessRunnerDoctorJSON:
			return `{"status":"ok","runtime":{"foreign_keys":true,"required_schema_ok":true,"fts_ready":true},"migrations":{"pending":0},"audit":{"note_provenance_ready":true,"exclusion_audit_ready":true,"import_audit_ready":true},"follow_imports":{"health_present":true,"last_updated_at":"2026-03-16T08:00:00Z","status":"partial","source":"watcher_import","input_count":1,"continuous":true,"poll_interval_seconds":5,"snapshot_age_seconds":12,"health_stale":true,"requested_watch_mode":"auto","active_watch_mode":"notify","watch_fallbacks":1,"watch_transitions":2,"watch_poll_catchups":3,"watch_poll_catchup_bytes":128,"warnings":[{"code":"WARN_FOLLOW_IMPORTS_HEALTH_STALE"}]},"mcp":{"transport":"stdio","tool_count":11}}`, "", nil
		case readinessRunnerStdioSmoke:
			return "mcp smoke test passed\n", "", nil
		case readinessRunnerHTTPSmoke:
			return readinessHTTPSmokePassedLine, "", nil
		default:
			t.Fatalf("unexpected command: %v", args)
			return "", "", nil
		}
	}

	report, err := runReadinessCheckWithRunner(context.Background(), ".", options, runner)
	if err == nil {
		t.Fatal("expected readiness failure from warning policy")
	}
	if report.Status != readinessStatusFailed {
		t.Fatalf("status mismatch: %q", report.Status)
	}
	if report.Summary != readinessSummaryWarningPolicyFailed {
		t.Fatalf("summary mismatch: %q", report.Summary)
	}
	if !report.WarningPolicyFailed {
		t.Fatalf("expected warning policy failure, got %+v", report)
	}
	if len(report.MatchedWarningCodes) != 1 || report.MatchedWarningCodes[0] != readinessWarnFollowHealthStale {
		t.Fatalf("unexpected matched warning codes: %+v", report.MatchedWarningCodes)
	}
	if len(report.AllWarningCodes) != 1 || report.AllWarningCodes[0] != readinessWarnFollowHealthStale {
		t.Fatalf("unexpected all warning codes: %+v", report.AllWarningCodes)
	}
	if report.Phases[0].Status != readinessStatusOK || report.Phases[1].Status != readinessStatusOK || report.Phases[2].Status != readinessStatusOK {
		t.Fatalf("expected all phases to pass before warning policy failed: %+v", report.Phases)
	}
}

func TestApplySlowWarningsSkipsNotRunPhases(t *testing.T) {
	report := newReadinessReport(readinessOptions{SlowRunThresholdMS: 1, SlowPhaseThresholdMS: 1})
	report.Status = readinessStatusFailed
	report.Summary = "readiness phases failed: doctor"
	startedAt := time.Date(2026, time.March, 16, 8, 3, 0, 0, time.UTC)
	completedAt := startedAt.Add(3 * time.Second)
	completeReadinessPhase(&report, readinessPhaseDoctor, readinessStatusFailed, "doctor check failed", startedAt, completedAt)
	completeReadinessRun(&report, startedAt, completedAt)
	refreshWarningState(&report)

	if len(report.Warnings) != 2 {
		t.Fatalf("unexpected warnings: %+v", report.Warnings)
	}
	if len(report.Phases[1].WarningCodes) != 0 || len(report.Phases[2].WarningCodes) != 0 {
		t.Fatalf("expected not-run phases to stay warning-free: %+v", report.Phases)
	}
}

func assertSuccessfulReadinessJSONReport(t *testing.T, decoded readinessReport) {
	t.Helper()

	assertSuccessfulReadinessJSONMeta(t, decoded)
	assertSuccessfulReadinessJSONDoctor(t, decoded)
	assertSuccessfulReadinessJSONSmoke(t, decoded)
	assertSuccessfulReadinessJSONPhases(t, decoded)
}

func assertSuccessfulReadinessJSONMeta(t *testing.T, decoded readinessReport) {
	t.Helper()

	if decoded.Status != readinessStatusOK {
		t.Fatalf("status mismatch: %q", decoded.Status)
	}
	if decoded.Summary != readinessSummaryAllPhasesPassed {
		t.Fatalf("summary mismatch: %q", decoded.Summary)
	}
	if !decoded.KeepGoing {
		t.Fatal("expected keep_going=true")
	}
	if decoded.PolicyProfile != "" {
		t.Fatalf("expected empty policy profile, got %q", decoded.PolicyProfile)
	}
	if decoded.SlowRunThresholdMS != 8000 || decoded.SlowPhaseThresholdMS != 1000 {
		t.Fatalf("unexpected thresholds: run=%d phase=%d", decoded.SlowRunThresholdMS, decoded.SlowPhaseThresholdMS)
	}
	if len(decoded.FailOnWarningCodes) != 0 {
		t.Fatalf("expected no fail-on-warning codes, got %+v", decoded.FailOnWarningCodes)
	}
	if decoded.StartedAt == nil || decoded.CompletedAt == nil || decoded.DurationMS != 9000 {
		t.Fatalf("unexpected run timing: started=%v completed=%v duration=%d", decoded.StartedAt, decoded.CompletedAt, decoded.DurationMS)
	}
	if len(decoded.Warnings) != 4 || decoded.Warnings[0].Code != "WARN_READINESS_RUN_SLOW" {
		t.Fatalf("unexpected readiness warnings: %+v", decoded.Warnings)
	}
	if len(decoded.AllWarningCodes) != 4 || decoded.AllWarningCodes[2] != readinessWarnFollowPollCatchup || decoded.WarningPolicyFailed {
		t.Fatalf("unexpected warning policy state: all=%+v matched=%+v failed=%t", decoded.AllWarningCodes, decoded.MatchedWarningCodes, decoded.WarningPolicyFailed)
	}
}

func assertSuccessfulReadinessJSONDoctor(t *testing.T, decoded readinessReport) {
	t.Helper()

	if decoded.Doctor == nil || !decoded.Doctor.Follow.HealthPresent || decoded.Doctor.Follow.Status != "partial" || decoded.Doctor.Follow.Source != "watcher_import" {
		t.Fatalf("unexpected doctor follow report: %+v", decoded.Doctor)
	}
	if len(decoded.Doctor.Follow.Warnings) != 2 || decoded.Doctor.Follow.Warnings[0].Code != readinessWarnFollowPollCatchup {
		t.Fatalf("unexpected warning codes: %+v", decoded.Doctor.Follow.Warnings)
	}
}

func assertSuccessfulReadinessJSONSmoke(t *testing.T, decoded readinessReport) {
	t.Helper()

	if decoded.Stdio.Status != readinessStatusOK || decoded.Stdio.Summary != "mcp smoke test passed" {
		t.Fatalf("unexpected stdio smoke summary: %+v", decoded.Stdio)
	}
	if decoded.HTTP.Status != readinessStatusOK || decoded.HTTP.Summary != "http mcp smoke test passed" {
		t.Fatalf("unexpected http smoke summary: %+v", decoded.HTTP)
	}
}

func assertSuccessfulReadinessJSONPhases(t *testing.T, decoded readinessReport) {
	t.Helper()

	if len(decoded.Phases) != 3 {
		t.Fatalf("unexpected phases: %+v", decoded.Phases)
	}
	if decoded.Phases[0].StartedAt == nil || decoded.Phases[0].CompletedAt == nil || decoded.Phases[0].DurationMS != 1500 {
		t.Fatalf("unexpected phase timing: %+v", decoded.Phases[0])
	}
	if len(decoded.Phases[0].WarningCodes) != 1 || decoded.Phases[0].WarningCodes[0] != "WARN_READINESS_PHASE_SLOW" {
		t.Fatalf("unexpected phase warning codes: %+v", decoded.Phases[0])
	}
}
