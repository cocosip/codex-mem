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
	InputPaths         []string
	StatePaths         []string
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

const (
	followImportsStatusIdle    = "idle"
	followImportsStatusFailed  = "failed"
	followImportsStatusPartial = "partial"
	followImportsEventRecovery = "watch_recovery"
	followImportsEventCatchup  = "watch_poll_catchup"
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
	WatchTransitions   int                  `json:"watch_transitions,omitempty"`
	LastFallbackReason string               `json:"last_fallback_reason,omitempty"`
	WatchEventCount    int                  `json:"watch_event_count,omitempty"`
	WatchEvents        []followImportsEvent `json:"watch_events,omitempty"`
	WatchPollCatchups  int                  `json:"watch_poll_catchups,omitempty"`
	WatchCatchupBytes  int                  `json:"watch_poll_catchup_bytes,omitempty"`
	Warnings           []common.Warning     `json:"warnings,omitempty"`
	Offset             int64                `json:"offset"`
	ConsumedBytes      int                  `json:"consumed_bytes"`
	PendingBytes       int                  `json:"pending_bytes"`
	Truncated          bool                 `json:"truncated,omitempty"`
	CheckpointReset    bool                 `json:"checkpoint_reset,omitempty"`
	ResetReason        string               `json:"reset_reason,omitempty"`
	Batch              *ingestImportsReport `json:"batch,omitempty"`
	BatchError         *common.ErrorPayload `json:"batch_error,omitempty"`
}

type followImportsAggregateReport struct {
	Status             string                `json:"status"`
	Source             string                `json:"source"`
	InputCount         int                   `json:"input_count"`
	ConsumedInputs     int                   `json:"consumed_inputs"`
	IdleInputs         int                   `json:"idle_inputs"`
	FailedInputs       int                   `json:"failed_inputs"`
	PartialInputs      int                   `json:"partial_inputs,omitempty"`
	RequestedWatchMode string                `json:"requested_watch_mode,omitempty"`
	ActiveWatchMode    string                `json:"active_watch_mode,omitempty"`
	WatchFallbacks     int                   `json:"watch_fallbacks,omitempty"`
	WatchTransitions   int                   `json:"watch_transitions,omitempty"`
	LastFallbackReason string                `json:"last_fallback_reason,omitempty"`
	WatchEventCount    int                   `json:"watch_event_count,omitempty"`
	WatchEvents        []followImportsEvent  `json:"watch_events,omitempty"`
	WatchPollCatchups  int                   `json:"watch_poll_catchups,omitempty"`
	WatchCatchupBytes  int                   `json:"watch_poll_catchup_bytes,omitempty"`
	Warnings           []common.Warning      `json:"warnings,omitempty"`
	TotalConsumedBytes int                   `json:"total_consumed_bytes"`
	TotalPendingBytes  int                   `json:"total_pending_bytes"`
	Inputs             []followImportsReport `json:"inputs"`
}

type followImportsHealthSnapshot struct {
	Status              string           `json:"status"`
	UpdatedAt           time.Time        `json:"updated_at"`
	Source              string           `json:"source"`
	InputCount          int              `json:"input_count"`
	Continuous          bool             `json:"continuous"`
	PollIntervalSeconds int64            `json:"poll_interval_seconds,omitempty"`
	RequestedWatchMode  string           `json:"requested_watch_mode,omitempty"`
	ActiveWatchMode     string           `json:"active_watch_mode,omitempty"`
	WatchFallbacks      int              `json:"watch_fallbacks,omitempty"`
	WatchTransitions    int              `json:"watch_transitions,omitempty"`
	LastFallbackReason  string           `json:"last_fallback_reason,omitempty"`
	WatchPollCatchups   int              `json:"watch_poll_catchups,omitempty"`
	WatchCatchupBytes   int              `json:"watch_poll_catchup_bytes,omitempty"`
	Warnings            []common.Warning `json:"warnings,omitempty"`
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

type followImportsEvent struct {
	At                 time.Time `json:"at"`
	Kind               string    `json:"kind"`
	RequestedWatchMode string    `json:"requested_watch_mode,omitempty"`
	PreviousWatchMode  string    `json:"previous_watch_mode,omitempty"`
	ActiveWatchMode    string    `json:"active_watch_mode,omitempty"`
	Reason             string    `json:"reason,omitempty"`
	Fallbacks          int       `json:"fallbacks,omitempty"`
	ConsumedInputs     int       `json:"consumed_inputs,omitempty"`
	ConsumedBytes      int       `json:"consumed_bytes,omitempty"`
}

type followImportsRuntimeState struct {
	Requested          followImportsWatchMode
	Active             followImportsWatchMode
	Fallbacks          int
	Transitions        int
	LastFallbackReason string
	PollCatchups       int
	PollCatchupBytes   int
	PendingEvents      []followImportsEvent
}

type followImportsRunTrigger string

const (
	followImportsRunTriggerInitial     followImportsRunTrigger = "initial"
	followImportsRunTriggerNotifyEvent followImportsRunTrigger = "notify_event"
	followImportsRunTriggerPollTick    followImportsRunTrigger = "poll_tick"
)

const followImportsCheckpointWindow = 256
const followImportsPollCatchupWarningThreshold = 3

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

	inputs, watchPaths, err := buildFollowImportsInputs(options)
	if err != nil {
		return err
	}
	runtimeState := followImportsRuntimeState{
		Requested: options.WatchMode,
		Active:    followImportsWatchModePoll,
	}

	runOnce := func(trigger followImportsRunTrigger) error {
		reports, err := runFollowImportsInputsOnce(ctx, instance, inputs)
		if err != nil {
			return err
		}
		if trigger == followImportsRunTriggerPollTick && runtimeState.Active == followImportsWatchModeNotify {
			consumedInputs, consumedBytes := summarizeFollowImportsConsumption(reports)
			if consumedBytes > 0 {
				markFollowImportsPollCatchup(&runtimeState, consumedInputs, consumedBytes)
			}
		}
		if len(reports) == 1 {
			runtimeState.Apply(&reports[0])
			if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, newFollowImportsHealthSnapshotFromReport(reports[0], options)); err != nil {
				return err
			}
			if shouldWriteFollowImportsReport(reports[0], options.Once) {
				return writeFollowImportsReport(stdout, reports[0], options.JSON)
			}
			return nil
		}

		report := newFollowImportsAggregateReport(options.Source, reports)
		runtimeState.ApplyAggregate(&report)
		if err := saveFollowImportsHealthSnapshot(cfg.Meta.LogDir, newFollowImportsHealthSnapshotFromAggregate(report, options)); err != nil {
			return err
		}
		if shouldWriteFollowImportsAggregateReport(report, options.Once) {
			return writeFollowImportsAggregateReport(stdout, report, options.JSON)
		}
		return nil
	}

	if options.Once {
		return runOnce(followImportsRunTriggerInitial)
	}

	if options.WatchMode == followImportsWatchModePoll {
		runtimeState.Active = followImportsWatchModePoll
		return runFollowImportsPollingLoop(ctx, options.PollInterval, runOnce)
	}
	if options.WatchMode == followImportsWatchModeAuto {
		return runFollowImportsAutoLoop(ctx, watchPaths, options.PollInterval, runOnce, &runtimeState)
	}

	return runFollowImportsNotifyLoop(ctx, watchPaths, options.WatchMode, options.PollInterval, runOnce, &runtimeState)
}

func runFollowImportsPollingLoop(ctx context.Context, pollInterval time.Duration, runOnce func(trigger followImportsRunTrigger) error) error {
	if err := runOnce(followImportsRunTriggerInitial); err != nil {
		return err
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := runOnce(followImportsRunTriggerPollTick); err != nil {
				return err
			}
		}
	}
}

func runFollowImportsAutoLoop(ctx context.Context, inputPaths []string, pollInterval time.Duration, runOnce func(trigger followImportsRunTrigger) error, runtimeState *followImportsRuntimeState) error {
	for {
		watcher, err := newFollowImportsWatcher(inputPaths)
		if err != nil {
			markFollowImportsFallback(runtimeState, "watcher_unavailable")
			slog.Default().Warn("follow-imports watcher unavailable, falling back to polling", "inputs", inputPaths, "err", err, "requested_watch_mode", followImportsWatchModeAuto, "fallbacks", runtimeStateFallbackCount(runtimeState), "fallback_reason", runtimeStateLastFallback(runtimeState))
		} else {
			recoverable, err := runFollowImportsNotifySession(ctx, inputPaths, followImportsWatchModeAuto, pollInterval, runOnce, runtimeState, watcher)
			if err != nil {
				return err
			}
			if !recoverable {
				return nil
			}
		}
		if err := runFollowImportsPollingRecoveryLoop(ctx, inputPaths, pollInterval, runOnce, runtimeState); err != nil {
			return err
		}
	}
}

func runFollowImportsPollingRecoveryLoop(ctx context.Context, inputPaths []string, pollInterval time.Duration, runOnce func(trigger followImportsRunTrigger) error, runtimeState *followImportsRuntimeState) error {
	if err := runOnce(followImportsRunTriggerInitial); err != nil {
		return err
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := runOnce(followImportsRunTriggerPollTick); err != nil {
				return err
			}
			watcher, err := newFollowImportsWatcher(inputPaths)
			if err != nil {
				continue
			}
			_ = watcher.Close()
			markFollowImportsRecovery(runtimeState, "watcher_recovered")
			return nil
		}
	}
}

func runFollowImportsNotifySession(ctx context.Context, inputPaths []string, watchMode followImportsWatchMode, pollInterval time.Duration, runOnce func(trigger followImportsRunTrigger) error, runtimeState *followImportsRuntimeState, watcher *fsnotify.Watcher) (bool, error) {
	if watcher == nil {
		return false, fmt.Errorf("follow-imports watcher is required")
	}
	defer func() {
		_ = watcher.Close()
	}()
	enterFollowImportsNotifyMode(runtimeState)
	if err := runOnce(followImportsRunTriggerInitial); err != nil {
		return false, err
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	events := watcher.Events
	errors := watcher.Errors
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case event, ok := <-events:
			if !ok {
				events = nil
				if errors == nil {
					if watchMode == followImportsWatchModeNotify {
						return false, fmt.Errorf("follow-imports watcher closed")
					}
					markFollowImportsFallback(runtimeState, "watcher_closed")
					slog.Default().Warn("follow-imports watcher closed, falling back to polling", "inputs", inputPaths, "requested_watch_mode", watchMode, "fallbacks", runtimeStateFallbackCount(runtimeState), "fallback_reason", runtimeStateLastFallback(runtimeState))
					return true, nil
				}
				continue
			}
			if shouldTriggerFollowImportsEvent(inputPaths, event) {
				if err := runOnce(followImportsRunTriggerNotifyEvent); err != nil {
					return false, err
				}
			}
		case err, ok := <-errors:
			if !ok {
				errors = nil
				if events == nil {
					if watchMode == followImportsWatchModeNotify {
						return false, fmt.Errorf("follow-imports watcher closed")
					}
					markFollowImportsFallback(runtimeState, "watcher_closed")
					slog.Default().Warn("follow-imports watcher closed, falling back to polling", "inputs", inputPaths, "requested_watch_mode", watchMode, "fallbacks", runtimeStateFallbackCount(runtimeState), "fallback_reason", runtimeStateLastFallback(runtimeState))
					return true, nil
				}
				continue
			}
			if err == nil {
				continue
			}
			if watchMode == followImportsWatchModeNotify {
				return false, fmt.Errorf("follow-imports watcher: %w", err)
			}
			markFollowImportsFallback(runtimeState, "watcher_error")
			slog.Default().Warn("follow-imports watcher error, falling back to polling", "inputs", inputPaths, "err", err, "requested_watch_mode", watchMode, "fallbacks", runtimeStateFallbackCount(runtimeState), "fallback_reason", runtimeStateLastFallback(runtimeState))
			return true, nil
		case <-ticker.C:
			if err := runOnce(followImportsRunTriggerPollTick); err != nil {
				return false, err
			}
		}
	}
}

func runFollowImportsNotifyLoop(ctx context.Context, inputPaths []string, watchMode followImportsWatchMode, pollInterval time.Duration, runOnce func(trigger followImportsRunTrigger) error, runtimeState *followImportsRuntimeState) error {
	watcher, err := newFollowImportsWatcher(inputPaths)
	if err != nil {
		if watchMode == followImportsWatchModeNotify {
			return err
		}
		markFollowImportsFallback(runtimeState, "watcher_unavailable")
		slog.Default().Warn("follow-imports watcher unavailable, falling back to polling", "inputs", inputPaths, "err", err, "requested_watch_mode", watchMode, "fallbacks", runtimeStateFallbackCount(runtimeState), "fallback_reason", runtimeStateLastFallback(runtimeState))
		return runFollowImportsPollingLoop(ctx, pollInterval, runOnce)
	}
	_, err = runFollowImportsNotifySession(ctx, inputPaths, watchMode, pollInterval, runOnce, runtimeState, watcher)
	return err
}

func enterFollowImportsNotifyMode(state *followImportsRuntimeState) {
	if state == nil {
		return
	}
	if state.Active == followImportsWatchModeNotify {
		return
	}
	if state.Fallbacks > 0 && state.Requested == followImportsWatchModeAuto {
		markFollowImportsRecovery(state, "watcher_recovered")
		return
	}
	setFollowImportsActiveMode(state, followImportsWatchModeNotify)
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
			options.InputPaths = append(options.InputPaths, value)
			i = next
		case "--state-file":
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.StatePaths = append(options.StatePaths, value)
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
	if len(options.InputPaths) == 0 {
		return fmt.Errorf("follow-imports input is required")
	}
	if len(options.StatePaths) > 0 && len(options.StatePaths) != len(options.InputPaths) {
		return fmt.Errorf("follow-imports state-file count (%d) must match input count (%d)", len(options.StatePaths), len(options.InputPaths))
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
		Status:          followImportsStatusIdle,
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
	report.WatchTransitions = s.Transitions
	report.LastFallbackReason = s.LastFallbackReason
	report.WatchEventCount = len(s.PendingEvents)
	report.WatchPollCatchups = s.PollCatchups
	report.WatchCatchupBytes = s.PollCatchupBytes
	report.Warnings = followImportsRuntimeWarnings(s)
	if report.WatchEventCount > 0 {
		report.WatchEvents = append([]followImportsEvent(nil), s.PendingEvents...)
		s.PendingEvents = nil
	}
}

func (s *followImportsRuntimeState) ApplyAggregate(report *followImportsAggregateReport) {
	if s == nil || report == nil {
		return
	}
	report.RequestedWatchMode = string(s.Requested)
	report.ActiveWatchMode = string(s.Active)
	report.WatchFallbacks = s.Fallbacks
	report.WatchTransitions = s.Transitions
	report.LastFallbackReason = s.LastFallbackReason
	report.WatchEventCount = len(s.PendingEvents)
	report.WatchPollCatchups = s.PollCatchups
	report.WatchCatchupBytes = s.PollCatchupBytes
	report.Warnings = followImportsRuntimeWarnings(s)
	if report.WatchEventCount > 0 {
		report.WatchEvents = append([]followImportsEvent(nil), s.PendingEvents...)
		s.PendingEvents = nil
	}
}

func markFollowImportsFallback(state *followImportsRuntimeState, reason string) {
	if state == nil {
		return
	}
	reason = strings.TrimSpace(reason)
	state.Fallbacks++
	state.LastFallbackReason = reason
	previous := state.Active
	state.Active = followImportsWatchModePoll
	if previous != followImportsWatchModePoll {
		state.Transitions++
	}
	state.PendingEvents = append(state.PendingEvents, followImportsEvent{
		At:                 time.Now().UTC(),
		Kind:               "watch_fallback",
		RequestedWatchMode: string(state.Requested),
		PreviousWatchMode:  string(previous),
		ActiveWatchMode:    string(state.Active),
		Reason:             reason,
		Fallbacks:          state.Fallbacks,
	})
}

func markFollowImportsRecovery(state *followImportsRuntimeState, reason string) {
	if state == nil {
		return
	}
	reason = strings.TrimSpace(reason)
	previous := state.Active
	state.Active = followImportsWatchModeNotify
	if previous != followImportsWatchModeNotify {
		state.Transitions++
	}
	state.PendingEvents = append(state.PendingEvents, followImportsEvent{
		At:                 time.Now().UTC(),
		Kind:               followImportsEventRecovery,
		RequestedWatchMode: string(state.Requested),
		PreviousWatchMode:  string(previous),
		ActiveWatchMode:    string(state.Active),
		Reason:             reason,
		Fallbacks:          state.Fallbacks,
	})
}

func markFollowImportsPollCatchup(state *followImportsRuntimeState, consumedInputs int, consumedBytes int) {
	if state == nil || consumedBytes <= 0 {
		return
	}
	state.PollCatchups++
	state.PollCatchupBytes += consumedBytes
	state.PendingEvents = append(state.PendingEvents, followImportsEvent{
		At:                 time.Now().UTC(),
		Kind:               followImportsEventCatchup,
		RequestedWatchMode: string(state.Requested),
		ActiveWatchMode:    string(state.Active),
		Reason:             "notify_safety_poll_consumed_bytes",
		Fallbacks:          state.Fallbacks,
		ConsumedInputs:     consumedInputs,
		ConsumedBytes:      consumedBytes,
	})
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

func setFollowImportsActiveMode(state *followImportsRuntimeState, next followImportsWatchMode) {
	if state == nil {
		return
	}
	previous := state.Active
	if previous == next {
		return
	}
	state.Active = next
	state.Transitions++
	state.PendingEvents = append(state.PendingEvents, followImportsEvent{
		At:                 time.Now().UTC(),
		Kind:               "watch_mode_transition",
		RequestedWatchMode: string(state.Requested),
		PreviousWatchMode:  string(previous),
		ActiveWatchMode:    string(next),
		Fallbacks:          state.Fallbacks,
	})
}

func newFollowImportsWatcher(inputPaths []string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create follow-imports watcher: %w", err)
	}
	seen := make(map[string]struct{})
	for _, inputPath := range inputPaths {
		watchPath := filepath.Dir(inputPath)
		if strings.TrimSpace(watchPath) == "" {
			watchPath = "."
		}
		key := followImportsPathKey(watchPath)
		if _, ok := seen[key]; ok {
			continue
		}
		if err := watcher.Add(watchPath); err != nil {
			_ = watcher.Close()
			return nil, fmt.Errorf("watch follow-imports directory %q: %w", watchPath, err)
		}
		seen[key] = struct{}{}
	}
	return watcher, nil
}

func shouldTriggerFollowImportsEvent(inputPaths []string, event fsnotify.Event) bool {
	for _, inputPath := range inputPaths {
		if followImportsPathsEqual(inputPath, event.Name) {
			return event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename)
		}
	}
	return false
}

func followImportsRuntimeWarnings(state *followImportsRuntimeState) []common.Warning {
	if state == nil {
		return nil
	}
	var warnings []common.Warning
	if state.PollCatchups >= followImportsPollCatchupWarningThreshold {
		warnings = append(warnings, common.Warning{
			Code:    common.WarnFollowImportsPollCatchup,
			Message: fmt.Sprintf("notify mode required poll catchup %d times and %d bytes so far", state.PollCatchups, state.PollCatchupBytes),
		})
	}
	return warnings
}

func followImportsPathsEqual(left string, right string) bool {
	left = filepath.Clean(strings.TrimSpace(left))
	right = filepath.Clean(strings.TrimSpace(right))
	return followImportsPathKey(left) == followImportsPathKey(right)
}

func followImportsPathKey(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if os.PathSeparator == '\\' {
		return strings.ToLower(path)
	}
	return path
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

func shouldWriteFollowImportsReport(report followImportsReport, once bool) bool {
	return once || report.Status != followImportsStatusIdle || report.WatchEventCount > 0
}

func shouldWriteFollowImportsAggregateReport(report followImportsAggregateReport, once bool) bool {
	return once || report.Status != followImportsStatusIdle || report.WatchEventCount > 0
}

func formatFollowImportsReport(report followImportsReport) string {
	lines := append([]string{fmt.Sprintf("follow imports %s", report.Status)}, followImportsReportLines(report)...)
	return strings.Join(lines, "\n") + "\n"
}

func formatFollowImportsAggregateReport(report followImportsAggregateReport) string {
	lines := []string{
		fmt.Sprintf("follow imports %s", report.Status),
		fmt.Sprintf("status=%s", report.Status),
		fmt.Sprintf("source=%s", report.Source),
		fmt.Sprintf("input_count=%d", report.InputCount),
		fmt.Sprintf("consumed_inputs=%d", report.ConsumedInputs),
		fmt.Sprintf("idle_inputs=%d", report.IdleInputs),
		fmt.Sprintf("failed_inputs=%d", report.FailedInputs),
		fmt.Sprintf("partial_inputs=%d", report.PartialInputs),
		fmt.Sprintf("requested_watch_mode=%s", fallbackString(report.RequestedWatchMode)),
		fmt.Sprintf("active_watch_mode=%s", fallbackString(report.ActiveWatchMode)),
		fmt.Sprintf("watch_fallbacks=%d", report.WatchFallbacks),
		fmt.Sprintf("watch_transitions=%d", report.WatchTransitions),
		fmt.Sprintf("last_fallback_reason=%s", fallbackString(report.LastFallbackReason)),
		fmt.Sprintf("watch_event_count=%d", report.WatchEventCount),
		fmt.Sprintf("watch_poll_catchups=%d", report.WatchPollCatchups),
		fmt.Sprintf("watch_poll_catchup_bytes=%d", report.WatchCatchupBytes),
		fmt.Sprintf("warnings=%d", len(report.Warnings)),
		fmt.Sprintf("total_consumed_bytes=%d", report.TotalConsumedBytes),
		fmt.Sprintf("total_pending_bytes=%d", report.TotalPendingBytes),
	}
	for i, event := range report.WatchEvents {
		prefix := fmt.Sprintf("watch_event_%d", i+1)
		lines = append(lines,
			fmt.Sprintf("%s_at=%s", prefix, event.At.UTC().Format(time.RFC3339)),
			fmt.Sprintf("%s_kind=%s", prefix, fallbackString(event.Kind)),
			fmt.Sprintf("%s_requested_watch_mode=%s", prefix, fallbackString(event.RequestedWatchMode)),
			fmt.Sprintf("%s_previous_watch_mode=%s", prefix, fallbackString(event.PreviousWatchMode)),
			fmt.Sprintf("%s_active_watch_mode=%s", prefix, fallbackString(event.ActiveWatchMode)),
			fmt.Sprintf("%s_reason=%s", prefix, fallbackString(event.Reason)),
			fmt.Sprintf("%s_fallbacks=%d", prefix, event.Fallbacks),
			fmt.Sprintf("%s_consumed_inputs=%d", prefix, event.ConsumedInputs),
			fmt.Sprintf("%s_consumed_bytes=%d", prefix, event.ConsumedBytes),
		)
	}
	for i, warning := range report.Warnings {
		prefix := fmt.Sprintf("warning_%d", i+1)
		lines = append(lines,
			fmt.Sprintf("%s_code=%s", prefix, warning.Code),
			fmt.Sprintf("%s_message=%s", prefix, warning.Message),
		)
	}
	for i, inputReport := range report.Inputs {
		prefix := fmt.Sprintf("input_%d_", i+1)
		for _, line := range followImportsReportLines(inputReport) {
			lines = append(lines, prefix+line)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func followImportsReportLines(report followImportsReport) []string {
	lines := []string{
		fmt.Sprintf("status=%s", report.Status),
		fmt.Sprintf("source=%s", report.Source),
		fmt.Sprintf("input=%s", report.Input),
		fmt.Sprintf("state_file=%s", report.StateFile),
		fmt.Sprintf("requested_watch_mode=%s", fallbackString(report.RequestedWatchMode)),
		fmt.Sprintf("active_watch_mode=%s", fallbackString(report.ActiveWatchMode)),
		fmt.Sprintf("watch_fallbacks=%d", report.WatchFallbacks),
		fmt.Sprintf("watch_transitions=%d", report.WatchTransitions),
		fmt.Sprintf("last_fallback_reason=%s", fallbackString(report.LastFallbackReason)),
		fmt.Sprintf("watch_event_count=%d", report.WatchEventCount),
		fmt.Sprintf("watch_poll_catchups=%d", report.WatchPollCatchups),
		fmt.Sprintf("watch_poll_catchup_bytes=%d", report.WatchCatchupBytes),
		fmt.Sprintf("warnings=%d", len(report.Warnings)),
		fmt.Sprintf("offset=%d", report.Offset),
		fmt.Sprintf("consumed_bytes=%d", report.ConsumedBytes),
		fmt.Sprintf("pending_bytes=%d", report.PendingBytes),
		fmt.Sprintf("truncated=%t", report.Truncated),
		fmt.Sprintf("checkpoint_reset=%t", report.CheckpointReset),
		fmt.Sprintf("reset_reason=%s", fallbackString(report.ResetReason)),
	}
	for i, event := range report.WatchEvents {
		prefix := fmt.Sprintf("watch_event_%d", i+1)
		lines = append(lines,
			fmt.Sprintf("%s_at=%s", prefix, event.At.UTC().Format(time.RFC3339)),
			fmt.Sprintf("%s_kind=%s", prefix, fallbackString(event.Kind)),
			fmt.Sprintf("%s_requested_watch_mode=%s", prefix, fallbackString(event.RequestedWatchMode)),
			fmt.Sprintf("%s_previous_watch_mode=%s", prefix, fallbackString(event.PreviousWatchMode)),
			fmt.Sprintf("%s_active_watch_mode=%s", prefix, fallbackString(event.ActiveWatchMode)),
			fmt.Sprintf("%s_reason=%s", prefix, fallbackString(event.Reason)),
			fmt.Sprintf("%s_fallbacks=%d", prefix, event.Fallbacks),
			fmt.Sprintf("%s_consumed_inputs=%d", prefix, event.ConsumedInputs),
			fmt.Sprintf("%s_consumed_bytes=%d", prefix, event.ConsumedBytes),
		)
	}
	for i, warning := range report.Warnings {
		prefix := fmt.Sprintf("warning_%d", i+1)
		lines = append(lines,
			fmt.Sprintf("%s_code=%s", prefix, warning.Code),
			fmt.Sprintf("%s_message=%s", prefix, warning.Message),
		)
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
	return lines
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

func writeFollowImportsAggregateReport(stdout io.Writer, report followImportsAggregateReport, jsonOutput bool) error {
	if jsonOutput {
		body, err := marshalIndented(report)
		if err != nil {
			return err
		}
		_, err = io.WriteString(stdout, body)
		return err
	}

	_, err := io.WriteString(stdout, formatFollowImportsAggregateReport(report))
	return err
}

func buildFollowImportsInputs(options followImportsOptions) ([]FollowImportsInput, []string, error) {
	cwd, err := resolveIngestImportsCWD(options.CWD)
	if err != nil {
		return nil, nil, err
	}

	inputs := make([]FollowImportsInput, 0, len(options.InputPaths))
	watchPaths := make([]string, 0, len(options.InputPaths))
	seen := make(map[string]struct{}, len(options.InputPaths))
	for i, rawInputPath := range options.InputPaths {
		inputPath, err := resolveIngestImportsPath(rawInputPath, cwd)
		if err != nil {
			return nil, nil, err
		}
		key := followImportsPathKey(inputPath)
		if _, ok := seen[key]; ok {
			return nil, nil, fmt.Errorf("follow-imports input %q is duplicated", inputPath)
		}
		seen[key] = struct{}{}

		statePath := ""
		if len(options.StatePaths) > 0 {
			statePath = options.StatePaths[i]
		}

		inputs = append(inputs, FollowImportsInput{
			Source:             options.Source,
			InputPath:          inputPath,
			StatePath:          statePath,
			CWD:                cwd,
			BranchName:         options.BranchName,
			RepoRemote:         options.RepoRemote,
			Task:               options.Task,
			FailedOutputPath:   deriveFollowImportsInputBasePath(options.FailedOutputPath, inputPath, len(options.InputPaths)),
			FailedManifestPath: deriveFollowImportsInputBasePath(options.FailedManifestPath, inputPath, len(options.InputPaths)),
		})
		watchPaths = append(watchPaths, inputPath)
	}
	return inputs, watchPaths, nil
}

func runFollowImportsInputsOnce(ctx context.Context, instance *App, inputs []FollowImportsInput) ([]followImportsReport, error) {
	reports := make([]followImportsReport, 0, len(inputs))
	for _, input := range inputs {
		report, err := instance.FollowImportsOnce(ctx, input)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func summarizeFollowImportsConsumption(reports []followImportsReport) (int, int) {
	consumedInputs := 0
	consumedBytes := 0
	for _, report := range reports {
		if report.ConsumedBytes > 0 {
			consumedInputs++
			consumedBytes += report.ConsumedBytes
		}
	}
	return consumedInputs, consumedBytes
}

func newFollowImportsAggregateReport(source imports.Source, reports []followImportsReport) followImportsAggregateReport {
	aggregate := followImportsAggregateReport{
		Status:     followImportsStatusIdle,
		Source:     string(source),
		InputCount: len(reports),
		Inputs:     append([]followImportsReport(nil), reports...),
	}
	for _, report := range reports {
		aggregate.TotalConsumedBytes += report.ConsumedBytes
		aggregate.TotalPendingBytes += report.PendingBytes
		switch report.Status {
		case followImportsStatusIdle:
			aggregate.IdleInputs++
		case followImportsStatusFailed:
			aggregate.FailedInputs++
			aggregate.ConsumedInputs++
		case followImportsStatusPartial:
			aggregate.PartialInputs++
			aggregate.ConsumedInputs++
		default:
			aggregate.ConsumedInputs++
		}
	}
	switch {
	case aggregate.InputCount == 0:
		aggregate.Status = followImportsStatusIdle
	case aggregate.FailedInputs == aggregate.InputCount:
		aggregate.Status = followImportsStatusFailed
	case aggregate.FailedInputs > 0 || aggregate.PartialInputs > 0:
		aggregate.Status = followImportsStatusPartial
	case aggregate.ConsumedInputs == 0:
		aggregate.Status = followImportsStatusIdle
	default:
		aggregate.Status = "ok"
	}
	return aggregate
}

func deriveFollowImportsInputBasePath(base string, inputPath string, inputCount int) string {
	base = strings.TrimSpace(base)
	if base == "" || inputCount <= 1 {
		return base
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	label := sanitizeFollowImportsPathLabel(inputPath)
	return fmt.Sprintf("%s.%s%s", stem, label, ext)
}

func sanitizeFollowImportsPathLabel(path string) string {
	base := filepath.Base(strings.TrimSpace(path))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if base == "" {
		return "input"
	}
	var builder strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('-')
		}
	}
	label := strings.Trim(builder.String(), "-")
	if label == "" {
		return "input"
	}
	return label
}

func followImportsHealthPath(logDir string) string {
	logDir = strings.TrimSpace(logDir)
	if logDir == "" {
		return "follow-imports.health.json"
	}
	return filepath.Join(logDir, "follow-imports.health.json")
}

func saveFollowImportsHealthSnapshot(logDir string, snapshot followImportsHealthSnapshot) error {
	path := followImportsHealthPath(logDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("prepare follow-imports health directory: %w", err)
	}
	body, err := marshalIndented(snapshot)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write follow-imports health snapshot: %w", err)
	}
	return nil
}

func loadFollowImportsHealthSnapshot(logDir string) (*followImportsHealthSnapshot, error) {
	path := followImportsHealthPath(logDir)
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read follow-imports health snapshot: %w", err)
	}
	var snapshot followImportsHealthSnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		return nil, fmt.Errorf("decode follow-imports health snapshot: %w", err)
	}
	return &snapshot, nil
}

func newFollowImportsHealthSnapshotFromReport(report followImportsReport, options followImportsOptions) followImportsHealthSnapshot {
	return followImportsHealthSnapshot{
		Status:              report.Status,
		UpdatedAt:           time.Now().UTC(),
		Source:              report.Source,
		InputCount:          1,
		Continuous:          !options.Once,
		PollIntervalSeconds: followImportsSnapshotPollIntervalSeconds(options),
		RequestedWatchMode:  report.RequestedWatchMode,
		ActiveWatchMode:     report.ActiveWatchMode,
		WatchFallbacks:      report.WatchFallbacks,
		WatchTransitions:    report.WatchTransitions,
		LastFallbackReason:  report.LastFallbackReason,
		WatchPollCatchups:   report.WatchPollCatchups,
		WatchCatchupBytes:   report.WatchCatchupBytes,
		Warnings:            append([]common.Warning(nil), report.Warnings...),
	}
}

func newFollowImportsHealthSnapshotFromAggregate(report followImportsAggregateReport, options followImportsOptions) followImportsHealthSnapshot {
	return followImportsHealthSnapshot{
		Status:              report.Status,
		UpdatedAt:           time.Now().UTC(),
		Source:              report.Source,
		InputCount:          report.InputCount,
		Continuous:          !options.Once,
		PollIntervalSeconds: followImportsSnapshotPollIntervalSeconds(options),
		RequestedWatchMode:  report.RequestedWatchMode,
		ActiveWatchMode:     report.ActiveWatchMode,
		WatchFallbacks:      report.WatchFallbacks,
		WatchTransitions:    report.WatchTransitions,
		LastFallbackReason:  report.LastFallbackReason,
		WatchPollCatchups:   report.WatchPollCatchups,
		WatchCatchupBytes:   report.WatchCatchupBytes,
		Warnings:            append([]common.Warning(nil), report.Warnings...),
	}
}

func followImportsSnapshotPollIntervalSeconds(options followImportsOptions) int64 {
	if options.Once {
		return 0
	}
	return int64(options.PollInterval / time.Second)
}
