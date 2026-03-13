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

## Appendices

Supporting appendices are available under [appendices/](./appendices/README.md), including:

- warning and error taxonomy
- example payloads
- onboarding flows
- migration examples
- conformance matrix

## Source Of Truth

When there is a conflict between implementation details and these documents, the spec in this directory should be treated as the authoritative v1 reference.
