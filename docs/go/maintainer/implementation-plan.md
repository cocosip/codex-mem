# codex-mem Go Implementation Plan

## Purpose

This document maps the `codex-mem` v1 specification to a Go-oriented implementation plan.

Audience:

- maintainers
- contributors designing or reviewing Go implementation structure

Use this when:

- deciding how the Go implementation should be structured
- reviewing package boundaries, service responsibilities, and delivery order
- checking whether a code change matches the intended architecture

Do not use this for:

- first-time user onboarding
- runtime troubleshooting
- registering a packaged binary with a client

It is not the normative product spec.
Instead, it describes how a Go implementation can realize the required behavior in a maintainable and testable way.

Normative references:

- [Spec Index](../../spec/README.md)
- [Implementation Backlog](./implementation-backlog.md)

## Implementation Goals

The Go implementation should:

- satisfy the `codex-mem` v1 baseline
- remain local-first
- work well as a single-binary local service
- support SQLite-backed durable memory
- expose the required MCP tool surface
- preserve room for future watcher/import extensions without forcing them into v1

## High-Level Architecture

Recommended shape:

1. CLI entrypoint
2. configuration loader
3. storage layer
4. domain services
5. MCP adapter layer
6. optional background jobs

Recommended runtime modes:

- `serve` for the local MCP server
- `migrate` for schema initialization and upgrades
- `doctor` for local diagnostics
- `agents install` or equivalent for AGENTS template installation

Recommended configuration convention:

- load project-level configuration from the repository `configs/` directory
- use `viper` as the Go configuration library for file loading and environment overrides
- keep precedence as `defaults < config file < environment variables`

## Recommended Repository Layout

Suggested layout:

```text
cmd/
  codex-mem/
    main.go

internal/
  app/
  config/
  db/
  domain/
    scope/
    session/
    memory/
    handoff/
    retrieval/
    agents/
    imports/
  mcp/
  observability/
  privacy/
  identity/

migrations/
templates/
docs/
```

### Why This Layout

- `cmd/` keeps the binary entrypoint thin
- `internal/` keeps application logic encapsulated
- `domain/` separates core behavior from transport and storage
- `migrations/` makes schema evolution explicit
- `templates/` already fits AGENTS template storage

## Package Responsibilities

### `internal/app`

Purpose:

- application wiring
- dependency construction
- lifecycle management

Responsibilities:

- load config
- open database
- initialize services
- register MCP tools
- start server mode
- expose reusable app-level workflows such as import ingestion for future in-process integrations

### `internal/config`

Purpose:

- configuration loading and validation

Responsibilities:

- product defaults
- config file parsing
- env var overrides
- runtime path resolution
- policy precedence helpers

Recommended implementation details:

- treat `configs/` as the default home for repository-local configuration
- let `viper` load the config file and bind environment variables into the same config model
- keep file parsing format-flexible through `viper`, even if the first checked-in example uses JSON

Suggested outputs:

- storage path
- log level
- default bootstrap limits
- privacy settings
- AGENTS install defaults

### `internal/db`

Purpose:

- database connection and migration handling

Responsibilities:

- open SQLite connection
- configure pragmas
- run migrations
- expose transaction helpers

Recommended SQLite concerns:

- enable foreign keys
- configure WAL mode if appropriate
- use busy timeout
- support FTS5

### `internal/domain/scope`

Purpose:

- scope resolution and scope consistency validation

Responsibilities:

- resolve `system/project/workspace`
- apply identity evidence priority
- validate scope-chain consistency
- create missing scope records when policy allows

### `internal/domain/session`

Purpose:

- session lifecycle management

Responsibilities:

- create sessions
- update session status
- enforce no revival of ended sessions
- query recent sessions when needed

### `internal/domain/memory`

Purpose:

- memory note write and read behavior

Responsibilities:

- validate note input
- normalize tags and file paths
- dedupe notes conservatively
- store and retrieve notes

### `internal/domain/handoff`

Purpose:

- handoff write and continuity behavior

Responsibilities:

- validate handoff payloads
- enforce `next_steps`
- detect same-task open handoff conflicts
- store and retrieve handoffs

### `internal/domain/retrieval`

Purpose:

- retrieval orchestration and ranking

Responsibilities:

- bootstrap retrieval sequence
- search retrieval
- recent retrieval
- related-project expansion
- ranking and dedupe in result sets
- startup brief synthesis

### `internal/domain/agents`

Purpose:

- AGENTS template installation and file update policy

Responsibilities:

- choose target paths
- load templates
- fill placeholders
- apply create/append/overwrite modes
- report changed and skipped files

### `internal/domain/imports`

Purpose:

- provenance and dedupe support for imported artifacts

Responsibilities:

- store import records
- track external ids and payload hashes
- suppress duplicate imports

This package may be implemented lightly in v1 and expanded later.

### `internal/mcp`

Purpose:

- transport adapter for MCP tool registration and request handling

Responsibilities:

- define MCP tool handlers
- convert MCP input to domain requests
- map domain results to MCP responses
- preserve warning and error taxonomy

### `internal/observability`

Purpose:

- logging, warnings, audit helpers, and optional debug traces

Responsibilities:

- structured logs
- provenance helpers
- optional retrieval trace assembly

### `internal/privacy`

Purpose:

- privacy and exclusion policy enforcement

Responsibilities:

- do-not-store checks
- exclusion markers
- optional redaction helpers
- search filtering for excluded records

### `internal/identity`

Purpose:

- canonical identity normalization and repository fingerprint logic

Responsibilities:

- normalize repo remotes
- compare identity evidence
- detect rename/move continuity
- warn on conflicts

## Recommended Storage Mapping

Recommended persistent tables:

- `systems`
- `projects`
- `workspaces`
- `sessions`
- `memory_items`
- `handoffs`
- `project_relations`
- `imports`

Recommended FTS:

- `memory_items_fts`
- optional later `handoffs_fts`

### Go Storage Strategy

Recommended split:

- repository interfaces in domain packages
- concrete SQLite repositories in `internal/db` or a `internal/store/sqlite` package

Suggested style:

- domain services depend on interfaces
- SQLite implementation satisfies those interfaces

This keeps the core behavior portable and testable.

## Canonical Go Type Strategy

Recommended approach:

- define canonical request/response and entity structs close to the domain layer
- avoid transport-specific structs leaking into the core services

Suggested type families:

- `Scope`
- `Session`
- `MemoryNote`
- `Handoff`
- `StartupBrief`
- `SearchResult`
- tool-specific request/response structs

Recommended enum handling:

- use typed string aliases for statuses and kinds
- validate at boundaries

Examples:

- `type SessionStatus string`
- `type NoteType string`
- `type HandoffStatus string`

## Service Layer Design

Recommended pattern:

- one service per major behavior area
- orchestration service for bootstrap and retrieval

Suggested services:

- `ScopeService`
- `SessionService`
- `MemoryService`
- `HandoffService`
- `RetrievalService`
- `AgentsService`
- `ImportService`

### Why Services Instead Of Direct DB Calls

- keeps business rules centralized
- makes MCP handlers thin
- makes testing easier
- preserves portability if storage changes later

## MCP Tool Implementation Mapping

Recommended Go handler mapping:

- `memory_bootstrap_session` -> `RetrievalService.BootstrapSession`
- `memory_resolve_scope` -> `ScopeService.Resolve`
- `memory_start_session` -> `SessionService.Start`
- `memory_save_note` -> `MemoryService.SaveNote`
- `memory_save_handoff` -> `HandoffService.SaveHandoff`
- `memory_search` -> `RetrievalService.Search`
- `memory_get_recent` -> `RetrievalService.GetRecent`
- `memory_get_note` -> `RetrievalService.GetRecordByID`
- `memory_install_agents` -> `AgentsService.Install`

## Bootstrap Flow In Go

Recommended internal sequence:

1. MCP handler parses request
2. `ScopeService.Resolve` resolves or creates scope
3. `SessionService.Start` creates new session
4. `RetrievalService.BootstrapSession`:
   - get latest open workspace handoff
   - fallback to project handoff
   - get workspace notes
   - fallback to project notes
   - optionally fetch related-project notes
   - synthesize startup brief
5. MCP handler returns canonical response

Recommended implementation detail:

- keep bootstrap orchestration inside one service call so transaction and warning flow stay coherent

## Search and Ranking In Go

Recommended implementation split:

- SQL/FTS retrieves candidate rows
- Go ranking logic applies scope/state/importance/recency weighting
- Go layer applies dedupe and final result shaping

Why:

- SQL is efficient at candidate filtering
- Go is better for policy-heavy ranking logic that may evolve

Recommended ranking pipeline:

1. candidate fetch by scope and text filter
2. state filtering
3. related-project policy gate
4. score computation
5. dedupe collapse
6. result shaping with provenance labels

## Privacy Enforcement In Go

Recommended enforcement points:

### Before durable writes

- validate explicit private or do-not-store input
- reject or redact sensitive content before persistence

### Before indexing

- ensure excluded records do not enter FTS

### Before retrieval output

- exclude records not meant to be searchable
- label inferred or imported records clearly

Recommended design:

- centralize this in `privacy` helpers used by write and search services

## AGENTS Installation In Go

Recommended implementation shape:

- read template files from `templates/`
- resolve target paths
- inspect existing file state
- apply selected mode:
  - create if missing
  - append block
  - overwrite template
- return structured result

Recommended file handling behavior:

- default to non-destructive
- preserve unresolved placeholders
- do not attempt complex markdown merging in v1

## Observability In Go

Recommended baseline:

- structured logger
- request-scoped warning collection
- audit fields for writes
- optional debug traces for retrieval

Recommended logging fields:

- tool name
- session id
- project id
- workspace id
- result counts
- warning codes
- error codes

Recommended caution:

- do not log private payload contents

## Testing Strategy

Recommended test layers:

### Unit tests

Focus:

- scope resolution logic
- state validation
- privacy filtering
- ranking and dedupe helpers

### Service tests

Focus:

- bootstrap flow
- note save flow
- handoff save flow
- AGENTS install modes

### Repository/storage tests

Focus:

- SQLite persistence
- FTS search behavior
- migration correctness
- foreign key and scope consistency behavior

### Conformance tests

Focus:

- scenarios from [conformance-matrix.md](../../spec/appendices/conformance-matrix.md)

Recommended approach:

- keep language-neutral scenario names
- implement them as Go test cases

## Suggested Delivery Order In Go

Recommended order:

1. config + db initialization
2. scope + identity services
3. session service
4. note and handoff persistence
5. bootstrap flow
6. search and recent retrieval
7. privacy enforcement
8. AGENTS installer
9. conformance test coverage

This order mirrors the language-neutral backlog while fitting Go service construction well.

## Recommended V1 Go Milestones

### Milestone 1: foundation ready

- database opens and migrates
- scope can be resolved
- sessions can be created
- notes and handoffs can be stored

### Milestone 2: continuity loop ready

- bootstrap works
- startup brief is produced
- later session can recover prior continuity

### Milestone 3: safe retrieval ready

- search works
- recent retrieval works
- project isolation is enforced
- privacy exclusions are enforced

### Milestone 4: workflow integration ready

- AGENTS installation works
- warnings and provenance are visible
- conformance matrix passes

## Future-Proofing Recommendations

To preserve room for v2+ work:

- keep import support behind clear interfaces
- keep ranking logic outside raw SQL where possible
- keep provenance explicit in the core data model
- avoid coupling all behavior directly to MCP transport structs
- avoid assuming transcript capture exists

## Final Recommendation

If Go is chosen, implement `codex-mem` as:

- a single local binary
- with SQLite as the durable store
- domain services at the core
- MCP handlers as adapters
- spec-driven behavior and conformance tests as the quality gate
