# codex-mem Go Development Tracker

Last updated: 2026-03-13
Status: active

## Purpose

This file is the working execution tracker for the Go implementation of `codex-mem`.

Use it to:

- track current implementation phase
- record progress across sessions
- keep next steps explicit
- reduce restart cost when a new Codex session begins

Normative references:

- [Spec Index](../spec/README.md)
- [Implementation Backlog](../implementation-backlog.md)
- [Go Implementation Plan](./implementation-plan.md)
- [Go Development Kickoff](./dev-kickoff.md)

## Working Rules

- Update this file after each meaningful coding session.
- Mark task status explicitly as `todo`, `doing`, `done`, or `blocked`.
- Record blockers and decisions briefly.
- Keep it concise and execution-focused.

## Current Target

Current target: Maintain v1 conformance coverage and follow-up implementation polish.

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

## Current Session Plan

Current session focus:

- Remote MCP transport, client-integration examples, and packaging follow-up

Immediate next tasks:

1. Decide whether the readiness check should be wrapped by a CI workflow or release script
2. Decide whether the readiness check should become the default release gate
3. See whether any client integration work still exposes MCP compatibility polish gaps
4. Revisit richer retrieval or audit traces only if troubleshooting data shows a real need

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

## Blockers

Current blockers:

- none recorded yet

### 2026-03-13 Session Update

- Completed: Initialized the Go module and repository skeleton; added `slog`-based startup logging; implemented SQLite open/init with embedded migrations; implemented scope/session domain types and services; added Git-based identity discovery; implemented scope/session SQLite repositories; added scope-chain validation for session writes; added foundation tests; verified with `go test ./...`.
- In progress: Preparing Phase 2 note and handoff persistence.
- Blockers: none.
- Next step: Add note/handoff schema, domain types, repositories, and the first save-tool handlers.

### 2026-03-13 Session Update

- Completed: Added `memory` and `handoff` domain packages with canonical types and validation; added SQLite migration `002_memory_and_handoffs.sql`; implemented note and handoff repositories with scope/session consistency checks; wired `memory_save_note` and `memory_save_handoff` through the app and MCP handlers; added service and repository tests; verified with `go test ./...`.
- In progress: Phase 2 continuity bootstrap work.
- Blockers: An untracked backup file `internal/db/handoff_repository.go.1923079326463469687` is currently locked by the local environment and could not be removed even though it does not affect builds.
- Next step: Implement latest handoff lookup, recent note retrieval, startup brief synthesis, and `memory_bootstrap_session`.

### 2026-03-13 Session Update

- Completed: Added retrieval bootstrap orchestration with fresh-session startup, latest open handoff lookup, recent high-value note lookup, and startup brief synthesis; wired `memory_bootstrap_session` through the app and MCP handlers; added retrieval service tests for workspace preference, project fallback, and empty-history warnings; verified with `go test ./...`.
- In progress: Phase 3 retrieval and safety planning.
- Blockers: An untracked backup file `internal/db/handoff_repository.go.1923079326463469687` is currently locked by the local environment and could not be removed even though it does not affect builds.
- Next step: Implement `memory_get_recent`, `memory_get_note`, and the first safe search/FTS slice.

### 2026-03-13 Session Update

- Completed: Added recent-activity and record-by-id retrieval support; extended note and handoff repositories with recent/project-fallback and `GetByID` queries; wired `memory_get_recent` and `memory_get_note` through the retrieval service and MCP handlers; added retrieval and repository tests for ordering, include flags, and not-found behavior; verified with `go test ./...`.
- In progress: Phase 3 search and indexing foundation.
- Blockers: An untracked backup file `internal/db/handoff_repository.go.1923079326463469687` is currently locked by the local environment and could not be removed even though it does not affect builds.
- Next step: Add FTS5 support and implement the first `memory_search` slice.

### 2026-03-13 Session Update

- Completed: Added SQLite FTS5 support for note indexing via migration `003_memory_search_fts.sql`; implemented project-scoped `memory_search` with FTS-backed note lookup, handoff text matching, ranking, and basic dedupe; wired search through the retrieval service and MCP handlers; added repository and retrieval tests covering FTS hits, zero-result success, project isolation, and search ordering; verified with `go test ./...`.
- In progress: Phase 3 safety and policy tightening.
- Blockers: An untracked backup file `internal/db/handoff_repository.go.1923079326463469687` is currently locked by the local environment and could not be removed even though it does not affect builds.
- Next step: Add privacy/search exclusion checks and improve related-project retrieval gating.

### 2026-03-13 Session Update

- Completed: Implemented opt-in related-project expansion for bootstrap, recent retrieval, and search using explicit `related_project_ids` references from current-project notes; added relation labeling for cross-project notes; added explicit privacy-intent rejection for durable note and handoff writes; added repository and service tests for related-project lookup and privacy rejection; verified with `go test ./...`.
- In progress: Phase 3 privacy/search exclusion hardening.
- Blockers: An untracked backup file `internal/domain/retrieval/service.go.7617514228505897916` is currently locked by the local environment and could not be removed even though it does not affect builds.
- Next step: Add storage-level/search-level exclusion controls so private or archived records can exist without being indexed or returned.

### 2026-03-13 Session Update

- Completed: Added migration `004_searchability_controls.sql`; introduced durable `searchable` and `exclusion_reason` fields for notes and handoffs; filtered recent/bootstrap/search reads to exclude non-searchable records; rebuilt FTS synchronization so excluded notes do not enter the note index; preserved `GetByID` access for non-searchable records; added repository tests covering exclusion from recent/search plus by-id visibility; verified with `go test ./...`.
- In progress: Phase 3 warning/error and conformance tightening.
- Blockers: An untracked backup file `internal/domain/retrieval/service.go.7617514228505897916` is currently locked by the local environment and could not be removed even though it does not affect builds.
- Next step: Finish warning/error taxonomy mapping and round out Phase 3 conformance-oriented tests.

### 2026-03-13 Session Update

- Completed: Finished Phase 3 warning/error taxonomy mapping by adding shared coded-error helpers, completing stable taxonomy constants, normalizing uncoded service/retrieval failures, emitting warning codes for related-project and search-dedupe paths, and adding MCP-ready response envelopes that surface top-level warnings and stable error payloads; added retrieval and MCP tests; verified with `go test ./...`.
- In progress: Phase 4 AGENTS integration planning.
- Blockers: Untracked backup files created by the local environment are still present and may remain locked, but they do not affect builds or tests.
- Next step: Start packaging AGENTS templates for runtime installation and implement the first `memory_install_agents` slice.

### 2026-03-13 Session Update

- Completed: Recorded the Go config-loading decision in the development docs: repository-local config should load from `configs/`, and Go config loading should use `viper` with environment variables overriding file values.
- In progress: Phase 4 AGENTS integration planning.
- Blockers: none new.
- Next step: Reflect the same config decision in implementation code when the config loader is expanded beyond env/default-only behavior.

### 2026-03-13 Session Update

- Completed: Finished Phase 4 AGENTS integration by embedding global/project templates for runtime use; implementing `memory_install_agents` with `safe`, `append`, and `overwrite` modes; filling project/system/tag/repository placeholders; defaulting global AGENTS installs to `~/.codex/AGENTS.md` and project installs to `<cwd>/AGENTS.md`; wiring the service through app construction and MCP handlers; and adding domain plus MCP tests; verified with `go test ./...`.
- In progress: Phase 5 conformance and hardening planning.
- Blockers: Untracked backup files created by the local environment may still remain locked, but they do not affect builds or tests.
- Next step: Add conformance tests for AGENTS safe install, bootstrap recovery scenarios, and warning/provenance visibility.

### 2026-03-13 Session Update

- Completed: Added explicit Phase 5 conformance and hardening tests covering empty-store bootstrap, same-workspace/project recovery, related-project retrieval labeling, privacy exclusion behavior, AGENTS safe install, identity conflicts, project rename continuity, remote URL normalization continuity, and warning/provenance visibility; verified with `go test ./...`.
- In progress: Post-conformance polish and config-loader follow-up.
- Blockers: Untracked backup files created by the local environment may still remain locked, but they do not affect builds or tests.
- Next step: Implement repository-local config-file loading with `viper` so runtime behavior matches the documented config policy.

### 2026-03-13 Session Update

- Completed: Implemented repository-local config loading with `viper` in `internal/config`, preserving `defaults < config file < environment variables`; added tests for default loading, `configs/` file loading, environment overrides, and explicit config-path selection; added `configs/codex-mem.example.json`; verified with `go test ./...`.
- In progress: Post-conformance polish and runtime integration follow-up.
- Blockers: Untracked backup files created by the local environment may still remain locked, but they do not affect builds or tests.
- Next step: Review remaining CLI and diagnostics polish, especially whether `doctor` should surface effective config information.

### 2026-03-13 Session Update

- Completed: Refactored `internal/config` so `Config` now separates user-configurable file/env values from runtime-derived metadata, and updated app/logger/CLI call sites plus tests to use the split structure; verified with `go test ./...`.
- In progress: Post-conformance polish and runtime integration follow-up.
- Blockers: Untracked backup files created by the local environment may still remain locked, but they do not affect builds or tests.
- Next step: Decide whether to expose an explicit effective-config summary in `doctor` or a dedicated diagnostics command.

### 2026-03-13 Session Update

- Completed: Implemented a native MCP stdio transport in `internal/mcp` with framed JSON-RPC handling for `initialize`, `ping`, `tools/list`, and `tools/call`; registered all v1 memory tools with JSON schemas and structured tool responses; wired `serve` to run the MCP server instead of a placeholder skeleton; added transport tests and a real `go run ./cmd/codex-mem serve` initialize smoke check; verified with `go test ./...`.
- In progress: Post-conformance diagnostics and observability follow-up.
- Blockers: none new.
- Next step: Decide whether `doctor` should expose an effective-config summary and review any remaining client-facing observability gaps.

### 2026-03-13 Session Update

- Completed: Expanded `doctor` to print an effective configuration summary including config precedence, selected config file, database path, SQLite settings, and log rotation settings; added `internal/app` tests covering config-file-present and config-file-absent diagnostics; verified with `go test ./...` and a real `go run ./cmd/codex-mem doctor` smoke check.
- In progress: Residual observability review and release/readiness follow-up.
- Blockers: none new.
- Next step: Review whether `doctor` should also surface broader runtime health and provenance diagnostics beyond effective config.

### 2026-03-13 Session Update

- Completed: Expanded `doctor` again to include runtime readiness diagnostics for SQLite pragmas, required schema presence, FTS availability, applied/available migration status, and MCP tool count; added `internal/db` diagnostics support plus database and app tests; verified with `go test ./...` and a real `go run ./cmd/codex-mem doctor` smoke check.
- In progress: Residual observability review and release/readiness follow-up.
- Blockers: none new.
- Next step: Decide whether the next diagnostics slice should focus on provenance/audit visibility or on release/readiness packaging.

### 2026-03-13 Session Update

- Completed: Expanded `doctor` provenance/audit diagnostics to report durable note and handoff counts, note source-category coverage, exclusion counts, missing exclusion-reason checks, recovery/open handoff counts, and readiness booleans for provenance/exclusion audit posture; added runtime inspection queries plus focused `internal/db` and `internal/app` tests; verified with `go test ./...` and a real `go run ./cmd/codex-mem doctor` smoke check.
- In progress: Residual observability review and release/readiness follow-up.
- Blockers: none new.
- Next step: Decide whether the next polish slice should focus on release/readiness packaging or on richer retrieval/audit traces.

### 2026-03-13 Session Update

- Completed: Added release/readiness packaging docs by creating a root `README.md` with command, MCP, onboarding, and diagnostics guidance; added `docs/go/release-readiness.md` with a practical pre-release checklist and smoke-test flow; updated the Go docs index to include the new readiness document.
- In progress: Residual observability review and packaging follow-up.
- Blockers: none new.
- Next step: Decide whether the next polish slice should focus on troubleshooting guidance, machine-readable diagnostics output, or richer retrieval/audit traces.

### 2026-03-13 Session Update

- Completed: Chose machine-readable diagnostics as the next polish slice and implemented `doctor --json`; refactored doctor output through a shared report model so text and JSON stay aligned; added CLI tests for default text output, JSON output, and unknown-flag rejection; updated README and release-readiness docs to include the new automation path.
- In progress: Troubleshooting and release-readiness follow-up.
- Blockers: none new.
- Next step: Add a concise troubleshooting guide for config resolution, database path/setup failures, and MCP startup/client integration issues.

### 2026-03-13 Session Update

- Completed: Added [troubleshooting.md](./troubleshooting.md) covering config discovery and precedence, invalid config values, database path and SQLite readiness failures, logging visibility, MCP stdio framing, initialize/tool-call failures, and minimal recovery recipes; updated the Go docs index, root README, and release-readiness doc to point at the troubleshooting guide.
- In progress: Packaging and client-integration follow-up.
- Blockers: none new.
- Next step: Add a client-facing MCP integration example or a scripted smoke-test recipe that uses `serve`, `initialize`, `tools/list`, and one real tool call.

### 2026-03-13 Session Update

- Completed: Added a runnable MCP smoke-test program under `scripts/mcp-smoke-test` that launches `go run ./cmd/codex-mem serve`, performs `initialize`, `notifications/initialized`, `tools/list`, and a real `memory_install_agents` tool call, then verifies that a temporary `AGENTS.md` file was written; added [mcp-integration.md](./mcp-integration.md) with the transport summary, expected request order, and usage guidance; updated the Go docs index, root README, and release-readiness doc to point at the new integration path.
- In progress: Release-readiness and CI automation follow-up.
- Blockers: none new.
- Next step: Decide whether to wire the MCP smoke test and `doctor --json` into a scripted CI-style check, or to add external-client-specific examples next.

### 2026-03-13 Session Update

- Completed: Added a combined readiness-check runner under `scripts/readiness-check` that executes `go run ./cmd/codex-mem doctor --json`, validates the key readiness fields, then runs `go run ./scripts/mcp-smoke-test`; updated README, MCP integration guidance, and release-readiness docs to point at the single-command readiness path.
- In progress: Client-specific integration and packaging follow-up.
- Blockers: none new.
- Next step: Add one or more external-client-specific MCP setup examples, or wire the readiness check into a CI/release workflow.

### 2026-03-13 Session Update

- Completed: Added [client-examples.md](./client-examples.md) with a concrete Codex CLI local stdio setup flow based on the locally available `codex mcp add` command shape, plus a ChatGPT connector limitation note that documents the current remote-only MCP support boundary and points to the OpenAI Apps SDK docs; updated the Go docs index, MCP integration guide, root README, and release-readiness doc to point at the client examples.
- In progress: Packaging and transport-scope follow-up.
- Blockers: none new.
- Next step: Decide whether to wire `go run ./scripts/readiness-check` into CI/release automation, or to open a separate implementation slice for remote MCP transport if ChatGPT support is required.

### 2026-03-13 Session Update

- Completed: Implemented a minimal remote HTTP MCP transport in `internal/mcp` with `POST` JSON request handling, JSON-RPC method reuse over the existing handlers, notification-only `202 Accepted` behavior, explicit `GET`/`DELETE` method rejection, and optional origin allowlisting; added the `serve-http` CLI command with `--listen`, `--path`, and repeated `--allow-origin` flags; added transport and option parser tests; updated MCP integration, client examples, release-readiness, and root README docs to describe both stdio and HTTP deployment paths.
- In progress: Remote transport hardening and packaging follow-up.
- Blockers: none new.
- Next step: Add an HTTP transport smoke-test runner and decide whether CI should call it together with `go run ./scripts/readiness-check`.

### 2026-03-13 Session Update

- Completed: Added a real HTTP transport smoke test under `scripts/http-mcp-smoke-test` that launches `serve-http`, performs `initialize`, `notifications/initialized`, `tools/list`, and a real `memory_install_agents` call over `POST /mcp`, then verifies that a temporary `AGENTS.md` file was written; extended `scripts/readiness-check` so it now validates `doctor --json`, the stdio smoke test, and the HTTP smoke test in one run; updated MCP integration, release-readiness, and root README docs to describe the expanded readiness gate.
- In progress: CI/release gate follow-up.
- Blockers: none new.
- Next step: Decide whether to wire `go run ./scripts/readiness-check` directly into CI or a release script, and whether any additional HTTP transport hardening is needed before that.

## Recommended Next Step

Recommended next implementation slice:

1. Wire `go run ./scripts/readiness-check` into CI or a release checklist runner.
2. Keep both stdio and HTTP smoke tests as required gates.
3. After that, decide whether HTTP transport needs:
   auth hardening
   richer capability signaling
   stream support
4. Revisit richer retrieval or audit traces only if real troubleshooting cases require them.

Why this is the best next step now:

- it builds directly on the new dual-transport readiness coverage
- it converts the current manual validation path into an enforceable release gate
- it improves release-readiness for private deployment scenarios without changing core logic
- it keeps richer trace work demand-driven instead of speculative

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
Read docs/go/development-tracker.md, docs/go/dev-kickoff.md, and docs/go/implementation-plan.md, then continue the current Go implementation from the listed next tasks and update the tracker as you make progress.
```
