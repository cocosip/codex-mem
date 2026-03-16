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
Use `follow-imports` when another process keeps appending to the same JSONL file and you want `codex-mem` to checkpoint progress between polling passes.

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
codex-mem.exe follow-imports --source relay_import --input .\relay-events.jsonl --state-file .\relay-events.offset.json --poll-interval 10s
```

Useful flags:

- `--source watcher_import|relay_import`
  Required. Declares the provenance source for every event in the input stream.
- `--input <path>`
  Optional for `ingest-imports`. Reads JSONL from a file instead of stdin.
  Required for `follow-imports`.
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
- `--poll-interval <duration>`
  `follow-imports` only. Optional. Controls how often the input file is polled for appended complete lines. Defaults to `5s`.
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
`follow-imports` reports the input path, checkpoint file, consumed offset, pending trailing bytes, truncation detection, and the nested batch report for whatever newly appended complete lines were imported during that poll.

## Operational Notes

- `ingest-imports` starts one fresh session for the whole batch after resolving scope.
- `follow-imports` starts one fresh session per consumed polling batch, not one session for the lifetime of the process.
- Each event uses the same imported-note workflow as `memory_save_imported_note`.
- Existing explicit memory wins over weaker imported duplicates in the same project.
- The default implementation is fail-fast: the first invalid line stops the batch and returns an error.
- `--continue-on-error` preserves successful lines, reports per-line failures, and still exits with an error if nothing in the batch imports successfully.
- `--failed-output` writes the original failed JSONL lines without wrapping them, so operators can edit that file and replay it through the same command later.
- `--failed-manifest` writes a structured JSON sidecar with line numbers, error codes, error messages, raw failed lines, and failed-output line numbers when available.
- `follow-imports` only consumes complete newline-terminated lines. A partially written trailing line is left in place until a later poll sees its terminating newline.
- If the followed input file is truncated or rotated to a smaller size, `follow-imports` resets its checkpoint to byte offset `0` and continues from the start of the new file contents.
