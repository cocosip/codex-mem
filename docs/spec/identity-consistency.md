# Identity and Consistency

## Core Principle

Human-readable names are useful, but not sufficient as primary identity.

Stable identity should use canonical project evidence where possible.

## Identity Layers

- `system_id`
- `project_id`
- `workspace_id`
- `session_id`

## Project Identity Evidence Priority

Recommended order:

1. explicit configured identity
2. canonical normalized repository remote identity
3. normalized repository fingerprint
4. canonical root fallback
5. local directory fallback

Rules:

- Prefer stability over convenience.
- Warn when using weak identity evidence.

## System Identity

`system_id` should usually come from explicit metadata or installation-time configuration.

Rule:

- Do not aggressively infer system identity from folder names alone.

## Workspace Identity

`workspace_id` represents one concrete local working copy.

Rules:

- Two local clones of the same repo usually have different `workspace_id` values.
- Two worktrees of the same project are usually different workspaces under the same project.
- Branch changes do not create a new project.

## Session Identity

Rules:

- Every new session gets a new `session_id`.
- `session_id` must never be reused.

## Rename and Move Semantics

### Project Rename

If canonical identity remains the same:

- keep `project_id`
- update display metadata only

### Workspace Move

If evidence supports continuity:

- preserve `workspace_id`

Otherwise:

- create a new workspace with warning or explicit link

### Remote Rename or Transfer

If the logical project is clearly the same:

- preserve `project_id`
- update remote metadata

## Split and Merge

Rules:

- Historical identity changes must be explicit.
- Do not silently merge old project histories into new identities.

## Deduplication

Goals:

- reduce noisy duplication
- preserve meaningful history

Domains:

- import dedupe
- explicit note dedupe
- handoff dedupe

Rules:

- Prefer idempotence for repeated imports.
- Do not collapse clearly different records just because titles are similar.
- Dedupe keys are helpers, not primary identity.

## Scope Consistency

Every stored note and handoff must consistently reference:

- one session
- one workspace
- one project
- one system

Rules:

- Reject cross-scope inconsistencies on write.
- Warn or error on identity conflicts instead of silently merging.
