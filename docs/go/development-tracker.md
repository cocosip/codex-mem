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

Current target: Begin Phase 3 retrieval and safety work on top of the completed continuity loop.

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

Status: todo

Tasks:

- [ ] Add FTS5 search support
- [ ] Implement `memory_search`
- [ ] Implement `memory_get_recent`
- [ ] Implement `memory_get_note`
- [ ] Implement related-project retrieval gating
- [ ] Implement result ranking policy
- [ ] Implement dedupe behavior for search results
- [ ] Implement privacy exclusion checks before indexing
- [ ] Implement warning/error taxonomy mapping

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

- Phase 3 retrieval and safety work starting from the completed continuity loop

Immediate next tasks:

1. Add retrieval-facing recent activity APIs on top of the current note and handoff repositories
2. Implement `memory_get_recent`
3. Implement `memory_get_note`
4. Prepare the Phase 3 search foundation and FTS5 schema changes

## Decisions Log

### 2026-03-13

- The project will follow the `codex-mem` v1 spec under `docs/spec/`.
- Dynamic continuity data should stay in durable memory and MCP responses, not in `AGENTS.md`.
- `AGENTS.md` should remain cache-friendly and stable.
- The first coding slice should focus on Phase 1 foundation work.
- Go logging should use `log/slog`.
- `slog` output should be written to a rotated log file with retention/compression support.
- If a config file is introduced, it should live under the repository `configs/` directory.
- The SQLite driver is `modernc.org/sqlite`.
- Schema migrations are embedded SQL files under `migrations/` and are applied automatically on startup.
- Scope resolution prefers normalized repo remote identity, then Git root fallback, then local directory fallback.
- Session writes validate the stored `workspace -> project -> system` scope chain before insert.
- Bootstrap retrieval prefers the latest open workspace handoff, then project fallback, then recent notes ordered by scope and importance.

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
