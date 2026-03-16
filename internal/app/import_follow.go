package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codex-mem/internal/config"
	"codex-mem/internal/domain/common"
	"codex-mem/internal/domain/imports"
	"github.com/fsnotify/fsnotify"
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
	WatchMode          followImportsWatchMode
}

type followImportsWatchMode string

const (
	followImportsWatchModeAuto   followImportsWatchMode = "auto"
	followImportsWatchModePoll   followImportsWatchMode = "poll"
	followImportsWatchModeNotify followImportsWatchMode = "notify"
)

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
	Status             string               `json:"status"`
	Source             string               `json:"source"`
	Input              string               `json:"input"`
	StateFile          string               `json:"state_file"`
	RequestedWatchMode string               `json:"requested_watch_mode,omitempty"`
	ActiveWatchMode    string               `json:"active_watch_mode,omitempty"`
	WatchFallbacks     int                  `json:"watch_fallbacks,omitempty"`
	LastFallbackReason string               `json:"last_fallback_reason,omitempty"`
	Offset             int64                `json:"offset"`
	ConsumedBytes      int                  `json:"consumed_bytes"`
	PendingBytes       int                  `json:"pending_bytes"`
	Truncated          bool                 `json:"truncated,omitempty"`
	CheckpointReset    bool                 `json:"checkpoint_reset,omitempty"`
	ResetReason        string               `json:"reset_reason,omitempty"`
	Batch              *ingestImportsReport `json:"batch,omitempty"`
	BatchError         *common.ErrorPayload `json:"batch_error,omitempty"`
}

type followImportsCheckpoint struct {
	WindowSize int       `json:"window_size,omitempty"`
	TailSHA256 string    `json:"tail_sha256,omitempty"`
	FileSize   int64     `json:"file_size,omitempty"`
	ModifiedAt time.Time `json:"modified_at,omitempty"`
}

type followImportsState struct {
	Input      string                   `json:"input"`
	Offset     int64                    `json:"offset"`
	Checkpoint *followImportsCheckpoint `json:"checkpoint,omitempty"`
	UpdatedAt  time.Time                `json:"updated_at"`
}

type followImportsChunk struct {
	StartOffset     int64
	NextOffset      int64
	Body            []byte
	PendingBytes    int
	Truncated       bool
	CheckpointReset bool
	ResetReason     string
}

type followImportsRuntimeState struct {
	Requested          followImportsWatchMode
	Active             followImportsWatchMode
	Fallbacks          int
	LastFallbackReason string
}

const followImportsCheckpointWindow = 256

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
	watchPath, err := resolveIngestImportsPath(options.InputPath, options.CWD)
	if err != nil {
		return err
	}
	runtimeState := followImportsRuntimeState{
		Requested: options.WatchMode,
		Active:    followImportsWatchModePoll,
	}

	runOnce := func() error {
		report, err := instance.FollowImportsOnce(ctx, input)
		if err != nil {
			return err
		}
		runtimeState.Apply(&report)
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

	if options.WatchMode == followImportsWatchModePoll {
		runtimeState.Active = followImportsWatchModePoll
		return runFollowImportsPollingLoop(ctx, options.PollInterval, runOnce)
	}

	return runFollowImportsNotifyLoop(ctx, watchPath, options.WatchMode, options.PollInterval, runOnce, &runtimeState)
}

func runFollowImportsPollingLoop(ctx context.Context, pollInterval time.Duration, runOnce func() error) error {
	ticker := time.NewTicker(pollInterval)
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

func runFollowImportsNotifyLoop(ctx context.Context, inputPath string, watchMode followImportsWatchMode, pollInterval time.Duration, runOnce func() error, runtimeState *followImportsRuntimeState) error {
	if runtimeState != nil {
		runtimeState.Active = followImportsWatchModeNotify
	}
	watcher, err := newFollowImportsWatcher(inputPath)
	if err != nil {
		if watchMode == followImportsWatchModeNotify {
			return err
		}
		markFollowImportsFallback(runtimeState, "watcher_unavailable")
		slog.Default().Warn("follow-imports watcher unavailable, falling back to polling", "input", inputPath, "err", err, "requested_watch_mode", watchMode, "fallbacks", runtimeStateFallbackCount(runtimeState), "fallback_reason", runtimeStateLastFallback(runtimeState))
		return runFollowImportsPollingLoop(ctx, pollInterval, runOnce)
	}
	defer func() {
		_ = watcher.Close()
	}()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	events := watcher.Events
	errors := watcher.Errors
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				events = nil
				if errors == nil {
					if watchMode == followImportsWatchModeNotify {
						return fmt.Errorf("follow-imports watcher closed")
					}
					markFollowImportsFallback(runtimeState, "watcher_closed")
					slog.Default().Warn("follow-imports watcher closed, falling back to polling", "input", inputPath, "requested_watch_mode", watchMode, "fallbacks", runtimeStateFallbackCount(runtimeState), "fallback_reason", runtimeStateLastFallback(runtimeState))
					return runFollowImportsPollingLoop(ctx, pollInterval, runOnce)
				}
				continue
			}
			if shouldTriggerFollowImportsEvent(inputPath, event) {
				if err := runOnce(); err != nil {
					return err
				}
			}
		case err, ok := <-errors:
			if !ok {
				errors = nil
				if events == nil {
					if watchMode == followImportsWatchModeNotify {
						return fmt.Errorf("follow-imports watcher closed")
					}
					markFollowImportsFallback(runtimeState, "watcher_closed")
					slog.Default().Warn("follow-imports watcher closed, falling back to polling", "input", inputPath, "requested_watch_mode", watchMode, "fallbacks", runtimeStateFallbackCount(runtimeState), "fallback_reason", runtimeStateLastFallback(runtimeState))
					return runFollowImportsPollingLoop(ctx, pollInterval, runOnce)
				}
				continue
			}
			if err == nil {
				continue
			}
			if watchMode == followImportsWatchModeNotify {
				return fmt.Errorf("follow-imports watcher: %w", err)
			}
			markFollowImportsFallback(runtimeState, "watcher_error")
			slog.Default().Warn("follow-imports watcher error, falling back to polling", "input", inputPath, "err", err, "requested_watch_mode", watchMode, "fallbacks", runtimeStateFallbackCount(runtimeState), "fallback_reason", runtimeStateLastFallback(runtimeState))
			return runFollowImportsPollingLoop(ctx, pollInterval, runOnce)
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
		WatchMode:    followImportsWatchModeAuto,
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
		case "--watch-mode":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.WatchMode = followImportsWatchMode(strings.ToLower(strings.TrimSpace(value)))
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
	switch options.WatchMode {
	case followImportsWatchModeAuto, followImportsWatchModePoll, followImportsWatchModeNotify:
	default:
		return fmt.Errorf("invalid follow-imports watch mode %q", options.WatchMode)
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

	chunk, err := readFollowImportsChunk(inputPath, state)
	if err != nil {
		return followImportsReport{}, err
	}

	report := followImportsReport{
		Status:          "idle",
		Source:          string(input.Source),
		Input:           inputPath,
		StateFile:       statePath,
		Offset:          chunk.NextOffset,
		PendingBytes:    chunk.PendingBytes,
		Truncated:       chunk.Truncated,
		CheckpointReset: chunk.CheckpointReset,
		ResetReason:     chunk.ResetReason,
	}

	if len(chunk.Body) == 0 && (chunk.CheckpointReset || chunk.NextOffset != state.Offset) {
		checkpoint, err := buildFollowImportsCheckpoint(inputPath, chunk.NextOffset)
		if err != nil {
			return report, err
		}
		if err := saveFollowImportsState(statePath, followImportsState{
			Input:      inputPath,
			Offset:     chunk.NextOffset,
			Checkpoint: checkpoint,
			UpdatedAt:  time.Now().UTC(),
		}); err != nil {
			return report, err
		}
	}

	if len(chunk.Body) == 0 {
		return report, nil
	}

	batchFailedOutput := deriveFollowImportsBatchPath(failedOutputBase, chunk.StartOffset, chunk.NextOffset)
	batchFailedManifest := deriveFollowImportsBatchPath(failedManifestBase, chunk.StartOffset, chunk.NextOffset)

	batch, batchErr := a.IngestImports(ctx, IngestImportsInput{
		Source:             input.Source,
		Reader:             bytes.NewReader(chunk.Body),
		InputLabel:         fmt.Sprintf("%s:%d-%d", inputPath, chunk.StartOffset, chunk.NextOffset),
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
	report.ConsumedBytes = len(chunk.Body)
	report.Batch = &batch
	if batchErr != nil {
		payload := common.ErrorDetails(batchErr, common.ErrWriteFailed, "follow-imports batch failed")
		report.BatchError = &payload
	}

	checkpoint, err := buildFollowImportsCheckpoint(inputPath, chunk.NextOffset)
	if err != nil {
		return report, err
	}
	if err := saveFollowImportsState(statePath, followImportsState{
		Input:      inputPath,
		Offset:     chunk.NextOffset,
		Checkpoint: checkpoint,
		UpdatedAt:  time.Now().UTC(),
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

func readFollowImportsChunk(path string, state followImportsState) (followImportsChunk, error) {
	info, err := os.Stat(path)
	if err != nil {
		return followImportsChunk{}, fmt.Errorf("stat follow-imports input: %w", err)
	}
	offset := state.Offset
	if offset < 0 {
		offset = 0
	}
	truncated := false
	checkpointReset := false
	resetReason := ""
	if info.Size() < offset {
		offset = 0
		truncated = true
		checkpointReset = true
		resetReason = "truncated"
	} else if offset > 0 && state.Checkpoint != nil {
		match, err := followImportsCheckpointMatches(path, offset, *state.Checkpoint)
		if err != nil {
			return followImportsChunk{}, err
		}
		if !match {
			offset = 0
			checkpointReset = true
			resetReason = "file_replaced"
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return followImportsChunk{}, fmt.Errorf("open follow-imports input: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return followImportsChunk{}, fmt.Errorf("seek follow-imports input: %w", err)
	}
	body, err := io.ReadAll(file)
	if err != nil {
		return followImportsChunk{}, fmt.Errorf("read follow-imports input: %w", err)
	}
	lastNewline := bytes.LastIndexByte(body, '\n')
	if lastNewline < 0 {
		return followImportsChunk{
			StartOffset:     offset,
			NextOffset:      offset,
			Body:            nil,
			PendingBytes:    len(body),
			Truncated:       truncated,
			CheckpointReset: checkpointReset,
			ResetReason:     resetReason,
		}, nil
	}

	consumed := append([]byte(nil), body[:lastNewline+1]...)
	nextOffset := offset + int64(lastNewline+1)
	pendingBytes := len(body) - lastNewline - 1
	return followImportsChunk{
		StartOffset:     offset,
		NextOffset:      nextOffset,
		Body:            consumed,
		PendingBytes:    pendingBytes,
		Truncated:       truncated,
		CheckpointReset: checkpointReset,
		ResetReason:     resetReason,
	}, nil
}

func buildFollowImportsCheckpoint(path string, offset int64) (*followImportsCheckpoint, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat follow-imports input for checkpoint: %w", err)
	}
	checkpoint := &followImportsCheckpoint{
		FileSize:   info.Size(),
		ModifiedAt: info.ModTime().UTC(),
	}
	if offset <= 0 {
		return checkpoint, nil
	}

	window, err := readFollowImportsWindow(path, offset, followImportsCheckpointWindow)
	if err != nil {
		return nil, err
	}
	checkpoint.WindowSize = len(window)
	checkpoint.TailSHA256 = hashFollowImportsBytes(window)
	return checkpoint, nil
}

func followImportsCheckpointMatches(path string, offset int64, checkpoint followImportsCheckpoint) (bool, error) {
	if offset <= 0 {
		return true, nil
	}
	if checkpoint.WindowSize <= 0 || strings.TrimSpace(checkpoint.TailSHA256) == "" {
		return true, nil
	}
	window, err := readFollowImportsWindow(path, offset, checkpoint.WindowSize)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(hashFollowImportsBytes(window), checkpoint.TailSHA256), nil
}

func readFollowImportsWindow(path string, offset int64, maxWindow int) ([]byte, error) {
	if offset <= 0 || maxWindow <= 0 {
		return nil, nil
	}
	start := offset - int64(maxWindow)
	if start < 0 {
		start = 0
	}
	length := offset - start
	if length <= 0 {
		return nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open follow-imports checkpoint window: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek follow-imports checkpoint window: %w", err)
	}
	window := make([]byte, length)
	if _, err := io.ReadFull(file, window); err != nil {
		return nil, fmt.Errorf("read follow-imports checkpoint window: %w", err)
	}
	return window, nil
}

func hashFollowImportsBytes(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (s *followImportsRuntimeState) Apply(report *followImportsReport) {
	if s == nil || report == nil {
		return
	}
	report.RequestedWatchMode = string(s.Requested)
	report.ActiveWatchMode = string(s.Active)
	report.WatchFallbacks = s.Fallbacks
	report.LastFallbackReason = s.LastFallbackReason
}

func markFollowImportsFallback(state *followImportsRuntimeState, reason string) {
	if state == nil {
		return
	}
	state.Active = followImportsWatchModePoll
	state.Fallbacks++
	state.LastFallbackReason = strings.TrimSpace(reason)
}

func runtimeStateFallbackCount(state *followImportsRuntimeState) int {
	if state == nil {
		return 0
	}
	return state.Fallbacks
}

func runtimeStateLastFallback(state *followImportsRuntimeState) string {
	if state == nil {
		return ""
	}
	return state.LastFallbackReason
}

func newFollowImportsWatcher(inputPath string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create follow-imports watcher: %w", err)
	}
	watchPath := filepath.Dir(inputPath)
	if strings.TrimSpace(watchPath) == "" {
		watchPath = "."
	}
	if err := watcher.Add(watchPath); err != nil {
		_ = watcher.Close()
		return nil, fmt.Errorf("watch follow-imports directory %q: %w", watchPath, err)
	}
	return watcher, nil
}

func shouldTriggerFollowImportsEvent(inputPath string, event fsnotify.Event) bool {
	if !followImportsPathsEqual(inputPath, event.Name) {
		return false
	}
	return event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename)
}

func followImportsPathsEqual(left string, right string) bool {
	left = filepath.Clean(strings.TrimSpace(left))
	right = filepath.Clean(strings.TrimSpace(right))
	if os.PathSeparator == '\\' {
		return strings.EqualFold(left, right)
	}
	return left == right
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
		fmt.Sprintf("requested_watch_mode=%s", fallbackString(report.RequestedWatchMode)),
		fmt.Sprintf("active_watch_mode=%s", fallbackString(report.ActiveWatchMode)),
		fmt.Sprintf("watch_fallbacks=%d", report.WatchFallbacks),
		fmt.Sprintf("last_fallback_reason=%s", fallbackString(report.LastFallbackReason)),
		fmt.Sprintf("offset=%d", report.Offset),
		fmt.Sprintf("consumed_bytes=%d", report.ConsumedBytes),
		fmt.Sprintf("pending_bytes=%d", report.PendingBytes),
		fmt.Sprintf("truncated=%t", report.Truncated),
		fmt.Sprintf("checkpoint_reset=%t", report.CheckpointReset),
		fmt.Sprintf("reset_reason=%s", fallbackString(report.ResetReason)),
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
			fmt.Sprintf("failed_output=%s", fallbackString(report.Batch.FailedOutput)),
			fmt.Sprintf("failed_manifest=%s", fallbackString(report.Batch.FailedManifest)),
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
