# codex-mem v2 Migration Sequencing Draft

## Status

This document is a draft for potential `codex-mem` v2 work.

It is exploratory and non-normative.
The v1 documents under this directory remain the source of truth for implemented behavior unless a future v2 spec adopts this material.

## Purpose

The v2 drafts now describe:

- hybrid retrieval
- runtime resurfacing
- semantic storage and backfill
- conformance expectations

This document narrows the rollout question:

- in what order schema, config, backfill, and retrieval changes should land
- how to keep lexical retrieval stable throughout that rollout
- how to avoid making semantic state a correctness dependency too early

Primary references:

- [V2 Draft Outline](./v2-outline.md)
- [V2 Config Draft](./v2-config-draft.md)
- [V2 Embedding Storage And Backfill Draft](./v2-embedding-storage-draft.md)
- [V2 Conformance Scenarios Draft](./v2-conformance-scenarios-draft.md)
- [V2 Semantic Interfaces Draft](../go/maintainer/v2-semantic-interfaces-draft.md)

## Rollout Principles

Any migration sequence should preserve these invariants:

- existing v1 data remains readable
- lexical retrieval remains available throughout rollout
- missing semantic state is degraded, not fatal
- sidecar rebuild remains possible from canonical SQLite data
- implicit runtime resurfacing stays disabled until semantic retrieval itself is proven

## Recommended Migration Phases

### Phase 0: Interface-first code prep

Goal:

- land retrieval and memory-domain interfaces before semantic behavior is enabled

Suggested outputs:

- nil-safe semantic collaborators in retrieval
- note embedding projection and metadata types
- config fields parsed but defaulting to disabled

Why first:

- it lets the codebase absorb new concepts without changing runtime behavior yet

### Phase 1: Additive SQLite metadata migration

Goal:

- add embedding lifecycle metadata to canonical note storage without altering v1 reads

Suggested migration:

- create a new additive migration after `005_import_records.sql`
- recommended filename: `006_note_embedding_metadata.sql`

Suggested columns on `memory_items`:

- `embedding_eligible`
- `embedding_status`
- `embedding_model_id`
- `embedding_content_hash`
- `embedding_content_version`
- `embedded_at`
- `embedding_error_code`
- `embedding_error_at`

Recommended safety posture:

- keep new columns nullable or safely defaulted
- avoid destructive rewrites of note content
- do not require every existing row to be embedded before startup succeeds

### Phase 2: Metadata-aware but lexical-only deployment

Goal:

- ship a build that understands new metadata but still behaves lexically by default

Expected behavior:

- v1 retrieval stays unchanged
- new binaries can read and write embedding metadata
- no sidecar index is required yet

Why this phase matters:

- it proves the schema change is boring before semantic features depend on it

### Phase 3: Backfill and sidecar bootstrap tooling

Goal:

- add explicit operator workflows for sidecar creation, health, and rebuild

Suggested outputs:

- semantic health command or workflow
- one-shot backfill command
- rebuild command

Expected behavior:

- semantic state can be created from canonical SQLite records
- lexical retrieval remains correct even when the sidecar is absent

### Phase 4: Notes-only hybrid retrieval behind config

Goal:

- enable semantic note retrieval as an additive candidate path

Guardrails:

- notes only
- current-project-first behavior
- handoffs remain lexical-only
- semantic retrieval off by default

Expected behavior:

- hybrid retrieval works only when config enables it and sidecar health is sufficient
- degraded cases still fall back to lexical retrieval

### Phase 5: Diagnostics and conformance hardening

Goal:

- expose explanation and health signals needed to trust the hybrid path

Suggested outputs:

- reason labels for lexical versus semantic contribution
- health summaries for ready, stale, and failed coverage
- named conformance scenarios for degraded states and rebuild recovery

### Phase 6: Optional implicit runtime resurfacing

Goal:

- enable task-conditioned resurfacing only after semantic retrieval itself is stable

Guardrails:

- still off by default
- shaped-summary injection only
- current-project-first thresholds
- suppression-cache repeat prevention

This should not land in the same first patch that introduces semantic search itself.

## Recommended Schema Posture

The first migration should be additive and reversible in practice.

That means:

- do not rename existing v1 columns
- do not remove FTS or lexical search structures
- do not introduce startup checks that require semantic metadata completeness
- do not store the canonical durable record only inside the sidecar

The canonical record must remain the SQLite note.

## Eligibility Initialization Strategy

The first rollout should avoid a heavyweight migration that tries to classify every historical note eagerly.

Safer options:

- set safe defaults in migration and let application code compute eligibility lazily
- or run an explicit post-migration reconciliation step during backfill preparation

Recommended current posture:

- migration adds metadata fields
- app code or backfill workflow computes notes-first eligibility explicitly
- semantic readiness becomes true only after successful backfill writes

This avoids turning schema migration into a long-running semantic job.

## Sidecar Introduction Strategy

The sidecar should not be part of schema migration itself.

Instead:

- schema migration makes canonical metadata possible
- operator tooling creates or rebuilds the sidecar later
- retrieval checks sidecar availability at runtime and falls back cleanly when it is absent

This keeps database upgrades independent from ANN index lifecycle concerns.

## Compatibility Expectations

During rollout, the system should preserve these compatibility expectations:

- older data remains readable by newer binaries
- newer binaries remain usable when the sidecar has not been created yet
- semantic-disabled installs do not need to care about backfill progress
- hybrid-enabled installs can survive sidecar loss by falling back to lexical retrieval

## Suggested Migration Checkpoints

The first implementation should probably pause and validate after these checkpoints:

1. additive metadata migration exists and ordinary v1 regression tests still pass
2. metadata-aware builds run with semantic retrieval still disabled
3. sidecar bootstrap and rebuild tooling work on a current-project note set
4. notes-only hybrid retrieval passes lexical-fallback and policy-guardrail conformance scenarios
5. implicit runtime resurfacing remains disabled until the previous checkpoint is stable

## Explicit Non-Goals During Early Migration

The early migration sequence should avoid:

- semantic support for handoffs in the first wave
- mandatory startup backfill
- destructive schema rewrites
- coupling sidecar creation to ordinary database migration
- enabling implicit runtime resurfacing by default

## Current Recommendation

Use this document as the rollout order for the first v2 implementation attempt.

The safest sequence is:

- interfaces first
- additive metadata migration second
- backfill and health workflows third
- notes-only hybrid retrieval fourth
- implicit runtime resurfacing later
