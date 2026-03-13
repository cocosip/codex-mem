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

Current target: Begin Phase 4 AGENTS integration on top of the completed retrieval and safety slice.

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

Status: todo

Tasks:

- [ ] Package AGENTS templates for runtime use
- [ ] Implement `memory_install_agents`
- [ ] Support safe create-if-missing behavior
- [ ] Support append mode
- [ ] Support explicit overwrite mode
- [ ] Implement project/system placeholder filling

### Phase 5: Conformance and Hardening

Status: todo

Note: This phase is the remaining validation layer for already-implemented Phase 1-4 features and should not be confused with missing feature implementation work.

Tasks:

- [ ] Add conformance tests for empty-store bootstrap
- [ ] Add conformance tests for same-project recovery
- [ ] Add conformance tests for related-project retrieval
- [ ] Add conformance tests for privacy exclusion
- [ ] Add conformance tests for AGENTS safe install
- [ ] Add identity conflict tests
- [ ] Add migration edge-case tests
- [ ] Verify provenance and warning visibility

## Current Session Plan

Current session focus:

- Close out Phase 3 warning/error taxonomy work and prepare the handoff into Phase 4

Immediate next tasks:

1. Package AGENTS templates for runtime use
2. Implement `memory_install_agents` with safe default write behavior
3. Add conformance-oriented tests for warning visibility and AGENTS safety
4. Decide whether handoffs need additional lifecycle/archive controls beyond `searchable`

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
