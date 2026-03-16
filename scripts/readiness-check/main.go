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
	"strings"
	"time"
)

const stringNone = "none"

type readinessOptions struct {
	JSON bool
}

type readinessReport struct {
	Status  string                 `json:"status"`
	Summary string                 `json:"summary"`
	Doctor  *doctorReport          `json:"doctor,omitempty"`
	Stdio   readinessSmokeTest     `json:"stdio_mcp_smoke_test"`
	HTTP    readinessSmokeTest     `json:"http_mcp_smoke_test"`
	Phases  []readinessPhaseResult `json:"phases"`
}

type readinessSmokeTest struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

type readinessPhaseResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
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

	report, runErr := runReadinessCheck(ctx, repoRoot)
	if err := writeReadinessOutput(os.Stdout, report, options); err != nil {
		failf("write readiness summary: %v", err)
	}
	if runErr != nil {
		os.Exit(1)
	}
}

func parseOptions(args []string) (readinessOptions, error) {
	var options readinessOptions
	for _, arg := range args {
		switch strings.TrimSpace(arg) {
		case "":
			continue
		case "--json":
			options.JSON = true
		default:
			return readinessOptions{}, fmt.Errorf("unknown readiness-check flag %q", arg)
		}
	}
	return options, nil
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

func runReadinessCheck(ctx context.Context, repoRoot string) (readinessReport, error) {
	report := newReadinessReport()

	doctorStdout, doctorStderr, err := runGo(ctx, repoRoot, "run", "./cmd/codex-mem", "doctor", "--json")
	if err != nil {
		failure := fmt.Errorf("doctor check failed: %v", err)
		report.Status = readinessStatusFailed
		report.Summary = failure.Error()
		setReadinessPhase(&report, readinessPhaseDoctor, readinessStatusFailed, summarizeCommandFailure(failure.Error(), doctorStdout, doctorStderr))
		return report, failure
	}

	var doctor doctorReport
	if err := json.Unmarshal([]byte(doctorStdout), &doctor); err != nil {
		failure := fmt.Errorf("decode doctor JSON: %v", err)
		report.Status = readinessStatusFailed
		report.Summary = failure.Error()
		setReadinessPhase(&report, readinessPhaseDoctor, readinessStatusFailed, summarizeCommandFailure(failure.Error(), doctorStdout, doctorStderr))
		return report, failure
	}
	if err := validateDoctor(doctor); err != nil {
		report.Doctor = &doctor
		report.Status = readinessStatusFailed
		report.Summary = err.Error()
		setReadinessPhase(&report, readinessPhaseDoctor, readinessStatusFailed, err.Error())
		return report, err
	}
	report.Doctor = &doctor
	setReadinessPhase(&report, readinessPhaseDoctor, readinessStatusOK, "doctor --json passed")

	smokeStdout, smokeStderr, err := runGo(ctx, repoRoot, "run", "./scripts/mcp-smoke-test")
	if err != nil {
		failure := fmt.Errorf("stdio mcp smoke test failed: %v", err)
		report.Stdio = readinessSmokeTest{
			Status:  readinessStatusFailed,
			Summary: summarizeCommandFailure(failure.Error(), smokeStdout, smokeStderr),
		}
		report.Status = readinessStatusFailed
		report.Summary = failure.Error()
		setReadinessPhase(&report, readinessPhaseStdio, readinessStatusFailed, report.Stdio.Summary)
		return report, failure
	}
	if !strings.Contains(smokeStdout, "mcp smoke test passed") {
		failure := errors.New("stdio mcp smoke test did not report success")
		report.Stdio = readinessSmokeTest{
			Status:  readinessStatusFailed,
			Summary: summarizeCommandFailure(failure.Error(), smokeStdout, smokeStderr),
		}
		report.Status = readinessStatusFailed
		report.Summary = failure.Error()
		setReadinessPhase(&report, readinessPhaseStdio, readinessStatusFailed, report.Stdio.Summary)
		return report, failure
	}
	report.Stdio = readinessSmokeTest{
		Status:  readinessStatusOK,
		Summary: firstLine(smokeStdout),
	}
	setReadinessPhase(&report, readinessPhaseStdio, readinessStatusOK, report.Stdio.Summary)

	httpSmokeStdout, httpSmokeStderr, err := runGo(ctx, repoRoot, "run", "./scripts/http-mcp-smoke-test")
	if err != nil {
		failure := fmt.Errorf("http mcp smoke test failed: %v", err)
		report.HTTP = readinessSmokeTest{
			Status:  readinessStatusFailed,
			Summary: summarizeCommandFailure(failure.Error(), httpSmokeStdout, httpSmokeStderr),
		}
		report.Status = readinessStatusFailed
		report.Summary = failure.Error()
		setReadinessPhase(&report, readinessPhaseHTTP, readinessStatusFailed, report.HTTP.Summary)
		return report, failure
	}
	if !strings.Contains(httpSmokeStdout, "http mcp smoke test passed") {
		failure := errors.New("http mcp smoke test did not report success")
		report.HTTP = readinessSmokeTest{
			Status:  readinessStatusFailed,
			Summary: summarizeCommandFailure(failure.Error(), httpSmokeStdout, httpSmokeStderr),
		}
		report.Status = readinessStatusFailed
		report.Summary = failure.Error()
		setReadinessPhase(&report, readinessPhaseHTTP, readinessStatusFailed, report.HTTP.Summary)
		return report, failure
	}
	report.HTTP = readinessSmokeTest{
		Status:  readinessStatusOK,
		Summary: firstLine(httpSmokeStdout),
	}
	setReadinessPhase(&report, readinessPhaseHTTP, readinessStatusOK, report.HTTP.Summary)

	report.Status = readinessStatusOK
	report.Summary = "all readiness phases passed"
	return report, nil
}

func newReadinessReport() readinessReport {
	report := readinessReport{
		Status:  readinessStatusNotRun,
		Summary: "readiness phases have not run",
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
	return report
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
