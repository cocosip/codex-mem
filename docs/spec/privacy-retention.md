# Privacy and Retention

## Core Principle

`codex-mem` should be local-first and store only durable, high-value, necessary memory by default.

## Sensitive Data

Sensitive material should not enter the main memory index by default.

Examples:

- API keys
- tokens
- passwords
- credentials
- raw personal data
- private customer payloads

## Private / Do-Not-Store

The system should support a privacy intent equivalent to:

- `private`
- `do_not_store`
- `ephemeral_only`

Rules:

- Marked private content must not be written into durable searchable memory.
- Private exclusions outrank automated enrichment.
- Imports must not reintroduce excluded content.

## Retention Classes

### Class A: Core Continuity Records

Includes:

- handoffs
- high-value notes
- scope metadata

Retention:

- retained by default until user deletion or archival

### Class B: Import Provenance

Includes:

- import tracking
- payload hashes
- external ids

Retention:

- retained long enough for dedupe usefulness

### Class C: Inferred or Recovery Artifacts

Includes:

- recovery handoffs
- inferred summaries

Retention:

- retained with clear provenance labels

### Class D: Raw Transcript-Like Artifacts

Includes:

- raw imports
- event fragments
- temporary parsed cache content

Retention:

- not part of the primary memory index by default
- may have shorter retention windows

## Deletion Policy

Preferred behavior:

- state transitions for normal lifecycle
- hard deletion only for explicit user request, sensitive data removal, or maintenance cleanup

Rules:

- Do not use hard deletion as the normal way to complete work.

## Redaction

If a stored record contains sensitive data:

1. redact or replace sensitive fields when possible
2. preserve non-sensitive structure if still useful
3. record provenance of modification
4. hard delete if redaction is insufficient

## Search Exclusions

Search must not surface records that are:

- explicitly private
- redacted into non-retrievable form
- moved to a non-searchable archival tier

## AGENTS and Privacy

`AGENTS.md` must not instruct Codex to persist secrets or private payloads.
