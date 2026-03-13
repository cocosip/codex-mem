# Observability and Provenance

## Core Principle

Users and downstream tools should be able to understand:

- what was stored
- why it was stored
- what was retrieved
- why it was retrieved
- what was excluded
- why it was excluded

## Provenance Requirement

Every durable memory record should carry provenance sufficient to explain its origin.

Recommended provenance dimensions:

- source category
- source session
- source scope
- creation time
- whether the record was explicit, imported, inferred, or recovery-generated

## Source Categories

Recommended stable categories:

- `codex_explicit`
- `watcher_import`
- `relay_import`
- `recovery_generated`

## Retrieval Explainability

Implementations should be able to explain retrieval at a high level using:

- matched scope
- matched state
- matched type
- importance contribution
- recency contribution
- relation-based inclusion when cross-project results appear

Rule:

- The exact scoring formula does not need to be exposed.
- Enough explanation must exist to justify inclusion and ordering.

## Write Auditability

Write operations should make it possible to understand:

- what tool wrote the record
- when it was written
- which session and scope it belongs to
- whether it was deduplicated
- whether it superseded or replaced earlier memory

## Retrieval Traces

Optional debug traces may include:

- effective scope used
- whether related-project expansion occurred
- record families searched
- fallback reasons
- exclusion reasons

Rule:

- Retrieval traces are optional and not required in normal user-facing flows.

## Import Audit

Imported artifacts should be auditable separately from explicit memory.

Recommended fields:

- import source
- external identifier
- import time
- payload hash
- whether durable memory was created
- whether the artifact was suppressed

## Exclusion Explainability

The system should be able to explain exclusions such as:

- excluded due to privacy policy
- excluded due to unrelated project scope
- excluded due to superseded lower-rank status
- excluded due to dedupe
- excluded due to result limit

## Privacy Balance

Observability must not leak sensitive content.

Rule:

- Explain exclusion or redaction decisions without exposing the private material itself.
