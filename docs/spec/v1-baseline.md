# V1 Baseline

## Conformance Language

- `MUST`: required for v1 conformance
- `SHOULD`: strongly recommended
- `MAY`: optional enhancement
- `MUST NOT`: prohibited

## Scope and Identity

A v1 implementation:

- MUST support `system`, `project`, `workspace`, and `session`
- MUST bind every durable note and handoff to one full scope chain
- MUST default retrieval to the current project boundary
- MUST NOT include unrelated project memory in default retrieval
- SHOULD support controlled related-project retrieval within the same system

## Session Continuity

A v1 implementation:

- MUST provide bootstrap-based session recovery
- MUST create a fresh session for each new Codex session
- MUST prefer the latest open handoff during bootstrap
- MUST succeed even when prior memory is absent
- SHOULD synthesize a compact startup brief

## Durable Memory

A v1 implementation:

- MUST support structured notes
- MUST support structured handoffs
- MUST distinguish explicit memory from imported or inferred memory
- SHOULD preserve import-audit provenance for secondary artifacts
- MAY materialize imported artifacts into durable notes when explicit-memory precedence is preserved
- MUST support normalized importance `1..5`
- MUST support note and handoff lifecycle states

## Retrieval

A v1 implementation:

- MUST support scoped search
- MUST support recent retrieval
- MUST rank results using scope-aware rules
- MUST label cross-project results clearly
- MUST treat zero results as success

## Tool Surface

A v1 implementation:

- MUST expose the conceptual tool contracts defined in [tool-contracts.md](./tool-contracts.md)
- SHOULD keep import-audit and imported-note workflows aligned with the same scope, privacy, and provenance rules as explicit memory
- MUST preserve scope-safety, warning, and error semantics

## AGENTS Integration

A v1 implementation:

- MUST support AGENTS-based workflow guidance
- MUST NOT use AGENTS as the primary dynamic memory store
- MUST support safe AGENTS installation or generation
- MUST default AGENTS installation to non-destructive behavior

## Privacy and Retention

A v1 implementation:

- MUST be local-first by default
- MUST provide a way to exclude private content from durable searchable memory
- MUST ensure imports do not reintroduce excluded content
- MUST keep raw transcript-like artifacts out of the primary memory index by default

## Observability

A v1 implementation:

- MUST preserve provenance for durable records
- MUST surface warnings for degraded but usable outcomes
- MUST make cross-project results traceable to their source project

## Consistency

A v1 implementation:

- MUST validate scope consistency before storing notes or handoffs
- MUST prevent invalid cross-scope references
- MUST treat identity conflicts as warnings or errors rather than silently merging data

## Non-Goals

V1 does not require:

- full passive transcript capture
- vector embeddings
- semantic search
- automatic LLM importance classification
- web UI
- team synchronization
- aggressive automatic merge of arbitrary AGENTS files

## Prohibitions

A v1 implementation:

- MUST NOT default to a global undifferentiated memory pool
- MUST NOT durably store explicitly excluded private content
- MUST NOT revive ended sessions as active sessions
- MUST NOT treat transcript fragments as first-class durable notes by default
- MUST NOT silently overwrite existing AGENTS files in safe/default mode

## Acceptance Criteria

An implementation is v1-ready when it can:

1. Safely install global and/or project AGENTS templates.
2. Start a new session and bootstrap prior context.
3. Save at least one structured note during work.
4. Save a handoff before pausing or ending.
5. Start a later session in the same workspace or project and recover useful continuity.
6. Search prior memory by scoped query.
7. Keep unrelated project memory out of default retrieval.
8. Preserve privacy exclusions and provenance throughout the workflow.
