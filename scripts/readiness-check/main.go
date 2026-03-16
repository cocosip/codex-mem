// Package main runs the local readiness smoke checks for codex-mem.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const stringNone = "none"

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

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	repoRoot, err := os.Getwd()
	if err != nil {
		failf("resolve working directory: %v", err)
	}

	doctorStdout, doctorStderr, err := runGo(ctx, repoRoot, "run", "./cmd/codex-mem", "doctor", "--json")
	if err != nil {
		failf("doctor check failed: %v\n%s", err, strings.TrimSpace(doctorStderr))
	}

	var doctor doctorReport
	if err := json.Unmarshal([]byte(doctorStdout), &doctor); err != nil {
		failf("decode doctor JSON: %v\nstdout:\n%s\nstderr:\n%s", err, doctorStdout, doctorStderr)
	}
	assertDoctor(doctor)

	smokeStdout, smokeStderr, err := runGo(ctx, repoRoot, "run", "./scripts/mcp-smoke-test")
	if err != nil {
		failf("stdio mcp smoke test failed: %v\nstdout:\n%s\nstderr:\n%s", err, smokeStdout, smokeStderr)
	}
	if !strings.Contains(smokeStdout, "mcp smoke test passed") {
		failf("stdio mcp smoke test did not report success\nstdout:\n%s\nstderr:\n%s", smokeStdout, smokeStderr)
	}

	httpSmokeStdout, httpSmokeStderr, err := runGo(ctx, repoRoot, "run", "./scripts/http-mcp-smoke-test")
	if err != nil {
		failf("http mcp smoke test failed: %v\nstdout:\n%s\nstderr:\n%s", err, httpSmokeStdout, httpSmokeStderr)
	}
	if !strings.Contains(httpSmokeStdout, "http mcp smoke test passed") {
		failf("http mcp smoke test did not report success\nstdout:\n%s\nstderr:\n%s", httpSmokeStdout, httpSmokeStderr)
	}

	if err := writeReadinessSummary(os.Stdout, doctor, smokeStdout, httpSmokeStdout); err != nil {
		failf("write readiness summary: %v", err)
	}
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

func assertDoctor(report doctorReport) {
	if report.Status != "ok" {
		failf("doctor status mismatch: got %q", report.Status)
	}
	if report.MCP.Transport != "stdio" {
		failf("doctor mcp transport mismatch: got %q", report.MCP.Transport)
	}
	if report.MCP.ToolCount != 11 {
		failf("doctor tool count mismatch: got %d want 11", report.MCP.ToolCount)
	}
	if !report.Runtime.ForeignKeys {
		failf("doctor foreign_keys=false")
	}
	if !report.Runtime.RequiredSchemaOK {
		failf("doctor required_schema_ok=false")
	}
	if !report.Runtime.FTSReady {
		failf("doctor fts_ready=false")
	}
	if report.Migrations.Pending != 0 {
		failf("doctor migrations pending: %d", report.Migrations.Pending)
	}
	if !report.Audit.NoteProvenanceReady {
		failf("doctor note_provenance_ready=false")
	}
	if !report.Audit.ExclusionAuditReady {
		failf("doctor exclusion_audit_ready=false")
	}
	if !report.Audit.ImportAuditReady {
		failf("doctor import_audit_ready=false")
	}
}

func writeReadinessSummary(w io.Writer, doctor doctorReport, smokeStdout, httpSmokeStdout string) error {
	lines := []string{
		"readiness check passed",
		fmt.Sprintf("doctor_status=%s", doctor.Status),
		fmt.Sprintf("doctor_mcp_transport=%s", doctor.MCP.Transport),
		fmt.Sprintf("doctor_mcp_tool_count=%d", doctor.MCP.ToolCount),
		fmt.Sprintf("doctor_schema_ready=%t", doctor.Runtime.RequiredSchemaOK),
		fmt.Sprintf("doctor_fts_ready=%t", doctor.Runtime.FTSReady),
		fmt.Sprintf("doctor_migrations_pending=%d", doctor.Migrations.Pending),
		fmt.Sprintf("doctor_provenance_ready=%t", doctor.Audit.NoteProvenanceReady),
		fmt.Sprintf("doctor_exclusion_audit_ready=%t", doctor.Audit.ExclusionAuditReady),
		fmt.Sprintf("doctor_import_audit_ready=%t", doctor.Audit.ImportAuditReady),
		fmt.Sprintf("doctor_follow_imports_health_present=%t", doctor.Follow.HealthPresent),
		fmt.Sprintf("doctor_follow_imports_status=%s", fallbackString(doctor.Follow.Status)),
		fmt.Sprintf("doctor_follow_imports_source=%s", fallbackString(doctor.Follow.Source)),
		fmt.Sprintf("doctor_follow_imports_input_count=%d", doctor.Follow.InputCount),
		fmt.Sprintf("doctor_follow_imports_last_updated_at=%s", timePointerOrNone(doctor.Follow.LastUpdatedAt)),
		fmt.Sprintf("doctor_follow_imports_continuous=%t", doctor.Follow.Continuous),
		fmt.Sprintf("doctor_follow_imports_poll_interval_seconds=%d", doctor.Follow.PollIntervalSeconds),
		fmt.Sprintf("doctor_follow_imports_snapshot_age_seconds=%d", doctor.Follow.SnapshotAgeSeconds),
		fmt.Sprintf("doctor_follow_imports_health_stale=%t", doctor.Follow.HealthStale),
		fmt.Sprintf("doctor_follow_imports_requested_watch_mode=%s", fallbackString(doctor.Follow.RequestedWatchMode)),
		fmt.Sprintf("doctor_follow_imports_active_watch_mode=%s", fallbackString(doctor.Follow.ActiveWatchMode)),
		fmt.Sprintf("doctor_follow_imports_watch_fallbacks=%d", doctor.Follow.WatchFallbacks),
		fmt.Sprintf("doctor_follow_imports_watch_transitions=%d", doctor.Follow.WatchTransitions),
		fmt.Sprintf("doctor_follow_imports_watch_poll_catchups=%d", doctor.Follow.WatchPollCatchups),
		fmt.Sprintf("doctor_follow_imports_watch_poll_catchup_bytes=%d", doctor.Follow.WatchPollCatchupBytes),
		fmt.Sprintf("doctor_follow_imports_warning_codes=%s", warningCodes(doctor.Follow.Warnings)),
		fmt.Sprintf("stdio_mcp_smoke_test=%s", firstLine(smokeStdout)),
		fmt.Sprintf("http_mcp_smoke_test=%s", firstLine(httpSmokeStdout)),
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
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

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
