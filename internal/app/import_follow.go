package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codex-mem/internal/config"
	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/imports"
)

type followImportsOptions struct {
	Source             imports.Source
	InputPath          string
	StatePath          string
	FailedOutputPath   string
	FailedManifestPath string
	CWD                string
	BranchName         string
	RepoRemote         string
	Task               string
	JSON               bool
	Once               bool
	PollInterval       time.Duration
}

// FollowImportsInput configures one polling-based import-follow pass.
type FollowImportsInput struct {
	Source             imports.Source
	InputPath          string
	StatePath          string
	CWD                string
	BranchName         string
	RepoRemote         string
	Task               string
	FailedOutputPath   string
	FailedManifestPath string
}

// FollowImportsReport summarizes one import-follow polling pass.
type FollowImportsReport = followImportsReport

type followImportsReport struct {
	Status        string               `json:"status"`
	Source        string               `json:"source"`
	Input         string               `json:"input"`
	StateFile     string               `json:"state_file"`
	Offset        int64                `json:"offset"`
	ConsumedBytes int                  `json:"consumed_bytes"`
	PendingBytes  int                  `json:"pending_bytes"`
	Truncated     bool                 `json:"truncated,omitempty"`
	Batch         *ingestImportsReport `json:"batch,omitempty"`
	BatchError    *common.ErrorPayload `json:"batch_error,omitempty"`
}

type followImportsState struct {
	Input     string    `json:"input"`
	Offset    int64     `json:"offset"`
	UpdatedAt time.Time `json:"updated_at"`
}

func runFollowImports(ctx context.Context, cfg config.Config, stdout io.Writer, args []string) error {
	options, err := parseFollowImportsOptions(args)
	if err != nil {
		return err
	}

	instance, err := New(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		_ = instance.Close()
	}()

	input := FollowImportsInput{
		Source:             options.Source,
		InputPath:          options.InputPath,
		StatePath:          options.StatePath,
		CWD:                options.CWD,
		BranchName:         options.BranchName,
		RepoRemote:         options.RepoRemote,
		Task:               options.Task,
		FailedOutputPath:   options.FailedOutputPath,
		FailedManifestPath: options.FailedManifestPath,
	}

	runOnce := func() error {
		report, err := instance.FollowImportsOnce(ctx, input)
		if err != nil {
			return err
		}
		if options.Once || report.Status != "idle" {
			return writeFollowImportsReport(stdout, report, options.JSON)
		}
		return nil
	}

	if err := runOnce(); err != nil {
		return err
	}
	if options.Once {
		return nil
	}

	ticker := time.NewTicker(options.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := runOnce(); err != nil {
				return err
			}
		}
	}
}

func parseFollowImportsOptions(args []string) (followImportsOptions, error) {
	options := followImportsOptions{
		PollInterval: 5 * time.Second,
	}

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch arg {
		case "":
			continue
		case "--source":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.Source = imports.Source(value)
			i = next
		case "--input":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.InputPath = value
			i = next
		case "--state-file":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.StatePath = value
			i = next
		case "--failed-output":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.FailedOutputPath = value
			i = next
		case "--failed-manifest":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.FailedManifestPath = value
			i = next
		case "--cwd":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.CWD = value
			i = next
		case "--branch-name":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.BranchName = value
			i = next
		case "--repo-remote":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.RepoRemote = value
			i = next
		case "--task":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.Task = value
			i = next
		case "--poll-interval":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			interval, err := time.ParseDuration(value)
			if err != nil {
				return followImportsOptions{}, fmt.Errorf("invalid follow-imports poll interval %q", value)
			}
			options.PollInterval = interval
			i = next
		case "--json":
			options.JSON = true
		case "--once":
			options.Once = true
		default:
			return followImportsOptions{}, fmt.Errorf("unknown follow-imports flag %q", arg)
		}
	}

	if err := validateFollowImportsOptions(options); err != nil {
		return followImportsOptions{}, err
	}
	return options, nil
}

func validateFollowImportsOptions(options followImportsOptions) error {
	if err := options.Source.Validate(); err != nil {
		if strings.TrimSpace(string(options.Source)) == "" {
			return fmt.Errorf("follow-imports source is required")
		}
		return err
	}
	if strings.TrimSpace(options.InputPath) == "" {
		return fmt.Errorf("follow-imports input is required")
	}
	if options.PollInterval <= 0 && !options.Once {
		return fmt.Errorf("follow-imports poll interval must be greater than zero")
	}
	return nil
}

// FollowImportsOnce consumes newly appended complete lines from a JSONL file and checkpoints the new offset.
func (a *App) FollowImportsOnce(ctx context.Context, input FollowImportsInput) (FollowImportsReport, error) {
	if a == nil {
		return followImportsReport{}, fmt.Errorf("app is required")
	}
	if err := input.Source.Validate(); err != nil {
		if strings.TrimSpace(string(input.Source)) == "" {
			return followImportsReport{}, fmt.Errorf("follow-imports source is required")
		}
		return followImportsReport{}, err
	}

	cwd, err := resolveIngestImportsCWD(input.CWD)
	if err != nil {
		return followImportsReport{}, err
	}
	inputPath, err := resolveIngestImportsPath(input.InputPath, cwd)
	if err != nil {
		return followImportsReport{}, err
	}
	if strings.TrimSpace(inputPath) == "" {
		return followImportsReport{}, fmt.Errorf("follow-imports input is required")
	}
	statePath, err := resolveFollowImportsStatePath(inputPath, input.StatePath, cwd)
	if err != nil {
		return followImportsReport{}, err
	}
	failedOutputBase, err := resolveIngestImportsPath(input.FailedOutputPath, cwd)
	if err != nil {
		return followImportsReport{}, err
	}
	failedManifestBase, err := resolveIngestImportsPath(input.FailedManifestPath, cwd)
	if err != nil {
		return followImportsReport{}, err
	}

	state, err := loadFollowImportsState(statePath)
	if err != nil {
		return followImportsReport{}, err
	}

	chunk, nextOffset, pendingBytes, truncated, err := readFollowImportsChunk(inputPath, state.Offset)
	if err != nil {
		return followImportsReport{}, err
	}

	report := followImportsReport{
		Status:       "idle",
		Source:       string(input.Source),
		Input:        inputPath,
		StateFile:    statePath,
		Offset:       nextOffset,
		PendingBytes: pendingBytes,
		Truncated:    truncated,
	}

	if len(chunk) == 0 && (truncated || nextOffset != state.Offset) {
		if err := saveFollowImportsState(statePath, followImportsState{
			Input:     inputPath,
			Offset:    nextOffset,
			UpdatedAt: time.Now().UTC(),
		}); err != nil {
			return report, err
		}
	}

	if len(chunk) == 0 {
		return report, nil
	}

	startOffset := state.Offset
	if truncated {
		startOffset = 0
	}
	batchFailedOutput := deriveFollowImportsBatchPath(failedOutputBase, startOffset, nextOffset)
	batchFailedManifest := deriveFollowImportsBatchPath(failedManifestBase, startOffset, nextOffset)

	batch, batchErr := a.IngestImports(ctx, IngestImportsInput{
		Source:             input.Source,
		Reader:             bytes.NewReader(chunk),
		InputLabel:         fmt.Sprintf("%s:%d-%d", inputPath, startOffset, nextOffset),
		CWD:                cwd,
		BranchName:         input.BranchName,
		RepoRemote:         input.RepoRemote,
		Task:               strings.TrimSpace(input.Task),
		ContinueOnError:    true,
		FailedOutputPath:   batchFailedOutput,
		FailedManifestPath: batchFailedManifest,
	})
	if batchErr != nil && batch.Attempted == 0 {
		return report, batchErr
	}

	report.Status = batch.Status
	report.ConsumedBytes = len(chunk)
	report.Batch = &batch
	if batchErr != nil {
		payload := common.ErrorDetails(batchErr, common.ErrWriteFailed, "follow-imports batch failed")
		report.BatchError = &payload
	}

	if err := saveFollowImportsState(statePath, followImportsState{
		Input:     inputPath,
		Offset:    nextOffset,
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		return report, err
	}

	return report, nil
}

func resolveFollowImportsStatePath(inputPath string, statePath string, cwd string) (string, error) {
	if strings.TrimSpace(statePath) == "" {
		return inputPath + ".offset.json", nil
	}
	return resolveIngestImportsPath(statePath, cwd)
}

func loadFollowImportsState(path string) (followImportsState, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return followImportsState{}, nil
		}
		return followImportsState{}, fmt.Errorf("read follow-imports state: %w", err)
	}

	var state followImportsState
	if err := json.Unmarshal(body, &state); err != nil {
		return followImportsState{}, fmt.Errorf("decode follow-imports state: %w", err)
	}
	if state.Offset < 0 {
		state.Offset = 0
	}
	return state, nil
}

func saveFollowImportsState(path string, state followImportsState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prepare follow-imports state directory: %w", err)
	}
	body, err := marshalIndented(state)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write follow-imports state: %w", err)
	}
	return nil
}

func readFollowImportsChunk(path string, offset int64) ([]byte, int64, int, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("stat follow-imports input: %w", err)
	}
	if offset < 0 {
		offset = 0
	}
	truncated := info.Size() < offset
	if truncated {
		offset = 0
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("open follow-imports input: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, 0, 0, false, fmt.Errorf("seek follow-imports input: %w", err)
	}
	body, err := io.ReadAll(file)
	if err != nil {
		return nil, 0, 0, false, fmt.Errorf("read follow-imports input: %w", err)
	}
	lastNewline := bytes.LastIndexByte(body, '\n')
	if lastNewline < 0 {
		return nil, offset, len(body), truncated, nil
	}

	consumed := append([]byte(nil), body[:lastNewline+1]...)
	nextOffset := offset + int64(lastNewline+1)
	pendingBytes := len(body) - lastNewline - 1
	return consumed, nextOffset, pendingBytes, truncated, nil
}

func deriveFollowImportsBatchPath(base string, start int64, end int64) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return fmt.Sprintf("%s.%d-%d%s", stem, start, end, ext)
}

func formatFollowImportsReport(report followImportsReport) string {
	lines := []string{
		fmt.Sprintf("follow imports %s", report.Status),
		fmt.Sprintf("status=%s", report.Status),
		fmt.Sprintf("source=%s", report.Source),
		fmt.Sprintf("input=%s", report.Input),
		fmt.Sprintf("state_file=%s", report.StateFile),
		fmt.Sprintf("offset=%d", report.Offset),
		fmt.Sprintf("consumed_bytes=%d", report.ConsumedBytes),
		fmt.Sprintf("pending_bytes=%d", report.PendingBytes),
		fmt.Sprintf("truncated=%t", report.Truncated),
	}
	if report.Batch != nil {
		lines = append(lines,
			fmt.Sprintf("batch_status=%s", report.Batch.Status),
			fmt.Sprintf("session_id=%s", report.Batch.Session.ID),
			fmt.Sprintf("attempted=%d", report.Batch.Attempted),
			fmt.Sprintf("processed=%d", report.Batch.Processed),
			fmt.Sprintf("failed=%d", report.Batch.Failed),
			fmt.Sprintf("materialized=%d", report.Batch.Materialized),
			fmt.Sprintf("suppressed=%d", report.Batch.Suppressed),
			fmt.Sprintf("failed_output=%s", fallbackString(report.Batch.FailedOutput, "none")),
			fmt.Sprintf("failed_manifest=%s", fallbackString(report.Batch.FailedManifest, "none")),
		)
	}
	if report.BatchError != nil {
		lines = append(lines,
			fmt.Sprintf("batch_error_code=%s", report.BatchError.Code),
			fmt.Sprintf("batch_error_message=%s", report.BatchError.Message),
		)
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeFollowImportsReport(stdout io.Writer, report followImportsReport, jsonOutput bool) error {
	if jsonOutput {
		body, err := marshalIndented(report)
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, body)
		return err
	}

	_, err := io.WriteString(stdout, formatFollowImportsReport(report))
	return err
}
