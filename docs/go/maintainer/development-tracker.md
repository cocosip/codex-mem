# codex-mem Go Development Tracker

Last updated: 2026-03-16
Status: active

## Purpose

This file is the working execution tracker for the Go implementation of `codex-mem`.

Use it to:

- track current implementation phase
- record progress across sessions
- keep next steps explicit
- reduce restart cost when a new Codex session begins

Normative references:

- [Spec Index](../../spec/README.md)
- [Implementation Backlog](./implementation-backlog.md)
- [Go Implementation Plan](./implementation-plan.md)
- [Go Development Kickoff](./dev-kickoff.md)

## Working Rules

- Update this file after each meaningful coding session.
- Mark task status explicitly as `todo`, `doing`, `done`, or `blocked`.
- Record blockers and decisions briefly.
- Keep it concise and execution-focused.

## Current Target

Current target: Maintain the completed v1 implementation, keep `modelcontextprotocol/go-sdk` as the only MCP runtime, and evolve import auditing into a durable imported-note workflow with explicit-memory precedence.

## Phase Progress

### Phase 1: Foundation

Status: done

Tasks:

- [x] Initialize Go module and base repository layout
- [x] Choose SQLite driver and migration approach
- [x] Implement database open/init flow
- [x] Create initial schema migrations
- [x] Implement canonical scope types
- [x] Implement canonical session types
- [x] Implement scope resolution inputs and fallback order
- [x] Implement scope consistency validation
- [x] Implement `memory_resolve_scope`
- [x] Implement `memory_start_session`

### Phase 2: Core Continuity Loop

Status: done

Tasks:

- [x] Implement canonical memory note types
- [x] Implement canonical handoff types
- [x] Implement note storage
- [x] Implement handoff storage
- [x] Implement `memory_save_note`
- [x] Implement `memory_save_handoff`
- [x] Implement latest open handoff lookup
- [x] Implement recent high-value note lookup
- [x] Implement startup brief synthesis
- [x] Implement `memory_bootstrap_session`

### Phase 3: Retrieval and Safety

Status: done

Note: Phase 3 feature implementation is complete, but broader conformance and hardening verification still remains under Phase 5.

Tasks:

- [x] Add FTS5 search support
- [x] Implement `memory_search`
- [x] Implement `memory_get_recent`
- [x] Implement `memory_get_note`
- [x] Implement related-project retrieval gating
- [x] Implement result ranking policy
- [x] Implement dedupe behavior for search results
- [x] Implement privacy exclusion checks before indexing
- [x] Implement warning/error taxonomy mapping

### Phase 4: AGENTS Integration

Status: done

Tasks:

- [x] Package AGENTS templates for runtime use
- [x] Implement `memory_install_agents`
- [x] Support safe create-if-missing behavior
- [x] Support append mode
- [x] Support explicit overwrite mode
- [x] Implement project/system placeholder filling

### Phase 5: Conformance and Hardening

Status: done

Note: This phase is the remaining validation layer for already-implemented Phase 1-4 features and should not be confused with missing feature implementation work.

Tasks:

- [x] Add conformance tests for empty-store bootstrap
- [x] Add conformance tests for same-project recovery
- [x] Add conformance tests for related-project retrieval
- [x] Add conformance tests for privacy exclusion
- [x] Add conformance tests for AGENTS safe install
- [x] Add identity conflict tests
- [x] Add migration edge-case tests
- [x] Verify provenance and warning visibility

### Phase 6: MCP SDK Migration

Status: done

Note: This phase replaces the hand-rolled MCP transport/runtime code with `modelcontextprotocol/go-sdk` without changing the domain service contracts or the exposed v1 tool names.

Tasks:

- [x] Add and pin the `modelcontextprotocol/go-sdk` dependency
- [x] Keep `internal/mcp/handlers.go` as the transport-agnostic boundary for domain calls
- [x] Introduce SDK-backed tool registration that preserves the current tool names, descriptions, and JSON schemas
- [x] Replace the custom stdio server loop with the SDK stdio transport
- [x] Replace the custom HTTP handler with the SDK streamable HTTP handler while preserving JSON-response compatibility and enabling session-aware SSE support
- [x] Preserve the existing `/mcp` endpoint and configurable origin allowlist behavior
- [x] Update doctor/smoke-test coverage to validate the SDK-backed stdio and HTTP paths
- [x] Update maintainer/operator docs to describe the new transport behavior and any stream/SSE capability changes
- [x] Remove obsolete custom transport code after parity tests pass

## Current Session Plan

Current session focus:

- Keep the imports workflow aligned across audit-only and imported-note materialization paths, including explicit-memory precedence.

Immediate next tasks:

1. Keep `go test ./...` plus the readiness/smoke checks in the normal regression path for code-bearing changes.
2. Keep the import MCP workflows aligned with the underlying import audit schema and project-scoped dedupe/precedence rules.
3. Preserve retrieval preference for explicit memory when imported artifacts overlap with the same project-level durable note.
4. Do not reintroduce a parallel in-tree MCP runtime, and only revisit transport internals if a real client compatibility issue appears.

## Decisions Log

### 2026-03-13

- The project will follow the `codex-mem` v1 spec under `docs/spec/`.
- Dynamic continuity data should stay in durable memory and MCP responses, not in `AGENTS.md`.
- `AGENTS.md` should remain cache-friendly and stable.
- The first coding slice should focus on Phase 1 foundation work.
- Go logging should use `log/slog`.
- `slog` output should be written to a rotated log file with retention/compression support.
- Repository-local config should load from the repository `configs/` directory.
- Go config loading should use `viper`, with environment variables overriding file values.
- The SQLite driver is `modernc.org/sqlite`.
- Schema migrations are embedded SQL files under `migrations/` and are applied automatically on startup.
- Scope resolution prefers normalized repo remote identity, then Git root fallback, then local directory fallback.
- Session writes validate the stored `workspace -> project -> system` scope chain before insert.
- Bootstrap retrieval prefers the latest open workspace handoff, then project fallback, then recent notes ordered by scope and importance.
- Recent retrieval reuses the same workspace-first/project-fallback scope ordering and full-record reads return `ERR_RECORD_NOT_FOUND` on misses.
- Search is project-scoped by default, uses FTS5-backed note matching plus SQL handoff matching, and ranks results by scope, state, importance, recency, and text overlap.
- Related-project expansion is now opt-in and currently derives allowed related projects from explicit `related_project_ids` references stored on current-project notes.
- Explicit `private`, `do_not_store`, and `ephemeral_only` intents are rejected before durable note or handoff writes.
- Durable notes and handoffs now carry `searchable` controls; non-searchable records stay readable by id but are excluded from recent/bootstrap/search paths and from the note FTS index.
- Warning/error taxonomy now flows through shared coded-error helpers, retrieval warning codes, and MCP-ready response envelopes that promote warnings to the top level while preserving stable error codes.
- Phase completion in this tracker refers to implementation status; broader conformance, hardening, and verification status remains tracked separately under Phase 5.
- AGENTS templates are now embedded for runtime use, and `memory_install_agents` defaults global installs to `~/.codex/AGENTS.md` plus project installs to `<cwd>/AGENTS.md`.
- AGENTS append mode uses managed comment markers so repeated installs do not duplicate appended template blocks.
- Phase 5 now includes explicit conformance-named Go tests for bootstrap, recovery, retrieval isolation/expansion, privacy exclusion, AGENTS safe install, identity conflict handling, migration continuity, and warning/provenance visibility.
- Runtime config now loads through `viper` from `configs/codex-mem.*`, with environment variables overriding file values and a checked-in example config under `configs/codex-mem.example.json`.
- Go config is now split between user-configurable file/env settings and runtime-derived metadata so fields like `ConfigFileUsed` and `LogDir` are not treated as file-configurable inputs.
- `doctor` should support machine-readable diagnostics output via `--json`, while keeping the existing human-readable report as the default.
- The Go implementation should support both local stdio MCP and remote HTTP MCP transports over the same tool surface.

### 2026-03-14

- The project will migrate MCP transport/runtime wiring from the custom in-tree implementation to `modelcontextprotocol/go-sdk`.
- `internal/mcp/handlers.go` and the existing domain request/response envelopes should remain the adapter boundary so domain services do not depend on SDK structs.
- Migration order should be stdio parity first, then HTTP parity on the SDK streamable HTTP handler, then removal of obsolete custom transport code.
- HTTP migration should preserve the current `/mcp` endpoint contract and configurable origin checks while moving toward SDK-managed streamable HTTP behavior.
- The first migration slice should add a go-sdk-backed server builder and parity tests before changing any CLI transport entrypoint.
- `modelcontextprotocol/go-sdk` v1.4.1 documents stdio as newline-delimited JSON, and the `codex-mcp-server` implementation summary for `openai/codex` also describes stdio as line-delimited JSON messages.
- The repository's existing `Content-Length` stdio framing should now be treated as a legacy in-tree implementation detail rather than the expected Codex-compatible transport shape.
- The remaining shared MCP protocol/tool metadata should live in neutral package files, and the old `internal/mcp/server.go` hand-written runtime entrypoint should be removed rather than kept as a dormant fallback.
- `modelcontextprotocol/go-sdk` is now the only MCP runtime in the repository; future work should extend the SDK-backed path instead of reviving parallel custom transport code.

## Blockers

Current blockers:

- none currently

### 2026-03-14 Session Update

- Completed: Reviewed the current custom MCP transport wiring in `internal/mcp` and `internal/app/run.go`; drafted a phased migration plan to move to `modelcontextprotocol/go-sdk`; retargeted this tracker from release follow-up to MCP transport migration; added `modelcontextprotocol/go-sdk` v1.4.1 to the module; introduced an SDK-backed server builder that registers the existing nine tools without changing handler/domain boundaries; added in-memory parity coverage for `tools/list` and `memory_install_agents`; switched `serve-http` to an SDK-backed streamable HTTP handler in compatibility-focused JSON-response/stateless mode while preserving `/mcp` and origin validation; added SDK HTTP transport tests; passed `go run ./scripts/http-mcp-smoke-test`; and verified the full repository with `go test ./...`.
- In progress: Updating the remaining stdio path from the legacy `Content-Length` framing to the newline-delimited JSON transport shape used by `modelcontextprotocol/go-sdk` and described for `codex-mcp-server`.
- Blockers: the repository still has legacy stdio code and tests that assume `Content-Length` framing.
- Next step: Switch `serve` to SDK stdio, update stdio smoke coverage and related docs, then remove or quarantine the old framed stdio implementation.

### 2026-03-14 Session Handoff

- Known good:
  - `serve-http` is now backed by `modelcontextprotocol/go-sdk` through `internal/mcp/sdk_http.go`.
  - The SDK-backed HTTP path preserves `/mcp`, preserves the existing origin allowlist behavior, and passes `go run ./scripts/http-mcp-smoke-test`.
  - The repository passes `go test ./...` after the SDK introduction and HTTP cutover.
- Known unknown:
  - `serve` stdio is still on the legacy implementation.
  - `modelcontextprotocol/go-sdk` v1.4.1 documents stdio as newline-delimited JSON.
  - The repository's current stdio implementation uses `Content-Length` framing.
  - Source-tree docs and tests still describe the legacy framing in a few places and must be brought in line with the newline-delimited target.
- Recommended restart point:
  - Start by switching `serve` to the SDK stdio transport.
  - Update the stdio smoke test and any CLI-facing docs that still assume `Content-Length`.
  - Remove or isolate the legacy framed stdio implementation once parity checks are green.

### 2026-03-13 Session Update

- Completed: Rewrote the Chinese portion of [prompt-examples.md](../user/prompt-examples.md) to use more user-facing wording for the current repository/project context instead of leaning on raw `scope` terminology; added [how-memory-works.md](../user/how-memory-works.md) as a quick explainer for what mem does, what gets saved, when scope matters, and which commands are for normal users versus operators; linked the new explainer from the Go docs index and the root README.
- In progress: User-facing documentation clarity follow-up.
- Blockers: none new.
- Next step: Decide whether the quick explainer should be expanded into a full onboarding doc with screenshots or client-specific walkthroughs.

### 2026-03-13 Session Update

- Completed: Reorganized the `docs/go` documentation entry points by audience; updated [docs/go/README.md](../README.md) to route users, operators, and maintainers to different docs; clarified [client-examples.md](../operator/client-examples.md), [mcp-integration.md](./mcp-integration.md), [release-readiness.md](../operator/release-readiness.md), [troubleshooting.md](../operator/troubleshooting.md), [dev-kickoff.md](./dev-kickoff.md), [implementation-plan.md](./implementation-plan.md), [how-memory-works.md](../user/how-memory-works.md), and [prompt-examples.md](../user/prompt-examples.md) with explicit audience and use-case framing so source-tree maintainer docs are less likely to be mistaken for end-user setup guides.
- In progress: Go docs information-architecture polish follow-up.
- Blockers: none new.
- Next step: Decide whether the `docs/go` area should stay flat with audience-based navigation or be physically split into `user`, `operator`, and `maintainer` subdirectories.

### 2026-03-13 Session Update

- Completed: Physically split `docs/go` into [user](../user/README.md), [operator](../operator/README.md), and [maintainer](./README.md) directories; moved the existing docs into those directories; added per-audience index files; and updated cross-links plus root README references so the new layout is navigable without relying on one flat directory.
- In progress: Post-split information architecture polish.
- Blockers: none new.
- Next step: Decide whether the user and operator areas need more task-oriented onboarding guides now that the audience boundaries are enforced by path.

### 2026-03-14 Session Update

- Completed: Switched serve to the go-sdk stdio transport using newline-delimited JSON over sdkmcp.IOTransport; rewrote the stdio smoke test and unit coverage to exchange one JSON-RPC message per line instead of Content-Length frames; refreshed maintainer/operator docs to describe the SDK-backed stdio behavior.
- In progress: Removing or isolating the remaining obsolete custom MCP request-routing/runtime helpers that are no longer on the CLI path.
- Blockers: none beyond deciding how far to push the legacy cleanup in this slice.
- Next step: Extract shared tool/schema registration out of internal/mcp/server.go, then delete or quarantine the remaining unused custom transport code.
### 2026-03-14 Session Update

- Completed: Extracted the shared MCP tool catalog away from the legacy runtime wiring so doctor and the SDK-backed server share one definition source; removed the obsolete custom HTTP handler/tests; reduced internal/mcp/http.go to shared HTTP/origin helpers plus server startup; and verified the repository with go test ./..., go run ./scripts/mcp-smoke-test, and go run ./scripts/http-mcp-smoke-test.
- In progress: none.
- Blockers: none.
- Next step: move on from transport migration work unless a regression is found.
### 2026-03-14 Session Update

- Completed: Split the remaining MCP protocol constants/types and tool catalog out of `internal/mcp/server.go` into neutral shared files; deleted the last `server.go` / `server_test.go` naming leftovers from the old hand-written runtime path; kept the SDK-backed stdio and HTTP paths green with `go test ./internal/mcp ./internal/app`.
- In progress: none.
- Blockers: none.
- Next step: pick the next feature or operator-facing enhancement without reopening MCP runtime replacement work.
### 2026-03-14 Session Update

- Completed: Switched the SDK-backed HTTP transport from stateless JSON-only compatibility mode to session-aware streamable HTTP with standalone SSE support on `GET /mcp`; kept JSON POST compatibility for ordinary tool calls; rewrote both source-tree smoke tests to use the go-sdk client transports instead of hand-written JSON-RPC message structs; and re-verified with `go run ./scripts/mcp-smoke-test`, `go run ./scripts/http-mcp-smoke-test`, `go run ./scripts/readiness-check`, and `go test ./...`.
- In progress: none.
- Blockers: none.
- Next step: move back to product-facing work unless a concrete HTTP client compatibility issue appears.
### 2026-03-14 Session Update

- Completed: Fixed the remaining `golangci-lint` dead-code issue after the MCP cleanup; added `serve-http --session-timeout <duration>` so operators can automatically expire idle HTTP MCP sessions; wired that timeout through the SDK streamable HTTP handler; updated maintainer/operator docs for HTTP session headers, SSE behavior, and idle expiry; and kept `golangci-lint`, `go test ./...`, `go run ./scripts/mcp-smoke-test`, `go run ./scripts/http-mcp-smoke-test`, and `go run ./scripts/readiness-check` green.
- In progress: none.
- Blockers: none.
- Next step: choose a product-facing feature slice outside MCP transport and operator hardening.
### 2026-03-16 Session Update

- Completed: Refreshed root `AGENTS.md` to point at the moved `docs/go/maintainer/` planning documents; updated this tracker so the current target/session plan reflects the completed SDK-backed v1 baseline instead of an already-finished transport migration; and synced [release-readiness.md](../operator/release-readiness.md) with the current HTTP session, SSE, and session-timeout behavior.
- In progress: none.
- Blockers: none.
- Next step: choose a small product-facing or operator-facing follow-up slice and keep the regression harness in place for any code-bearing change.
### 2026-03-16 Session Update

- Completed: Added imports MVP plumbing with embedded `005_import_records.sql`, a new `internal/domain/imports` service, SQLite-backed import audit persistence in `internal/db/import_repository.go`, and `doctor`/runtime diagnostics coverage for import audit counts and readiness.
- In progress: none.
- Blockers: none.
- Next step: choose the first caller for the import plumbing, such as an MCP-exposed import flow, a CLI command, or a watcher-side ingestion path that can create durable memory from imported artifacts.
### 2026-03-16 Session Update

- Completed: Exposed the import audit flow as a new MCP tool `memory_save_import` instead of a CLI-specific workflow. The tool uses the existing imports service/repository, carries project-scoped dedupe and privacy suppression forward into the MCP surface, and increases the advertised tool count to ten; smoke/readiness/app tests and docs were updated accordingly.
- In progress: none.
- Blockers: none.
- Next step: decide whether imported artifacts should remain audit-only or whether a follow-up tool/workflow should materialize them into durable notes or handoffs.
### 2026-03-16 Session Update

- Completed: Added `memory_save_imported_note` as an eleven-tool MCP workflow that atomically materializes imported artifacts into durable notes plus import audit. The imports service now suppresses imported-note materialization when a stronger project-level explicit note already exists, links import audit to reused imported notes, and keeps retrieval ranking biased toward explicit notes over imported artifacts. Repository, domain, MCP, doctor-count, and smoke-test expectations were updated for the expanded tool surface.
- In progress: none.
- Blockers: none.
- Next step: decide whether imported handoff materialization is worth supporting later, or whether imported artifacts should stay note-only beyond audit records.
### 2026-03-16 Session Update

- Completed: Made `memory_save_imported_note` truly transactional instead of only sequential. The DB layer now supports tx-bound memory/import repositories plus an imported-note transaction runner, and regression coverage forces an import-audit failure after note creation to verify the note is rolled back instead of being left orphaned.
- In progress: none.
- Blockers: none.
- Next step: decide whether to keep polishing the import workflow through spec-facing docs and watcher/relay integration, or move on to a different product-facing slice.
### 2026-03-16 Session Update

- Completed: Added a real CLI-side caller for the imported-note workflow with `codex-mem ingest-imports`. The command resolves scope, starts one ingestion session, reads newline-delimited JSON import events from stdin or `--input`, and routes each event through `memory_save_imported_note` semantics so watcher/relay batches can create imported notes plus audit records without going through MCP. Added app-level coverage for text/JSON summaries and persisted note/import counts.
- In progress: none.
- Blockers: none.
- Next step: decide whether to keep this CLI batch path as the main watcher/relay bridge for now, or add a more direct integration path on top of the same imported-note service.
### 2026-03-16 Session Update

- Completed: Updated the normative spec to include `memory_save_import` and `memory_save_imported_note` tool contracts plus example request/response payloads, and added an operator-facing [import-ingestion.md](../operator/import-ingestion.md) guide that documents the `ingest-imports` JSONL schema, example invocations, output shape, and current fail-fast batch semantics.
- In progress: none.
- Blockers: none.
- Next step: decide whether `ingest-imports` should remain the main watcher/relay bridge for now or whether a more direct long-lived integration path is worth adding later.
### 2026-03-16 Session Update

- Completed: Added `ingest-imports --continue-on-error` so watcher/relay batches can keep importing valid lines while collecting per-line decode/write failures in the text/JSON report. Default behavior remains fail-fast for compatibility, but partial-success mode now reports `status`, attempted/failed counts, and structured line errors; app coverage verifies partial success and the all-failed path.
- In progress: none.
- Blockers: none.
- Next step: decide whether partial-success mode is enough for watcher/relay operators for now, or whether they also need richer retry/export behavior for failed lines.
### 2026-03-16 Session Update

- Completed: Added `ingest-imports --failed-output <path>` for `--continue-on-error` batches so failed raw JSONL lines can be exported unchanged for later replay. The CLI report now includes the resolved failed-output path plus written count, and app coverage verifies both partial-success export and all-failed export behavior.
- In progress: none.
- Blockers: none.
- Next step: decide whether failed-line export is enough for operators or whether the next slice should add a richer retry manifest with error metadata alongside the raw replay file.
### 2026-03-16 Session Update

- Completed: Added `ingest-imports --failed-manifest <path>` so `--continue-on-error` batches can emit a JSON retry manifest with line numbers, error payloads, raw failed lines, and failed-output line numbers. The main report now surfaces the manifest path/count, and app coverage verifies manifest validation plus partial/all-failed export behavior.
- In progress: none.
- Blockers: none.
- Next step: decide whether the operator path is now sufficient, or whether the next import slice should focus on a more direct watcher/relay integration instead of more CLI/reporting polish.
### 2026-03-16 Session Update

- Completed: Extracted the import batch workflow into a reusable app-level entrypoint `(*App).IngestImports(...)` so future in-process watcher/relay integrations can reuse the same scope resolution, session creation, imported-note materialization, and failure-export behavior without shelling out to the CLI. The CLI command now delegates to that method, and app coverage verifies the embedded path directly.
- In progress: none.
- Blockers: none.
- Next step: decide whether to build an actual in-tree watcher/relay adapter on top of `App.IngestImports`, or stop here and treat the reusable app method as sufficient integration scaffolding for now.
### 2026-03-16 Session Update

- Completed: Added `follow-imports` as a checkpointed long-lived adapter on top of `App.IngestImports(...)`. The new command polls a JSONL file for newly appended complete lines, persists byte-offset state in a sidecar checkpoint file, resets cleanly on truncation, and derives per-batch failed-output / failed-manifest paths so operator retry artifacts are not overwritten.
- In progress: none.
- Blockers: none.
- Next step: decide whether polling-based follow mode is sufficient for watcher/relay integration for now, or whether a later slice should add native filesystem notifications, rotation metadata, or multi-input fan-in.
### 2026-03-16 Session Update

- Completed: Strengthened `follow-imports` checkpoint recovery so it no longer relies only on `size < offset` truncation detection. The sidecar now stores a hash of the last consumed boundary bytes plus file metadata, and follow mode resets to offset `0` when the current file no longer matches that checkpoint, including same-size replacement cases. App coverage now verifies checkpoint hashes, normal restart recovery, truncation resets, and replacement without shrink.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current boundary-hash approach is enough for production watcher/relay rotation patterns, or whether a later slice should add stronger file identity metadata or native filesystem notifications.
### 2026-03-16 Session Update

- Completed: Added filesystem notification support to `follow-imports` with a new `--watch-mode auto|notify|poll` operator switch. Auto mode now prefers `fsnotify` on the input directory and still keeps the polling timer as a safety net, notify mode fails hard on watcher setup/runtime errors, and poll mode preserves the previous timer-only behavior. App coverage now includes watch-mode parsing and event filtering for input-file notifications.
- In progress: none.
- Blockers: none.
- Next step: decide whether `follow-imports` needs richer observability for watcher fallbacks and dropped-event recovery, or whether the current auto/notify/poll split is enough for operators.
### 2026-03-16 Session Update

- Completed: Added follow-mode observability for watcher state. `follow-imports` reports now include the requested watch mode, the currently active mode, fallback count, and last fallback reason, so operators can tell when auto mode has degraded to polling. Text output and JSON output both expose those fields, and app coverage verifies runtime-state injection plus report formatting.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next long-lived import slice should expose these watch-state transitions as structured metrics/events, or move on to multi-input fan-in support.
### 2026-03-16 Session Update

- Completed: Extended `follow-imports` watch observability from state snapshots to structured events. Reports now carry cumulative watch transition counts plus per-report `watch_events` entries for notify activation and polling fallbacks, and long-lived follow mode emits an otherwise-idle report when one of those watch events occurs so operators do not have to wait for the next imported batch to see a mode change.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next import follow-up should focus on multi-input fan-in support or on stronger dropped-event / watcher-recovery behavior on top of the new watch-event surface.
### 2026-03-16 Session Update

- Completed: Added multi-input fan-in support to `follow-imports`. Operators can now repeat `--input` in one process, each input keeps its own checkpoint sidecar, notify mode watches the union of parent directories, and multi-input runs emit an aggregate report with command-level watch state plus one nested per-input report. Shared failed-output and failed-manifest base paths now derive input-specific filenames before the byte-range suffix so retry artifacts from different inputs do not collide.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next `follow-imports` slice should strengthen dropped-event / watcher-recovery behavior now that multi-input fan-in and watch-event observability are both in place.
### 2026-03-16 Session Update

- Completed: Strengthened `follow-imports` auto-mode recovery so watcher failures no longer leave the process permanently degraded to polling. Auto mode now retries watcher setup on later poll intervals, emits a structured `watch_recovery` event when notify mode is restored, and switches back to filesystem notifications once the watcher path becomes available again. App coverage now verifies the recovery event helpers plus polling-loop watcher recovery.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next `follow-imports` slice should focus on explicitly detecting poll-caught dropped events during notify mode, or whether the current fallback-plus-recovery behavior is sufficient for operators.
### 2026-03-16 Session Update

- Completed: Added explicit notify safety-poll catchup observability to `follow-imports`. The runtime loop now distinguishes notify-event-triggered runs from poll-tick runs, and when notify mode remains active but a poll tick consumes appended bytes, the report emits a structured `watch_poll_catchup` event with consumed input and byte counts. This makes it visible when the polling safety net materially contributed to ingestion even though the watcher never fully fell back.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next import slice should escalate `watch_poll_catchup` repeated occurrences into stronger operator warnings/metrics, or whether the current event stream is enough.
### 2026-03-16 Session Update

- Completed: Escalated repeated notify safety-poll catchup into summary-level warnings. `follow-imports` now keeps cumulative `watch_poll_catchups` and `watch_poll_catchup_bytes` counters in runtime state, surfaces them on both single-input and aggregate reports, and emits `WARN_FOLLOW_IMPORTS_POLL_CATCHUP` once the same process has needed poll catchup at least three times. App coverage now verifies the counters, threshold warning, and text output fields.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next import slice should export these watch-health counters through `doctor` or another machine-readable diagnostics surface, or whether keeping them scoped to runtime follow reports is enough.
### 2026-03-16 Session Update

- Completed: Exposed last-known `follow-imports` watch health through `doctor`. Each emitted follow report now refreshes a `follow-imports.health.json` snapshot in the configured log directory, and `doctor` now reports whether that snapshot exists plus its last-known watch mode, fallback counts, poll-catchup counters, and follow-level warnings. App coverage now verifies both the empty-doctor case and the populated follow-health case.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next import slice should age or prune stale follow-health snapshots, or whether simple last-known state is sufficient for operators.
### 2026-03-16 Session Update

- Completed: Added stale-snapshot detection to the `doctor` follow-health view. Follow-health snapshots now persist whether they came from continuous mode and which poll interval they used, and `doctor` marks a snapshot stale when a continuous follow process has not refreshed it for roughly three poll intervals with a 30-second minimum freshness window. Stale snapshots now add `WARN_FOLLOW_IMPORTS_HEALTH_STALE`, and app coverage verifies both fresh and stale follow-health reporting.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next import slice should prune stale health files automatically, or whether operators should keep last-known stale state available until the next follow run overwrites it.
### 2026-03-16 Session Update

- Completed: Surfaced `doctor` follow-health into the broader `scripts/readiness-check` machine-readable summary without creating a second runtime source of truth. The readiness helper now echoes flat `doctor_follow_imports_*` lines for last-known follow status, staleness, watch mode, fallback/catchup counters, and warning codes straight from `doctor --json`; script-level tests cover both populated and missing follow-health cases; and maintainer/operator docs now call out that these lines are informational by default rather than a hard readiness gate.
- In progress: none.
- Blockers: none.
- Next step: decide whether a later slice should add an explicit JSON summary mode for `scripts/readiness-check`, or whether the flat key/value output is enough for current automation consumers.
### 2026-03-16 Session Update

- Completed: Added an explicit `--json` mode to `scripts/readiness-check`. The helper still defaults to flat key/value text, but `--json` now emits one structured readiness summary that embeds the parsed `doctor --json` payload plus compact stdio/HTTP smoke-test results; script-level tests now cover flag parsing and JSON output; and maintainer/operator docs now describe when to use the text versus JSON forms.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next automation-facing slice should let `readiness-check` downgrade or annotate individual smoke-test phases instead of only succeeding when all phases pass.
### 2026-03-16 Session Update

- Completed: Added explicit per-phase results to `scripts/readiness-check` for `doctor`, stdio smoke, and HTTP smoke. The helper now collects phase status/summary metadata before exiting, so failed runs still emit a usable text or JSON summary naming the phase that stopped progress; success output now includes stable `phase_*` lines and JSON `phases` entries; and the maintainer/operator docs now describe the new phase annotations alongside the existing follow-health summary fields.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next automation-facing slice should add an option to continue running later readiness phases after an earlier failure, or whether fail-fast execution plus explicit phase reporting is enough.
### 2026-03-16 Session Update

- Completed: Added `--keep-going` to `scripts/readiness-check` so operators can choose between the default fail-fast execution and a mode that still attempts later phases after an earlier failure. The readiness helper now carries `keep_going` through both text and JSON output, tests cover continued execution after both doctor and stdio-phase failures, and maintainer/operator docs now describe when to use the new mode.
- In progress: none.
- Blockers: none.
- Next step: decide whether a later readiness slice should also expose per-phase durations or timestamps for automation timelines, or whether the current phase status/summary surface is sufficient.
### 2026-03-16 Session Update

- Completed: Added per-phase timing metadata to `scripts/readiness-check`. Each phase now records started/completed timestamps plus elapsed milliseconds, the text output now emits stable `phase_*_started_at`, `phase_*_completed_at`, and `phase_*_duration_ms` lines, JSON output carries the same fields in `phases`, and tests now cover both successful timing output and failed/not-run phase cases.
- In progress: none.
- Blockers: none.
- Next step: decide whether a later readiness slice should also record an overall command duration and wall-clock window, or whether phase-level timing is enough for current automation consumers.
### 2026-03-16 Session Update

- Completed: Added overall readiness-run timing metadata on top of the existing phase timing. `scripts/readiness-check` now emits top-level `started_at`, `completed_at`, and `duration_ms` fields in both text and JSON output, tests verify successful and failing runs include the overall timing surface, and the docs now describe that the helper exposes both whole-run and per-phase timing for automation.
- In progress: none.
- Blockers: none.
- Next step: decide whether a later readiness slice should start classifying slow overall runs or phases into warnings, or keep timing purely observational for now.
### 2026-03-16 Session Update

- Completed: Added optional slow-run warning thresholds to `scripts/readiness-check`. `--slow-run-ms` and `--slow-phase-ms` now annotate both text and JSON output with informational readiness warnings plus per-phase warning codes when elapsed time exceeds configured thresholds, while preserving the existing exit-status semantics so slow checks do not fail readiness on their own.
- In progress: none.
- Blockers: none.
- Next step: decide whether downstream automation wants fixed local threshold presets or a richer policy layer that can optionally fail on specific warning codes.
### 2026-03-16 Session Update

- Completed: Added warning-policy escalation to `scripts/readiness-check`. Operators can now pass `--fail-on-warning-code` one or more times, including comma-separated codes, to make specific emitted warning codes fail readiness while leaving the default behavior unchanged. The helper now reports configured fail-on codes, the combined warning-code set it observed, the subset that matched policy, and whether policy caused the final failure in both text and JSON output; tests cover flag parsing, summary serialization, and a clean phase pass that still exits non-zero because a configured doctor follow-health warning code matched.
- In progress: none.
- Blockers: none.
- Next step: decide whether maintainers want a small set of documented threshold/policy presets for CI and release workflows, or whether explicit flags remain the right level of control.

## Recommended Next Step

Recommended next implementation slice:

1. Return to feature or product follow-up work outside the completed MCP transport migration.
2. Keep the stdio and HTTP smoke tests in the normal regression path.
3. Only revisit transport internals if a real client compatibility issue appears.
4. Prefer small, documented follow-ups instead of re-introducing parallel transport implementations.

Why this is the best next step now:

- the repository already has both stdio and HTTP MCP entrypoints on the SDK-backed path, so transport replacement is no longer the highest-value area
- the hand-written MCP runtime has been removed, which reduces maintenance cost and ambiguity about which path is authoritative
- the existing handler/domain boundary remains stable, so new work can focus on user-visible behavior instead of transport churn
- the readiness check plus stdio/HTTP smoke tests already provide a practical regression harness for future follow-up changes

## Session Handoff Template

When ending a coding session, append a short update in this format:

```md
### YYYY-MM-DD Session Update

- Completed:
- In progress:
- Blockers:
- Next step:
```

## Suggested Prompt For A New Session

```text
Read docs/go/maintainer/development-tracker.md, docs/go/maintainer/dev-kickoff.md, and docs/go/maintainer/implementation-plan.md, then continue the current Go implementation from the listed next tasks and update the tracker as you make progress.
```



