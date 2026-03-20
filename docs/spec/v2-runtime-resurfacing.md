# codex-mem v2 Runtime Resurfacing Draft

## Status

This document is a draft for potential `codex-mem` v2 work.

It is exploratory and non-normative.
The v1 documents under this directory remain the source of truth for implemented behavior unless a future v2 spec adopts this material.

## Purpose

This document narrows one specific part of the broader v2 outline:

- task-conditioned retrieval while a request is already being handled
- confidence-gated resurfacing of prior durable memory
- compact working-context injection instead of raw durable-record dumping

Primary references:

- [V2 Draft Outline](./v2-outline.md)
- [Retrieval Policy](./retrieval-policy.md)
- [V2 Hybrid Retrieval Roadmap](../go/maintainer/v2-hybrid-retrieval-roadmap.md)

## Relationship To Explicit Search

This draft distinguishes three modes:

1. bootstrap recovery at session start
2. explicit search or recovery when the user or agent asks for memory lookup
3. implicit runtime resurfacing when active request handling appears likely to benefit from prior durable memory

This document is about the third mode.

The system should remain fully usable when implicit resurfacing is disabled, unavailable, or skipped by confidence policy.

## Suggested First Implementation Boundary

The first v2 resurfacing rollout should stay intentionally narrow:

- notes are eligible for semantic candidate generation and implicit resurfacing
- handoffs remain available for bootstrap recovery and explicit retrieval, but stay lexical or structured only in the first semantic rollout
- imported artifacts should influence ranking only through their durable note forms, not as a new direct resurfacing object type
- current-project candidates should be the default implicit resurfacing scope
- related-project expansion should require a stronger threshold and can remain disabled in the first production slice
- implicit resurfacing should inject shaped summaries only, not full durable records

This keeps the first rollout aligned with the current v1 object model while avoiding a large surface-area increase.

## Request Classification

Before consulting durable memory, the runtime should classify the current turn into one of three request classes:

### Class A: Explicit memory request

Examples:

- "restore the previous task context"
- "search memory for the last fix"
- "what did we decide about retrieval scoring?"

Expected behavior:

- retrieval is allowed to use a larger candidate budget
- the caller may inspect full durable records
- suppression-cache cooling should not block retrieval

### Class B: Implicit resurfacing candidate

Examples:

- the request looks like a continuation of recent tracked work
- the request mentions a bug, decision, or design topic likely solved before
- the request is underspecified and prior durable memory would likely reduce repeated investigation cost

Expected behavior:

- run a bounded consult decision
- inject only compact shaped summaries when confidence is high enough

### Class C: No retrieval needed

Examples:

- casual conversation
- self-contained requests fully answered by current turn context
- repeated follow-ups where the same memory was just injected and no stronger new signal exists

Expected behavior:

- skip memory consult and continue normal request handling

## Consult Decision Pipeline

Implicit runtime resurfacing should use a two-stage decision pipeline:

1. hard bypass checks
2. scored consult decision

### Stage 1: Hard bypass checks

Skip implicit resurfacing immediately when any of the following is true:

- implicit resurfacing is disabled by configuration
- the request is already an explicit memory operation
- the current turn already includes sufficient retrieved durable context
- the request class is clearly non-project or non-memory-dependent
- the session-local suppression cache marks the same task fingerprint as recently satisfied without stronger new evidence

### Stage 2: Scored consult decision

If stage 1 does not bypass, compute a consult confidence score in the range `0.0` to `1.0`.

Suggested signals:

- lexical overlap with note titles, tags, or compact summaries
- semantic similarity to prior notes
- same-task continuity from the current active session
- scope proximity favoring current workspace and current project
- record-type relevance such as decision, discovery, or bugfix
- importance and recency
- provenance preference for explicit durable memory over weaker imported material

Suggested prototype weighting:

- lexical overlap: up to `0.30`
- semantic similarity: up to `0.25`
- same-task continuity: up to `0.20`
- scope proximity: up to `0.10`
- note-type relevance: up to `0.05`
- importance and recency: up to `0.05`
- provenance preference: up to `0.05`

These weights are draft defaults for experiments, not a normative scoring contract.

## Suggested Confidence Thresholds

The first prototype should use conservative defaults:

- below `0.55`: skip implicit resurfacing
- `0.55` to below `0.72`: consult current-project note candidates only
- `0.72` and above: allow current-project resurfacing with normal summary injection
- `0.82` and above: allow optional related-project candidate consideration when policy permits

Explicit search should bypass these implicit thresholds and follow explicit retrieval policy instead.

These numbers are intended to keep implicit resurfacing rare and useful rather than broad and noisy.

## Candidate Retrieval And Budgeting

When implicit resurfacing is allowed to consult memory, the initial candidate pass should remain small:

- lexical candidates: up to `8`
- semantic candidates: up to `8`
- fused ranked candidates before shaping: up to `5`
- injected working-context summaries: default `1` to `3`

Ranking should still apply hard policy filters before candidate generation:

1. scope eligibility
2. privacy and searchability eligibility
3. lifecycle and provenance eligibility
4. lexical and semantic candidate generation
5. fusion, dedupe, and final ranking

If semantic coverage is absent or stale, the system should continue with the lexical path only.

## Session-Local Suppression Cache

Implicit resurfacing should maintain a session-local suppression cache so the same durable records do not keep reappearing on every turn.

Recommended cache key:

- task fingerprint
- record id
- retrieval mode

Recommended cache value:

- last injected time
- last injected turn index if available
- last consult confidence
- last relevance reason label

Suggested cooling rules:

- suppress re-injection of the same record for roughly `5` turns or `15` minutes, whichever ends later
- allow re-injection earlier if consult confidence increases by at least `0.15`
- allow re-injection when the task fingerprint changes materially
- never let the suppression cache block explicit search or explicit record inspection

Suggested task-fingerprint ingredients:

- normalized user request summary
- active task summary
- current scope identifiers
- recent retrieved-memory ids

The fingerprint does not need to be perfect.
It only needs to suppress obvious repeats within one active line of work.

## Working-Context Shaping

Implicit resurfacing should inject compact shaped memory, not the raw durable record body.

The first rollout should use a working-context payload shaped like this:

```json
{
  "record_id": "note_20260320_005246_52109d3a",
  "kind": "note",
  "title": "v2 draft should include task-conditioned runtime resurfacing of durable memory",
  "scope": {
    "workspace_id": "ws_d04ba8d70ab5",
    "project_id": "proj_972b12a48f61",
    "relation": "current_project"
  },
  "provenance": {
    "source": "codex_explicit",
    "status": "active"
  },
  "retrieval": {
    "mode": "implicit_runtime_resurfacing",
    "confidence": 0.78,
    "reason_label": "same_task_design_continuation",
    "reason_summary": "Matches the active v2 retrieval design task",
    "signals": ["same_task", "lexical", "semantic"]
  },
  "durable_summary": "Prior draft work already established runtime resurfacing as a bounded, disableable part of the v2 direction.",
  "freshness": {
    "created_at": "2026-03-20T00:52:46Z"
  },
  "full_record_hint": "Use memory_get_note with the record_id for full detail"
}
```

Required first-cut fields:

- `record_id`
- `kind`
- `title`
- `scope`
- `provenance`
- `retrieval.reason_label`
- `retrieval.reason_summary`
- `durable_summary`

Recommended constraints:

- `durable_summary` should usually stay within `1` to `3` compact sentences
- `reason_summary` should explain relevance now, not restate the whole note
- raw note content should not be injected by default
- callers should retain a path to inspect the full record explicitly

## Full-Record Expansion Policy

The first v2 resurfacing slice should not auto-expand full durable records into active context.

Full-record expansion should be reserved for:

- explicit user or agent requests
- debug or diagnostics tooling
- rare future cases where a separate budgeted policy is adopted intentionally

This keeps the implicit path safe for prompt budget and provenance clarity.

## Degraded Behavior

Implicit resurfacing should degrade gracefully:

- embeddings unavailable: use lexical-only consult
- embedding coverage incomplete: use covered records and continue
- consult confidence too low: skip implicit resurfacing
- ranking diagnostics unavailable: omit diagnostics, keep normal behavior
- suppression cache unavailable: continue without repeat suppression rather than failing request handling

No degraded state here should block ordinary request handling by default.

## Diagnostics

Implementations should expose enough debug information to explain why implicit resurfacing did or did not run.

Useful diagnostics include:

- request class
- consult confidence
- triggered signal labels
- whether semantic retrieval was available
- whether suppression cache prevented reinjection
- injected record ids and reason labels

These details should default to diagnostics or debug views rather than ordinary user-facing responses.

## Open Questions

The following questions remain open even with the suggested defaults above:

1. Should the consult score be a hand-tuned weighted sum, a calibrated model output, or a simpler rule engine at first?
2. Should related-project implicit resurfacing be allowed in the first prototype, or wait until current-project behavior is proven?
3. Should the suppression cache be persisted across restarts, or remain session-local only?
4. What is the best compact-summary length target before prompt-budget pressure outweighs retrieval value?

## Current Recommendation

Use this document as the working draft for runtime resurfacing behavior inside the broader v2 effort.

For the first implementation attempt:

- keep implicit resurfacing note-only
- keep it current-project-first
- gate it behind configuration
- inject shaped summaries only
- keep lexical-only fallback always available
