# V2 Hybrid Retrieval Roadmap

## Purpose

This document turns the early v2 draft direction into a maintainer-oriented implementation roadmap.

It is not a normative spec.
It exists to help maintainers discuss and stage v2 work without destabilizing the completed v1 baseline.

Primary references:

- [Spec Index](../../spec/README.md)
- [V2 Draft Outline](../../spec/v2-outline.md)
- [V2 Runtime Resurfacing Draft](../../spec/v2-runtime-resurfacing.md)
- [V2 Config Draft](../../spec/v2-config-draft.md)
- [V2 Embedding Storage Draft](../../spec/v2-embedding-storage-draft.md)
- [V2 Conformance Scenarios Draft](../../spec/v2-conformance-scenarios-draft.md)
- [V2 Operator Backfill And Health Draft](../operator/v2-embedding-backfill-health-draft.md)
- [V2 Semantic Interfaces Draft](./v2-semantic-interfaces-draft.md)
- [V2 Migration Sequencing Draft](../../spec/v2-migration-sequencing-draft.md)
- [Implementation Plan](./implementation-plan.md)
- [Development Tracker](./development-tracker.md)

## Current Position

The repository currently ships a complete v1 memory loop built around:

- SQLite as the durable source of truth
- FTS5-backed note search
- scope-aware retrieval and ranking in Go
- explicit privacy, provenance, and related-project controls

The roadmap below assumes those v1 properties remain the baseline.

## Roadmap Principles

Any v2 retrieval work should follow these principles:

- keep v1 behavior available behind a stable fallback path
- do not weaken scope or privacy rules to improve recall
- keep semantic retrieval optional and configuration-gated at first
- avoid coupling the design to a single vector backend too early
- keep final ranking and policy decisions in Go
- preserve local-first operation for the default install path

## Recommended Delivery Slices

### Slice 0: Design and contracts

Goal:

- define the minimal interfaces needed for embedding generation, semantic candidate retrieval, and runtime memory resurfacing

Suggested outputs:

- draft service interfaces for embedding providers and semantic indexes
- proposed config flags for enabling semantic retrieval
- explicit fallback semantics when embeddings are absent
- explicit trigger semantics for when active-session requests should consult durable memory automatically

Why this slice comes first:

- it creates architectural clarity without committing to one storage backend

### Slice 1: Retrieval abstraction layer

Goal:

- separate lexical candidate generation from future semantic candidate generation and runtime resurfacing triggers

Likely code areas:

- `internal/domain/retrieval`
- `internal/db`

Suggested changes:

- extract lexical candidate fetch behind clearer internal interfaces
- define semantic candidate types that carry score plus explanation metadata
- keep final ranking in the existing retrieval service
- add an internal path for request-conditioned retrieval that can be used during active task handling, not only at bootstrap

Success criteria:

- current v1 retrieval still behaves the same
- semantic retrieval can be added without rewriting the whole service
- runtime resurfacing can be introduced without forcing every caller to manually orchestrate separate search calls

### Slice 1a: Runtime trigger policy

Goal:

- define when active request handling should consult durable memory automatically

Suggested outputs:

- a request-conditioned trigger evaluator
- confidence thresholds for skip versus consult
- explicit distinctions between implicit resurfacing and explicit search requests

Success criteria:

- automatic resurfacing happens often enough to be useful
- automatic resurfacing is rare enough to avoid context spam

### Slice 2: Embedding metadata and eligibility

Goal:

- track which durable records are eligible for embeddings and whether embeddings are current

Likely code areas:

- migrations
- `internal/domain/memory`
- `internal/db`

Suggested additions:

- embedding model id or version
- embedding status
- last embedded time
- optional content hash used for stale detection

Success criteria:

- records remain usable even when no embeddings exist
- the system can tell which records need backfill

### Slice 3: Backfill and update pipeline

Goal:

- generate embeddings for eligible records incrementally

Likely code areas:

- `internal/app`
- `internal/domain/memory`
- possible future background-job package

Suggested design:

- one-shot backfill command first
- incremental updates later
- explicit operator control over cost and throughput

Success criteria:

- semantic infrastructure can be built without blocking normal v1 operation
- stale or missing embeddings degrade gracefully

### Slice 4: Semantic candidate retrieval

Goal:

- retrieve note candidates by semantic similarity

Suggested scope for first implementation:

- notes only
- current project only
- configuration-gated
- handoffs remain lexical-only

Why start there:

- it limits blast radius
- it exercises the main retrieval path without reopening every object type at once

Success criteria:

- semantic retrieval returns plausible candidates
- lexical-only installs still behave normally

### Slice 5: Fusion and reranking

Goal:

- merge lexical and semantic candidates into one policy-aware ranking path

Suggested approach:

- keep simple fusion first, such as weighted merge or reciprocal-rank-style combination
- continue applying scope, state, importance, provenance, and recency weighting in Go
- dedupe candidates before final shaping

Success criteria:

- exact keyword matches still perform well
- semantically similar phrasing improves recall
- current-project exact results still outrank weak cross-project semantic matches

### Slice 6: Runtime resurfacing policy

Goal:

- consult durable memory during active request handling when the current task appears related to prior solved work

Suggested behavior:

- detect whether the current request is likely to benefit from prior durable memory
- run a bounded retrieval pass automatically
- surface only a small set of high-confidence records
- avoid repeated noisy reloads of the same memory within one active task
- maintain a session-local memory of which records were already injected recently

Success criteria:

- the system can help with "we solved something like this before" cases without forcing an explicit search step every time
- runtime retrieval remains scope-safe and explainable
- low-confidence cases do not flood the context with irrelevant memory

### Slice 6a: Working-context shaping

Goal:

- convert resurfaced durable records into compact working-context snippets instead of raw record dumps

Suggested behavior:

- keep the full durable record in storage
- inject only a concise working summary into the active reasoning path
- preserve record ids, provenance, and relevance reasons alongside the summary

Success criteria:

- runtime resurfacing improves response quality without exhausting prompt budget
- callers can still inspect the original durable record when they need full detail

### Slice 7: Retrieval explanations and diagnostics

Goal:

- make hybrid retrieval debuggable

Suggested outputs:

- debug-level retrieval traces
- reason labels such as lexical, semantic, fused, related_project
- operator diagnostics for embedding coverage and staleness
- visibility into implicit resurfacing hit rate, suppression rate, and repeated-record suppression

Success criteria:

- maintainers can explain why a result appeared
- degraded semantic state is visible without becoming a hard failure by default

### Slice 8: Broader scope expansion

Goal:

- decide whether hybrid retrieval should expand beyond notes

Candidates:

- handoffs
- imported notes
- future chunked artifacts

Recommendation:

- do not include these in the first semantic rollout unless note-only retrieval proves insufficient

## Suggested First Implementation Boundary

The safest first v2 implementation slice is:

- keep SQLite as the canonical store
- keep FTS5 as the lexical path
- add semantic retrieval for notes only
- keep semantic retrieval behind config
- backfill embeddings offline
- fuse results in `internal/domain/retrieval`

This keeps the first v2 experiment small enough to test without changing the user-facing memory workflow too aggressively.

## Recommended Compatibility Rules

V2 rollout should follow these rules:

- no existing v1 tool should require semantic infrastructure
- lexical retrieval remains the default fallback
- missing embeddings should produce warnings or degraded results, not fatal startup failures
- operators should be able to disable semantic retrieval completely
- current conformance coverage for scope and privacy must continue to pass

## Open Design Decisions

Maintainers should explicitly decide:

1. whether semantic retrieval requires a new operator command for backfill
2. whether a local sidecar index or optional external backend is the better first target
3. when if ever handoffs should move beyond lexical-only after the notes-first rollout
4. whether explanation metadata belongs in normal responses or diagnostics only
5. which embedding provider assumptions are acceptable for local installs

## Suggested Documentation Follow-Ups

If this roadmap becomes active work, the next docs to add should be:

- an implementation spike plan tied to concrete packages and conformance checkpoints
- an operator-facing readiness checklist once semantic commands become real

## Current Recommendation

Use this roadmap to shape discussion and low-risk prototypes.

Do not treat it as approval to replace the v1 retrieval model.
The current best direction is incremental hybrid retrieval layered on top of the existing v1 architecture.
