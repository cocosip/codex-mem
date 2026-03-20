# codex-mem v2 Embedding Storage And Backfill Draft

## Status

This document is a draft for potential `codex-mem` v2 work.

It is exploratory and non-normative.
The v1 documents under this directory remain the source of truth for implemented behavior unless a future v2 spec adopts this material.

## Purpose

The v2 outline establishes hybrid retrieval as the leading direction, but it leaves open a practical question:

- where embedding state should live
- how semantic candidate retrieval should be wired in
- how embeddings should be backfilled and refreshed without destabilizing the v1 baseline

This document narrows that storage and lifecycle question into a concrete draft direction.

Primary references:

- [V2 Draft Outline](./v2-outline.md)
- [V2 Config Draft](./v2-config-draft.md)
- [V2 Runtime Resurfacing Draft](./v2-runtime-resurfacing.md)
- [V2 Hybrid Retrieval Roadmap](../go/maintainer/v2-hybrid-retrieval-roadmap.md)

## Design Constraints

Any v2 embedding-storage design should preserve the core v1 properties:

- SQLite remains the canonical durable store for notes and handoffs
- lexical retrieval remains available without semantic infrastructure
- scope, privacy, searchability, and lifecycle policy remain hard filters before semantic ranking
- local-first installation remains the default experience
- missing or stale embeddings degrade retrieval quality, not baseline product correctness

These constraints matter more than choosing one fashionable vector backend.

## What Must Be Stored

V2 does not need to make embeddings part of the canonical note body.

Instead, it needs two classes of state:

### 1. Canonical embedding metadata

This state should remain in SQLite because it is part of the system of record for retrieval readiness.

Recommended metadata fields per eligible durable record:

- durable record id
- durable record kind
- embedding eligibility state
- embedding model id
- embedding content version
- embedding content hash
- embedding status
- last embedded at
- last embedding error code
- last embedding error time

### 2. Non-canonical vector material

This state is the high-dimensional embedding payload used for semantic retrieval.

It does not need to become part of the canonical product record.
It only needs stable linkage back to the canonical durable record id plus enough versioning metadata to detect staleness.

## Storage Options

The v2 outline already allows multiple backend shapes.
This draft compares them more directly.

### Option A: Store vectors directly in SQLite

Description:

- keep metadata and vector payloads in the same SQLite database
- attempt to support nearest-neighbor lookup from SQLite itself or from application-side scanning

Strengths:

- operationally simple on paper
- one file remains the full local data footprint
- backup story is straightforward

Weaknesses:

- practical vector search support inside the current SQLite setup is not yet established in this repository
- pure application-side scanning does not look attractive once the note set grows
- coupling vector experimentation to the canonical SQLite file raises upgrade and portability risk earlier than necessary

Current assessment:

- acceptable as a future specialized path
- not the best first target for low-risk v2 exploration

### Option B: SQLite metadata plus local sidecar semantic index

Description:

- keep canonical metadata in SQLite
- store vector payloads and nearest-neighbor data structures in a sidecar local index managed by a semantic-index interface
- link sidecar entries back to durable record ids and embedding content versions

Strengths:

- preserves SQLite as the canonical source of truth
- keeps the semantic path pluggable and easier to replace
- supports local-first installs without requiring a remote service
- lets the project evolve semantic internals without rewriting the canonical durable schema

Weaknesses:

- introduces another local artifact that must stay in sync
- requires explicit rebuild or repair workflows
- backup and diagnostics need to acknowledge a two-part local state model

Current assessment:

- best first-fit option for the project's current goals
- offers the cleanest separation between canonical memory state and experimental semantic infrastructure

### Option C: SQLite metadata plus external vector backend

Description:

- keep canonical metadata in SQLite
- delegate vector storage and nearest-neighbor search to a configurable external system

Strengths:

- can unlock more mature ANN capabilities quickly
- may scale better for advanced or centralized deployments

Weaknesses:

- weakens the default local-first story
- adds deployment and connectivity complexity
- risks making semantic retrieval feel like a separate product subsystem

Current assessment:

- useful as an advanced optional backend later
- not a good baseline for the first v2 rollout

## Draft Recommendation

The best first v2 direction is:

- keep canonical embedding metadata in SQLite
- use a pluggable semantic-index interface
- make the default implementation a local sidecar index
- allow optional future external backends only behind the same interface

This gives the project a narrow and reversible first step.

## Recommended First-Cut Architecture

The first implementation should separate responsibilities clearly:

### SQLite responsibilities

SQLite should continue to own:

- durable note and handoff records
- scope and lifecycle fields
- privacy and searchability state
- lexical retrieval structures
- embedding readiness metadata
- backfill queue visibility or status metadata if needed

### Semantic index responsibilities

The semantic index should own:

- vector payload persistence
- nearest-neighbor candidate lookup
- index-local rebuild state
- index version compatibility checks

### Retrieval service responsibilities

The Go retrieval layer should own:

- hard policy filtering before semantic candidate use
- requesting semantic candidates for eligible records
- fusion and reranking across lexical and semantic paths
- degraded fallback behavior when semantic state is absent or stale

This keeps policy in Go and storage-specific behavior in the semantic backend.

## Suggested Metadata Model

The first v2 metadata extension can stay intentionally small.

Recommended per-record fields:

- `embedding_eligible`
- `embedding_status`
- `embedding_model_id`
- `embedding_content_hash`
- `embedding_content_version`
- `embedded_at`
- `embedding_error_code`
- `embedding_error_at`

Suggested status values:

- `not_applicable`
- `pending`
- `ready`
- `stale`
- `failed`

Suggested semantics:

- `not_applicable`: the record type or policy is not eligible for embeddings
- `pending`: eligible but not yet embedded
- `ready`: embedding is available and matches current content version
- `stale`: embedding exists but content or model version changed
- `failed`: the last embedding attempt failed and needs operator attention or retry

The content hash should be derived from the embedding source text, not from unrelated record metadata.

## Suggested Embedding Source Text

For the first notes-only rollout, embedding text should be derived from a stable, bounded projection of the durable record.

Recommended source ingredients:

- note title
- note type
- compact normalized body content
- optional tags if the product later adopts them explicitly

It should not depend on:

- volatile session-local context
- diagnostic fields
- retrieval scores

This keeps staleness detection predictable and avoids unnecessary re-embedding.

## Backfill Model

The first backfill workflow should be explicit and operator-controlled.

Recommended first phase:

- one-shot backfill command
- current-project scope first
- notes only
- resumable progress by status and content hash

Recommended later phase:

- incremental background updates when durable notes change
- optional scheduled maintenance workflows

This matches the project's cautious rollout posture.

## Suggested Backfill Pipeline

The first operator backfill flow should look like this:

1. enumerate eligible records from SQLite
2. skip records already marked `ready` for the current content hash and model id
3. select a bounded batch of `pending`, `stale`, or retryable `failed` records
4. generate embeddings
5. write vector payloads into the semantic index
6. update SQLite metadata only after the semantic write succeeds
7. emit a summary of processed, skipped, failed, and stale-cleared records

This ordering avoids claiming a record is ready before the semantic index actually contains the vector.

## Failure And Repair Semantics

Sidecar designs only work well if repair is explicit.

Recommended failure handling:

- semantic write fails after embedding generation: leave SQLite status as `pending` or mark `failed`
- SQLite metadata update fails after sidecar write: treat the sidecar entry as orphan-repairable and detect it during health checks
- sidecar index missing or incompatible: mark semantic retrieval unavailable and fall back to lexical retrieval
- content hash mismatch: mark record `stale` and exclude stale semantic candidates unless degraded policy says otherwise

Recommended repair workflows:

- rebuild the entire sidecar index from canonical SQLite metadata and durable records
- reconcile sidecar entries against SQLite embedding metadata
- clear failed states for retryable records without touching canonical note content

## Degraded Retrieval Rules

Retrieval behavior should remain predictable in partially-built states:

- no semantic index present: lexical-only retrieval
- semantic index present but empty: lexical-only retrieval plus diagnostics
- semantic index partially populated: hybrid retrieval over only `ready` records
- semantic index incompatible with current model version: lexical-only retrieval until rebuild or repair

The fallback should be deterministic and boring.

## Health And Diagnostics

Any first v2 implementation should expose enough observability for maintainers to understand semantic readiness.

Useful health signals include:

- eligible record count
- ready record count
- stale record count
- failed record count
- sidecar index version
- sidecar orphans count
- current embedding model id
- last successful backfill time

These can remain maintainer-facing before any operator UX is formalized.

## Suggested Interface Shape

The implementation should likely split the semantic path into two interfaces:

### Embedding provider

Responsibilities:

- turn bounded source text into vectors
- expose provider or model identity
- surface retryable versus permanent failures

### Semantic index

Responsibilities:

- upsert vectors keyed by durable record id and content version
- fetch nearest candidates for a query vector
- report readiness or compatibility state
- rebuild, repair, or clear index-local state when needed

This keeps provider and storage concerns separate.

## First Implementation Boundary

The first real implementation should keep all of these limits:

- notes only
- one embedding source-text shape
- one local sidecar implementation
- explicit backfill command first
- lexical fallback always available
- no mandatory startup indexing
- no semantic dependence for bootstrap recovery correctness

These limits are a feature, not a temporary embarrassment.

## Open Questions

The following design questions remain open:

1. What sidecar format best matches the project's portability goals: another SQLite file, a purpose-built local index file, or a small embedded library format?
2. Should stale vectors ever remain queryable in degraded mode, or should only `ready` vectors participate in candidate generation?
3. How much backfill progress state belongs in SQLite metadata versus index-local bookkeeping?
4. Should the first backfill command operate only on the current project scope, or also allow related-project batches when policy permits?

## Current Recommendation

Use this document as the working draft for v2 semantic storage and backfill design.

For the first implementation attempt:

- keep SQLite as the canonical metadata source
- add only minimal embedding lifecycle fields there
- use a local sidecar semantic index behind an interface
- start with one-shot operator backfill
- treat rebuild and lexical fallback as first-class paths
