# Go Import Ingestion

## Purpose

This document explains how operators can use `codex-mem ingest-imports` for one-shot batches and `codex-mem follow-imports` for long-lived incremental consumption of watcher or relay JSONL feeds.

Audience:

- operators wiring watcher or relay output into `codex-mem`
- maintainers validating packaged-binary ingestion behavior

Use this when:

- you need a one-shot batch bridge into the imported-note workflow
- you need a checkpointed long-lived bridge for a growing JSONL file
- your upstream process can emit newline-delimited JSON events

Do not use this for:

- normal day-to-day Codex prompting
- direct MCP tool calls from a client

## Command Shape

Use `ingest-imports` when you already have a bounded batch to replay.
Use `follow-imports` when another process keeps appending to the same JSONL file and you want `codex-mem` to checkpoint progress between notification or polling passes.
`follow-imports` can now fan in multiple files by repeating `--input`.

Minimal stdin example:

```powershell
Get-Content .\events.jsonl | codex-mem.exe ingest-imports --source watcher_import
```

Read from a file and print JSON:

```powershell
codex-mem.exe ingest-imports --source relay_import --input .\relay-events.jsonl --json
```

Continue past bad lines and keep successful imports:

```powershell
codex-mem.exe ingest-imports --source watcher_import --input .\events.jsonl --continue-on-error --json
```

Export failed lines for retry after the batch finishes:

```powershell
codex-mem.exe ingest-imports --source watcher_import --input .\events.jsonl --continue-on-error --failed-output .\failed-events.jsonl --json
```

Export a machine-readable retry manifest alongside the raw failed lines:

```powershell
codex-mem.exe ingest-imports --source watcher_import --input .\events.jsonl --continue-on-error --failed-output .\failed-events.jsonl --failed-manifest .\failed-events.json --json
```

Follow a growing JSONL file once and checkpoint the consumed offset:

```powershell
codex-mem.exe follow-imports --source watcher_import --input .\events.jsonl --once --json
```

Run as a long-lived poller with an explicit checkpoint file:

```powershell
codex-mem.exe follow-imports --source relay_import --input .\relay-events.jsonl --state-file .\relay-events.offset.json --watch-mode poll --poll-interval 10s
```

Run in notify-first mode and let polling stay as a safety fallback:

```powershell
codex-mem.exe follow-imports --source watcher_import --input .\events.jsonl --watch-mode auto --poll-interval 5s
```

Follow two growing JSONL files in one process:

```powershell
codex-mem.exe follow-imports --source watcher_import --input .\events-a.jsonl --input .\events-b.jsonl --watch-mode auto --poll-interval 5s --json
```

Useful flags:

- `--source watcher_import|relay_import`
  Required. Declares the provenance source for every event in the input stream.
- `--input <path>`
  Optional for `ingest-imports`. Reads JSONL from a file instead of stdin.
  Required for `follow-imports`. Repeat it to follow multiple files in one process.
- `--cwd <path>`
  Optional. Resolves scope from a specific workspace root.
- `--branch-name <name>`
  Optional. Carries branch metadata into the ingestion session.
- `--repo-remote <url>`
  Optional. Strengthens scope resolution with the repository remote.
- `--task <text>`
  Optional. Overrides the default ingestion session task summary.
- `--json`
  Optional. Prints a structured report instead of line-oriented text output.
- `--continue-on-error`
  `ingest-imports` only. Keeps scanning after per-line decode or import failures and returns a partial-success report when at least one event succeeds.
- `--failed-output <path>`
  Optional. For `ingest-imports`, requires `--continue-on-error` and writes the original failed input lines to a JSONL file for manual fix-up or replay.
  For `follow-imports`, each polling batch derives a range-suffixed file from the provided base path so earlier failures are not overwritten.
- `--failed-manifest <path>`
  Optional. For `ingest-imports`, requires `--continue-on-error` and writes a JSON manifest with per-line error metadata and raw failed input.
  For `follow-imports`, each polling batch derives a range-suffixed manifest path from the provided base path.
- `--state-file <path>`
  `follow-imports` only. Optional. Stores the consumed byte offset checkpoint. Defaults to `<input>.offset.json`.
  When `follow-imports` uses multiple `--input` flags, either omit `--state-file` and let each input use its own default sidecar, or repeat `--state-file` once per `--input` in the same order.
- `--poll-interval <duration>`
  `follow-imports` only. Optional. Controls how often the input file is polled for appended complete lines and how often notify mode performs a safety poll. Defaults to `5s`.
- `--watch-mode auto|notify|poll`
  `follow-imports` only. Optional. `auto` prefers filesystem notifications and falls back to polling on watcher setup/runtime issues. `notify` requires filesystem notifications and fails if they cannot be used. `poll` disables notifications and uses polling only. Defaults to `auto`.
- `--once`
  `follow-imports` only. Optional. Runs one poll/ingest pass and exits instead of staying in the polling loop.

## Event Schema

Each non-empty line must be one JSON object.

Required fields:

- `type`
  Canonical note type: `decision`, `bugfix`, `discovery`, `constraint`, `preference`, or `todo`.
- `title`
  Short imported note title.
- `content`
  Durable imported note body.
- `importance`
  Integer importance from `1` to `5`.

At least one of:

- `external_id`
  Stable upstream artifact id used for import dedupe.
- `payload_hash`
  Stable content hash used when no external id exists.

Optional fields:

- `tags`
  String array of note tags.
- `file_paths`
  String array of touched or relevant paths.
- `related_project_ids`
  String array of related project ids for cross-project retrieval links.
- `status`
  Note lifecycle state. Defaults to `active` when omitted.
- `privacy_intent`
  When set to `private`, `do_not_store`, or `ephemeral_only`, the import is audited but note materialization is suppressed.

## Example JSONL

```jsonl
{"external_id":"watcher:1","type":"discovery","title":"Imported discovery","content":"Useful watcher discovery.","importance":4,"tags":["watcher"]}
{"external_id":"watcher:2","type":"todo","title":"Private follow-up","content":"Should stay audit-only.","importance":3,"privacy_intent":"private"}
```

Behavior to expect from this batch:

- the first event creates an imported durable note plus an import audit record
- the second event creates only a suppressed import audit record

## Output Semantics

Text mode prints a compact summary such as:

```text
ingest imports ok
status=ok
source=watcher_import
input=stdin
session_id=sess_20260316_001
resolved_by=repo_remote
continue_on_error=false
attempted=2
processed=2
failed=0
materialized=1
suppressed=1
note_deduplicated=0
import_deduplicated=0
warnings=1
```

JSON mode returns the same summary plus per-line results, including the created or reused `note_id` and `import_id`.
When a line fails in `--continue-on-error` mode, that result entry includes a structured `error` payload instead.
If `--failed-output` is set, the report also includes the resolved output path and how many failed lines were written there.
If `--failed-manifest` is set, the report also includes the manifest path and how many failures were captured there.
Single-input `follow-imports` reports the input path, checkpoint file, requested watch mode, active watch mode, fallback count, transition count, cumulative poll-catchup count and bytes, any warning summaries, any structured watch events since the previous emitted report, consumed offset, pending trailing bytes, whether the checkpoint was reset, the reset reason, truncation detection, and the nested batch report for whatever newly appended complete lines were imported during that poll.
Multi-input `follow-imports` returns one aggregate report with command-level watch state, cumulative poll-catchup counters, warning summaries, per-process watch events, total consumed and pending bytes, and one nested per-input report for each followed file.

## Operational Notes

- `ingest-imports` starts one fresh session for the whole batch after resolving scope.
- `follow-imports` starts one fresh session per consumed polling batch, not one session for the lifetime of the process.
- When `follow-imports` fans in multiple files, each input keeps its own checkpoint sidecar and each consumed input still starts its own ingestion session for that batch.
- In `auto` mode, `follow-imports` prefers filesystem notifications for lower latency and keeps the poll timer as a safety net in case a platform drops an event.
- In `auto` mode, if watcher setup fails or a running watcher later closes/errors, `follow-imports` falls back to polling and keeps retrying watcher setup on later poll intervals. When watcher setup succeeds again, the process switches back to notify mode instead of staying degraded forever.
- In `notify` mode, watcher setup or runtime failures stop the command instead of silently switching to polling.
- The follow-mode report now exposes both the requested watch mode and the currently active mode, so operators can tell when `auto` has fallen back to polling and how many fallbacks have happened in the current process.
- Follow-mode reports now also emit structured watch events when the active mode changes, a fallback occurs, auto mode successfully recovers from polling back to notify, or notify mode's safety poll is the thing that actually catches newly appended bytes. In JSON mode these appear under `watch_events`; in text mode they are flattened as `watch_event_<n>_*` lines.
- A watch-state transition or fallback now forces one emitted `follow-imports` report even when the ingestion pass itself is otherwise idle, so long-lived operators can observe notify activation and fallback transitions without waiting for the next imported batch.
- When a notify-mode safety poll catches appended bytes, `follow-imports` records a `watch_poll_catchup` event with consumed input and byte counts. Treat that as evidence that the polling safety net was materially useful, not necessarily proof that the platform dropped an fsnotify event.
- `follow-imports` also keeps cumulative `watch_poll_catchups` and `watch_poll_catchup_bytes` counters for the lifetime of the process. Once poll catchup happens at least three times in the same process, the report adds a `WARN_FOLLOW_IMPORTS_POLL_CATCHUP` warning so operators and automation can treat notify mode as degraded even if it never fully falls back.
- Each emitted `follow-imports` report also refreshes a last-known health sidecar under the normal log directory. `codex-mem doctor` reads that snapshot so operators can inspect the most recent follow-mode watch health even after the long-lived process has already exited.
- For continuous follow mode, `doctor` now marks that sidecar as stale when it has not been refreshed for roughly three poll intervals, with a minimum freshness window of 30 seconds. Stale follow health adds `WARN_FOLLOW_IMPORTS_HEALTH_STALE` so operators can distinguish a healthy last-known state from an old snapshot left behind by a stopped process.
- When multi-input follow mode shares `--failed-output` or `--failed-manifest` base paths, `codex-mem` derives per-input file names before adding the byte-range suffix so retry artifacts from different inputs do not overwrite each other.
- Each event uses the same imported-note workflow as `memory_save_imported_note`.
- Existing explicit memory wins over weaker imported duplicates in the same project.
- The default implementation is fail-fast: the first invalid line stops the batch and returns an error.
- `--continue-on-error` preserves successful lines, reports per-line failures, and still exits with an error if nothing in the batch imports successfully.
- `--failed-output` writes the original failed JSONL lines without wrapping them, so operators can edit that file and replay it through the same command later.
- `--failed-manifest` writes a structured JSON sidecar with line numbers, error codes, error messages, raw failed lines, and failed-output line numbers when available.
- `follow-imports` only consumes complete newline-terminated lines. A partially written trailing line is left in place until a later poll sees its terminating newline.
- The `follow-imports` checkpoint sidecar stores both the consumed byte offset and a hash of the last consumed boundary bytes so replacement or rotation can be detected even when the new file does not shrink first.
- If the followed input file is truncated, rotated, or replaced with different bytes before the saved offset, `follow-imports` resets its checkpoint to byte offset `0` and continues from the start of the new file contents.
