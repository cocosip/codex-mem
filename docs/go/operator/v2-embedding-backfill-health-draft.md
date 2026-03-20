# V2 Embedding Backfill And Health Draft

## Status

This document is a draft for potential `codex-mem` v2 work.

It is exploratory and non-normative.
The active operator contract remains the implemented v1 behavior unless a future v2 design is adopted.

## Purpose

The v2 storage draft recommends:

- SQLite as the canonical metadata source
- a local sidecar semantic index for the first rollout
- explicit backfill before semantic retrieval is expected to be useful

This document describes the operator view of that design:

- how semantic retrieval should be enabled safely
- what health signals should be visible
- what repair workflows should exist when semantic state degrades

Primary references:

- [V2 Config Draft](../../spec/v2-config-draft.md)
- [V2 Embedding Storage And Backfill Draft](../../spec/v2-embedding-storage-draft.md)
- [V2 Conformance Scenarios Draft](../../spec/v2-conformance-scenarios-draft.md)
- [V2 Hybrid Retrieval Roadmap](../maintainer/v2-hybrid-retrieval-roadmap.md)

## Operator Goals

The first v2 operator workflow should make these properties obvious:

- semantic retrieval is optional
- lexical retrieval remains the safe baseline
- backfill progress is inspectable
- degraded semantic state is diagnosable
- rebuild and repair paths are clearer than silent drift

## Safe Rollout Sequence

The recommended first rollout sequence is:

1. deploy a build that still defaults to lexical-only retrieval
2. verify ordinary readiness and lexical behavior
3. enable semantic retrieval for notes only in a controlled environment
4. run explicit backfill for the current project scope
5. inspect health signals until ready coverage is acceptable
6. optionally enable implicit runtime resurfacing after semantic retrieval is already behaving well

This sequence keeps semantic retrieval from becoming the first thing operators debug in production.

## Recommended Health States

Operators should be able to distinguish these semantic states clearly:

- `disabled`
- `ready`
- `degraded_missing_index`
- `degraded_partial_coverage`
- `degraded_stale_embeddings`
- `degraded_failed_embeddings`
- `degraded_incompatible_index`
- `rebuilding`

These are operator-facing health concepts, not necessarily final internal enum names.

## Minimum Health Signals

Any first v2 operator view should surface at least:

- retrieval mode
- semantic retrieval enabled or disabled
- implicit runtime resurfacing enabled or disabled
- embedding model id
- eligible note count
- ready note count
- pending note count
- stale note count
- failed note count
- sidecar index presence
- sidecar index compatibility state
- last successful backfill time
- last rebuild time if available

These signals should be available in a human-readable summary and a structured machine-readable form.

## Recommended Operator Workflows

### Workflow 1: Initial semantic enablement

Recommended steps:

1. keep lexical-only mode active
2. verify the deployment passes normal readiness checks
3. enable semantic retrieval for notes only
4. confirm that semantic health reports `disabled` or `degraded_missing_index` rather than crashing
5. run the initial backfill
6. inspect health until a useful amount of note coverage is `ready`

Expected outcome:

- hybrid retrieval can be turned on without breaking ordinary request handling

### Workflow 2: Routine backfill maintenance

Recommended steps:

1. inspect semantic health
2. run backfill for `pending`, `stale`, or retryable `failed` notes
3. confirm ready coverage improves and failures are visible

Expected outcome:

- semantic readiness drifts gradually instead of silently collapsing

### Workflow 3: Rebuild after sidecar drift or corruption

Recommended steps:

1. confirm canonical SQLite data is intact
2. mark semantic retrieval degraded rather than hard-failing the service
3. rebuild the sidecar index from canonical notes and embedding metadata
4. rerun health checks

Expected outcome:

- semantic retrieval can be restored without rewriting canonical durable memory

## Illustrative Command Shapes

These command names are placeholders, not final contract commitments.

Useful operator flows would likely need command families shaped roughly like:

```powershell
codex-mem semantic-health --json
codex-mem semantic-backfill --scope current-project
codex-mem semantic-rebuild --scope current-project
codex-mem semantic-repair --json
```

The important design point is the workflow, not the exact command spelling.

## Suggested Backfill Summary Fields

A useful backfill report should include at least:

- scope processed
- embedding model id
- eligible count
- skipped_ready count
- processed_pending count
- processed_stale count
- retried_failed count
- newly_ready count
- failed count
- started_at
- completed_at

Optional but helpful fields:

- sample failure codes
- whether a rebuild was required first
- sidecar version before and after the run

## Suggested Health Summary Fields

A useful semantic health report should include at least:

- retrieval mode
- semantic enabled
- implicit resurfacing enabled
- current embedding model id
- sidecar present
- sidecar compatible
- eligible count
- ready count
- pending count
- stale count
- failed count
- orphan count if known
- last successful backfill time

## Failure Response Guide

### Missing sidecar index

Expected operator response:

- keep lexical retrieval active
- initialize or rebuild the sidecar
- rerun semantic health

### Partial coverage

Expected operator response:

- continue lexical retrieval normally
- run backfill for remaining eligible notes
- investigate repeated failures if ready coverage stalls

### Stale embeddings

Expected operator response:

- treat lexical retrieval as the reliable fallback
- run targeted backfill or rebuild
- confirm stale count decreases

### Incompatible sidecar version

Expected operator response:

- do not trust semantic retrieval
- rebuild the sidecar against the current semantic implementation
- verify compatibility before re-enabling hybrid expectations

## Rollout Guardrails

The first operator-facing v2 rollout should preserve these rules:

- do not require semantic infrastructure for service startup correctness
- do not default implicit runtime resurfacing to on
- do not require operators to backfill handoffs or imported artifacts in the first rollout
- do not hide degraded semantic states behind generic success messages

## Current Recommendation

Treat this document as the draft operator playbook for v2 semantic rollout.

The best first operator posture is:

- lexical-first by default
- notes-only semantic enablement
- explicit backfill and explicit health checks
- rebuild-friendly recovery when the sidecar drifts
