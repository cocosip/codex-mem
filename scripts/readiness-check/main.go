// Package main runs the local readiness smoke checks for codex-mem.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const stringNone = "none"

type readinessOptions struct {
	JSON                 bool
	KeepGoing            bool
	SlowRunThresholdMS   int64
	SlowPhaseThresholdMS int64
	FailOnWarningCodes   []string
}

type readinessReport struct {
	Status               string                 `json:"status"`
	Summary              string                 `json:"summary"`
	KeepGoing            bool                   `json:"keep_going"`
	SlowRunThresholdMS   int64                  `json:"slow_run_threshold_ms,omitempty"`
	SlowPhaseThresholdMS int64                  `json:"slow_phase_threshold_ms,omitempty"`
	FailOnWarningCodes   []string               `json:"fail_on_warning_codes,omitempty"`
	StartedAt            *time.Time             `json:"started_at,omitempty"`
	CompletedAt          *time.Time             `json:"completed_at,omitempty"`
	DurationMS           int64                  `json:"duration_ms,omitempty"`
	Warnings             []readinessWarning     `json:"warnings,omitempty"`
	AllWarningCodes      []string               `json:"all_warning_codes,omitempty"`
	MatchedWarningCodes  []string               `json:"matched_warning_codes,omitempty"`
	WarningPolicyFailed  bool                   `json:"warning_policy_failed"`
	Doctor               *doctorReport          `json:"doctor,omitempty"`
	Stdio                readinessSmokeTest     `json:"stdio_mcp_smoke_test"`
	HTTP                 readinessSmokeTest     `json:"http_mcp_smoke_test"`
	Phases               []readinessPhaseResult `json:"phases"`
}

type readinessSmokeTest struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

type readinessPhaseResult struct {
	Name         string     `json:"name"`
	Status       string     `json:"status"`
	Summary      string     `json:"summary"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	DurationMS   int64      `json:"duration_ms,omitempty"`
	WarningCodes []string   `json:"warning_codes,omitempty"`
}

type readinessWarning struct {
	Code             string `json:"code"`
	Summary          string `json:"summary"`
	Scope            string `json:"scope"`
	Phase            string `json:"phase,omitempty"`
	ThresholdMS      int64  `json:"threshold_ms"`
	ActualDurationMS int64  `json:"actual_duration_ms"`
}

type doctorReport struct {
	Status  string `json:"status"`
	Runtime struct {
		ForeignKeys      bool `json:"foreign_keys"`
		RequiredSchemaOK bool `json:"required_schema_ok"`
		FTSReady         bool `json:"fts_ready"`
	} `json:"runtime"`
	Migrations struct {
		Pending int `json:"pending"`
	} `json:"migrations"`
	Audit struct {
		NoteProvenanceReady bool `json:"note_provenance_ready"`
		ExclusionAuditReady bool `json:"exclusion_audit_ready"`
		ImportAuditReady    bool `json:"import_audit_ready"`
	} `json:"audit"`
	Follow struct {
		HealthPresent         bool            `json:"health_present"`
		LastUpdatedAt         *time.Time      `json:"last_updated_at"`
		Status                string          `json:"status"`
		Source                string          `json:"source"`
		InputCount            int             `json:"input_count"`
		Continuous            bool            `json:"continuous"`
		PollIntervalSeconds   int64           `json:"poll_interval_seconds"`
		SnapshotAgeSeconds    int64           `json:"snapshot_age_seconds"`
		HealthStale           bool            `json:"health_stale"`
		RequestedWatchMode    string          `json:"requested_watch_mode"`
		ActiveWatchMode       string          `json:"active_watch_mode"`
		WatchFallbacks        int             `json:"watch_fallbacks"`
		WatchTransitions      int             `json:"watch_transitions"`
		WatchPollCatchups     int             `json:"watch_poll_catchups"`
		WatchPollCatchupBytes int             `json:"watch_poll_catchup_bytes"`
		Warnings              []doctorWarning `json:"warnings"`
	} `json:"follow_imports"`
	MCP struct {
		Transport string `json:"transport"`
		ToolCount int    `json:"tool_count"`
	} `json:"mcp"`
}

type doctorWarning struct {
	Code string `json:"code"`
}

type goRunner func(ctx context.Context, dir string, args ...string) (string, string, error)

const (
	readinessStatusOK     = "ok"
	readinessStatusFailed = "failed"
	readinessStatusNotRun = "not_run"

	readinessPhaseDoctor = "doctor"
	readinessPhaseStdio  = "stdio_mcp_smoke_test"
	readinessPhaseHTTP   = "http_mcp_smoke_test"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	options, err := parseOptions(os.Args[1:])
	if err != nil {
		failf("%v", err)
	}

	repoRoot, err := os.Getwd()
	if err != nil {
		failf("resolve working directory: %v", err)
	}

	report, runErr := runReadinessCheck(ctx, repoRoot, options)
	if err := writeReadinessOutput(os.Stdout, report, options); err != nil {
		failf("write readiness summary: %v", err)
	}
	if runErr != nil {
		os.Exit(1)
	}
}

func parseOptions(args []string) (readinessOptions, error) {
	var options readinessOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch strings.TrimSpace(arg) {
		case "":
			continue
		case "--json":
			options.JSON = true
		case "--keep-going":
			options.KeepGoing = true
		default:
			var err error
			options, i, err = parseExtendedOption(args, i, options)
			if err != nil {
				return readinessOptions{}, err
			}
		}
	}
	return options, nil
}

func parseExtendedOption(args []string, index int, options readinessOptions) (readinessOptions, int, error) {
	arg := strings.TrimSpace(args[index])
	switch {
	case strings.HasPrefix(arg, "--slow-run-ms="):
		value, err := parsePositiveInt64Flag("slow-run-ms", strings.TrimSpace(strings.TrimPrefix(arg, "--slow-run-ms=")))
		if err != nil {
			return readinessOptions{}, index, err
		}
		options.SlowRunThresholdMS = value
		return options, index, nil
	case strings.HasPrefix(arg, "--slow-phase-ms="):
		value, err := parsePositiveInt64Flag("slow-phase-ms", strings.TrimSpace(strings.TrimPrefix(arg, "--slow-phase-ms=")))
		if err != nil {
			return readinessOptions{}, index, err
		}
		options.SlowPhaseThresholdMS = value
		return options, index, nil
	case arg == "--slow-run-ms":
		if index+1 >= len(args) {
			return readinessOptions{}, index, errors.New(`missing value for "--slow-run-ms"`)
		}
		value, err := parsePositiveInt64Flag("slow-run-ms", strings.TrimSpace(args[index+1]))
		if err != nil {
			return readinessOptions{}, index, err
		}
		options.SlowRunThresholdMS = value
		return options, index + 1, nil
	case arg == "--slow-phase-ms":
		if index+1 >= len(args) {
			return readinessOptions{}, index, errors.New(`missing value for "--slow-phase-ms"`)
		}
		value, err := parsePositiveInt64Flag("slow-phase-ms", strings.TrimSpace(args[index+1]))
		if err != nil {
			return readinessOptions{}, index, err
		}
		options.SlowPhaseThresholdMS = value
		return options, index + 1, nil
	case strings.HasPrefix(arg, "--fail-on-warning-code="):
		values, err := parseWarningCodeFlagValues(strings.TrimSpace(strings.TrimPrefix(arg, "--fail-on-warning-code=")))
		if err != nil {
			return readinessOptions{}, index, err
		}
		options.FailOnWarningCodes = appendNormalizedWarningCodes(options.FailOnWarningCodes, values...)
		return options, index, nil
	case arg == "--fail-on-warning-code":
		if index+1 >= len(args) {
			return readinessOptions{}, index, errors.New(`missing value for "--fail-on-warning-code"`)
		}
		values, err := parseWarningCodeFlagValues(strings.TrimSpace(args[index+1]))
		if err != nil {
			return readinessOptions{}, index, err
		}
		options.FailOnWarningCodes = appendNormalizedWarningCodes(options.FailOnWarningCodes, values...)
		return options, index + 1, nil
	default:
		return readinessOptions{}, index, fmt.Errorf("unknown readiness-check flag %q", arg)
	}
}

func parsePositiveInt64Flag(name string, raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf(`invalid value for "--%s": empty`, name)
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(`invalid value for "--%s": %q`, name, raw)
	}
	if value <= 0 {
		return 0, fmt.Errorf(`invalid value for "--%s": %q must be > 0`, name, raw)
	}
	return value, nil
}

func parseWarningCodeFlagValues(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New(`invalid value for "--fail-on-warning-code": empty`)
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		code := normalizeWarningCode(part)
		if code == "" {
			return nil, fmt.Errorf(`invalid value for "--fail-on-warning-code": %q`, raw)
		}
		values = append(values, code)
	}
	return values, nil
}

func normalizeWarningCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func appendNormalizedWarningCodes(existing []string, values ...string) []string {
	result := append([]string(nil), existing...)
	seen := make(map[string]struct{}, len(result))
	for _, value := range result {
		seen[normalizeWarningCode(value)] = struct{}{}
	}
	for _, value := range values {
		normalized := normalizeWarningCode(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		result = append(result, normalized)
		seen[normalized] = struct{}{}
	}
	return result
}

func runGo(ctx context.Context, dir string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(stdout), string(exitErr.Stderr), err
		}
		return string(stdout), "", err
	}
	return string(stdout), "", nil
}

func runReadinessCheck(ctx context.Context, repoRoot string, options readinessOptions) (readinessReport, error) {
	return runReadinessCheckWithRunner(ctx, repoRoot, options, runGo)
}

func runReadinessCheckWithRunner(ctx context.Context, repoRoot string, options readinessOptions, runner goRunner) (readinessReport, error) {
	report := newReadinessReport(options)
	startedAt := time.Now().UTC()
	failedPhases := make([]string, 0, len(report.Phases))

	if err := runDoctorPhase(ctx, repoRoot, runner, &report); err != nil {
		failedPhases = append(failedPhases, readinessPhaseDoctor)
		if !options.KeepGoing {
			return finalizeReadinessReport(report, failedPhases, startedAt, time.Now().UTC())
		}
	}
	if err := runSmokePhase(ctx, repoRoot, runner, &report, readinessPhaseStdio, "stdio mcp smoke test", "mcp smoke test passed", "run", "./scripts/mcp-smoke-test"); err != nil {
		failedPhases = append(failedPhases, readinessPhaseStdio)
		if !options.KeepGoing {
			return finalizeReadinessReport(report, failedPhases, startedAt, time.Now().UTC())
		}
	}
	if err := runSmokePhase(ctx, repoRoot, runner, &report, readinessPhaseHTTP, "http mcp smoke test", "http mcp smoke test passed", "run", "./scripts/http-mcp-smoke-test"); err != nil {
		failedPhases = append(failedPhases, readinessPhaseHTTP)
		if !options.KeepGoing {
			return finalizeReadinessReport(report, failedPhases, startedAt, time.Now().UTC())
		}
	}
	return finalizeReadinessReport(report, failedPhases, startedAt, time.Now().UTC())
}

func newReadinessReport(options readinessOptions) readinessReport {
	return readinessReport{
		Status:               readinessStatusNotRun,
		Summary:              "readiness phases have not run",
		KeepGoing:            options.KeepGoing,
		SlowRunThresholdMS:   options.SlowRunThresholdMS,
		SlowPhaseThresholdMS: options.SlowPhaseThresholdMS,
		FailOnWarningCodes:   append([]string(nil), options.FailOnWarningCodes...),
		Stdio: readinessSmokeTest{
			Status:  readinessStatusNotRun,
			Summary: stringNone,
		},
		HTTP: readinessSmokeTest{
			Status:  readinessStatusNotRun,
			Summary: stringNone,
		},
		Phases: []readinessPhaseResult{
			{Name: readinessPhaseDoctor, Status: readinessStatusNotRun, Summary: stringNone},
			{Name: readinessPhaseStdio, Status: readinessStatusNotRun, Summary: stringNone},
			{Name: readinessPhaseHTTP, Status: readinessStatusNotRun, Summary: stringNone},
		},
	}
}

func runDoctorPhase(ctx context.Context, repoRoot string, runner goRunner, report *readinessReport) error {
	startedAt := time.Now().UTC()
	doctorStdout, doctorStderr, err := runner(ctx, repoRoot, "run", "./cmd/codex-mem", "doctor", "--json")
	if err != nil {
		failure := fmt.Errorf("doctor check failed: %v", err)
		completeReadinessPhase(report, readinessPhaseDoctor, readinessStatusFailed, summarizeCommandFailure(failure.Error(), doctorStdout, doctorStderr), startedAt, time.Now().UTC())
		return failure
	}

	var doctor doctorReport
	if err := json.Unmarshal([]byte(doctorStdout), &doctor); err != nil {
		failure := fmt.Errorf("decode doctor JSON: %v", err)
		completeReadinessPhase(report, readinessPhaseDoctor, readinessStatusFailed, summarizeCommandFailure(failure.Error(), doctorStdout, doctorStderr), startedAt, time.Now().UTC())
		return failure
	}
	if err := validateDoctor(doctor); err != nil {
		report.Doctor = &doctor
		completeReadinessPhase(report, readinessPhaseDoctor, readinessStatusFailed, err.Error(), startedAt, time.Now().UTC())
		return err
	}
	report.Doctor = &doctor
	completeReadinessPhase(report, readinessPhaseDoctor, readinessStatusOK, "doctor --json passed", startedAt, time.Now().UTC())
	return nil
}

func runSmokePhase(ctx context.Context, repoRoot string, runner goRunner, report *readinessReport, phaseName string, label string, successMarker string, args ...string) error {
	startedAt := time.Now().UTC()
	stdout, stderr, err := runner(ctx, repoRoot, args...)
	if err != nil {
		failure := fmt.Errorf("%s failed: %v", label, err)
		setSmokeTestFailure(report, phaseName, summarizeCommandFailure(failure.Error(), stdout, stderr), startedAt, time.Now().UTC())
		return failure
	}
	if !strings.Contains(stdout, successMarker) {
		failure := fmt.Errorf("%s did not report success", label)
		setSmokeTestFailure(report, phaseName, summarizeCommandFailure(failure.Error(), stdout, stderr), startedAt, time.Now().UTC())
		return failure
	}
	setSmokeTestSuccess(report, phaseName, firstLine(stdout), startedAt, time.Now().UTC())
	return nil
}

func setSmokeTestFailure(report *readinessReport, phaseName string, summary string, startedAt, completedAt time.Time) {
	summary = fallbackString(summary)
	switch phaseName {
	case readinessPhaseStdio:
		report.Stdio = readinessSmokeTest{Status: readinessStatusFailed, Summary: summary}
	case readinessPhaseHTTP:
		report.HTTP = readinessSmokeTest{Status: readinessStatusFailed, Summary: summary}
	}
	completeReadinessPhase(report, phaseName, readinessStatusFailed, summary, startedAt, completedAt)
}

func setSmokeTestSuccess(report *readinessReport, phaseName string, summary string, startedAt, completedAt time.Time) {
	summary = fallbackString(summary)
	switch phaseName {
	case readinessPhaseStdio:
		report.Stdio = readinessSmokeTest{Status: readinessStatusOK, Summary: summary}
	case readinessPhaseHTTP:
		report.HTTP = readinessSmokeTest{Status: readinessStatusOK, Summary: summary}
	}
	completeReadinessPhase(report, phaseName, readinessStatusOK, summary, startedAt, completedAt)
}

func finalizeReadinessReport(report readinessReport, failedPhases []string, startedAt, completedAt time.Time) (readinessReport, error) {
	completeReadinessRun(&report, startedAt, completedAt)
	if len(failedPhases) > 0 {
		report.Status = readinessStatusFailed
		report.Summary = fmt.Sprintf("readiness phases failed: %s", strings.Join(failedPhases, ","))
		refreshWarningState(&report)
		return report, errors.New(report.Summary)
	}

	report.Status = readinessStatusOK
	report.Summary = "all readiness phases passed"
	refreshWarningState(&report)
	if report.WarningPolicyFailed {
		report.Status = readinessStatusFailed
		report.Summary = fmt.Sprintf("warning policy failed: %s", strings.Join(report.MatchedWarningCodes, ","))
		return report, errors.New(report.Summary)
	}
	return report, nil
}

func completeReadinessRun(report *readinessReport, startedAt, completedAt time.Time) {
	start := startedAt.UTC()
	end := completedAt.UTC()
	if end.Before(start) {
		end = start
	}
	duration := end.Sub(start)
	report.StartedAt = &start
	report.CompletedAt = &end
	report.DurationMS = duration.Milliseconds()
}

func setReadinessPhase(report *readinessReport, name, status, summary string) {
	for i := range report.Phases {
		if report.Phases[i].Name == name {
			report.Phases[i].Status = status
			report.Phases[i].Summary = fallbackString(summary)
			return
		}
	}
	report.Phases = append(report.Phases, readinessPhaseResult{
		Name:    name,
		Status:  status,
		Summary: fallbackString(summary),
	})
}

func completeReadinessPhase(report *readinessReport, name, status, summary string, startedAt, completedAt time.Time) {
	setReadinessPhase(report, name, status, summary)
	for i := range report.Phases {
		if report.Phases[i].Name != name {
			continue
		}
		start := startedAt.UTC()
		end := completedAt.UTC()
		if end.Before(start) {
			end = start
		}
		duration := end.Sub(start)
		report.Phases[i].StartedAt = &start
		report.Phases[i].CompletedAt = &end
		report.Phases[i].DurationMS = duration.Milliseconds()
		return
	}
}

func applySlowWarnings(report *readinessReport) {
	report.Warnings = nil
	for i := range report.Phases {
		report.Phases[i].WarningCodes = nil
	}

	if report.SlowRunThresholdMS > 0 && report.DurationMS > report.SlowRunThresholdMS {
		report.Warnings = append(report.Warnings, readinessWarning{
			Code:             "WARN_READINESS_RUN_SLOW",
			Summary:          fmt.Sprintf("overall readiness run exceeded %dms threshold: %dms", report.SlowRunThresholdMS, report.DurationMS),
			Scope:            "run",
			ThresholdMS:      report.SlowRunThresholdMS,
			ActualDurationMS: report.DurationMS,
		})
	}

	if report.SlowPhaseThresholdMS <= 0 {
		return
	}
	for i := range report.Phases {
		phase := &report.Phases[i]
		if phase.Status == readinessStatusNotRun || phase.DurationMS <= report.SlowPhaseThresholdMS {
			continue
		}
		phase.WarningCodes = append(phase.WarningCodes, "WARN_READINESS_PHASE_SLOW")
		report.Warnings = append(report.Warnings, readinessWarning{
			Code:             "WARN_READINESS_PHASE_SLOW",
			Summary:          fmt.Sprintf("phase %s exceeded %dms threshold: %dms", phase.Name, report.SlowPhaseThresholdMS, phase.DurationMS),
			Scope:            "phase",
			Phase:            phase.Name,
			ThresholdMS:      report.SlowPhaseThresholdMS,
			ActualDurationMS: phase.DurationMS,
		})
	}
}

func refreshWarningState(report *readinessReport) {
	applySlowWarnings(report)
	report.AllWarningCodes = collectAllWarningCodes(*report)
	report.MatchedWarningCodes = matchWarningCodes(report.FailOnWarningCodes, report.AllWarningCodes)
	report.WarningPolicyFailed = len(report.MatchedWarningCodes) > 0
}

func collectAllWarningCodes(report readinessReport) []string {
	codes := make([]string, 0, len(report.Warnings))
	for _, warning := range report.Warnings {
		codes = appendNormalizedWarningCodes(codes, warning.Code)
	}
	if report.Doctor != nil {
		for _, warning := range report.Doctor.Follow.Warnings {
			codes = appendNormalizedWarningCodes(codes, warning.Code)
		}
	}
	return codes
}

func matchWarningCodes(configured []string, available []string) []string {
	if len(configured) == 0 || len(available) == 0 {
		return nil
	}
	availableSet := make(map[string]struct{}, len(available))
	for _, code := range available {
		normalized := normalizeWarningCode(code)
		if normalized == "" {
			continue
		}
		availableSet[normalized] = struct{}{}
	}
	matched := make([]string, 0, len(configured))
	for _, code := range configured {
		normalized := normalizeWarningCode(code)
		if normalized == "" {
			continue
		}
		if _, ok := availableSet[normalized]; ok {
			matched = appendNormalizedWarningCodes(matched, normalized)
		}
	}
	return matched
}

func validateDoctor(report doctorReport) error {
	if report.Status != "ok" {
		return fmt.Errorf("doctor status mismatch: got %q", report.Status)
	}
	if report.MCP.Transport != "stdio" {
		return fmt.Errorf("doctor mcp transport mismatch: got %q", report.MCP.Transport)
	}
	if report.MCP.ToolCount != 11 {
		return fmt.Errorf("doctor tool count mismatch: got %d want 11", report.MCP.ToolCount)
	}
	if !report.Runtime.ForeignKeys {
		return errors.New("doctor foreign_keys=false")
	}
	if !report.Runtime.RequiredSchemaOK {
		return errors.New("doctor required_schema_ok=false")
	}
	if !report.Runtime.FTSReady {
		return errors.New("doctor fts_ready=false")
	}
	if report.Migrations.Pending != 0 {
		return fmt.Errorf("doctor migrations pending: %d", report.Migrations.Pending)
	}
	if !report.Audit.NoteProvenanceReady {
		return errors.New("doctor note_provenance_ready=false")
	}
	if !report.Audit.ExclusionAuditReady {
		return errors.New("doctor exclusion_audit_ready=false")
	}
	if !report.Audit.ImportAuditReady {
		return errors.New("doctor import_audit_ready=false")
	}
	return nil
}

func writeReadinessOutput(w io.Writer, report readinessReport, options readinessOptions) error {
	if options.JSON {
		return writeReadinessJSON(w, report)
	}
	return writeReadinessSummary(w, report)
}

func writeReadinessSummary(w io.Writer, report readinessReport) error {
	doctor := report.Doctor
	lines := []string{
		fmt.Sprintf("readiness check %s", statusLabel(report.Status)),
		fmt.Sprintf("status=%s", report.Status),
		fmt.Sprintf("summary=%s", fallbackString(report.Summary)),
		fmt.Sprintf("keep_going=%t", report.KeepGoing),
		fmt.Sprintf("slow_run_threshold_ms=%d", report.SlowRunThresholdMS),
		fmt.Sprintf("slow_phase_threshold_ms=%d", report.SlowPhaseThresholdMS),
		fmt.Sprintf("fail_on_warning_codes=%s", csvOrNone(report.FailOnWarningCodes)),
		fmt.Sprintf("started_at=%s", timePointerOrNone(report.StartedAt)),
		fmt.Sprintf("completed_at=%s", timePointerOrNone(report.CompletedAt)),
		fmt.Sprintf("duration_ms=%d", report.DurationMS),
		fmt.Sprintf("warning_codes=%s", readinessWarningCodes(report.Warnings)),
		fmt.Sprintf("warning_count=%d", len(report.Warnings)),
		fmt.Sprintf("all_warning_codes=%s", csvOrNone(report.AllWarningCodes)),
		fmt.Sprintf("matched_warning_codes=%s", csvOrNone(report.MatchedWarningCodes)),
		fmt.Sprintf("warning_policy_failed=%t", report.WarningPolicyFailed),
		fmt.Sprintf("doctor_status=%s", doctorFieldString(doctor, func(value doctorReport) string { return fallbackString(value.Status) })),
		fmt.Sprintf("doctor_mcp_transport=%s", doctorFieldString(doctor, func(value doctorReport) string { return fallbackString(value.MCP.Transport) })),
		fmt.Sprintf("doctor_mcp_tool_count=%d", doctorFieldInt(doctor, func(value doctorReport) int { return value.MCP.ToolCount })),
		fmt.Sprintf("doctor_schema_ready=%t", doctorFieldBool(doctor, func(value doctorReport) bool { return value.Runtime.RequiredSchemaOK })),
		fmt.Sprintf("doctor_fts_ready=%t", doctorFieldBool(doctor, func(value doctorReport) bool { return value.Runtime.FTSReady })),
		fmt.Sprintf("doctor_migrations_pending=%d", doctorFieldInt(doctor, func(value doctorReport) int { return value.Migrations.Pending })),
		fmt.Sprintf("doctor_provenance_ready=%t", doctorFieldBool(doctor, func(value doctorReport) bool { return value.Audit.NoteProvenanceReady })),
		fmt.Sprintf("doctor_exclusion_audit_ready=%t", doctorFieldBool(doctor, func(value doctorReport) bool { return value.Audit.ExclusionAuditReady })),
		fmt.Sprintf("doctor_import_audit_ready=%t", doctorFieldBool(doctor, func(value doctorReport) bool { return value.Audit.ImportAuditReady })),
		fmt.Sprintf("doctor_follow_imports_health_present=%t", doctorFieldBool(doctor, func(value doctorReport) bool { return value.Follow.HealthPresent })),
		fmt.Sprintf("doctor_follow_imports_status=%s", doctorFieldString(doctor, func(value doctorReport) string { return fallbackString(value.Follow.Status) })),
		fmt.Sprintf("doctor_follow_imports_source=%s", doctorFieldString(doctor, func(value doctorReport) string { return fallbackString(value.Follow.Source) })),
		fmt.Sprintf("doctor_follow_imports_input_count=%d", doctorFieldInt(doctor, func(value doctorReport) int { return value.Follow.InputCount })),
		fmt.Sprintf("doctor_follow_imports_last_updated_at=%s", doctorFieldString(doctor, func(value doctorReport) string { return timePointerOrNone(value.Follow.LastUpdatedAt) })),
		fmt.Sprintf("doctor_follow_imports_continuous=%t", doctorFieldBool(doctor, func(value doctorReport) bool { return value.Follow.Continuous })),
		fmt.Sprintf("doctor_follow_imports_poll_interval_seconds=%d", doctorFieldInt64(doctor, func(value doctorReport) int64 { return value.Follow.PollIntervalSeconds })),
		fmt.Sprintf("doctor_follow_imports_snapshot_age_seconds=%d", doctorFieldInt64(doctor, func(value doctorReport) int64 { return value.Follow.SnapshotAgeSeconds })),
		fmt.Sprintf("doctor_follow_imports_health_stale=%t", doctorFieldBool(doctor, func(value doctorReport) bool { return value.Follow.HealthStale })),
		fmt.Sprintf("doctor_follow_imports_requested_watch_mode=%s", doctorFieldString(doctor, func(value doctorReport) string { return fallbackString(value.Follow.RequestedWatchMode) })),
		fmt.Sprintf("doctor_follow_imports_active_watch_mode=%s", doctorFieldString(doctor, func(value doctorReport) string { return fallbackString(value.Follow.ActiveWatchMode) })),
		fmt.Sprintf("doctor_follow_imports_watch_fallbacks=%d", doctorFieldInt(doctor, func(value doctorReport) int { return value.Follow.WatchFallbacks })),
		fmt.Sprintf("doctor_follow_imports_watch_transitions=%d", doctorFieldInt(doctor, func(value doctorReport) int { return value.Follow.WatchTransitions })),
		fmt.Sprintf("doctor_follow_imports_watch_poll_catchups=%d", doctorFieldInt(doctor, func(value doctorReport) int { return value.Follow.WatchPollCatchups })),
		fmt.Sprintf("doctor_follow_imports_watch_poll_catchup_bytes=%d", doctorFieldInt(doctor, func(value doctorReport) int { return value.Follow.WatchPollCatchupBytes })),
		fmt.Sprintf("doctor_follow_imports_warning_codes=%s", doctorFieldString(doctor, func(value doctorReport) string { return warningCodes(value.Follow.Warnings) })),
		fmt.Sprintf("%s_status=%s", readinessPhaseStdio, report.Stdio.Status),
		fmt.Sprintf("%s_summary=%s", readinessPhaseStdio, fallbackString(report.Stdio.Summary)),
		fmt.Sprintf("%s_status=%s", readinessPhaseHTTP, report.HTTP.Status),
		fmt.Sprintf("%s_summary=%s", readinessPhaseHTTP, fallbackString(report.HTTP.Summary)),
	}
	for _, phase := range report.Phases {
		lines = append(lines,
			fmt.Sprintf("phase_%s_status=%s", phase.Name, phase.Status),
			fmt.Sprintf("phase_%s_summary=%s", phase.Name, fallbackString(phase.Summary)),
			fmt.Sprintf("phase_%s_started_at=%s", phase.Name, timePointerOrNone(phase.StartedAt)),
			fmt.Sprintf("phase_%s_completed_at=%s", phase.Name, timePointerOrNone(phase.CompletedAt)),
			fmt.Sprintf("phase_%s_duration_ms=%d", phase.Name, phase.DurationMS),
			fmt.Sprintf("phase_%s_warning_codes=%s", phase.Name, csvOrNone(phase.WarningCodes)),
		)
	}
	for i, warning := range report.Warnings {
		lines = append(lines,
			fmt.Sprintf("warning_%d_code=%s", i, warning.Code),
			fmt.Sprintf("warning_%d_summary=%s", i, fallbackString(warning.Summary)),
			fmt.Sprintf("warning_%d_scope=%s", i, warning.Scope),
			fmt.Sprintf("warning_%d_phase=%s", i, fallbackString(warning.Phase)),
			fmt.Sprintf("warning_%d_threshold_ms=%d", i, warning.ThresholdMS),
			fmt.Sprintf("warning_%d_actual_duration_ms=%d", i, warning.ActualDurationMS),
		)
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func writeReadinessJSON(w io.Writer, report readinessReport) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func firstLine(value string) string {
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "none"
}

func fallbackString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return stringNone
	}
	return value
}

func timePointerOrNone(value *time.Time) string {
	if value == nil {
		return stringNone
	}
	return value.UTC().Format(time.RFC3339)
}

func warningCodes(warnings []doctorWarning) string {
	if len(warnings) == 0 {
		return stringNone
	}
	codes := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		code := strings.TrimSpace(warning.Code)
		if code == "" {
			continue
		}
		codes = append(codes, code)
	}
	if len(codes) == 0 {
		return stringNone
	}
	return strings.Join(codes, ",")
}

func readinessWarningCodes(warnings []readinessWarning) string {
	if len(warnings) == 0 {
		return stringNone
	}
	codes := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		code := strings.TrimSpace(warning.Code)
		if code == "" {
			continue
		}
		codes = append(codes, code)
	}
	return csvOrNone(codes)
}

func csvOrNone(values []string) string {
	if len(values) == 0 {
		return stringNone
	}
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		filtered = append(filtered, value)
	}
	if len(filtered) == 0 {
		return stringNone
	}
	return strings.Join(filtered, ",")
}

func summarizeCommandFailure(baseSummary, stdout, stderr string) string {
	for _, candidate := range []string{firstLine(stderr), firstLine(stdout)} {
		if candidate != stringNone {
			return fmt.Sprintf("%s (%s)", baseSummary, candidate)
		}
	}
	return baseSummary
}

func doctorFieldString(report *doctorReport, getter func(doctorReport) string) string {
	if report == nil {
		return stringNone
	}
	return fallbackString(getter(*report))
}

func doctorFieldInt(report *doctorReport, getter func(doctorReport) int) int {
	if report == nil {
		return 0
	}
	return getter(*report)
}

func doctorFieldInt64(report *doctorReport, getter func(doctorReport) int64) int64 {
	if report == nil {
		return 0
	}
	return getter(*report)
}

func doctorFieldBool(report *doctorReport, getter func(doctorReport) bool) bool {
	if report == nil {
		return false
	}
	return getter(*report)
}

func statusLabel(status string) string {
	if status == readinessStatusOK {
		return "passed"
	}
	return "failed"
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
