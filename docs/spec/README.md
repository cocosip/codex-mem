# codex-mem v1 Spec

This directory contains the formal v1 specification for `codex-mem`.

These files are derived from the design discussion log, but they focus on the current normative standard rather than the discussion history.

## Spec Index

- [glossary.md](./glossary.md)
- [domain-model.md](./domain-model.md)
- [state-model.md](./state-model.md)
- [tool-contracts.md](./tool-contracts.md)
- [retrieval-policy.md](./retrieval-policy.md)
- [privacy-retention.md](./privacy-retention.md)
- [agents-policy.md](./agents-policy.md)
- [configuration-precedence.md](./configuration-precedence.md)
- [identity-consistency.md](./identity-consistency.md)
- [observability-provenance.md](./observability-provenance.md)
- [v1-baseline.md](./v1-baseline.md)
- [appendices/README.md](./appendices/README.md)

## Reading Order

Recommended reading order:

1. Glossary
2. Domain Model
3. State Model
4. Tool Contracts
5. Retrieval Policy
6. Privacy and Retention
7. AGENTS Policy
8. Configuration and Precedence
9. Identity and Consistency
10. Observability and Provenance
11. V1 Baseline

## Scope

This spec defines the `codex-mem` v1 standard:

- external memory for Codex relay environments
- scoped continuity across restarted sessions
- structured durable memory
- handoff-based recovery
- controlled related-project retrieval

This spec does not require:

- full passive transcript capture
- vector embeddings
- semantic search infrastructure
- web UI
- team synchronization

## Draft Future Work

The following documents are exploratory and non-normative:

- [v2-outline.md](./v2-outline.md)
  Early draft direction for possible v2 hybrid retrieval work.
- [v2-runtime-resurfacing.md](./v2-runtime-resurfacing.md)
  Draft runtime algorithm and working-context shape for implicit memory resurfacing during active work.
- [v2-config-draft.md](./v2-config-draft.md)
  Draft configuration direction for hybrid retrieval and implicit runtime resurfacing gates.
- [v2-embedding-storage-draft.md](./v2-embedding-storage-draft.md)
  Draft storage and backfill direction for embeddings, semantic indexes, and degraded local-first operation.
- [v2-conformance-scenarios-draft.md](./v2-conformance-scenarios-draft.md)
  Draft verification matrix for lexical fallback, hybrid retrieval, degraded semantic states, and implicit resurfacing controls.
- [v2-migration-sequencing-draft.md](./v2-migration-sequencing-draft.md)
  Draft rollout order for additive metadata migration, sidecar bootstrap, hybrid retrieval gating, and later resurfacing.

## Appendices

Supporting appendices are available under [appendices/](./appendices/README.md), including:

- warning and error taxonomy
- example payloads
- onboarding flows
- migration examples
- conformance matrix

## Source Of Truth

When there is a conflict between implementation details and these documents, the spec in this directory should be treated as the authoritative v1 reference.
