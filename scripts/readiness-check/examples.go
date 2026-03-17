package main

import (
	"bytes"
	"os"
	"path/filepath"
	"time"
)

const (
	readinessExampleDirName             = "testdata"
	readinessFollowSourceWatcherImport  = "watcher_import"
	readinessSummaryAllPhasesPassed     = "all readiness phases passed"
	readinessSummaryWarningPolicyFailed = "warning policy failed: WARN_FOLLOW_IMPORTS_HEALTH_STALE"
)

type readinessExampleFixture struct {
	Name         string
	RelativePath string
	JSON         bool
	Report       readinessReport
}

func readinessExampleFixtures() []readinessExampleFixture {
	return []readinessExampleFixture{
		{
			Name:         "slow-ci-text",
			RelativePath: "example-slow-ci-success.txt",
			JSON:         false,
			Report:       exampleSlowCISuccessReport(),
		},
		{
			Name:         "ci-json",
			RelativePath: "example-ci-success.json",
			JSON:         true,
			Report:       exampleCISuccessReport(),
		},
		{
			Name:         "release-warning-failure-text",
			RelativePath: "example-release-warning-failure.txt",
			JSON:         false,
			Report:       exampleReleaseWarningFailureReport(),
		},
	}
}

func renderReadinessExample(report readinessReport, jsonOutput bool) ([]byte, error) {
	var buffer bytes.Buffer
	var err error
	if jsonOutput {
		err = writeReadinessJSON(&buffer, report)
	} else {
		err = writeReadinessSummary(&buffer, report)
	}
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func writeReadinessExampleFixtures(baseDir string) ([]string, error) {
	fixtures := readinessExampleFixtures()
	writtenPaths := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		body, err := renderReadinessExample(fixture.Report, fixture.JSON)
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

func exampleCISuccessReport() readinessReport {
	startedAt := time.Date(2026, time.March, 17, 9, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(3 * time.Second)
	doctorStartedAt := startedAt
	doctorCompletedAt := doctorStartedAt.Add(400 * time.Millisecond)
	stdioCompletedAt := doctorCompletedAt.Add(600 * time.Millisecond)
	httpCompletedAt := stdioCompletedAt.Add(800 * time.Millisecond)

	report := readinessReport{
		Status:               readinessStatusOK,
		Summary:              readinessSummaryAllPhasesPassed,
		KeepGoing:            false,
		PolicyProfile:        readinessPolicyProfileCI,
		SlowRunThresholdMS:   8000,
		SlowPhaseThresholdMS: 1000,
		StartedAt:            &startedAt,
		CompletedAt:          &completedAt,
		DurationMS:           3000,
		Doctor:               exampleHealthyDoctorReport(),
		Stdio: readinessSmokeTest{
			Status:  readinessStatusOK,
			Summary: "mcp smoke test passed",
		},
		HTTP: readinessSmokeTest{
			Status:  readinessStatusOK,
			Summary: "http mcp smoke test passed",
		},
		Phases: []readinessPhaseResult{
			{Name: readinessPhaseDoctor, Status: readinessStatusOK, Summary: "doctor --json passed", StartedAt: &doctorStartedAt, CompletedAt: &doctorCompletedAt, DurationMS: 400},
			{Name: readinessPhaseStdio, Status: readinessStatusOK, Summary: "mcp smoke test passed", StartedAt: &doctorCompletedAt, CompletedAt: &stdioCompletedAt, DurationMS: 600},
			{Name: readinessPhaseHTTP, Status: readinessStatusOK, Summary: "http mcp smoke test passed", StartedAt: &stdioCompletedAt, CompletedAt: &httpCompletedAt, DurationMS: 800},
		},
	}
	refreshWarningState(&report)
	return report
}

func exampleSlowCISuccessReport() readinessReport {
	report := exampleCISuccessReport()
	report.PolicyProfile = readinessPolicyProfileSlowCI
	report.SlowRunThresholdMS = 20000
	report.SlowPhaseThresholdMS = 4000
	refreshWarningState(&report)
	return report
}

func exampleReleaseWarningFailureReport() readinessReport {
	report := exampleCISuccessReport()
	report.Status = readinessStatusFailed
	report.PolicyProfile = readinessPolicyProfileRelease
	report.FailOnWarningCodes = []string{readinessWarnFollowHealthStale}
	report.Doctor = exampleHealthyDoctorReport()
	report.Doctor.Follow.HealthStale = true
	report.Doctor.Follow.Warnings = []doctorWarning{{Code: readinessWarnFollowHealthStale}}
	refreshWarningState(&report)
	report.Summary = readinessSummaryWarningPolicyFailed
	return report
}

func exampleHealthyDoctorReport() *doctorReport {
	updatedAt := time.Date(2026, time.March, 17, 8, 59, 55, 0, time.UTC)
	doctor := &doctorReport{
		Status: "ok",
	}
	doctor.Runtime.ForeignKeys = true
	doctor.Runtime.RequiredSchemaOK = true
	doctor.Runtime.FTSReady = true
	doctor.Migrations.Pending = 0
	doctor.Audit.NoteProvenanceReady = true
	doctor.Audit.ExclusionAuditReady = true
	doctor.Audit.ImportAuditReady = true
	doctor.Follow.HealthPresent = true
	doctor.Follow.LastUpdatedAt = &updatedAt
	doctor.Follow.Status = "ok"
	doctor.Follow.Source = readinessFollowSourceWatcherImport
	doctor.Follow.InputCount = 1
	doctor.Follow.Continuous = true
	doctor.Follow.PollIntervalSeconds = 5
	doctor.Follow.SnapshotAgeSeconds = 5
	doctor.Follow.HealthStale = false
	doctor.Follow.RequestedWatchMode = "auto"
	doctor.Follow.ActiveWatchMode = "notify"
	doctor.MCP.Transport = readinessMCPTransportStdio
	doctor.MCP.ToolCount = 11
	return doctor
}
