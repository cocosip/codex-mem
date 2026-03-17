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
	pathpkg "path"
	"path/filepath"
	"sort"
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

type cleanupFollowImportsOptions struct {
	followImportsHygieneOptions
	DryRun                 bool
	PruneState             bool
	PruneFailedOutput      bool
	PruneFailedManifest    bool
	PruneStaleFollowHealth bool
}

type auditFollowImportsOptions struct {
	followImportsHygieneOptions
	CheckState          bool
	CheckFailedOutput   bool
	CheckFailedManifest bool
	CheckFollowHealth   bool
}

type followImportsHygieneOptions struct {
	InputPaths         []string
	StatePaths         []string
	FailedOutputPath   string
	FailedManifestPath string
	IncludePatterns    []string
	ExcludePatterns    []string
	TargetProfile      string
	RetentionProfile   string
	CWD                string
	JSON               bool
	FailIfMatched      bool
	OlderThan          time.Duration
	olderThanExplicit  bool
}

type followImportsWatchMode string

const (
	followImportsWatchModeAuto   followImportsWatchMode = "auto"
	followImportsWatchModePoll   followImportsWatchMode = "poll"
	followImportsWatchModeNotify followImportsWatchMode = "notify"
)

const followImportsStateFileFlag = "--state-file"

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

type cleanupFollowImportsReport struct {
	DryRun           bool                                 `json:"dry_run"`
	FailIfMatched    bool                                 `json:"fail_if_matched"`
	MatchFound       bool                                 `json:"match_found"`
	TargetProfile    string                               `json:"target_profile,omitempty"`
	RetentionProfile string                               `json:"retention_profile,omitempty"`
	OlderThanSeconds int64                                `json:"older_than_seconds,omitempty"`
	IncludePatterns  []string                             `json:"include_patterns,omitempty"`
	ExcludePatterns  []string                             `json:"exclude_patterns,omitempty"`
	Status           string                               `json:"status"`
	StateFiles       cleanupFollowImportsPathSummary      `json:"state_files"`
	FailedOutputs    cleanupFollowImportsPatternSummary   `json:"failed_outputs"`
	FailedManifests  cleanupFollowImportsPatternSummary   `json:"failed_manifests"`
	FollowHealth     cleanupFollowImportsFollowHealthView `json:"follow_health"`
}

type cleanupFollowImportsPathSummary struct {
	Requested             int      `json:"requested"`
	Matched               int      `json:"matched"`
	Removed               int      `json:"removed"`
	Missing               int      `json:"missing"`
	SkippedByAge          int      `json:"skipped_by_age,omitempty"`
	SkippedByPattern      int      `json:"skipped_by_pattern,omitempty"`
	MatchedPaths          []string `json:"matched_paths,omitempty"`
	RemovedPaths          []string `json:"removed_paths,omitempty"`
	MissingPaths          []string `json:"missing_paths,omitempty"`
	SkippedByPatternPaths []string `json:"skipped_by_pattern_paths,omitempty"`
	SkippedByAgePaths     []string `json:"skipped_by_age_paths,omitempty"`
}

type cleanupFollowImportsPatternSummary struct {
	Bases                 int      `json:"bases"`
	Matched               int      `json:"matched"`
	Removed               int      `json:"removed"`
	SkippedByAge          int      `json:"skipped_by_age,omitempty"`
	SkippedByPattern      int      `json:"skipped_by_pattern,omitempty"`
	BasePaths             []string `json:"base_paths,omitempty"`
	MatchedPaths          []string `json:"matched_paths,omitempty"`
	RemovedPaths          []string `json:"removed_paths,omitempty"`
	SkippedByPatternPaths []string `json:"skipped_by_pattern_paths,omitempty"`
	SkippedByAgePaths     []string `json:"skipped_by_age_paths,omitempty"`
}

type cleanupFollowImportsFollowHealthView struct {
	File        string `json:"file"`
	WouldPrune  bool   `json:"would_prune"`
	Pruned      bool   `json:"pruned"`
	PruneReason string `json:"prune_reason,omitempty"`
}

type auditFollowImportsReport struct {
	FailIfMatched    bool                             `json:"fail_if_matched"`
	MatchFound       bool                             `json:"match_found"`
	TargetProfile    string                           `json:"target_profile,omitempty"`
	RetentionProfile string                           `json:"retention_profile,omitempty"`
	OlderThanSeconds int64                            `json:"older_than_seconds,omitempty"`
	IncludePatterns  []string                         `json:"include_patterns,omitempty"`
	ExcludePatterns  []string                         `json:"exclude_patterns,omitempty"`
	Status           string                           `json:"status"`
	StateFiles       auditFollowImportsPathSummary    `json:"state_files"`
	FailedOutputs    auditFollowImportsPatternSummary `json:"failed_outputs"`
	FailedManifests  auditFollowImportsPatternSummary `json:"failed_manifests"`
	FollowHealth     auditFollowImportsHealthView     `json:"follow_health"`
}

type auditFollowImportsPathSummary struct {
	Requested             int      `json:"requested"`
	Matched               int      `json:"matched"`
	Missing               int      `json:"missing"`
	SkippedByAge          int      `json:"skipped_by_age,omitempty"`
	SkippedByPattern      int      `json:"skipped_by_pattern,omitempty"`
	MatchedPaths          []string `json:"matched_paths,omitempty"`
	MissingPaths          []string `json:"missing_paths,omitempty"`
	SkippedByPatternPaths []string `json:"skipped_by_pattern_paths,omitempty"`
	SkippedByAgePaths     []string `json:"skipped_by_age_paths,omitempty"`
}

type auditFollowImportsPatternSummary struct {
	Bases                 int      `json:"bases"`
	Matched               int      `json:"matched"`
	SkippedByAge          int      `json:"skipped_by_age,omitempty"`
	SkippedByPattern      int      `json:"skipped_by_pattern,omitempty"`
	BasePaths             []string `json:"base_paths,omitempty"`
	MatchedPaths          []string `json:"matched_paths,omitempty"`
	SkippedByPatternPaths []string `json:"skipped_by_pattern_paths,omitempty"`
	SkippedByAgePaths     []string `json:"skipped_by_age_paths,omitempty"`
}

type auditFollowImportsHealthView struct {
	File                  string           `json:"file"`
	Present               bool             `json:"present"`
	LastUpdatedAt         *time.Time       `json:"last_updated_at,omitempty"`
	Status                string           `json:"status,omitempty"`
	Source                string           `json:"source,omitempty"`
	InputCount            int              `json:"input_count,omitempty"`
	Continuous            bool             `json:"continuous"`
	PollIntervalSeconds   int64            `json:"poll_interval_seconds,omitempty"`
	SnapshotAgeSeconds    int64            `json:"snapshot_age_seconds,omitempty"`
	Stale                 bool             `json:"stale"`
	RequestedWatchMode    string           `json:"requested_watch_mode,omitempty"`
	ActiveWatchMode       string           `json:"active_watch_mode,omitempty"`
	WatchFallbacks        int              `json:"watch_fallbacks,omitempty"`
	WatchTransitions      int              `json:"watch_transitions,omitempty"`
	LastFallbackReason    string           `json:"last_fallback_reason,omitempty"`
	WatchPollCatchups     int              `json:"watch_poll_catchups,omitempty"`
	WatchPollCatchupBytes int              `json:"watch_poll_catchup_bytes,omitempty"`
	Warnings              []common.Warning `json:"warnings,omitempty"`
}

type followImportsRunTrigger string

const (
	followImportsRunTriggerInitial     followImportsRunTrigger = "initial"
	followImportsRunTriggerNotifyEvent followImportsRunTrigger = "notify_event"
	followImportsRunTriggerPollTick    followImportsRunTrigger = "poll_tick"
)

const followImportsCheckpointWindow = 256
const followImportsPollCatchupWarningThreshold = 3

const (
	followImportsTargetProfileAll       = "all"
	followImportsTargetProfileArtifacts = "artifacts"
	followImportsTargetProfileState     = "state"
	followImportsTargetProfileRetry     = "retry"
	followImportsTargetProfileHealth    = "health"
)

const (
	cleanupFollowImportsRetentionProfileStale = "stale"
	cleanupFollowImportsRetentionProfileDaily = "daily"
	cleanupFollowImportsRetentionProfileReset = "reset"
)

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

func runCleanupFollowImports(cfg config.Config, stdout io.Writer, args []string) error {
	options, err := parseCleanupFollowImportsOptions(args)
	if err != nil {
		return err
	}

	report, err := cleanupFollowImports(cfg, options)
	if err != nil {
		return err
	}

	if options.JSON {
		body, err := marshalIndented(report)
		if err != nil {
			return err
		}
		if _, err = io.WriteString(stdout, body); err != nil {
			return err
		}
		return cleanupFollowImportsMatchError(options, report)
	}

	if _, err = io.WriteString(stdout, formatCleanupFollowImportsReport(report)); err != nil {
		return err
	}
	return cleanupFollowImportsMatchError(options, report)
}

func runAuditFollowImports(cfg config.Config, stdout io.Writer, args []string) error {
	options, err := parseAuditFollowImportsOptions(args)
	if err != nil {
		return err
	}

	report, err := auditFollowImports(cfg, options)
	if err != nil {
		return err
	}

	if options.JSON {
		body, err := marshalIndented(report)
		if err != nil {
			return err
		}
		if _, err = io.WriteString(stdout, body); err != nil {
			return err
		}
		return auditFollowImportsMatchError(options, report)
	}

	if _, err = io.WriteString(stdout, formatAuditFollowImportsReport(report)); err != nil {
		return err
	}
	return auditFollowImportsMatchError(options, report)
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
		case ingestImportsInputFlag:
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.InputPaths = append(options.InputPaths, value)
			i = next
		case followImportsStateFileFlag:
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.StatePaths = append(options.StatePaths, value)
			i = next
		case ingestImportsFailedOutputFlag:
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.FailedOutputPath = value
			i = next
		case ingestImportsFailedManifestFlag:
			value, next, err := optionValue(args, i)
			if err != nil {
				return followImportsOptions{}, err
			}
			options.FailedManifestPath = value
			i = next
		case ingestImportsCWDFlag:
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
		case doctorJSONFlag:
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

func parseCleanupFollowImportsOptions(args []string) (cleanupFollowImportsOptions, error) {
	var options cleanupFollowImportsOptions

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		next, err := parseCleanupFollowImportsOption(args, i, arg, &options)
		if err != nil {
			return cleanupFollowImportsOptions{}, err
		}
		i = next
	}

	applyCleanupFollowImportsTargetProfile(&options)
	applyFollowImportsHygieneRetentionProfile(&options.followImportsHygieneOptions)
	if err := validateCleanupFollowImportsOptions(options); err != nil {
		return cleanupFollowImportsOptions{}, err
	}
	return options, nil
}

func parseCleanupFollowImportsOption(args []string, index int, arg string, options *cleanupFollowImportsOptions) (int, error) {
	if options == nil {
		return index, fmt.Errorf("cleanup-follow-imports options are required")
	}
	switch arg {
	case "":
		return index, nil
	case "--dry-run":
		options.DryRun = true
		return index, nil
	case "--prune-state":
		options.PruneState = true
		return index, nil
	case "--prune-failed-output":
		options.PruneFailedOutput = true
		return index, nil
	case "--prune-failed-manifest":
		options.PruneFailedManifest = true
		return index, nil
	case "--prune-stale-follow-health":
		options.PruneStaleFollowHealth = true
		return index, nil
	}
	if parseFollowImportsHygieneBooleanFlag(arg, &options.followImportsHygieneOptions) {
		return index, nil
	}
	return parseFollowImportsHygieneOptionValue(args, index, arg, "cleanup-follow-imports", &options.followImportsHygieneOptions)
}

func parseFollowImportsHygieneBooleanFlag(arg string, options *followImportsHygieneOptions) bool {
	if options == nil {
		return false
	}
	switch arg {
	case "--json":
		options.JSON = true
		return true
	case "--fail-if-matched":
		options.FailIfMatched = true
		return true
	default:
		return false
	}
}

func parseFollowImportsHygieneOptionValue(args []string, index int, arg string, command string, options *followImportsHygieneOptions) (int, error) {
	value, next, err := optionValue(args, index)
	if err != nil {
		return index, err
	}
	if options == nil {
		return index, fmt.Errorf("%s options are required", command)
	}
	switch arg {
	case ingestImportsInputFlag:
		options.InputPaths = append(options.InputPaths, value)
	case followImportsStateFileFlag:
		options.StatePaths = append(options.StatePaths, value)
	case ingestImportsFailedOutputFlag:
		options.FailedOutputPath = value
	case ingestImportsFailedManifestFlag:
		options.FailedManifestPath = value
	case "--include":
		options.IncludePatterns = append(options.IncludePatterns, parseCleanupFollowImportsPatterns(value)...)
	case "--exclude":
		options.ExcludePatterns = append(options.ExcludePatterns, parseCleanupFollowImportsPatterns(value)...)
	case "--target-profile":
		profile, err := normalizeFollowImportsTargetProfile(value)
		if err != nil {
			return index, err
		}
		options.TargetProfile = profile
	case "--retention-profile":
		profile, err := normalizeCleanupFollowImportsRetentionProfile(value)
		if err != nil {
			return index, err
		}
		options.RetentionProfile = profile
	case ingestImportsCWDFlag:
		options.CWD = value
	case "--older-than":
		duration, err := time.ParseDuration(value)
		if err != nil {
			return index, fmt.Errorf("invalid %s older-than duration %q", command, value)
		}
		options.OlderThan = duration
		options.olderThanExplicit = true
	default:
		return index, fmt.Errorf("unknown %s flag %q", command, arg)
	}
	return next, nil
}

func parseAuditFollowImportsOptions(args []string) (auditFollowImportsOptions, error) {
	var options auditFollowImportsOptions

	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		next, err := parseAuditFollowImportsOption(args, i, arg, &options)
		if err != nil {
			return auditFollowImportsOptions{}, err
		}
		i = next
	}

	applyAuditFollowImportsTargetProfile(&options)
	applyFollowImportsHygieneRetentionProfile(&options.followImportsHygieneOptions)
	if err := validateAuditFollowImportsOptions(options); err != nil {
		return auditFollowImportsOptions{}, err
	}
	return options, nil
}

func parseAuditFollowImportsOption(args []string, index int, arg string, options *auditFollowImportsOptions) (int, error) {
	if options == nil {
		return index, fmt.Errorf("audit-follow-imports options are required")
	}
	switch arg {
	case "":
		return index, nil
	case "--check-state":
		options.CheckState = true
		return index, nil
	case "--check-failed-output":
		options.CheckFailedOutput = true
		return index, nil
	case "--check-failed-manifest":
		options.CheckFailedManifest = true
		return index, nil
	case "--check-follow-health":
		options.CheckFollowHealth = true
		return index, nil
	}
	if parseFollowImportsHygieneBooleanFlag(arg, &options.followImportsHygieneOptions) {
		return index, nil
	}
	return parseFollowImportsHygieneOptionValue(args, index, arg, "audit-follow-imports", &options.followImportsHygieneOptions)
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

func validateCleanupFollowImportsOptions(options cleanupFollowImportsOptions) error {
	if err := validateCleanupFollowImportsTargets(options); err != nil {
		return err
	}
	return validateFollowImportsHygieneCommonOptions("cleanup-follow-imports", options.followImportsHygieneOptions)
}

func validateCleanupFollowImportsTargets(options cleanupFollowImportsOptions) error {
	if !options.PruneState && !options.PruneFailedOutput && !options.PruneFailedManifest && !options.PruneStaleFollowHealth {
		return fmt.Errorf("cleanup-follow-imports requires at least one prune target")
	}
	return validateFollowImportsHygieneTargets(
		"cleanup-follow-imports",
		options.followImportsHygieneOptions,
		options.PruneState,
		"--prune-state",
		options.PruneFailedOutput,
		"--prune-failed-output",
		options.PruneFailedManifest,
		"--prune-failed-manifest",
	)
}

func validateFollowImportsHygieneCommonOptions(command string, options followImportsHygieneOptions) error {
	if err := validateCleanupFollowImportsPatterns(options.IncludePatterns, "--include"); err != nil {
		return err
	}
	if err := validateCleanupFollowImportsPatterns(options.ExcludePatterns, "--exclude"); err != nil {
		return err
	}
	if options.OlderThan < 0 {
		return fmt.Errorf("%s --older-than must be greater than or equal to zero", command)
	}
	return nil
}

func validateFollowImportsHygieneTargets(command string, options followImportsHygieneOptions, stateEnabled bool, stateActionFlag string, failedOutputEnabled bool, failedOutputActionFlag string, failedManifestEnabled bool, failedManifestActionFlag string) error {
	if len(options.StatePaths) > 0 && !stateEnabled {
		return fmt.Errorf("%s --state-file requires %s", command, stateActionFlag)
	}
	if options.FailedOutputPath != "" && !failedOutputEnabled {
		return fmt.Errorf("%s --failed-output requires %s", command, failedOutputActionFlag)
	}
	if options.FailedManifestPath != "" && !failedManifestEnabled {
		return fmt.Errorf("%s --failed-manifest requires %s", command, failedManifestActionFlag)
	}
	if stateEnabled && len(options.StatePaths) == 0 && len(options.InputPaths) == 0 {
		return fmt.Errorf("%s %s requires --input or --state-file", command, stateActionFlag)
	}
	if len(options.StatePaths) > 0 && len(options.InputPaths) > 0 && len(options.StatePaths) != len(options.InputPaths) {
		return fmt.Errorf("%s state-file count (%d) must match input count (%d)", command, len(options.StatePaths), len(options.InputPaths))
	}
	if failedOutputEnabled && strings.TrimSpace(options.FailedOutputPath) == "" {
		return fmt.Errorf("%s %s requires --failed-output", command, failedOutputActionFlag)
	}
	if failedManifestEnabled && strings.TrimSpace(options.FailedManifestPath) == "" {
		return fmt.Errorf("%s %s requires --failed-manifest", command, failedManifestActionFlag)
	}
	return nil
}

func validateAuditFollowImportsOptions(options auditFollowImportsOptions) error {
	if err := validateAuditFollowImportsTargets(options); err != nil {
		return err
	}
	return validateFollowImportsHygieneCommonOptions("audit-follow-imports", options.followImportsHygieneOptions)
}

func validateAuditFollowImportsTargets(options auditFollowImportsOptions) error {
	if !options.CheckState && !options.CheckFailedOutput && !options.CheckFailedManifest && !options.CheckFollowHealth {
		return fmt.Errorf("audit-follow-imports requires at least one check target")
	}
	return validateFollowImportsHygieneTargets(
		"audit-follow-imports",
		options.followImportsHygieneOptions,
		options.CheckState,
		"--check-state",
		options.CheckFailedOutput,
		"--check-failed-output",
		options.CheckFailedManifest,
		"--check-failed-manifest",
	)
}

func normalizeCleanupFollowImportsRetentionProfile(raw string) (string, error) {
	profile := strings.ToLower(strings.TrimSpace(raw))
	switch profile {
	case cleanupFollowImportsRetentionProfileStale, cleanupFollowImportsRetentionProfileDaily, cleanupFollowImportsRetentionProfileReset:
		return profile, nil
	case "":
		return "", fmt.Errorf(`invalid value for "--retention-profile": empty`)
	default:
		return "", fmt.Errorf(`invalid value for "--retention-profile": %q`, raw)
	}
}

func normalizeFollowImportsTargetProfile(raw string) (string, error) {
	profile := strings.ToLower(strings.TrimSpace(raw))
	switch profile {
	case followImportsTargetProfileAll, followImportsTargetProfileArtifacts, followImportsTargetProfileState, followImportsTargetProfileRetry, followImportsTargetProfileHealth:
		return profile, nil
	case "":
		return "", fmt.Errorf(`invalid value for "--target-profile": empty`)
	default:
		return "", fmt.Errorf(`invalid value for "--target-profile": %q`, raw)
	}
}

func applyCleanupFollowImportsTargetProfile(options *cleanupFollowImportsOptions) {
	if options == nil {
		return
	}
	switch options.TargetProfile {
	case followImportsTargetProfileAll:
		options.PruneState = true
		options.PruneFailedOutput = true
		options.PruneFailedManifest = true
		options.PruneStaleFollowHealth = true
	case followImportsTargetProfileArtifacts:
		options.PruneState = true
		options.PruneFailedOutput = true
		options.PruneFailedManifest = true
	case followImportsTargetProfileState:
		options.PruneState = true
	case followImportsTargetProfileRetry:
		options.PruneFailedOutput = true
		options.PruneFailedManifest = true
	case followImportsTargetProfileHealth:
		options.PruneStaleFollowHealth = true
	}
}

func applyAuditFollowImportsTargetProfile(options *auditFollowImportsOptions) {
	if options == nil {
		return
	}
	switch options.TargetProfile {
	case followImportsTargetProfileAll:
		options.CheckState = true
		options.CheckFailedOutput = true
		options.CheckFailedManifest = true
		options.CheckFollowHealth = true
	case followImportsTargetProfileArtifacts:
		options.CheckState = true
		options.CheckFailedOutput = true
		options.CheckFailedManifest = true
	case followImportsTargetProfileState:
		options.CheckState = true
	case followImportsTargetProfileRetry:
		options.CheckFailedOutput = true
		options.CheckFailedManifest = true
	case followImportsTargetProfileHealth:
		options.CheckFollowHealth = true
	}
}

func applyFollowImportsHygieneRetentionProfile(options *followImportsHygieneOptions) {
	if options == nil {
		return
	}
	if !options.olderThanExplicit {
		options.OlderThan = cleanupFollowImportsRetentionProfileOlderThan(options.RetentionProfile)
	}
}

func cleanupFollowImportsRetentionProfileOlderThan(profile string) time.Duration {
	switch profile {
	case cleanupFollowImportsRetentionProfileStale:
		return time.Hour
	case cleanupFollowImportsRetentionProfileDaily:
		return 24 * time.Hour
	case cleanupFollowImportsRetentionProfileReset:
		return 0
	default:
		return 0
	}
}

func parseCleanupFollowImportsPatterns(value string) []string {
	parts := strings.Split(value, ",")
	patterns := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		patterns = append(patterns, part)
	}
	return patterns
}

func validateCleanupFollowImportsPatterns(patterns []string, flag string) error {
	for _, pattern := range patterns {
		if _, err := pathpkg.Match(pattern, ""); err != nil {
			return fmt.Errorf("invalid cleanup-follow-imports %s pattern %q", flag, pattern)
		}
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

func cleanupFollowImports(cfg config.Config, options cleanupFollowImportsOptions) (cleanupFollowImportsReport, error) {
	return cleanupFollowImportsAt(cfg, options, time.Now().UTC())
}

func cleanupFollowImportsAt(cfg config.Config, options cleanupFollowImportsOptions, now time.Time) (cleanupFollowImportsReport, error) {
	targets, err := buildCleanupFollowImportsTargets(options)
	if err != nil {
		return cleanupFollowImportsReport{}, err
	}

	report := cleanupFollowImportsReport{
		DryRun:           options.DryRun,
		FailIfMatched:    options.FailIfMatched,
		TargetProfile:    options.TargetProfile,
		RetentionProfile: options.RetentionProfile,
		OlderThanSeconds: int64(options.OlderThan / time.Second),
		IncludePatterns:  append([]string(nil), options.IncludePatterns...),
		ExcludePatterns:  append([]string(nil), options.ExcludePatterns...),
		Status:           "ok",
		FollowHealth: cleanupFollowImportsFollowHealthView{
			File: followImportsHealthPath(cfg.Meta.LogDir),
		},
	}

	if options.PruneState {
		report.StateFiles, err = pruneCleanupFollowImportsPaths(targets.statePaths, options, now)
		if err != nil {
			return cleanupFollowImportsReport{}, err
		}
	}
	if options.PruneFailedOutput {
		report.FailedOutputs, err = pruneCleanupFollowImportsPatternTargets(targets.failedOutputBases, options, now)
		if err != nil {
			return cleanupFollowImportsReport{}, err
		}
	}
	if options.PruneFailedManifest {
		report.FailedManifests, err = pruneCleanupFollowImportsPatternTargets(targets.failedManifestBases, options, now)
		if err != nil {
			return cleanupFollowImportsReport{}, err
		}
	}
	if options.PruneStaleFollowHealth {
		report.FollowHealth.WouldPrune, report.FollowHealth.Pruned, report.FollowHealth.PruneReason, err = cleanupFollowImportsHealthSnapshot(cfg.Meta.LogDir, options.DryRun, now)
		if err != nil {
			return cleanupFollowImportsReport{}, err
		}
	}
	report.MatchFound = cleanupFollowImportsHasMatches(report)

	return report, nil
}

func cleanupFollowImportsHasMatches(report cleanupFollowImportsReport) bool {
	return report.StateFiles.Matched > 0 ||
		report.FailedOutputs.Matched > 0 ||
		report.FailedManifests.Matched > 0 ||
		report.FollowHealth.WouldPrune ||
		report.FollowHealth.Pruned
}

func cleanupFollowImportsMatchError(options cleanupFollowImportsOptions, report cleanupFollowImportsReport) error {
	if !options.FailIfMatched || !report.MatchFound {
		return nil
	}
	return fmt.Errorf("cleanup-follow-imports found matching artifacts; see report output")
}

func auditFollowImports(cfg config.Config, options auditFollowImportsOptions) (auditFollowImportsReport, error) {
	return auditFollowImportsAt(cfg, options, time.Now().UTC())
}

func auditFollowImportsAt(cfg config.Config, options auditFollowImportsOptions, now time.Time) (auditFollowImportsReport, error) {
	targets, err := buildAuditFollowImportsTargets(options)
	if err != nil {
		return auditFollowImportsReport{}, err
	}

	scanOptions := options.cleanupDryRunOptions()
	report := auditFollowImportsReport{
		FailIfMatched:    options.FailIfMatched,
		TargetProfile:    options.TargetProfile,
		RetentionProfile: options.RetentionProfile,
		OlderThanSeconds: int64(options.OlderThan / time.Second),
		IncludePatterns:  append([]string(nil), options.IncludePatterns...),
		ExcludePatterns:  append([]string(nil), options.ExcludePatterns...),
		Status:           "ok",
		FollowHealth: auditFollowImportsHealthView{
			File: followImportsHealthPath(cfg.Meta.LogDir),
		},
	}

	if options.CheckState {
		summary, err := pruneCleanupFollowImportsPaths(targets.statePaths, scanOptions, now)
		if err != nil {
			return auditFollowImportsReport{}, err
		}
		report.StateFiles = newAuditFollowImportsPathSummary(summary)
	}
	if options.CheckFailedOutput {
		summary, err := pruneCleanupFollowImportsPatternTargets(targets.failedOutputBases, scanOptions, now)
		if err != nil {
			return auditFollowImportsReport{}, err
		}
		report.FailedOutputs = newAuditFollowImportsPatternSummary(summary)
	}
	if options.CheckFailedManifest {
		summary, err := pruneCleanupFollowImportsPatternTargets(targets.failedManifestBases, scanOptions, now)
		if err != nil {
			return auditFollowImportsReport{}, err
		}
		report.FailedManifests = newAuditFollowImportsPatternSummary(summary)
	}
	if options.CheckFollowHealth {
		report.FollowHealth, err = inspectAuditFollowImportsHealth(cfg.Meta.LogDir, now)
		if err != nil {
			return auditFollowImportsReport{}, err
		}
	}

	report.MatchFound = auditFollowImportsHasMatches(report)
	return report, nil
}

func newAuditFollowImportsPathSummary(summary cleanupFollowImportsPathSummary) auditFollowImportsPathSummary {
	return auditFollowImportsPathSummary{
		Requested:             summary.Requested,
		Matched:               summary.Matched,
		Missing:               summary.Missing,
		SkippedByAge:          summary.SkippedByAge,
		SkippedByPattern:      summary.SkippedByPattern,
		MatchedPaths:          append([]string(nil), summary.MatchedPaths...),
		MissingPaths:          append([]string(nil), summary.MissingPaths...),
		SkippedByPatternPaths: append([]string(nil), summary.SkippedByPatternPaths...),
		SkippedByAgePaths:     append([]string(nil), summary.SkippedByAgePaths...),
	}
}

func newAuditFollowImportsPatternSummary(summary cleanupFollowImportsPatternSummary) auditFollowImportsPatternSummary {
	return auditFollowImportsPatternSummary{
		Bases:                 summary.Bases,
		Matched:               summary.Matched,
		SkippedByAge:          summary.SkippedByAge,
		SkippedByPattern:      summary.SkippedByPattern,
		BasePaths:             append([]string(nil), summary.BasePaths...),
		MatchedPaths:          append([]string(nil), summary.MatchedPaths...),
		SkippedByPatternPaths: append([]string(nil), summary.SkippedByPatternPaths...),
		SkippedByAgePaths:     append([]string(nil), summary.SkippedByAgePaths...),
	}
}

func inspectAuditFollowImportsHealth(logDir string, now time.Time) (auditFollowImportsHealthView, error) {
	view := auditFollowImportsHealthView{
		File: followImportsHealthPath(logDir),
	}
	followHealth, err := loadFollowImportsHealthSnapshot(logDir)
	if err != nil {
		return auditFollowImportsHealthView{}, err
	}
	if followHealth == nil {
		return view, nil
	}

	updatedAt := followHealth.UpdatedAt
	age, stale := evaluateFollowImportsHealthStaleness(*followHealth, now)
	view.Present = true
	view.LastUpdatedAt = &updatedAt
	view.Status = followHealth.Status
	view.Source = followHealth.Source
	view.InputCount = followHealth.InputCount
	view.Continuous = followHealth.Continuous
	view.PollIntervalSeconds = followHealth.PollIntervalSeconds
	view.SnapshotAgeSeconds = int64(age / time.Second)
	view.Stale = stale
	view.RequestedWatchMode = followHealth.RequestedWatchMode
	view.ActiveWatchMode = followHealth.ActiveWatchMode
	view.WatchFallbacks = followHealth.WatchFallbacks
	view.WatchTransitions = followHealth.WatchTransitions
	view.LastFallbackReason = followHealth.LastFallbackReason
	view.WatchPollCatchups = followHealth.WatchPollCatchups
	view.WatchPollCatchupBytes = followHealth.WatchCatchupBytes
	view.Warnings = common.MergeWarnings(
		append([]common.Warning(nil), followHealth.Warnings...),
		followImportsHealthStaleWarnings(*followHealth, age, stale),
	)
	return view, nil
}

func auditFollowImportsHasMatches(report auditFollowImportsReport) bool {
	return report.StateFiles.Matched > 0 ||
		report.FailedOutputs.Matched > 0 ||
		report.FailedManifests.Matched > 0 ||
		report.FollowHealth.Stale
}

func auditFollowImportsMatchError(options auditFollowImportsOptions, report auditFollowImportsReport) error {
	if !options.FailIfMatched || !report.MatchFound {
		return nil
	}
	return fmt.Errorf("audit-follow-imports found matching artifacts; see report output")
}

func (options auditFollowImportsOptions) cleanupDryRunOptions() cleanupFollowImportsOptions {
	return cleanupFollowImportsOptions{
		followImportsHygieneOptions: followImportsHygieneOptions{
			InputPaths:         append([]string(nil), options.InputPaths...),
			StatePaths:         append([]string(nil), options.StatePaths...),
			FailedOutputPath:   options.FailedOutputPath,
			FailedManifestPath: options.FailedManifestPath,
			IncludePatterns:    append([]string(nil), options.IncludePatterns...),
			ExcludePatterns:    append([]string(nil), options.ExcludePatterns...),
			TargetProfile:      options.TargetProfile,
			RetentionProfile:   options.RetentionProfile,
			CWD:                options.CWD,
			OlderThan:          options.OlderThan,
			olderThanExplicit:  options.olderThanExplicit,
		},
		DryRun:                 true,
		PruneState:             options.CheckState,
		PruneFailedOutput:      options.CheckFailedOutput,
		PruneFailedManifest:    options.CheckFailedManifest,
		PruneStaleFollowHealth: options.CheckFollowHealth,
	}
}

type cleanupFollowImportsTargets struct {
	statePaths          []string
	failedOutputBases   []string
	failedManifestBases []string
}

func buildCleanupFollowImportsTargets(options cleanupFollowImportsOptions) (cleanupFollowImportsTargets, error) {
	cwd, err := resolveIngestImportsCWD(options.CWD)
	if err != nil {
		return cleanupFollowImportsTargets{}, err
	}

	inputs, err := resolveCleanupFollowImportsInputs(options.InputPaths, cwd)
	if err != nil {
		return cleanupFollowImportsTargets{}, err
	}

	var targets cleanupFollowImportsTargets
	if options.PruneState {
		if len(options.StatePaths) > 0 {
			targets.statePaths, err = resolveCleanupFollowImportsPaths(options.StatePaths, cwd)
			if err != nil {
				return cleanupFollowImportsTargets{}, err
			}
		} else {
			for _, inputPath := range inputs {
				statePath, err := resolveFollowImportsStatePath(inputPath, "", cwd)
				if err != nil {
					return cleanupFollowImportsTargets{}, err
				}
				targets.statePaths = append(targets.statePaths, statePath)
			}
		}
		targets.statePaths = dedupeCleanupFollowImportsPaths(targets.statePaths)
	}
	if options.PruneFailedOutput {
		targets.failedOutputBases, err = resolveCleanupFollowImportsArtifactBases(options.FailedOutputPath, inputs, cwd)
		if err != nil {
			return cleanupFollowImportsTargets{}, err
		}
	}
	if options.PruneFailedManifest {
		targets.failedManifestBases, err = resolveCleanupFollowImportsArtifactBases(options.FailedManifestPath, inputs, cwd)
		if err != nil {
			return cleanupFollowImportsTargets{}, err
		}
	}
	return targets, nil
}

func buildAuditFollowImportsTargets(options auditFollowImportsOptions) (cleanupFollowImportsTargets, error) {
	cwd, err := resolveIngestImportsCWD(options.CWD)
	if err != nil {
		return cleanupFollowImportsTargets{}, err
	}

	inputs, err := resolveAuditFollowImportsInputs(options.InputPaths, cwd)
	if err != nil {
		return cleanupFollowImportsTargets{}, err
	}

	var targets cleanupFollowImportsTargets
	if options.CheckState {
		if len(options.StatePaths) > 0 {
			targets.statePaths, err = resolveCleanupFollowImportsPaths(options.StatePaths, cwd)
			if err != nil {
				return cleanupFollowImportsTargets{}, err
			}
		} else {
			for _, inputPath := range inputs {
				statePath, err := resolveFollowImportsStatePath(inputPath, "", cwd)
				if err != nil {
					return cleanupFollowImportsTargets{}, err
				}
				targets.statePaths = append(targets.statePaths, statePath)
			}
		}
		targets.statePaths = dedupeCleanupFollowImportsPaths(targets.statePaths)
	}
	if options.CheckFailedOutput {
		targets.failedOutputBases, err = resolveAuditFollowImportsArtifactBases(options.FailedOutputPath, inputs, cwd)
		if err != nil {
			return cleanupFollowImportsTargets{}, err
		}
	}
	if options.CheckFailedManifest {
		targets.failedManifestBases, err = resolveAuditFollowImportsArtifactBases(options.FailedManifestPath, inputs, cwd)
		if err != nil {
			return cleanupFollowImportsTargets{}, err
		}
	}
	return targets, nil
}

func resolveCleanupFollowImportsInputs(inputPaths []string, cwd string) ([]string, error) {
	resolved, err := resolveCleanupFollowImportsPaths(inputPaths, cwd)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(resolved))
	for _, inputPath := range resolved {
		key := followImportsPathKey(inputPath)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("cleanup-follow-imports input %q is duplicated", inputPath)
		}
		seen[key] = struct{}{}
	}
	return resolved, nil
}

func resolveAuditFollowImportsInputs(inputPaths []string, cwd string) ([]string, error) {
	resolved, err := resolveCleanupFollowImportsPaths(inputPaths, cwd)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(resolved))
	for _, inputPath := range resolved {
		key := followImportsPathKey(inputPath)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("audit-follow-imports input %q is duplicated", inputPath)
		}
		seen[key] = struct{}{}
	}
	return resolved, nil
}

func resolveCleanupFollowImportsPaths(paths []string, cwd string) ([]string, error) {
	resolved := make([]string, 0, len(paths))
	for _, path := range paths {
		resolvedPath, err := resolveIngestImportsPath(path, cwd)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(resolvedPath) == "" {
			continue
		}
		resolved = append(resolved, resolvedPath)
	}
	return resolved, nil
}

func resolveAuditFollowImportsArtifactBases(base string, inputs []string, cwd string) ([]string, error) {
	resolvedBase, err := resolveIngestImportsPath(base, cwd)
	if err != nil {
		return nil, err
	}
	if len(inputs) == 0 {
		return []string{resolvedBase}, nil
	}
	bases := make([]string, 0, len(inputs))
	for _, inputPath := range inputs {
		bases = append(bases, deriveFollowImportsInputBasePath(resolvedBase, inputPath, len(inputs)))
	}
	return dedupeCleanupFollowImportsPaths(bases), nil
}

func resolveCleanupFollowImportsArtifactBases(base string, inputs []string, cwd string) ([]string, error) {
	resolvedBase, err := resolveIngestImportsPath(base, cwd)
	if err != nil {
		return nil, err
	}
	if len(inputs) == 0 {
		return []string{resolvedBase}, nil
	}
	bases := make([]string, 0, len(inputs))
	for _, inputPath := range inputs {
		bases = append(bases, deriveFollowImportsInputBasePath(resolvedBase, inputPath, len(inputs)))
	}
	return dedupeCleanupFollowImportsPaths(bases), nil
}

func dedupeCleanupFollowImportsPaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(paths))
	deduped := make([]string, 0, len(paths))
	for _, path := range paths {
		key := followImportsPathKey(path)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, path)
	}
	sort.Strings(deduped)
	return deduped
}

func pruneCleanupFollowImportsPaths(paths []string, options cleanupFollowImportsOptions, now time.Time) (cleanupFollowImportsPathSummary, error) {
	summary := cleanupFollowImportsPathSummary{
		Requested: len(paths),
	}
	for _, path := range paths {
		matched, removed, missing, skippedByPattern, skippedByAge, err := maybeRemoveCleanupFollowImportsPath(path, options, now)
		if err != nil {
			return cleanupFollowImportsPathSummary{}, err
		}
		switch {
		case skippedByPattern:
			summary.SkippedByPatternPaths = append(summary.SkippedByPatternPaths, path)
		case missing:
			summary.MissingPaths = append(summary.MissingPaths, path)
		case skippedByAge:
			summary.SkippedByAgePaths = append(summary.SkippedByAgePaths, path)
		case matched:
			summary.MatchedPaths = append(summary.MatchedPaths, path)
		}
		if removed {
			summary.RemovedPaths = append(summary.RemovedPaths, path)
		}
	}
	sort.Strings(summary.MatchedPaths)
	sort.Strings(summary.RemovedPaths)
	sort.Strings(summary.MissingPaths)
	sort.Strings(summary.SkippedByPatternPaths)
	sort.Strings(summary.SkippedByAgePaths)
	summary.Matched = len(summary.MatchedPaths)
	summary.Removed = len(summary.RemovedPaths)
	summary.Missing = len(summary.MissingPaths)
	summary.SkippedByPattern = len(summary.SkippedByPatternPaths)
	summary.SkippedByAge = len(summary.SkippedByAgePaths)
	return summary, nil
}

func maybeRemoveCleanupFollowImportsPath(path string, options cleanupFollowImportsOptions, now time.Time) (matched bool, removed bool, missing bool, skippedByPattern bool, skippedByAge bool, err error) {
	included, err := cleanupFollowImportsPathIncluded(path, options.IncludePatterns, options.ExcludePatterns)
	if err != nil {
		return false, false, false, false, false, err
	}
	if !included {
		return false, false, false, true, false, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, true, false, false, nil
		}
		return false, false, false, false, false, fmt.Errorf("stat cleanup-follow-imports path %q: %w", path, err)
	}
	if cleanupFollowImportsTooNew(info.ModTime(), options.OlderThan, now) {
		return false, false, false, false, true, nil
	}
	if options.DryRun {
		return true, false, false, false, false, nil
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return false, false, true, false, false, nil
		}
		return false, false, false, false, false, fmt.Errorf("remove cleanup-follow-imports path %q: %w", path, err)
	}
	return true, true, false, false, false, nil
}

func pruneCleanupFollowImportsPatternTargets(bases []string, options cleanupFollowImportsOptions, now time.Time) (cleanupFollowImportsPatternSummary, error) {
	summary := cleanupFollowImportsPatternSummary{
		Bases:     len(bases),
		BasePaths: append([]string(nil), bases...),
	}
	sort.Strings(summary.BasePaths)
	for _, base := range bases {
		matches, err := listCleanupFollowImportsBatchArtifacts(base)
		if err != nil {
			return cleanupFollowImportsPatternSummary{}, err
		}
		for _, match := range matches {
			matched, removed, _, skippedByPattern, skippedByAge, err := maybeRemoveCleanupFollowImportsPath(match, options, now)
			if err != nil {
				return cleanupFollowImportsPatternSummary{}, err
			}
			if matched {
				summary.MatchedPaths = append(summary.MatchedPaths, match)
			}
			if removed {
				summary.RemovedPaths = append(summary.RemovedPaths, match)
			}
			if skippedByPattern {
				summary.SkippedByPatternPaths = append(summary.SkippedByPatternPaths, match)
			}
			if skippedByAge {
				summary.SkippedByAgePaths = append(summary.SkippedByAgePaths, match)
			}
		}
	}
	sort.Strings(summary.MatchedPaths)
	sort.Strings(summary.RemovedPaths)
	sort.Strings(summary.SkippedByPatternPaths)
	sort.Strings(summary.SkippedByAgePaths)
	summary.Matched = len(summary.MatchedPaths)
	summary.Removed = len(summary.RemovedPaths)
	summary.SkippedByPattern = len(summary.SkippedByPatternPaths)
	summary.SkippedByAge = len(summary.SkippedByAgePaths)
	return summary, nil
}

func listCleanupFollowImportsBatchArtifacts(base string) ([]string, error) {
	dir := filepath.Dir(base)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list cleanup-follow-imports artifacts for %q: %w", base, err)
	}

	baseName := filepath.Base(base)
	ext := filepath.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)
	var matches []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !cleanupFollowImportsBatchArtifactMatches(entry.Name(), stem, ext) {
			continue
		}
		matches = append(matches, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(matches)
	return matches, nil
}

func cleanupFollowImportsTooNew(modTime time.Time, olderThan time.Duration, now time.Time) bool {
	if olderThan <= 0 {
		return false
	}
	age := now.Sub(modTime)
	if age < 0 {
		age = 0
	}
	return age < olderThan
}

func cleanupFollowImportsPathIncluded(candidate string, includePatterns []string, excludePatterns []string) (bool, error) {
	matchesInclude, err := cleanupFollowImportsMatchAnyPattern(candidate, includePatterns)
	if err != nil {
		return false, err
	}
	if len(includePatterns) > 0 && !matchesInclude {
		return false, nil
	}
	matchesExclude, err := cleanupFollowImportsMatchAnyPattern(candidate, excludePatterns)
	if err != nil {
		return false, err
	}
	return !matchesExclude, nil
}

func cleanupFollowImportsMatchAnyPattern(candidate string, patterns []string) (bool, error) {
	if len(patterns) == 0 {
		return false, nil
	}
	slashCandidate := filepath.ToSlash(filepath.Clean(strings.TrimSpace(candidate)))
	baseCandidate := filepath.Base(slashCandidate)
	for _, pattern := range patterns {
		matched, err := pathpkg.Match(pattern, slashCandidate)
		if err != nil {
			return false, fmt.Errorf("match cleanup-follow-imports pattern %q against %q: %w", pattern, candidate, err)
		}
		if matched {
			return true, nil
		}
		matched, err = pathpkg.Match(pattern, baseCandidate)
		if err != nil {
			return false, fmt.Errorf("match cleanup-follow-imports pattern %q against %q: %w", pattern, candidate, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func cleanupFollowImportsHealthSnapshot(logDir string, dryRun bool, now time.Time) (bool, bool, string, error) {
	followHealth, err := loadFollowImportsHealthSnapshot(logDir)
	if err != nil {
		return false, false, "", err
	}
	if followHealth == nil {
		return false, false, "", nil
	}
	_, stale := evaluateFollowImportsHealthStaleness(*followHealth, now)
	if !stale {
		return false, false, "", nil
	}
	if dryRun {
		return true, false, "stale", nil
	}
	if err := pruneFollowImportsHealthSnapshot(logDir); err != nil {
		return false, false, "", err
	}
	return false, true, "stale", nil
}

func cleanupFollowImportsBatchArtifactMatches(name string, stem string, ext string) bool {
	if !strings.HasPrefix(name, stem+".") {
		return false
	}
	if ext != "" {
		if !strings.HasSuffix(name, ext) {
			return false
		}
		name = strings.TrimSuffix(name, ext)
	}
	name = strings.TrimPrefix(name, stem+".")
	if name == "" {
		return false
	}
	parts := strings.Split(name, ".")
	return cleanupFollowImportsIsRangeToken(parts[len(parts)-1])
}

func cleanupFollowImportsIsRangeToken(value string) bool {
	value = strings.TrimSpace(value)
	dash := strings.IndexByte(value, '-')
	if dash <= 0 || dash == len(value)-1 {
		return false
	}
	return cleanupFollowImportsDigitsOnly(value[:dash]) && cleanupFollowImportsDigitsOnly(value[dash+1:])
}

func cleanupFollowImportsDigitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func formatCleanupFollowImportsReport(report cleanupFollowImportsReport) string {
	lines := []string{
		"cleanup follow-imports ok",
		fmt.Sprintf("status=%s", report.Status),
		fmt.Sprintf("dry_run=%t", report.DryRun),
		fmt.Sprintf("fail_if_matched=%t", report.FailIfMatched),
		fmt.Sprintf("match_found=%t", report.MatchFound),
		fmt.Sprintf("retention_profile=%s", fallbackString(report.RetentionProfile)),
		fmt.Sprintf("older_than_seconds=%d", report.OlderThanSeconds),
		fmt.Sprintf("include_patterns=%d", len(report.IncludePatterns)),
		fmt.Sprintf("exclude_patterns=%d", len(report.ExcludePatterns)),
		fmt.Sprintf("state_files_requested=%d", report.StateFiles.Requested),
		fmt.Sprintf("state_files_matched=%d", report.StateFiles.Matched),
		fmt.Sprintf("state_files_removed=%d", report.StateFiles.Removed),
		fmt.Sprintf("state_files_missing=%d", report.StateFiles.Missing),
		fmt.Sprintf("state_files_skipped_by_pattern=%d", report.StateFiles.SkippedByPattern),
		fmt.Sprintf("state_files_skipped_by_age=%d", report.StateFiles.SkippedByAge),
		fmt.Sprintf("failed_output_bases=%d", report.FailedOutputs.Bases),
		fmt.Sprintf("failed_output_matched=%d", report.FailedOutputs.Matched),
		fmt.Sprintf("failed_output_removed=%d", report.FailedOutputs.Removed),
		fmt.Sprintf("failed_output_skipped_by_pattern=%d", report.FailedOutputs.SkippedByPattern),
		fmt.Sprintf("failed_output_skipped_by_age=%d", report.FailedOutputs.SkippedByAge),
		fmt.Sprintf("failed_manifest_bases=%d", report.FailedManifests.Bases),
		fmt.Sprintf("failed_manifest_matched=%d", report.FailedManifests.Matched),
		fmt.Sprintf("failed_manifest_removed=%d", report.FailedManifests.Removed),
		fmt.Sprintf("failed_manifest_skipped_by_pattern=%d", report.FailedManifests.SkippedByPattern),
		fmt.Sprintf("failed_manifest_skipped_by_age=%d", report.FailedManifests.SkippedByAge),
		fmt.Sprintf("follow_health_file=%s", report.FollowHealth.File),
		fmt.Sprintf("follow_health_would_prune=%t", report.FollowHealth.WouldPrune),
		fmt.Sprintf("follow_health_pruned=%t", report.FollowHealth.Pruned),
		fmt.Sprintf("follow_health_prune_reason=%s", fallbackString(report.FollowHealth.PruneReason)),
	}
	if report.TargetProfile != "" {
		lines = append(lines, fmt.Sprintf("target_profile=%s", report.TargetProfile))
	}
	for i, pattern := range report.IncludePatterns {
		lines = append(lines, fmt.Sprintf("include_pattern_%d=%s", i+1, pattern))
	}
	for i, pattern := range report.ExcludePatterns {
		lines = append(lines, fmt.Sprintf("exclude_pattern_%d=%s", i+1, pattern))
	}
	for i, path := range report.StateFiles.MatchedPaths {
		lines = append(lines, fmt.Sprintf("state_file_matched_%d=%s", i+1, path))
	}
	for i, path := range report.StateFiles.RemovedPaths {
		lines = append(lines, fmt.Sprintf("state_file_removed_%d=%s", i+1, path))
	}
	for i, path := range report.StateFiles.MissingPaths {
		lines = append(lines, fmt.Sprintf("state_file_missing_%d=%s", i+1, path))
	}
	for i, path := range report.StateFiles.SkippedByPatternPaths {
		lines = append(lines, fmt.Sprintf("state_file_skipped_by_pattern_%d=%s", i+1, path))
	}
	for i, path := range report.StateFiles.SkippedByAgePaths {
		lines = append(lines, fmt.Sprintf("state_file_skipped_by_age_%d=%s", i+1, path))
	}
	for i, path := range report.FailedOutputs.BasePaths {
		lines = append(lines, fmt.Sprintf("failed_output_base_%d=%s", i+1, path))
	}
	for i, path := range report.FailedOutputs.MatchedPaths {
		lines = append(lines, fmt.Sprintf("failed_output_matched_%d=%s", i+1, path))
	}
	for i, path := range report.FailedOutputs.RemovedPaths {
		lines = append(lines, fmt.Sprintf("failed_output_removed_%d=%s", i+1, path))
	}
	for i, path := range report.FailedOutputs.SkippedByPatternPaths {
		lines = append(lines, fmt.Sprintf("failed_output_skipped_by_pattern_%d=%s", i+1, path))
	}
	for i, path := range report.FailedOutputs.SkippedByAgePaths {
		lines = append(lines, fmt.Sprintf("failed_output_skipped_by_age_%d=%s", i+1, path))
	}
	for i, path := range report.FailedManifests.BasePaths {
		lines = append(lines, fmt.Sprintf("failed_manifest_base_%d=%s", i+1, path))
	}
	for i, path := range report.FailedManifests.MatchedPaths {
		lines = append(lines, fmt.Sprintf("failed_manifest_matched_%d=%s", i+1, path))
	}
	for i, path := range report.FailedManifests.RemovedPaths {
		lines = append(lines, fmt.Sprintf("failed_manifest_removed_%d=%s", i+1, path))
	}
	for i, path := range report.FailedManifests.SkippedByPatternPaths {
		lines = append(lines, fmt.Sprintf("failed_manifest_skipped_by_pattern_%d=%s", i+1, path))
	}
	for i, path := range report.FailedManifests.SkippedByAgePaths {
		lines = append(lines, fmt.Sprintf("failed_manifest_skipped_by_age_%d=%s", i+1, path))
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatAuditFollowImportsReport(report auditFollowImportsReport) string {
	lines := []string{
		"audit follow-imports ok",
		fmt.Sprintf("status=%s", report.Status),
		fmt.Sprintf("fail_if_matched=%t", report.FailIfMatched),
		fmt.Sprintf("match_found=%t", report.MatchFound),
		fmt.Sprintf("retention_profile=%s", fallbackString(report.RetentionProfile)),
		fmt.Sprintf("older_than_seconds=%d", report.OlderThanSeconds),
		fmt.Sprintf("include_patterns=%d", len(report.IncludePatterns)),
		fmt.Sprintf("exclude_patterns=%d", len(report.ExcludePatterns)),
		fmt.Sprintf("state_files_requested=%d", report.StateFiles.Requested),
		fmt.Sprintf("state_files_matched=%d", report.StateFiles.Matched),
		fmt.Sprintf("state_files_missing=%d", report.StateFiles.Missing),
		fmt.Sprintf("state_files_skipped_by_pattern=%d", report.StateFiles.SkippedByPattern),
		fmt.Sprintf("state_files_skipped_by_age=%d", report.StateFiles.SkippedByAge),
		fmt.Sprintf("failed_output_bases=%d", report.FailedOutputs.Bases),
		fmt.Sprintf("failed_output_matched=%d", report.FailedOutputs.Matched),
		fmt.Sprintf("failed_output_skipped_by_pattern=%d", report.FailedOutputs.SkippedByPattern),
		fmt.Sprintf("failed_output_skipped_by_age=%d", report.FailedOutputs.SkippedByAge),
		fmt.Sprintf("failed_manifest_bases=%d", report.FailedManifests.Bases),
		fmt.Sprintf("failed_manifest_matched=%d", report.FailedManifests.Matched),
		fmt.Sprintf("failed_manifest_skipped_by_pattern=%d", report.FailedManifests.SkippedByPattern),
		fmt.Sprintf("failed_manifest_skipped_by_age=%d", report.FailedManifests.SkippedByAge),
		fmt.Sprintf("follow_health_file=%s", report.FollowHealth.File),
		fmt.Sprintf("follow_health_present=%t", report.FollowHealth.Present),
		fmt.Sprintf("follow_health_last_updated_at=%s", pointerTimeOrNone(report.FollowHealth.LastUpdatedAt)),
		fmt.Sprintf("follow_health_status=%s", fallbackString(report.FollowHealth.Status)),
		fmt.Sprintf("follow_health_source=%s", fallbackString(report.FollowHealth.Source)),
		fmt.Sprintf("follow_health_input_count=%d", report.FollowHealth.InputCount),
		fmt.Sprintf("follow_health_continuous=%t", report.FollowHealth.Continuous),
		fmt.Sprintf("follow_health_poll_interval_seconds=%d", report.FollowHealth.PollIntervalSeconds),
		fmt.Sprintf("follow_health_snapshot_age_seconds=%d", report.FollowHealth.SnapshotAgeSeconds),
		fmt.Sprintf("follow_health_stale=%t", report.FollowHealth.Stale),
		fmt.Sprintf("follow_health_requested_watch_mode=%s", fallbackString(report.FollowHealth.RequestedWatchMode)),
		fmt.Sprintf("follow_health_active_watch_mode=%s", fallbackString(report.FollowHealth.ActiveWatchMode)),
		fmt.Sprintf("follow_health_watch_fallbacks=%d", report.FollowHealth.WatchFallbacks),
		fmt.Sprintf("follow_health_watch_transitions=%d", report.FollowHealth.WatchTransitions),
		fmt.Sprintf("follow_health_last_fallback_reason=%s", fallbackString(report.FollowHealth.LastFallbackReason)),
		fmt.Sprintf("follow_health_watch_poll_catchups=%d", report.FollowHealth.WatchPollCatchups),
		fmt.Sprintf("follow_health_watch_poll_catchup_bytes=%d", report.FollowHealth.WatchPollCatchupBytes),
		fmt.Sprintf("follow_health_warnings=%d", len(report.FollowHealth.Warnings)),
	}
	if report.TargetProfile != "" {
		lines = append(lines, fmt.Sprintf("target_profile=%s", report.TargetProfile))
	}
	for i, pattern := range report.IncludePatterns {
		lines = append(lines, fmt.Sprintf("include_pattern_%d=%s", i+1, pattern))
	}
	for i, pattern := range report.ExcludePatterns {
		lines = append(lines, fmt.Sprintf("exclude_pattern_%d=%s", i+1, pattern))
	}
	for i, path := range report.StateFiles.MatchedPaths {
		lines = append(lines, fmt.Sprintf("state_file_matched_%d=%s", i+1, path))
	}
	for i, path := range report.StateFiles.MissingPaths {
		lines = append(lines, fmt.Sprintf("state_file_missing_%d=%s", i+1, path))
	}
	for i, path := range report.StateFiles.SkippedByPatternPaths {
		lines = append(lines, fmt.Sprintf("state_file_skipped_by_pattern_%d=%s", i+1, path))
	}
	for i, path := range report.StateFiles.SkippedByAgePaths {
		lines = append(lines, fmt.Sprintf("state_file_skipped_by_age_%d=%s", i+1, path))
	}
	for i, path := range report.FailedOutputs.BasePaths {
		lines = append(lines, fmt.Sprintf("failed_output_base_%d=%s", i+1, path))
	}
	for i, path := range report.FailedOutputs.MatchedPaths {
		lines = append(lines, fmt.Sprintf("failed_output_matched_%d=%s", i+1, path))
	}
	for i, path := range report.FailedOutputs.SkippedByPatternPaths {
		lines = append(lines, fmt.Sprintf("failed_output_skipped_by_pattern_%d=%s", i+1, path))
	}
	for i, path := range report.FailedOutputs.SkippedByAgePaths {
		lines = append(lines, fmt.Sprintf("failed_output_skipped_by_age_%d=%s", i+1, path))
	}
	for i, path := range report.FailedManifests.BasePaths {
		lines = append(lines, fmt.Sprintf("failed_manifest_base_%d=%s", i+1, path))
	}
	for i, path := range report.FailedManifests.MatchedPaths {
		lines = append(lines, fmt.Sprintf("failed_manifest_matched_%d=%s", i+1, path))
	}
	for i, path := range report.FailedManifests.SkippedByPatternPaths {
		lines = append(lines, fmt.Sprintf("failed_manifest_skipped_by_pattern_%d=%s", i+1, path))
	}
	for i, path := range report.FailedManifests.SkippedByAgePaths {
		lines = append(lines, fmt.Sprintf("failed_manifest_skipped_by_age_%d=%s", i+1, path))
	}
	for i, warning := range report.FollowHealth.Warnings {
		prefix := fmt.Sprintf("follow_health_warning_%d", i+1)
		lines = append(lines,
			fmt.Sprintf("%s_code=%s", prefix, warning.Code),
			fmt.Sprintf("%s_message=%s", prefix, warning.Message),
		)
	}
	return strings.Join(lines, "\n") + "\n"
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

func pruneFollowImportsHealthSnapshot(logDir string) error {
	path := followImportsHealthPath(logDir)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("remove follow-imports health snapshot: %w", err)
	}
	return nil
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
