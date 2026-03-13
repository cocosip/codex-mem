# Warning and Error Taxonomy

## Purpose

This appendix defines a compact set of recommended warning and error identifiers for `codex-mem` v1.

The goal is consistency across implementations and easier debugging during conformance testing.

## Conventions

- Warnings indicate degraded but usable outcomes.
- Errors indicate that the intended operation could not complete safely or meaningfully.
- Identifiers are stable categories, not user-facing prose.

## Warning Codes

### `WARN_NO_PRIOR_HANDOFF`

Meaning:

- No open handoff was found for the current workspace or project during bootstrap.

### `WARN_NO_PRIOR_NOTES`

Meaning:

- No relevant high-value notes were found for the current scope.

### `WARN_SCOPE_AMBIGUOUS`

Meaning:

- Scope resolution succeeded, but only with partial confidence or fallback logic.

### `WARN_SCOPE_FALLBACK_USED`

Meaning:

- Weaker identity evidence was used, such as local path fallback.

### `WARN_RELATED_PROJECTS_SKIPPED`

Meaning:

- Related-project retrieval was requested or possible, but was skipped due to policy or missing relations.

### `WARN_RELATED_PROJECTS_EMPTY`

Meaning:

- Related-project retrieval was attempted but returned no results.

### `WARN_RECOVERY_HANDOFF_USED`

Meaning:

- Bootstrap used a recovery-generated handoff because no stronger continuation source was available.

### `WARN_DEDUPE_APPLIED`

Meaning:

- A write operation matched an existing record and deduplication logic was applied.

### `WARN_RELATED_REFERENCE_IGNORED`

Meaning:

- Optional related project ids or note references were provided, but some could not be resolved.

### `WARN_HANDOFF_SPARSE`

Meaning:

- A handoff was accepted, but missing optional fields may reduce future recovery quality.

### `WARN_EXISTING_AGENTS_SKIPPED`

Meaning:

- An AGENTS target file already existed and safe mode skipped modification.

### `WARN_PLACEHOLDERS_UNRESOLVED`

Meaning:

- AGENTS installation succeeded, but some placeholders remain for manual completion.

### `WARN_IMPORT_SUPPRESSED`

Meaning:

- An imported artifact was ignored due to privacy, dedupe, or stronger explicit memory already existing.

## Error Codes

### `ERR_INVALID_INPUT`

Meaning:

- Required input is missing, malformed, or semantically invalid.

### `ERR_INVALID_SCOPE`

Meaning:

- The provided or resolved scope is incomplete, inconsistent, or unsafe to use.

### `ERR_SCOPE_CONFLICT`

Meaning:

- Scope evidence or stored mappings conflict and safe identity cannot be maintained.

### `ERR_SESSION_NOT_FOUND`

Meaning:

- A referenced session does not exist.

### `ERR_RECORD_NOT_FOUND`

Meaning:

- A requested note or handoff id does not exist.

### `ERR_INVALID_STATE`

Meaning:

- A requested state or state transition is invalid for the operation.

### `ERR_STORAGE_UNAVAILABLE`

Meaning:

- The durable storage backend is unavailable or failed the operation.

### `ERR_WRITE_FAILED`

Meaning:

- A write operation could not be completed.

### `ERR_READ_FAILED`

Meaning:

- A read operation could not be completed.

### `ERR_AGENTS_WRITE_DENIED`

Meaning:

- AGENTS installation failed because the target path could not be written safely.

### `ERR_INVALID_TARGET`

Meaning:

- An invalid target or mode was supplied to `memory_install_agents`.

## Recommended Usage

- Return user-facing explanatory text alongside these identifiers.
- Preserve identifiers in debug or audit output where possible.
- Use one or more warnings rather than an error when the system can still proceed safely.
