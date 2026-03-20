# codex-mem v2 Conformance Scenarios Draft

## Status

This document is a draft for potential `codex-mem` v2 work.

It is exploratory and non-normative.
The v1 documents under this directory remain the source of truth for implemented behavior unless a future v2 spec adopts this material.

## Purpose

The v2 draft set now covers:

- hybrid retrieval direction
- runtime resurfacing policy
- configuration gates
- embedding storage and backfill behavior

This document adds a verification view.

Its goal is to describe the scenario coverage a first v2 implementation should satisfy before hybrid retrieval is treated as trustworthy.

Primary references:

- [V2 Draft Outline](./v2-outline.md)
- [V2 Runtime Resurfacing Draft](./v2-runtime-resurfacing.md)
- [V2 Config Draft](./v2-config-draft.md)
- [V2 Embedding Storage And Backfill Draft](./v2-embedding-storage-draft.md)
- [V2 Hybrid Retrieval Roadmap](../go/maintainer/v2-hybrid-retrieval-roadmap.md)

## Conformance Priorities

The first v2 conformance set should optimize for these outcomes:

- v1 lexical behavior remains available and stable
- semantic retrieval never bypasses scope, privacy, searchability, or lifecycle policy
- degraded semantic states stay usable and predictable
- runtime resurfacing remains bounded, explainable, and disableable
- notes-first rollout boundaries remain enforced

## Scenario Dimensions

The initial scenario matrix should vary these dimensions:

- retrieval mode: lexical-only or hybrid
- embedding coverage: none, partial, ready, stale, failed
- semantic backend state: available, missing, incompatible, rebuilt
- request mode: bootstrap, explicit search, implicit runtime resurfacing
- scope relation: current workspace, current project, related project
- policy eligibility: searchable, non-searchable, privacy-excluded, inactive

Not every pairwise combination needs its own first-pass scenario.
The first draft should cover the smallest set that still exercises all policy boundaries.

## Scenario Group 1: Lexical-Only Baseline

### Scenario 1.1: Semantic retrieval fully disabled

Preconditions:

- `retrieval.mode=lexical`
- semantic retrieval disabled
- runtime resurfacing disabled

Expected behavior:

- `memory_search` uses lexical retrieval only
- bootstrap recovery remains unchanged
- no semantic readiness dependency is introduced

### Scenario 1.2: Hybrid codepaths compiled but still off

Preconditions:

- semantic code is present in the build
- config still selects lexical-only mode

Expected behavior:

- results remain functionally equivalent to lexical-only v1 behavior
- no sidecar index is required for startup or request handling

## Scenario Group 2: Happy-Path Hybrid Retrieval

### Scenario 2.1: Notes-only semantic retrieval in current project

Preconditions:

- `retrieval.mode=hybrid`
- note embeddings are `ready`
- current-project semantic index available
- handoffs remain lexical-only

Expected behavior:

- note candidates may come from both lexical and semantic paths
- handoff retrieval remains lexical or structured only
- final ranking still prefers stronger exact current-project matches when appropriate

### Scenario 2.2: Explicit search with fused results

Preconditions:

- same as scenario 2.1
- the query contains both exact terms and semantically related phrasing

Expected behavior:

- lexical-only matches are not lost
- semantically relevant note candidates can be added
- returned results remain deduped and policy-filtered

## Scenario Group 3: Degraded Semantic States

### Scenario 3.1: Hybrid configured but semantic index missing

Preconditions:

- `retrieval.mode=hybrid`
- semantic retrieval enabled for notes
- semantic index absent

Expected behavior:

- request handling falls back to lexical retrieval
- the system does not fail startup or ordinary search
- diagnostics can report semantic unavailability

### Scenario 3.2: Partial embedding coverage

Preconditions:

- some notes are `ready`
- some notes are `pending` or `failed`

Expected behavior:

- only `ready` notes participate in semantic candidate generation
- lexical retrieval still sees all policy-eligible notes
- hybrid retrieval remains usable without claiming full semantic coverage

### Scenario 3.3: Stale embeddings present

Preconditions:

- semantic index exists
- some note embeddings are marked `stale`

Expected behavior:

- stale notes are excluded from default semantic candidate generation unless a future degraded policy explicitly allows them
- lexical retrieval remains available for the same durable notes
- diagnostics expose stale coverage clearly

### Scenario 3.4: Incompatible sidecar version

Preconditions:

- SQLite metadata exists
- the semantic sidecar index version is incompatible with the current semantic implementation

Expected behavior:

- semantic retrieval is treated as unavailable
- lexical retrieval continues normally
- repair or rebuild is surfaced as the recovery path

## Scenario Group 4: Runtime Resurfacing Controls

### Scenario 4.1: Implicit runtime resurfacing disabled

Preconditions:

- hybrid or lexical mode may be active
- `retrieval.runtime_resurfacing.enabled=false`

Expected behavior:

- bootstrap and explicit search remain available
- no implicit memory injection occurs during ordinary request handling

### Scenario 4.2: High-confidence current-project resurfacing

Preconditions:

- implicit runtime resurfacing enabled
- current request matches prior current-project notes strongly
- consult confidence clears the configured threshold

Expected behavior:

- only `1` to `3` shaped summaries are injected
- the injected objects preserve record ids and provenance
- full durable records are not auto-expanded into active context

### Scenario 4.3: Suppression-cache repeat prevention

Preconditions:

- a note was just injected implicitly for the same task fingerprint
- no meaningful new signal has appeared

Expected behavior:

- the same note is not re-injected immediately
- explicit search still bypasses suppression

### Scenario 4.4: Related-project threshold remains stricter

Preconditions:

- related-project retrieval is allowed by policy
- implicit resurfacing is enabled
- the request has only moderate current-task similarity to related-project notes

Expected behavior:

- related-project implicit resurfacing does not fire below the stronger threshold
- current-project results continue to outrank weaker related-project matches

## Scenario Group 5: Policy Guardrails

### Scenario 5.1: Privacy and searchability exclusion

Preconditions:

- a note exists with semantic coverage
- the same note is privacy-excluded or marked non-searchable

Expected behavior:

- the note is excluded before semantic ranking
- semantic availability does not make the note visible

### Scenario 5.2: Inactive or filtered lifecycle state

Preconditions:

- a note exists with semantic coverage
- lifecycle or retrieval policy marks it ineligible for the current request

Expected behavior:

- the note is filtered before candidate fusion
- lexical and semantic paths agree on ineligibility

## Scenario Group 6: Repair And Recovery

### Scenario 6.1: Sidecar rebuild restores hybrid readiness

Preconditions:

- canonical SQLite metadata and durable notes are intact
- semantic sidecar is missing, corrupted, or intentionally cleared

Expected behavior:

- a rebuild path can recreate semantic readiness from canonical state
- hybrid retrieval becomes available again after rebuild

### Scenario 6.2: Orphaned sidecar entries do not corrupt retrieval correctness

Preconditions:

- sidecar entries exist for notes whose metadata update failed or whose canonical state changed

Expected behavior:

- those sidecar entries do not cause ineligible results to leak into retrieval
- repair tooling can reconcile or remove orphaned entries

## Suggested First Implementation Coverage

The smallest useful first-pass v2 conformance set should include at least:

1. lexical-only baseline with semantic disabled
2. happy-path hybrid retrieval for ready current-project notes
3. hybrid fallback when the semantic index is missing
4. partial-coverage behavior with ready plus pending notes
5. stale-embedding exclusion behavior
6. implicit runtime resurfacing disabled by config
7. high-confidence current-project implicit resurfacing with shaped summaries
8. suppression-cache repeat prevention
9. privacy-excluded note with semantic coverage still filtered
10. sidecar rebuild recovery

## Suggested Evidence Shape

Conformance checks should validate more than top-line success or failure.

Useful assertions include:

- result ids and ordering
- reason labels or diagnostics for lexical versus semantic contribution
- absence of policy-ineligible records
- fallback behavior when semantic infrastructure is degraded
- shaped summary fields for implicit resurfacing

## Current Recommendation

Use this document as the working draft for v2 verification scope.

The safest initial posture is:

- prove lexical fallback first
- prove policy guardrails second
- prove hybrid recall improvements third
- prove implicit resurfacing only after the previous two are stable
