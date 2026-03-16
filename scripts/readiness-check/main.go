// Package main runs the local readiness smoke checks for codex-mem.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

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
	MCP struct {
		Transport string `json:"transport"`
		ToolCount int    `json:"tool_count"`
	} `json:"mcp"`
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

	fmt.Println("readiness check passed")
	fmt.Printf("doctor_status=%s\n", doctor.Status)
	fmt.Printf("doctor_mcp_transport=%s\n", doctor.MCP.Transport)
	fmt.Printf("doctor_mcp_tool_count=%d\n", doctor.MCP.ToolCount)
	fmt.Printf("doctor_schema_ready=%t\n", doctor.Runtime.RequiredSchemaOK)
	fmt.Printf("doctor_fts_ready=%t\n", doctor.Runtime.FTSReady)
	fmt.Printf("doctor_migrations_pending=%d\n", doctor.Migrations.Pending)
	fmt.Printf("doctor_provenance_ready=%t\n", doctor.Audit.NoteProvenanceReady)
	fmt.Printf("doctor_exclusion_audit_ready=%t\n", doctor.Audit.ExclusionAuditReady)
	fmt.Printf("doctor_import_audit_ready=%t\n", doctor.Audit.ImportAuditReady)
	fmt.Printf("stdio_mcp_smoke_test=%s\n", firstLine(smokeStdout))
	fmt.Printf("http_mcp_smoke_test=%s\n", firstLine(httpSmokeStdout))
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
	if report.MCP.ToolCount != 10 {
		failf("doctor tool count mismatch: got %d want 10", report.MCP.ToolCount)
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

func firstLine(value string) string {
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "none"
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
