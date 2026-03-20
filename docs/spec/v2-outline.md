# codex-mem v2 Draft Outline

## Status

This document is a draft for potential `codex-mem` v2 work.

It is not the current normative spec.
The v1 documents under this directory remain the source of truth for implemented behavior unless and until a future v2 spec is adopted.

Companion draft documents:

- [v2-runtime-resurfacing.md](./v2-runtime-resurfacing.md)
- [v2-config-draft.md](./v2-config-draft.md)
- [v2-embedding-storage-draft.md](./v2-embedding-storage-draft.md)
- [v2-conformance-scenarios-draft.md](./v2-conformance-scenarios-draft.md)
- [v2-migration-sequencing-draft.md](./v2-migration-sequencing-draft.md)

## Why A V2 Draft Exists

The current v1 system is intentionally local-first, scope-safe, and operationally simple:

- SQLite is the durable store
- notes use FTS5 for full-text search
- handoffs use structured retrieval and text matching
- ranking policy lives mostly in Go code

That design has worked well for structured note and handoff memory, but it leaves some room for improvement:

- lexical search misses semantically similar queries that do not share terms
- imported artifacts and larger free-form summaries may be harder to recover with keyword overlap alone
- retrieval quality may degrade when the same concept is described in very different wording across sessions

This draft describes a v2 direction that improves retrieval quality without discarding the scope, privacy, provenance, and local-first guarantees that define v1.

## Relationship To V1

V2 should be evolutionary, not a rewrite.

Required continuity assumptions:

- v1 data remains readable without migration to a different primary store
- v1 MCP tools continue to exist unless a future normative spec explicitly changes them
- v1 retrieval behavior remains available as a fallback
- scope, privacy, and provenance rules remain mandatory

V2 should treat v1 as the stable baseline and add retrieval capabilities around it rather than replacing it wholesale.

## Proposed V2 Themes

The current leading candidate for v2 is hybrid retrieval.

Hybrid retrieval means:

1. keep structured filters and lexical retrieval
2. add embedding-backed candidate retrieval for notes and selected future artifacts
3. fuse both candidate sets in the Go retrieval layer
4. preserve explicit explanations for why a result was returned

This is intentionally different from "switch everything to a vector database."

The likely v2 direction is:

- SQLite remains the canonical durable record store
- FTS remains useful for exact terminology, code identifiers, and operational queries
- vector retrieval becomes an additional recall path, not the only one
- final ranking remains policy-aware and scope-aware in Go

## Proposed V2 Goals

### 1. Hybrid note retrieval

V2 should support retrieving note candidates from both:

- lexical search such as SQLite FTS
- semantic similarity search using embeddings

The system should then merge and rerank results in a way that still respects:

- scope boundaries
- lifecycle state
- importance
- provenance
- recency
- explicit retrieval intent

### 2. Runtime memory resurfacing during active work

V2 should improve not only explicit search, but also retrieval during an active session.

Today, the system already supports:

- bootstrap-time recovery at session start
- explicit retrieval through tools such as search and recent-history lookup

But a stronger v2 experience should also support:

- retrieving relevant prior notes while a new user request is being handled
- resurfacing previously solved problems that match the current task, even when the user did not explicitly ask to search memory
- keeping that retrieval bounded by scope, privacy, and provenance rules

This should be treated as task-conditioned retrieval, not as an always-on global memory dump.

The system should be able to use the current request, active task, and recent session context to decide whether durable memory should be consulted before or during response generation.

### 3. Preserve local-first deployment

V2 should keep local-first operation as the default experience.

That means a v2 design should prefer:

- local embedding generation when practical
- local caching and backfill workflows
- storage layouts that do not force a remote SaaS dependency

V2 may allow external vector infrastructure, but it should not assume that a remote vector service is required for a normal local installation.

### 4. Keep scope safety as the first filter

Semantic similarity must not weaken project isolation.

The retrieval order should still begin with hard policy filters:

1. scope eligibility
2. privacy and searchability eligibility
3. lifecycle and provenance filters
4. candidate generation
5. ranking and dedupe

Semantic similarity should help rank eligible candidates.
It should not decide whether ineligible records become visible.

### 5. Preserve provenance and auditability

V2 should keep retrieval explainable enough for debugging and operator trust.

At minimum, a result should remain traceable by:

- source scope
- record kind
- record source or provenance
- relation type for cross-project results
- retrieval reason or score components at debug level

### 6. Avoid forcing a transcript-memory product shape

V2 may improve support for larger text summaries or imported artifacts, but it should not turn `codex-mem` into an indiscriminate transcript archive by default.

Structured notes and handoffs should remain the core durable objects.

## Proposed Runtime Retrieval Behavior

The v2 runtime experience should distinguish between three retrieval moments:

1. bootstrap retrieval at session start
2. explicit retrieval when the agent or user asks to search memory
3. implicit runtime resurfacing while handling a new request

The third mode is the important addition.

In that mode, the system should be able to:

- inspect the current task or user request
- fetch a small set of relevant prior durable records
- inject only the most relevant records into the active reasoning context
- avoid repeatedly reloading the same irrelevant memory on every turn

Recommended safeguards:

- keep the candidate set small
- prefer current workspace and current project first
- require stronger confidence for related-project expansion
- preserve provenance labels so the caller can distinguish retrieved memory from newly derived reasoning
- allow the caller or deployment to disable implicit runtime resurfacing if desired

### Proposed runtime resurfacing triggers

Implicit runtime resurfacing should not run blindly on every turn.

Recommended trigger classes:

- the current request appears to ask about a previously solved bug, fix, or design choice
- the current request contains terminology that strongly overlaps with prior durable memory
- the active task already has a durable history and the new request looks like a continuation of that work
- the request is ambiguous enough that prior project memory is likely to reduce repeated investigation cost

Recommended non-triggers:

- casual conversation that does not depend on project memory
- requests already fully satisfied by the current turn context
- repeated turns where the same memory was just surfaced and no meaningful new signal appeared

### Proposed confidence gate

Runtime resurfacing should use an explicit confidence gate before injecting durable memory into the active reasoning context.

Possible contributors to resurfacing confidence:

- lexical overlap with note title or content
- semantic similarity to prior notes
- same-task continuity signals from the current session
- scope proximity
- note type relevance such as decision, bugfix, or discovery
- recency and importance
- provenance preference for explicit memory over weaker imported artifacts

If the confidence is below a configured threshold, the system should skip automatic resurfacing and remain usable without it.

### Proposed candidate budget

The resurfacing path should stay intentionally small.

Recommended initial budget:

- default to 1 to 3 records for implicit runtime resurfacing
- allow a slightly larger budget when the user explicitly asks to search or recover memory
- prefer fewer high-confidence records over a larger noisy set

The system should optimize for relevance density, not recall at all costs.

### Proposed anti-repeat behavior

The system should avoid reloading the same memory repeatedly within one active task unless there is a good reason.

Suggested safeguards:

- keep a session-local resurfacing cache of recently injected record ids
- suppress the same records for a cooling period unless the request changed materially
- allow resurfacing again when the task shifts, the confidence rises meaningfully, or the user explicitly asks to search memory

This helps keep runtime retrieval useful rather than repetitive.

### Proposed injection strategy

Runtime resurfacing should not dump raw durable records into the live context without shaping.

Preferred behavior:

- retrieve a small set of records
- convert them into compact working-context summaries
- preserve the original record ids and provenance labels
- keep the full record available for explicit inspection when needed

Recommended working-context fields:

- record id
- kind
- source scope
- note or handoff title
- one compact reason why it is relevant now
- one compact summary of the durable content

This keeps the reasoning context compact while preserving traceability.

### Proposed degraded behavior

If the resurfacing path is unavailable or degraded, the system should still behave well.

Examples:

- embeddings unavailable: fall back to lexical-only resurfacing
- embedding coverage incomplete: use the records that do have coverage and continue
- confidence too low: skip implicit resurfacing
- indexing stale: warn in diagnostics, but do not fail ordinary request handling by default

## Suggested First Semantic Rollout Boundary

The safest first v2 implementation boundary is narrower than the full long-term direction.

Recommended starting assumptions:

- notes are the first and only semantic retrieval target
- handoffs remain lexical or structured only in early v2
- imported artifacts participate through durable notes rather than a new direct semantic path
- implicit runtime resurfacing stays current-project-first
- related-project implicit resurfacing remains optional and higher-threshold
- implicit runtime resurfacing injects shaped summaries, not full durable records

This preserves a small initial blast radius while still testing the most valuable new retrieval behavior.

## Proposed V2 Non-Goals

The following should remain out of scope unless separately justified:

- a global undifferentiated memory pool
- remote-first architecture
- mandatory hosted vector infrastructure
- automatic durable storage of raw transcripts by default
- black-box ranking with no retrieval explanation
- dropping SQLite as the canonical source of durable truth

## Proposed Retrieval Model

Recommended v2 retrieval pipeline:

1. Resolve the current scope and retrieval policy.
2. Apply hard filters for scope, privacy, searchability, and state.
3. Fetch lexical candidates.
4. Fetch semantic candidates.
5. Merge, dedupe, and rerank in Go.
6. Return results with provenance and relation labels.

### Lexical candidate path

The lexical path can continue to use:

- SQLite FTS for notes
- optional future FTS support for handoffs
- structured filters for type, state, importance, and scope

### Semantic candidate path

The semantic path should be interface-driven and pluggable.

The v2 spec should not yet require one storage backend.
Acceptable designs could include:

- SQLite plus a local sidecar vector index
- SQLite plus vector rows managed in a companion local store
- a configurable external vector backend for advanced deployments

The important requirement is not the brand of store.
It is that semantic retrieval integrates cleanly with the existing scope and policy model.

Current draft recommendation:

- keep embedding readiness metadata in SQLite
- use a pluggable semantic-index interface
- prefer a local sidecar index for the first rollout
- allow optional future external backends later without changing the canonical store

### Fusion and reranking

The final ranking layer should remain in Go, not be delegated entirely to raw backend scores.

Candidate ranking should consider at least:

- scope proximity
- lifecycle state
- importance
- recency
- source and provenance bias
- lexical relevance
- semantic relevance
- explicit search intent

This keeps retrieval policy portable even if the semantic backend changes later.

## Proposed Data Model Extensions

V2 likely needs additional metadata for embedding-backed retrieval.

Possible additions include:

- embedding model identifier
- embedding version
- embedding status for each eligible record
- embedding updated time
- chunk or segment metadata for larger future artifacts
- retrieval explanation metadata for debug output

These additions should not change the canonical meaning of `memory_items` and `handoffs`.
They should extend retrieval support around the existing durable objects.

## Proposed Tool Surface Direction

The safest v2 direction is to preserve the current tool names first.

Possible rollout order:

1. keep `memory_search` and `memory_bootstrap_session` unchanged
2. improve their retrieval quality internally behind configuration gates
3. add optional debug or explanation fields only when the contract is ready
4. consider new tool parameters only if the retrieval policy truly needs them

This minimizes client breakage and lets v2 mature behind stable entrypoints.

## Compatibility Requirements

Any v2 implementation should maintain:

- compatibility with existing v1 records
- compatibility with installations that only want lexical retrieval
- graceful operation when embeddings are unavailable or stale
- deterministic fallback to lexical-only retrieval when semantic infrastructure is disabled

## Migration Direction

Recommended migration posture:

- do not require destructive schema changes to keep v1 running
- allow embeddings to be backfilled incrementally
- treat missing embeddings as degraded-but-usable, not fatal
- keep operator control over backfill cost and retention

## Open Questions

The following questions should be resolved before a normative v2 spec is written:

1. Should v2 require a specific local embedding provider, or only define an interface?
2. After a notes-first rollout, when if ever should handoffs or imported artifacts join the semantic path?
3. What fusion strategy best balances exact keyword matches with semantic recall?
4. How much retrieval explanation should be returned by default versus debug mode only?
5. Should a single-binary local install remain the baseline even when semantic retrieval is enabled?
6. Which concrete sidecar format gives the best local-first tradeoff for the first semantic rollout?

## Suggested First V2 Work Items

If the project decides to pursue v2, a low-risk sequence would be:

1. write a maintainer roadmap for hybrid retrieval slices
2. add retrieval interfaces for semantic candidate generation
3. add optional embedding metadata and backfill state
4. implement semantic retrieval behind a config gate
5. fuse lexical and semantic candidates in the existing Go ranking layer
6. expand conformance coverage for lexical-only, semantic-enabled, and degraded fallback modes

## Current Recommendation

Treat this document as a planning anchor, not as a committed product contract.

For now:

- v1 remains the active implemented baseline
- hybrid retrieval is the most likely v2 direction
- SQLite should remain the canonical durable store unless a future v2 spec explicitly changes that decision
