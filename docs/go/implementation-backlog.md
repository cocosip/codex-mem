# codex-mem v1 Implementation Backlog

## Purpose

This document translates the `codex-mem` v1 specification into a language-neutral implementation backlog.

It is not the normative spec.
Instead, it provides a practical execution plan for building a conformant v1 implementation.

Normative reference:

- [Spec Index](../spec/README.md)

## How To Use This Backlog

- Use the spec to decide what is correct.
- Use this backlog to decide what to build first.
- Treat phases as dependency-ordered workstreams, not rigid release gates.

## Guiding Principle

Build the smallest sequence of capabilities that produces a complete continuity loop:

1. identify scope
2. create session
3. save structured notes
4. save handoff
5. bootstrap a later session from prior memory
6. search scoped memory safely

## Phase Overview

1. Foundation
2. Core Continuity Loop
3. Retrieval and Safety
4. AGENTS Integration
5. Conformance and Hardening

## Phase 1: Foundation

### Goal

Create the basic identity, storage, and canonical object layer required by all later work.

### Tasks

#### 1. Scope and identity model

- implement `system`, `project`, `workspace`, and `session` identity handling
- implement identity resolution inputs and fallback order
- implement scope consistency validation rules

Depends on:

- [domain-model.md](../spec/domain-model.md)
- [identity-consistency.md](../spec/identity-consistency.md)

#### 2. Durable storage model

- create storage for systems, projects, workspaces, sessions, notes, handoffs, relations, and imports
- persist canonical fields and lifecycle states
- support provenance and timestamps

Depends on:

- [domain-model.md](../spec/domain-model.md)
- [state-model.md](../spec/state-model.md)

#### 3. Canonical object mapping

- map canonical objects to implementation data models
- preserve required fields and enum meanings

Depends on:

- [tool-contracts.md](../spec/tool-contracts.md)
- [v1-baseline.md](../spec/v1-baseline.md)

### Phase 1 Exit Criteria

- scope can be resolved or safely degraded
- session records can be created
- notes and handoffs can be stored with full scope binding
- provenance and states are preserved

## Phase 2: Core Continuity Loop

### Goal

Deliver the first complete end-to-end continuity workflow.

### Tasks

#### 4. `memory_resolve_scope`

- implement canonical scope resolution
- support warnings for weak or ambiguous identity

#### 5. `memory_start_session`

- implement fresh session creation
- ensure no ended session is revived

#### 6. `memory_save_note`

- validate scope/session consistency
- store high-value note with importance, status, and provenance
- support conservative dedupe behavior

#### 7. `memory_save_handoff`

- validate required fields
- support `final`, `checkpoint`, and `recovery`
- preserve task summary, next steps, and status

#### 8. `memory_bootstrap_session`

- combine scope resolution, session creation, handoff retrieval, note retrieval, and startup brief synthesis
- ensure success even on first run with empty storage

Depends on:

- scope resolution
- session creation
- durable notes
- durable handoffs

### Phase 2 Exit Criteria

- one session can bootstrap, write note(s), write handoff, and end
- a later session in the same workspace or project can recover continuity
- startup brief is generated from structured memory

## Phase 3: Retrieval and Safety

### Goal

Make the memory system reliably searchable, scope-safe, and privacy-safe.

### Tasks

#### 9. Scoped retrieval engine

- implement current-workspace and current-project retrieval
- support related-project expansion only under explicit policy
- label cross-project results clearly

Depends on:

- [retrieval-policy.md](../spec/retrieval-policy.md)

#### 10. `memory_search`

- implement query-based search
- support zero-result success semantics
- rank by scope, state, importance, recency, and optional intent

#### 11. `memory_get_recent`

- implement recent handoff and note retrieval
- prioritize continuity-related items

#### 12. `memory_get_note`

- implement full retrieval by id
- return error when the requested record is absent

#### 13. Privacy and exclusion enforcement

- prevent private/do-not-store content from entering durable searchable memory
- ensure imports cannot reintroduce excluded content
- keep raw transcript-like artifacts out of the main memory index by default

Depends on:

- [privacy-retention.md](../spec/privacy-retention.md)

#### 14. Warning and error taxonomy

- map warning and error conditions to stable categories
- return consistent degraded-but-usable outcomes

Depends on:

- [warning-error-taxonomy.md](../spec/appendices/warning-error-taxonomy.md)

### Phase 3 Exit Criteria

- memory can be searched safely within scope
- privacy exclusions are enforced
- cross-project retrieval is controlled and labeled
- warnings and errors are consistent

## Phase 4: AGENTS Integration

### Goal

Integrate `codex-mem` into normal Codex workflow through safe AGENTS usage.

### Tasks

#### 15. Template packaging

- package global and project AGENTS templates
- keep placeholders and usage expectations clear

Depends on:

- [agents-policy.md](../spec/agents-policy.md)

#### 16. `memory_install_agents`

- support global, project, or both targets
- support safe default mode
- support append or explicit overwrite modes
- report exactly what changed or was skipped

#### 17. Onboarding flow support

- support single-repo setup
- support multi-repo system setup
- allow explicit system/project metadata declaration

Depends on:

- [onboarding-flows.md](../spec/appendices/onboarding-flows.md)

### Phase 4 Exit Criteria

- AGENTS templates can be installed safely
- Codex can be guided into the intended memory workflow
- onboarding is usable for both single-repo and multi-repo cases

## Phase 5: Conformance and Hardening

### Goal

Validate that the implementation is truly v1-ready and robust under edge cases.

### Tasks

#### 18. Conformance scenario coverage

- implement or execute the conformance scenarios from the matrix
- verify empty store, same-project recovery, related-project retrieval, privacy exclusion, and AGENTS safety

Depends on:

- [conformance-matrix.md](../spec/appendices/conformance-matrix.md)

#### 19. Migration edge cases

- test rename, move, split, and merge scenarios
- verify identity continuity and non-silent migration behavior

Depends on:

- [migration-examples.md](../spec/appendices/migration-examples.md)

#### 20. Observability and provenance inspection

- verify provenance is preserved for durable records
- verify exclusion and retrieval explanations exist in debug or audit paths

Depends on:

- [observability-provenance.md](../spec/observability-provenance.md)

### Phase 5 Exit Criteria

- implementation satisfies the v1 baseline
- conformance scenarios pass
- migration and identity edge cases behave predictably
- provenance and warnings are inspectable

## Priority Summary

### Must have before calling anything v1

- phase 1
- phase 2
- phase 3

### Strongly recommended before public or team usage

- phase 4
- phase 5

## Suggested Workstream Grouping

If multiple people are involved, work can be grouped as:

- Identity and scope
- Storage and canonical objects
- Session continuity tools
- Retrieval and ranking
- Privacy and policy enforcement
- AGENTS integration
- Conformance and testing

## Final Readiness Check

Do not claim v1 readiness until the implementation can:

1. bootstrap a new session with no prior memory
2. write structured notes and handoffs safely
3. recover useful continuity in a later session
4. search memory within correct scope boundaries
5. preserve privacy exclusions
6. explain degraded or cross-project results through warnings and provenance
