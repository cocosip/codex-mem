# Go Import Ingestion

## Purpose

This document explains how operators can use `codex-mem ingest-imports` to turn watcher or relay batches into durable imported notes plus import audit records.

Audience:

- operators wiring watcher or relay output into `codex-mem`
- maintainers validating packaged-binary ingestion behavior

Use this when:

- you need a one-shot batch bridge into the imported-note workflow
- your upstream process can emit newline-delimited JSON events

Do not use this for:

- normal day-to-day Codex prompting
- direct MCP tool calls from a client

## Command Shape

Minimal stdin example:

```powershell
Get-Content .\events.jsonl | codex-mem.exe ingest-imports --source watcher_import
```

Read from a file and print JSON:

```powershell
codex-mem.exe ingest-imports --source relay_import --input .\relay-events.jsonl --json
```

Useful flags:

- `--source watcher_import|relay_import`
  Required. Declares the provenance source for every event in the batch.
- `--input <path>`
  Optional. Reads JSONL from a file instead of stdin.
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
source=watcher_import
input=stdin
session_id=sess_20260316_001
resolved_by=repo_remote
processed=2
materialized=1
suppressed=1
note_deduplicated=0
import_deduplicated=0
warnings=1
```

JSON mode returns the same summary plus per-line results, including the created or reused `note_id` and `import_id`.

## Operational Notes

- `ingest-imports` starts one fresh session for the whole batch after resolving scope.
- Each event uses the same imported-note workflow as `memory_save_imported_note`.
- Existing explicit memory wins over weaker imported duplicates in the same project.
- The current implementation is fail-fast: the first invalid line stops the batch and returns an error.
