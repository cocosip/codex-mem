# Go Development Kickoff

## Purpose

This file is the fastest entry point for the next session if the goal is to begin actual Go implementation work for `codex-mem`.

Primary references:

- [Spec Index](../spec/README.md)
- [Implementation Backlog](../implementation-backlog.md)
- [Go Implementation Plan](./implementation-plan.md)
- [Go Development Tracker](./development-tracker.md)

## What Is Already Ready

The following are already defined well enough to begin implementation:

- v1 scope and identity model
- domain model
- state model
- tool contracts
- retrieval policy
- privacy and retention rules
- configuration precedence
- observability and provenance rules
- v1 baseline and conformance matrix
- Go implementation plan

## What Does Not Need More Design Before Coding

The next session does not need more product-level discussion before starting these:

- repository layout
- initial Go module setup
- SQLite migration structure
- core domain types
- service interfaces
- MCP handler skeletons

## Recommended First Coding Slice

Build this first end-to-end slice:

1. initialize Go module and project layout
2. add SQLite open + migration support
3. implement canonical scope and session types
4. implement `memory_resolve_scope`
5. implement `memory_start_session`
6. implement durable storage for sessions

This gives the first solid foundation without jumping too early into full retrieval complexity.

## Recommended Second Coding Slice

After the first slice works:

1. implement `MemoryNote` storage
2. implement `Handoff` storage
3. implement `memory_save_note`
4. implement `memory_save_handoff`
5. add basic scope consistency validation

## Recommended Third Coding Slice

Then implement the continuity loop:

1. implement latest open handoff lookup
2. implement recent note lookup
3. implement `startup_brief` synthesis
4. implement `memory_bootstrap_session`

## Recommended Fourth Coding Slice

Then implement safe retrieval:

1. add FTS5 support
2. implement `memory_search`
3. implement `memory_get_recent`
4. implement cross-project labeling
5. enforce privacy and exclusion filtering

## Immediate Open Questions That May Need Small Coding Decisions

These are not product blockers, but the next session may need to choose them during implementation:

- exact SQLite driver choice
- migration tool style
- exact MCP library choice
- exact package naming

These can be decided locally during implementation without reopening the spec.

Configuration decision already made for the Go implementation:

- repository-local configuration should load from `configs/`
- configuration loading should use `viper`

## Best Starting Documents For The Next Session

Read in this order:

1. [V1 Baseline](../spec/v1-baseline.md)
2. [Tool Contracts](../spec/tool-contracts.md)
3. [Domain Model](../spec/domain-model.md)
4. [Retrieval Policy](../spec/retrieval-policy.md)
5. [Implementation Backlog](../implementation-backlog.md)
6. [Go Implementation Plan](./implementation-plan.md)

## Suggested First Prompt For The Next Session

Use a prompt like:

```text
Read docs/go/dev-kickoff.md, docs/go/implementation-plan.md, and docs/spec/v1-baseline.md, then start implementing the Go project skeleton and the Phase 1 foundation work.
```
