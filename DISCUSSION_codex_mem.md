# codex-mem Design Log

Last updated: 2026-03-13

## Purpose

This file is the design log and decision index for `codex-mem`.

It is no longer the primary place for the formal v1 specification.
The normative spec now lives under [docs/spec/](D:/Code/go/codex-mem/docs/spec/README.md).

## Background

- Codex is being used through a relay/proxy endpoint.
- The relay does not support the compaction interface.
- When context fills up, sessions must be restarted.
- Restarting a session causes continuity loss unless prior work is recovered externally.

## Product Direction

`codex-mem` is intended to be an external memory and handoff system for Codex relay environments.

It is designed to:

- preserve continuity across restarted sessions
- store structured high-value memory
- recover recent context through bootstrap
- isolate memory by scope
- allow controlled related-project retrieval

It is not intended to:

- replace compaction inside a single live session
- behave as a global undifferentiated memory pool
- depend on full passive transcript capture for v1

## Key Design Decisions

### 1. v1 is specification-first

We chose to define a language-neutral v1 standard before binding to any implementation language or code structure.

### 2. Memory is scoped

The core scope hierarchy is:

1. `system`
2. `project`
3. `workspace`
4. `session`

Default retrieval is project-scoped.
Cross-project retrieval is controlled and never the default.

### 3. Structured memory is primary

The primary durable memory forms are:

- `MemoryNote`
- `Handoff`

These are preferred over raw transcript storage.

### 4. Handoff drives continuity

Session recovery is centered on:

- explicit handoff capture before pause/end
- explicit high-value note capture during work
- session bootstrap at the start of a new session

### 5. AGENTS.md is workflow guidance, not dynamic memory

`AGENTS.md` is used for stable rules such as:

- when to bootstrap
- when to save notes
- when to write handoff

It is not used as the primary store for dynamic session state.

### 6. Privacy and scope safety outrank convenience

The design assumes:

- local-first durable memory
- private/do-not-store content must remain excluded
- unrelated project memory must not appear in default retrieval

### 7. Observability matters

The system should explain:

- what was stored
- why it was retrieved
- why something was excluded
- where memory came from

without leaking sensitive content.

## Formal Spec Location

The formal `codex-mem v1` specification is split into these files:

- [Spec Index](D:/Code/go/codex-mem/docs/spec/README.md)
- [Glossary](D:/Code/go/codex-mem/docs/spec/glossary.md)
- [Domain Model](D:/Code/go/codex-mem/docs/spec/domain-model.md)
- [State Model](D:/Code/go/codex-mem/docs/spec/state-model.md)
- [Tool Contracts](D:/Code/go/codex-mem/docs/spec/tool-contracts.md)
- [Retrieval Policy](D:/Code/go/codex-mem/docs/spec/retrieval-policy.md)
- [Privacy and Retention](D:/Code/go/codex-mem/docs/spec/privacy-retention.md)
- [AGENTS Policy](D:/Code/go/codex-mem/docs/spec/agents-policy.md)
- [Configuration and Precedence](D:/Code/go/codex-mem/docs/spec/configuration-precedence.md)
- [Identity and Consistency](D:/Code/go/codex-mem/docs/spec/identity-consistency.md)
- [Observability and Provenance](D:/Code/go/codex-mem/docs/spec/observability-provenance.md)
- [V1 Baseline](D:/Code/go/codex-mem/docs/spec/v1-baseline.md)

## Supporting Assets

AGENTS templates are stored here:

- [Global AGENTS Template](D:/Code/go/codex-mem/templates/AGENTS.global.template.md)
- [Project AGENTS Template](D:/Code/go/codex-mem/templates/AGENTS.project.template.md)

Implementation planning reference:

- [Implementation Backlog](D:/Code/go/codex-mem/docs/implementation-backlog.md)
- [Go Implementation Plan](D:/Code/go/codex-mem/docs/go-implementation-plan.md)
- [Go Development Tracker](D:/Code/go/codex-mem/docs/go-development-tracker.md)

## Remaining Follow-Up Items

These items were identified as useful additions, but they do not block the v1 spec:

- canonical warning/error code taxonomy
- example request/response payloads
- onboarding flow examples
- migration examples for rename/move/split/merge
- conformance scenario matrix

## Recommended Next Step

Use `docs/spec/` as the normative reference and treat this file as the high-level design log.
