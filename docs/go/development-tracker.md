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

Current target: Start Phase 1 foundation work for the Go implementation.

## Phase Progress

### Phase 1: Foundation

Status: todo

Tasks:

- [ ] Initialize Go module and base repository layout
- [ ] Choose SQLite driver and migration approach
- [ ] Implement database open/init flow
- [ ] Create initial schema migrations
- [ ] Implement canonical scope types
- [ ] Implement canonical session types
- [ ] Implement scope resolution inputs and fallback order
- [ ] Implement scope consistency validation
- [ ] Implement `memory_resolve_scope`
- [ ] Implement `memory_start_session`

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

- not started yet

Immediate next tasks:

1. Initialize Go module
2. Decide SQLite driver
3. Create migration scaffolding
4. Add scope/session domain types

## Decisions Log

### 2026-03-13

- The project will follow the `codex-mem` v1 spec under `docs/spec/`.
- Dynamic continuity data should stay in durable memory and MCP responses, not in `AGENTS.md`.
- `AGENTS.md` should remain cache-friendly and stable.
- The first coding slice should focus on Phase 1 foundation work.

## Blockers

Current blockers:

- none recorded yet

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
