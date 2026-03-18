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
Add `--audit-only` to either import command when you want import-audit provenance without creating or reusing durable imported notes.
Use `audit-follow-imports` when you want a read-only hygiene report for pending checkpoint, retry-artifact, or stale-health cleanup work before deciding whether to run deletion.

Minimal stdin example:

```powershell
Get-Content .\events.jsonl | codex-mem.exe ingest-imports --source watcher_import
```

Read from a file and print JSON:

```powershell
codex-mem.exe ingest-imports --source relay_import --input .\relay-events.jsonl --json
```

Store only import-audit records for the batch while preserving the same privacy and explicit-memory precedence checks:

```powershell
codex-mem.exe ingest-imports --source watcher_import --input .\events.jsonl --audit-only --json
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

Follow the same stream in audit-only mode when another system should inspect imported provenance before materializing notes:

```powershell
codex-mem.exe follow-imports --source watcher_import --input .\events.jsonl --once --audit-only --json
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

Explicitly clean follow-imports checkpoint sidecars, derived retry artifacts, and stale health:

```powershell
codex-mem.exe cleanup-follow-imports --input .\events-a.jsonl --input .\events-b.jsonl --prune-state --failed-output .\failed-events.jsonl --prune-failed-output --failed-manifest .\failed-events.json --prune-failed-manifest --prune-stale-follow-health
```

Preview what cleanup would remove before deleting anything:

```powershell
codex-mem.exe cleanup-follow-imports --input .\events.jsonl --prune-state --failed-output .\failed-events.jsonl --prune-failed-output --failed-manifest .\failed-events.json --prune-failed-manifest --prune-stale-follow-health --older-than 1h --dry-run
```

Restrict cleanup to one input family while excluding another:

```powershell
codex-mem.exe cleanup-follow-imports --input .\events-a.jsonl --input .\events-b.jsonl --prune-state --failed-output .\failed-events.jsonl --prune-failed-output --failed-manifest .\failed-events.json --prune-failed-manifest --include "*events-a*" --exclude "*.43-84.*"
```

Use a named retention preset instead of spelling out an age threshold:

```powershell
codex-mem.exe cleanup-follow-imports --failed-output .\failed-events.jsonl --prune-failed-output --retention-profile daily --dry-run
```

Use a named target preset when the command should cover a common hygiene slice:

```powershell
codex-mem.exe cleanup-follow-imports --target-profile all --input .\events.jsonl --failed-output .\failed-events.jsonl --failed-manifest .\failed-events.json --dry-run
```

Audit whether cleanup work is pending and fail the command if anything in the selected target set matches:

```powershell
codex-mem.exe cleanup-follow-imports --input .\events.jsonl --prune-state --dry-run --fail-if-matched
```

Run the same hygiene audit as a dedicated read-only report command:

```powershell
codex-mem.exe audit-follow-imports --input .\events.jsonl --check-state --failed-output .\failed-events.jsonl --check-failed-output --failed-manifest .\failed-events.json --check-failed-manifest --check-follow-health --retention-profile daily --fail-if-matched
```

Focus that audit on retry exports only:

```powershell
codex-mem.exe audit-follow-imports --target-profile retry --failed-output .\failed-events.jsonl --failed-manifest .\failed-events.json --retention-profile daily --fail-if-matched
```

Keep scheduled hygiene logs compact while preserving the matched counts:

```powershell
codex-mem.exe audit-follow-imports --target-profile artifacts --input .\events.jsonl --failed-output .\failed-events.jsonl --failed-manifest .\failed-events.json --summary-only --json
```

Cover checkpoint plus retry artifacts together while leaving follow-health untouched:

```powershell
codex-mem.exe cleanup-follow-imports --target-profile artifacts --input .\events.jsonl --failed-output .\failed-events.jsonl --failed-manifest .\failed-events.json --dry-run
```

Useful flags:

- `--source watcher_import|relay_import`
  Required. Declares the provenance source for every event in the input stream.
- `--input <path>`
  Optional for `ingest-imports`. Reads JSONL from a file instead of stdin.
  Required for `follow-imports`. Repeat it to follow multiple files in one process.
  For `cleanup-follow-imports` and `audit-follow-imports`, pair it with checkpoint or retry-artifact target flags when you want the command to derive the same per-input sidecar and retry base paths that `follow-imports` would use.
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
- `--audit-only`
  `ingest-imports` and `follow-imports` only. Optional. Evaluates each event against the same imported-note dedupe, privacy, and explicit-memory precedence rules, but writes only the import-audit record instead of materializing or reusing a durable note. The event schema stays the same so the audit-only path can answer whether the artifact would have been suppressed or linked to an existing note.
- `--continue-on-error`
  `ingest-imports` only. Keeps scanning after per-line decode or import failures and returns a partial-success report when at least one event succeeds.
- `--failed-output <path>`
  Optional. For `ingest-imports`, requires `--continue-on-error` and writes the original failed input lines to a JSONL file for manual fix-up or replay.
  For `follow-imports`, each polling batch derives a range-suffixed file from the provided base path so earlier failures are not overwritten.
  For `cleanup-follow-imports` and `audit-follow-imports`, pass the same base path so the command can target the derived range-suffixed retry exports without touching the unsuffixed base file.
- `--failed-manifest <path>`
  Optional. For `ingest-imports`, requires `--continue-on-error` and writes a JSON manifest with per-line error metadata and raw failed input.
  For `follow-imports`, each polling batch derives a range-suffixed manifest path from the provided base path.
  For `cleanup-follow-imports` and `audit-follow-imports`, pass the same base path so the command can target the derived range-suffixed retry manifests without touching the unsuffixed base file.
- `--state-file <path>`
  `follow-imports` only. Optional. Stores the consumed byte offset checkpoint. Defaults to `<input>.offset.json`.
  When `follow-imports` uses multiple `--input` flags, either omit `--state-file` and let each input use its own default sidecar, or repeat `--state-file` once per `--input` in the same order.
  For `cleanup-follow-imports` and `audit-follow-imports`, pair explicit `--state-file` values with `--prune-state` or `--check-state` when you want to target checkpoint sidecars directly instead of deriving them from `--input`.
- `--poll-interval <duration>`
  `follow-imports` only. Optional. Controls how often the input file is polled for appended complete lines and how often notify mode performs a safety poll. Defaults to `5s`.
- `--watch-mode auto|notify|poll`
  `follow-imports` only. Optional. `auto` prefers filesystem notifications and falls back to polling on watcher setup/runtime issues. `notify` requires filesystem notifications and fails if they cannot be used. `poll` disables notifications and uses polling only. Defaults to `auto`.
- `--once`
  `follow-imports` only. Optional. Runs one poll/ingest pass and exits instead of staying in the polling loop.
- `--prune-state`
  `cleanup-follow-imports` only. Removes follow-imports checkpoint sidecars. Pair it with `--input` to remove the default `<input>.offset.json` files, or pair it with one or more explicit `--state-file` paths.
- `--prune-failed-output`
  `cleanup-follow-imports` only. Removes only the range-suffixed JSONL retry exports derived from the provided `--failed-output` base path. The unsuffixed base file itself is left alone.
- `--prune-failed-manifest`
  `cleanup-follow-imports` only. Removes only the range-suffixed JSON retry manifests derived from the provided `--failed-manifest` base path. The unsuffixed base file itself is left alone.
- `--prune-stale-follow-health`
  `cleanup-follow-imports` only. Reuses the same stale-health rule as `doctor --prune-stale-follow-health` and removes the `follow-imports.health.json` sidecar only when it is currently stale.
- `--check-state`
  `audit-follow-imports` only. Audits follow-imports checkpoint sidecars without deleting them. Pair it with `--input` to inspect the default `<input>.offset.json` files, or pair it with explicit `--state-file` paths.
- `--check-failed-output`
  `audit-follow-imports` only. Audits only the range-suffixed JSONL retry exports derived from the provided `--failed-output` base path and leaves the base file untouched.
- `--check-failed-manifest`
  `audit-follow-imports` only. Audits only the range-suffixed JSON retry manifests derived from the provided `--failed-manifest` base path and leaves the base file untouched.
- `--check-follow-health`
  `audit-follow-imports` only. Reports whether the last-known `follow-imports.health.json` snapshot is present, stale, and carrying any warnings, without deleting it.
- `--older-than <duration>`
  `cleanup-follow-imports` and `audit-follow-imports` only. Optional. Limits checkpoint-sidecar and retry-artifact matching to files at least this old based on filesystem modification time. Use values such as `30m`, `1h`, or `24h`.
- `--dry-run`
  `cleanup-follow-imports` only. Optional. Computes the same cleanup candidates and reports what would be removed, but leaves every file in place.
- `--fail-if-matched`
  `cleanup-follow-imports` and `audit-follow-imports` only. Optional. Returns a non-zero exit after writing the report when the selected target set matched at least one checkpoint sidecar, retry artifact, or stale follow-health snapshot. This is especially useful for CI or scheduled hygiene audits.
- `--summary-only`
  `cleanup-follow-imports` and `audit-follow-imports` only. Optional. Keeps the same aggregate counts, target metadata, and follow-health status, but omits enumerated checkpoint and retry-artifact path lists from the text or JSON report. This is useful for scheduled automation where path-by-path detail would create noisy logs.
- `--include <glob>`
  `cleanup-follow-imports` and `audit-follow-imports` only. Optional. Repeats or accepts comma-separated glob patterns that act as a whitelist for checkpoint and retry-artifact candidate paths. Patterns are matched against both the absolute path and the basename.
- `--exclude <glob>`
  `cleanup-follow-imports` and `audit-follow-imports` only. Optional. Repeats or accepts comma-separated glob patterns that remove checkpoint and retry-artifact candidates from the matched set after includes are considered. Excludes win over includes.
- `--target-profile all|artifacts|state|retry|health`
  `cleanup-follow-imports` and `audit-follow-imports` only. Optional. Expands to a documented default target set instead of making you spell out the target flags every time. `all` enables every target class, `artifacts` enables checkpoint plus retry-artifact targets while leaving follow-health alone, `state` enables only checkpoint sidecars, `retry` enables only failed-output plus failed-manifest artifacts, and `health` enables only follow-health inspection/pruning. Explicit `--prune-*` or `--check-*` flags still add extra targets on top.
- `--retention-profile stale|daily|reset`
  `cleanup-follow-imports` and `audit-follow-imports` only. Optional. Expands to a documented default age threshold instead of making you spell out `--older-than` every time. `stale` means `1h`, `daily` means `24h`, and `reset` means `0s`. An explicit `--older-than` still overrides the profile.

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

When `--audit-only` is set for `ingest-imports` or `follow-imports`:

- the same event schema is still required because the command evaluates imported-note precedence rather than dropping to raw `memory_save_import`
- new non-suppressed artifacts create only import-audit records and leave `materialized=0`
- imported duplicates can still link the created audit record to an existing imported note
- stronger explicit memory still suppresses the import audit with `suppression_reason=explicit_memory_exists`

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
When `--audit-only` is active, the report also includes `audit_only=true`, `materialized` stays `0`, `would_materialize` counts unsuppressed artifacts that would have created a new imported note, and `linked_existing_note` counts unsuppressed artifacts that only linked an already-imported durable note. Whenever any suppression happens, the JSON report also includes a `suppression_reasons` object keyed by normalized reason (for example `privacy_intent`, `explicit_memory_exists`, or the fallback `import_policy` bucket), and text mode flattens those same counts as `suppression_reason_<reason>=<count>`. Each per-line result can still surface `suppression_reason`, and `note_id` stays omitted for newly audited artifacts that would have taken the new-note path.
When a line fails in `--continue-on-error` mode, that result entry includes a structured `error` payload instead.
If `--failed-output` is set, the report also includes the resolved output path and how many failed lines were written there.
If `--failed-manifest` is set, the report also includes the manifest path and how many failures were captured there.
Single-input `follow-imports` reports the input path, checkpoint file, requested watch mode, active watch mode, fallback count, transition count, cumulative poll-catchup count and bytes, any warning summaries, any structured watch events since the previous emitted report, consumed offset, pending trailing bytes, whether the checkpoint was reset, the reset reason, truncation detection, and the nested batch report for whatever newly appended complete lines were imported during that poll.
Multi-input `follow-imports` returns one aggregate report with command-level watch state, cumulative poll-catchup counters, warning summaries, per-process watch events, total consumed and pending bytes, and one nested per-input report for each followed file.
`cleanup-follow-imports` reports whether the run was a dry-run, whether `--fail-if-matched` was active, whether the selected target set matched anything, whether `--summary-only` was active, the named retention profile when one is active, the configured age gate in seconds, any include/exclude patterns in effect, how many checkpoint sidecars and derived retry artifacts matched cleanup versus were actually removed, which files were skipped because they were filtered out by pattern or were too new, which explicit state files were already missing, and whether it pruned or would prune the stale follow-health sidecar.
`audit-follow-imports` reports the same target-selection metadata and matched-versus-skipped counts as a read-only hygiene pass, plus whether `--summary-only` was active, whether the follow-health snapshot is present, when it was last updated, whether it is stale, and any warning summaries carried by that snapshot.
When `--summary-only` is set, the aggregate counts stay the same but the detailed checkpoint and retry-artifact path lists are omitted from both text and JSON output.

Checked-in sample outputs for import and follow workflows live under [../../../internal/app/testdata](../../../internal/app/testdata/):

The full checked-in catalog is indexed in [command-example-manifest.txt](../../../internal/app/testdata/command-example-manifest.txt), which records each example's command, fixture name, output format, and relative path.

Packaged binaries can print that same embedded catalog on demand:

```powershell
codex-mem.exe list-command-examples
```

If automation needs a stable machine-readable catalog, use:

```powershell
codex-mem.exe list-command-examples --json
```

If a deliberate output change makes those fixtures drift, refresh the ingest fixtures from the repository root through the test-only maintainer helper:

```powershell
$env:CODEX_MEM_REFRESH_INGEST_EXAMPLES = "all"
go test ./internal/app -run TestRefreshIngestImportsExampleFixtures
Remove-Item Env:CODEX_MEM_REFRESH_INGEST_EXAMPLES
```

Refresh the cleanup fixtures the same way:

```powershell
$env:CODEX_MEM_REFRESH_FOLLOW_IMPORT_EXAMPLES = "all"
go test ./internal/app -run TestRefreshFollowImportsExampleFixtures
Remove-Item Env:CODEX_MEM_REFRESH_FOLLOW_IMPORT_EXAMPLES

$env:CODEX_MEM_REFRESH_CLEANUP_EXAMPLES = "all"
go test ./internal/app -run TestRefreshCleanupFollowImportsExampleFixtures
Remove-Item Env:CODEX_MEM_REFRESH_CLEANUP_EXAMPLES
```

Refresh the audit fixtures the same way:

```powershell
$env:CODEX_MEM_REFRESH_AUDIT_EXAMPLES = "all"
go test ./internal/app -run TestRefreshAuditFollowImportsExampleFixtures
Remove-Item Env:CODEX_MEM_REFRESH_AUDIT_EXAMPLES
```

Refresh the checked-in catalog after adding, removing, or renaming any example fixture:

```powershell
$env:CODEX_MEM_REFRESH_EXAMPLE_MANIFEST = "1"
go test ./internal/app -run TestRefreshCommandExampleManifest
Remove-Item Env:CODEX_MEM_REFRESH_EXAMPLE_MANIFEST
```

If you only need one fixture while iterating on a specific report shape, pass a comma-separated fixture-name subset instead of `all`:

```powershell
$env:CODEX_MEM_REFRESH_INGEST_EXAMPLES = "audit-only-linked-json"
go test ./internal/app -run TestRefreshIngestImportsExampleFixtures
Remove-Item Env:CODEX_MEM_REFRESH_INGEST_EXAMPLES

$env:CODEX_MEM_REFRESH_FOLLOW_IMPORT_EXAMPLES = "audit-only-single-text"
go test ./internal/app -run TestRefreshFollowImportsExampleFixtures
Remove-Item Env:CODEX_MEM_REFRESH_FOLLOW_IMPORT_EXAMPLES

$env:CODEX_MEM_REFRESH_CLEANUP_EXAMPLES = "filtered-cleanup-json"
go test ./internal/app -run TestRefreshCleanupFollowImportsExampleFixtures
Remove-Item Env:CODEX_MEM_REFRESH_CLEANUP_EXAMPLES

$env:CODEX_MEM_REFRESH_AUDIT_EXAMPLES = "filtered-audit-json"
go test ./internal/app -run TestRefreshAuditFollowImportsExampleFixtures
Remove-Item Env:CODEX_MEM_REFRESH_AUDIT_EXAMPLES
```

The fixture names live in [follow_import_example_fixtures_test.go](../../../internal/app/follow_import_example_fixtures_test.go), and the ordinary read-only guard remains:

```powershell
go test ./internal/app -run "Test(Audit|Cleanup)FollowImportsExampleOutputsStayInSync"
```

## Operational Notes

- `ingest-imports` starts one fresh session for the whole batch after resolving scope.
- `follow-imports` starts one fresh session per consumed polling batch, not one session for the lifetime of the process.
- `--audit-only` keeps the same session, checkpoint, retry-export, and follow-health behavior as the materializing path so operators can switch between audit-only and imported-note materialization without learning a second ingestion flow.
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
- If you want to clear only stale follow-health sidecars without touching fresh ones, run `codex-mem.exe doctor --prune-stale-follow-health`. The doctor report then tells you whether it actually removed a stale snapshot via `follow_imports_health_pruned` and `follow_imports_health_prune_reason`.
- If you want one explicit operator cleanup pass for follow-imports state, use `codex-mem.exe cleanup-follow-imports`. It removes only the artifacts you target with prune flags; it does not infer or delete anything unless you ask for that specific cleanup class.
- If you want the same target selection and age/pattern filtering logic without any deletion, use `codex-mem.exe audit-follow-imports`. It is the cleaner fit for scheduled reports, pre-cleanup review, and automation that should fail on pending hygiene work before anything is removed.
- Add `--dry-run` first when you are not fully sure about the target set. The report shows the same matched file list and stale-health outcome it would use for a real cleanup pass, but without deleting anything.
- Add `--fail-if-matched` when the command should act as a hygiene gate instead of only as an informational report. On `audit-follow-imports` the command stays read-only; on `cleanup-follow-imports --dry-run` it behaves the same way while still showing what a future deletion pass would remove.
- Add `--summary-only` when scheduled automation or CI should preserve the counts and status metadata but avoid printing long per-path lists into logs or inbox output.
- Add `--older-than <duration>` when you want cleanup or audit to ignore very recent checkpoint or retry files. This age gate applies to checkpoint sidecars plus range-suffixed retry artifacts, not to the stale-health decision, which still uses the existing follow-health staleness policy.
- Add `--include` when the cleanup or audit target should stay inside one file family, input label, or path prefix. This is especially useful for multi-input runs where you only want to inspect or clean artifacts related to one input at a time.
- Add `--exclude` for the final guardrail when a broad include or input set still catches more than you want. Exclude patterns override includes, so they are the safer place to carve out known paths or suffixes.
- Use `--target-profile all` when you want the old "turn on every target flag" behavior without spelling out each flag, `--target-profile artifacts` when the command should cover checkpoint and retry files but intentionally leave follow-health snapshots alone, `--target-profile retry` when scheduled audits should look only at range-suffixed retry exports, `--target-profile state` when checkpoint hygiene is the only concern, and `--target-profile health` when you only need the last-known follow-health snapshot.
- Use `--retention-profile stale` when you want a quick ad-hoc cleanup or audit pass that ignores artifacts newer than one hour, `--retention-profile daily` for a broader once-per-day sweep, and `--retention-profile reset` when you want the selected target set matched immediately. If one preset is close but not quite right, keep the profile for readability and add an explicit `--older-than` override.
- `cleanup-follow-imports --prune-state` derives the same default checkpoint path that `follow-imports` would use for each `--input`, so you can clean old sidecars without repeating every `.offset.json` name manually.
- `cleanup-follow-imports --prune-failed-output` and `--prune-failed-manifest` remove only batch-scoped retry artifacts whose names end in the byte-range suffix that `follow-imports` generates (for example `failed.events-a.0-42.jsonl`). The base path you pass stays untouched, which avoids deleting a placeholder or unrelated manually curated file with the unsuffixed name.
- When multi-input follow mode shares failed-output or failed-manifest bases, pass the same `--input` set to `cleanup-follow-imports` so it derives the same per-input filenames before scanning for range-suffixed artifacts.
- When multi-input follow mode shares `--failed-output` or `--failed-manifest` base paths, `codex-mem` derives per-input file names before adding the byte-range suffix so retry artifacts from different inputs do not overwrite each other.
- Each event uses the same imported-note workflow as `memory_save_imported_note`.
- `--audit-only` intentionally still uses that imported-note workflow instead of the lower-level `memory_save_import` contract, because operators usually want privacy suppression, explicit-memory precedence, and imported-note dedupe to stay aligned between audit-only and materializing runs.
- Existing explicit memory wins over weaker imported duplicates in the same project.
- The default implementation is fail-fast: the first invalid line stops the batch and returns an error.
- `--continue-on-error` preserves successful lines, reports per-line failures, and still exits with an error if nothing in the batch imports successfully.
- `--failed-output` writes the original failed JSONL lines without wrapping them, so operators can edit that file and replay it through the same command later.
- `--failed-manifest` writes a structured JSON sidecar with line numbers, error codes, error messages, raw failed lines, and failed-output line numbers when available.
- `follow-imports` only consumes complete newline-terminated lines. A partially written trailing line is left in place until a later poll sees its terminating newline.
- The `follow-imports` checkpoint sidecar stores both the consumed byte offset and a hash of the last consumed boundary bytes so replacement or rotation can be detected even when the new file does not shrink first.
- If the followed input file is truncated, rotated, or replaced with different bytes before the saved offset, `follow-imports` resets its checkpoint to byte offset `0` and continues from the start of the new file contents.
