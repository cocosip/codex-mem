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

Current target: Begin Phase 2 note and handoff persistence on top of the Phase 1 foundation.

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

Status: todo

Tasks:

- [ ] Implement canonical memory note types
- [ ] Implement canonical handoff types
- [ ] Implement note storage
- [ ] Implement handoff storage
- [ ] Implement `memory_save_note`
- [ ] Implement `memory_save_handoff`
- [ ] Implement latest open handoff lookup
- [ ] Implement recent high-value note lookup
- [ ] Implement startup brief synthesis
- [ ] Implement `memory_bootstrap_session`

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

- Phase 1 foundation skeleton, storage bootstrap, and scope/session flow

Immediate next tasks:

1. Add canonical note and handoff types
2. Create note and handoff tables plus repositories
3. Implement `memory_save_note`
4. Implement `memory_save_handoff`

## Decisions Log

### 2026-03-13

- The project will follow the `codex-mem` v1 spec under `docs/spec/`.
- Dynamic continuity data should stay in durable memory and MCP responses, not in `AGENTS.md`.
- `AGENTS.md` should remain cache-friendly and stable.
- The first coding slice should focus on Phase 1 foundation work.
- Go logging should use `log/slog`.
- If a config file is introduced, it should live under the repository `configs/` directory.
- The SQLite driver is `modernc.org/sqlite`.
- Schema migrations are embedded SQL files under `migrations/` and are applied automatically on startup.
- Scope resolution prefers normalized repo remote identity, then Git root fallback, then local directory fallback.
- Session writes validate the stored `workspace -> project -> system` scope chain before insert.

## Blockers

Current blockers:

- none recorded yet

### 2026-03-13 Session Update

- Completed: Initialized the Go module and repository skeleton; added `slog`-based startup logging; implemented SQLite open/init with embedded migrations; implemented scope/session domain types and services; added Git-based identity discovery; implemented scope/session SQLite repositories; added scope-chain validation for session writes; added foundation tests; verified with `go test ./...`.
- In progress: Preparing Phase 2 note and handoff persistence.
- Blockers: none.
- Next step: Add note/handoff schema, domain types, repositories, and the first save-tool handlers.

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
