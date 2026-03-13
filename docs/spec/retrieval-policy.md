# Retrieval Policy

## Core Principle

Prefer precise, recent, high-value, scope-local memory first.

## Scope Expansion Order

Required retrieval order:

1. current workspace
2. current project
3. explicitly allowed related projects in the same system
4. other systems only when explicitly requested

Rules:

- Do not jump directly to unrelated projects by default.

## Bootstrap Retrieval Order

When starting a new session:

1. latest open handoff in current workspace
2. latest open handoff in current project
3. recent high-value notes in current workspace
4. recent high-value notes in current project
5. related-project memory only if explicitly enabled

## Ranking Priorities

Recommended ranking order:

- scope
- state
- importance
- recency
- text overlap

This means correct scope outranks keyword coincidence.

## State Ranking

For notes:

1. `active`
2. `resolved`
3. `superseded`

For handoffs:

1. `open`
2. `completed`
3. `abandoned`

## Importance Scale

- `5`: critical future context
- `4`: important project-level context
- `3`: meaningful task-level context
- `2`: useful but narrow
- `1`: minor detail

## Handoff vs Note Preference

For bootstrap:

- handoffs outrank notes

For search:

- continuity queries favor handoffs
- technical memory queries favor notes

## Query Intent Categories

Recommended intent categories:

- `continuation`
- `decision`
- `bugfix`
- `discovery`
- `recent_activity`
- `general`

## Related Project Retrieval

Related-project retrieval requires at least one of:

- explicit tool parameter
- project policy allowing it
- relation type suggesting relevance

Safeguards:

- limit related-project result count
- label source project and relation
- do not outrank strong current-project results by default

## Deduplication in Retrieval

Rules:

- prefer newer notes over superseded duplicates
- prefer the latest open handoff for the same unfinished task
- prefer explicit notes over imported artifacts

## Output Labeling

Every result should include:

- source scope
- record kind
- record state
- created time

Cross-project results must also include:

- source project
- relation type when available
