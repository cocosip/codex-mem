# codex-mem v2 Config Draft

## Status

This document is a draft for possible `codex-mem` v2 configuration surfaces.

It is exploratory and non-normative.
The current v1 configuration behavior remains defined by the active v1 spec unless a future v2 contract adopts this material.

## Purpose

The v2 outline proposes optional hybrid retrieval and implicit runtime resurfacing.

Those features should be configuration-gated so operators can:

- keep lexical-only retrieval as the baseline
- enable semantic features incrementally
- observe degraded behavior without breaking normal request handling
- disable implicit resurfacing independently from explicit search

Primary references:

- [V2 Draft Outline](./v2-outline.md)
- [V2 Runtime Resurfacing Draft](./v2-runtime-resurfacing.md)
- [configuration-precedence.md](./configuration-precedence.md)

## Configuration Principles

Any future v2 config surface should follow these rules:

- lexical-only behavior must remain available
- semantic retrieval must be optional
- implicit resurfacing must be independently disableable
- missing embedding infrastructure must degrade, not crash normal retrieval
- current-project scope should remain the safest default for implicit resurfacing

## Suggested Config Areas

### 1. Retrieval mode

Suggested keys:

- `retrieval.mode`
- `retrieval.lexical.enabled`
- `retrieval.semantic.enabled`

Suggested defaults:

- `retrieval.mode=lexical`
- `retrieval.lexical.enabled=true`
- `retrieval.semantic.enabled=false`

Suggested modes:

- `lexical`
- `hybrid`

The first v2 rollout should keep `lexical` as the default mode even if semantic infrastructure is compiled in.

### 2. Semantic eligibility

Suggested keys:

- `retrieval.semantic.notes.enabled`
- `retrieval.semantic.handoffs.enabled`
- `retrieval.semantic.imports.enabled`

Suggested defaults:

- `retrieval.semantic.notes.enabled=true` when semantic retrieval is enabled
- `retrieval.semantic.handoffs.enabled=false`
- `retrieval.semantic.imports.enabled=false`

The draft assumption is that early v2 semantic retrieval targets notes only.

### 3. Embedding lifecycle

Suggested keys:

- `retrieval.semantic.embedding_model`
- `retrieval.semantic.backfill_on_start`
- `retrieval.semantic.allow_stale`
- `retrieval.semantic.max_staleness_hours`

Suggested defaults:

- no implicit startup backfill in the first rollout
- stale embeddings allowed for degraded operation
- stale coverage should warn in diagnostics rather than fail normal retrieval

### 4. Runtime resurfacing

Suggested keys:

- `retrieval.runtime_resurfacing.enabled`
- `retrieval.runtime_resurfacing.max_records`
- `retrieval.runtime_resurfacing.related_projects_enabled`
- `retrieval.runtime_resurfacing.min_confidence`
- `retrieval.runtime_resurfacing.related_project_min_confidence`

Suggested defaults:

- `retrieval.runtime_resurfacing.enabled=false`
- `retrieval.runtime_resurfacing.max_records=3`
- `retrieval.runtime_resurfacing.related_projects_enabled=false`
- `retrieval.runtime_resurfacing.min_confidence=0.55`
- `retrieval.runtime_resurfacing.related_project_min_confidence=0.82`

These defaults intentionally bias toward safety and low noise.

### 5. Diagnostics and explanations

Suggested keys:

- `retrieval.debug_explanations.enabled`
- `retrieval.debug_explanations.include_scores`
- `retrieval.debug_explanations.include_signal_labels`

Suggested defaults:

- explanation details off in ordinary responses
- debug explanation data available to maintainers and diagnostics tooling

## Suggested First-Rollout Profile

The safest early v2 profile would look like this:

```yaml
retrieval:
  mode: lexical
  lexical:
    enabled: true
  semantic:
    enabled: false
    notes:
      enabled: true
    handoffs:
      enabled: false
    imports:
      enabled: false
  runtime_resurfacing:
    enabled: false
    max_records: 3
    related_projects_enabled: false
    min_confidence: 0.55
    related_project_min_confidence: 0.82
  debug_explanations:
    enabled: false
    include_scores: false
    include_signal_labels: false
```

This means a build can carry the future config surface before semantic retrieval or implicit resurfacing are turned on by default.

## Suggested Progressive Enablement

Operators should be able to enable v2 features incrementally:

1. keep lexical-only mode as baseline
2. enable semantic retrieval for notes only
3. backfill embeddings offline
4. enable debug explanations for evaluation
5. enable implicit runtime resurfacing for current-project work only
6. consider related-project expansion later if current-project behavior proves useful

## Fallback Semantics

Any future implementation should preserve these fallback rules:

- semantic disabled: use lexical retrieval only
- semantic enabled but embeddings unavailable: use lexical retrieval and emit diagnostics
- runtime resurfacing disabled: keep bootstrap and explicit retrieval behavior only
- related-project resurfacing disabled: keep implicit resurfacing within current project

## Open Questions

The following configuration questions remain open:

1. Should retrieval mode be a single top-level switch, or should lexical and semantic toggles be independently combined?
2. Should `runtime_resurfacing.enabled` default to `false` even after hybrid retrieval is considered mature?
3. How much of the consult-confidence policy belongs in config versus fixed code defaults?
4. Should future operator profiles expose a single "safe local hybrid" preset in addition to raw flags?

## Current Recommendation

Treat this document as a planning scaffold for future v2 gating and rollout work.

The best current default posture is still:

- lexical-only by default
- notes-first semantic expansion
- implicit resurfacing off by default
- diagnostics available for maintainers before broader enablement
