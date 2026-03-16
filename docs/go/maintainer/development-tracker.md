# codex-mem Go Development Tracker

Last updated: 2026-03-16
Status: active

## Purpose

This file is the working execution tracker for the Go implementation of `codex-mem`.

Use it to:

- track current implementation phase
- record progress across sessions
- keep next steps explicit
- reduce restart cost when a new Codex session begins

Normative references:

- [Spec Index](../../spec/README.md)
- [Implementation Backlog](./implementation-backlog.md)
- [Go Implementation Plan](./implementation-plan.md)
- [Go Development Kickoff](./dev-kickoff.md)

## Working Rules

- Update this file after each meaningful coding session.
- Mark task status explicitly as `todo`, `doing`, `done`, or `blocked`.
- Record blockers and decisions briefly.
- Keep it concise and execution-focused.

## Current Target

Current target: Maintain the completed v1 implementation, expose import auditing through an MCP tool, and keep `modelcontextprotocol/go-sdk` as the only MCP runtime.

## Phase Progress

### Phase 1: Foundation

Status: done

Tasks:

- [x] Initialize Go module and base repository layout
- [x] Choose SQLite driver and migration approach
- [x] Implement database open/init flow
- [x] Create initial schema migrations
- [x] Implement canonical scope types
- [x] Implement canonical session types
- [x] Implement scope resolution inputs and fallback order
- [x] Implement scope consistency validation
- [x] Implement `memory_resolve_scope`
- [x] Implement `memory_start_session`

### Phase 2: Core Continuity Loop

Status: done

Tasks:

- [x] Implement canonical memory note types
- [x] Implement canonical handoff types
- [x] Implement note storage
- [x] Implement handoff storage
- [x] Implement `memory_save_note`
- [x] Implement `memory_save_handoff`
- [x] Implement latest open handoff lookup
- [x] Implement recent high-value note lookup
- [x] Implement startup brief synthesis
- [x] Implement `memory_bootstrap_session`

### Phase 3: Retrieval and Safety

Status: done

Note: Phase 3 feature implementation is complete, but broader conformance and hardening verification still remains under Phase 5.

Tasks:

- [x] Add FTS5 search support
- [x] Implement `memory_search`
- [x] Implement `memory_get_recent`
- [x] Implement `memory_get_note`
- [x] Implement related-project retrieval gating
- [x] Implement result ranking policy
- [x] Implement dedupe behavior for search results
- [x] Implement privacy exclusion checks before indexing
- [x] Implement warning/error taxonomy mapping

### Phase 4: AGENTS Integration

Status: done

Tasks:

- [x] Package AGENTS templates for runtime use
- [x] Implement `memory_install_agents`
- [x] Support safe create-if-missing behavior
- [x] Support append mode
- [x] Support explicit overwrite mode
- [x] Implement project/system placeholder filling

### Phase 5: Conformance and Hardening

Status: done

Note: This phase is the remaining validation layer for already-implemented Phase 1-4 features and should not be confused with missing feature implementation work.

Tasks:

- [x] Add conformance tests for empty-store bootstrap
- [x] Add conformance tests for same-project recovery
- [x] Add conformance tests for related-project retrieval
- [x] Add conformance tests for privacy exclusion
- [x] Add conformance tests for AGENTS safe install
- [x] Add identity conflict tests
- [x] Add migration edge-case tests
- [x] Verify provenance and warning visibility

### Phase 6: MCP SDK Migration

Status: done

Note: This phase replaces the hand-rolled MCP transport/runtime code with `modelcontextprotocol/go-sdk` without changing the domain service contracts or the exposed v1 tool names.

Tasks:

- [x] Add and pin the `modelcontextprotocol/go-sdk` dependency
- [x] Keep `internal/mcp/handlers.go` as the transport-agnostic boundary for domain calls
- [x] Introduce SDK-backed tool registration that preserves the current tool names, descriptions, and JSON schemas
- [x] Replace the custom stdio server loop with the SDK stdio transport
- [x] Replace the custom HTTP handler with the SDK streamable HTTP handler while preserving JSON-response compatibility and enabling session-aware SSE support
- [x] Preserve the existing `/mcp` endpoint and configurable origin allowlist behavior
- [x] Update doctor/smoke-test coverage to validate the SDK-backed stdio and HTTP paths
- [x] Update maintainer/operator docs to describe the new transport behavior and any stream/SSE capability changes
- [x] Remove obsolete custom transport code after parity tests pass

## Current Session Plan

Current session focus:

- Expose the imports plumbing through MCP, intentionally expanding the tool surface, and decide the next materialization behavior after that.

Immediate next tasks:

1. Keep `go test ./...` plus the readiness/smoke checks in the normal regression path for code-bearing changes.
2. Keep the new import MCP workflow aligned with the underlying import audit schema and project-scoped dedupe rules.
3. Expose import audit health through `doctor` so operators can inspect suppression/provenance readiness.
4. Do not reintroduce a parallel in-tree MCP runtime, and only revisit transport internals if a real client compatibility issue appears.

## Decisions Log

### 2026-03-13

- The project will follow the `codex-mem` v1 spec under `docs/spec/`.
- Dynamic continuity data should stay in durable memory and MCP responses, not in `AGENTS.md`.
- `AGENTS.md` should remain cache-friendly and stable.
- The first coding slice should focus on Phase 1 foundation work.
- Go logging should use `log/slog`.
- `slog` output should be written to a rotated log file with retention/compression support.
- Repository-local config should load from the repository `configs/` directory.
- Go config loading should use `viper`, with environment variables overriding file values.
- The SQLite driver is `modernc.org/sqlite`.
- Schema migrations are embedded SQL files under `migrations/` and are applied automatically on startup.
- Scope resolution prefers normalized repo remote identity, then Git root fallback, then local directory fallback.
- Session writes validate the stored `workspace -> project -> system` scope chain before insert.
- Bootstrap retrieval prefers the latest open workspace handoff, then project fallback, then recent notes ordered by scope and importance.
- Recent retrieval reuses the same workspace-first/project-fallback scope ordering and full-record reads return `ERR_RECORD_NOT_FOUND` on misses.
- Search is project-scoped by default, uses FTS5-backed note matching plus SQL handoff matching, and ranks results by scope, state, importance, recency, and text overlap.
- Related-project expansion is now opt-in and currently derives allowed related projects from explicit `related_project_ids` references stored on current-project notes.
- Explicit `private`, `do_not_store`, and `ephemeral_only` intents are rejected before durable note or handoff writes.
- Durable notes and handoffs now carry `searchable` controls; non-searchable records stay readable by id but are excluded from recent/bootstrap/search paths and from the note FTS index.
- Warning/error taxonomy now flows through shared coded-error helpers, retrieval warning codes, and MCP-ready response envelopes that promote warnings to the top level while preserving stable error codes.
- Phase completion in this tracker refers to implementation status; broader conformance, hardening, and verification status remains tracked separately under Phase 5.
- AGENTS templates are now embedded for runtime use, and `memory_install_agents` defaults global installs to `~/.codex/AGENTS.md` plus project installs to `<cwd>/AGENTS.md`.
- AGENTS append mode uses managed comment markers so repeated installs do not duplicate appended template blocks.
- Phase 5 now includes explicit conformance-named Go tests for bootstrap, recovery, retrieval isolation/expansion, privacy exclusion, AGENTS safe install, identity conflict handling, migration continuity, and warning/provenance visibility.
- Runtime config now loads through `viper` from `configs/codex-mem.*`, with environment variables overriding file values and a checked-in example config under `configs/codex-mem.example.json`.
- Go config is now split between user-configurable file/env settings and runtime-derived metadata so fields like `ConfigFileUsed` and `LogDir` are not treated as file-configurable inputs.
- `doctor` should support machine-readable diagnostics output via `--json`, while keeping the existing human-readable report as the default.
- The Go implementation should support both local stdio MCP and remote HTTP MCP transports over the same tool surface.

### 2026-03-14

- The project will migrate MCP transport/runtime wiring from the custom in-tree implementation to `modelcontextprotocol/go-sdk`.
- `internal/mcp/handlers.go` and the existing domain request/response envelopes should remain the adapter boundary so domain services do not depend on SDK structs.
- Migration order should be stdio parity first, then HTTP parity on the SDK streamable HTTP handler, then removal of obsolete custom transport code.
- HTTP migration should preserve the current `/mcp` endpoint contract and configurable origin checks while moving toward SDK-managed streamable HTTP behavior.
- The first migration slice should add a go-sdk-backed server builder and parity tests before changing any CLI transport entrypoint.
- `modelcontextprotocol/go-sdk` v1.4.1 documents stdio as newline-delimited JSON, and the `codex-mcp-server` implementation summary for `openai/codex` also describes stdio as line-delimited JSON messages.
- The repository's existing `Content-Length` stdio framing should now be treated as a legacy in-tree implementation detail rather than the expected Codex-compatible transport shape.
- The remaining shared MCP protocol/tool metadata should live in neutral package files, and the old `internal/mcp/server.go` hand-written runtime entrypoint should be removed rather than kept as a dormant fallback.
- `modelcontextprotocol/go-sdk` is now the only MCP runtime in the repository; future work should extend the SDK-backed path instead of reviving parallel custom transport code.

## Blockers

Current blockers:

- none currently

### 2026-03-14 Session Update

- Completed: Reviewed the current custom MCP transport wiring in `internal/mcp` and `internal/app/run.go`; drafted a phased migration plan to move to `modelcontextprotocol/go-sdk`; retargeted this tracker from release follow-up to MCP transport migration; added `modelcontextprotocol/go-sdk` v1.4.1 to the module; introduced an SDK-backed server builder that registers the existing nine tools without changing handler/domain boundaries; added in-memory parity coverage for `tools/list` and `memory_install_agents`; switched `serve-http` to an SDK-backed streamable HTTP handler in compatibility-focused JSON-response/stateless mode while preserving `/mcp` and origin validation; added SDK HTTP transport tests; passed `go run ./scripts/http-mcp-smoke-test`; and verified the full repository with `go test ./...`.
- In progress: Updating the remaining stdio path from the legacy `Content-Length` framing to the newline-delimited JSON transport shape used by `modelcontextprotocol/go-sdk` and described for `codex-mcp-server`.
- Blockers: the repository still has legacy stdio code and tests that assume `Content-Length` framing.
- Next step: Switch `serve` to SDK stdio, update stdio smoke coverage and related docs, then remove or quarantine the old framed stdio implementation.

### 2026-03-14 Session Handoff

- Known good:
  - `serve-http` is now backed by `modelcontextprotocol/go-sdk` through `internal/mcp/sdk_http.go`.
  - The SDK-backed HTTP path preserves `/mcp`, preserves the existing origin allowlist behavior, and passes `go run ./scripts/http-mcp-smoke-test`.
  - The repository passes `go test ./...` after the SDK introduction and HTTP cutover.
- Known unknown:
  - `serve` stdio is still on the legacy implementation.
  - `modelcontextprotocol/go-sdk` v1.4.1 documents stdio as newline-delimited JSON.
  - The repository's current stdio implementation uses `Content-Length` framing.
  - Source-tree docs and tests still describe the legacy framing in a few places and must be brought in line with the newline-delimited target.
- Recommended restart point:
  - Start by switching `serve` to the SDK stdio transport.
  - Update the stdio smoke test and any CLI-facing docs that still assume `Content-Length`.
  - Remove or isolate the legacy framed stdio implementation once parity checks are green.

### 2026-03-13 Session Update

- Completed: Rewrote the Chinese portion of [prompt-examples.md](../user/prompt-examples.md) to use more user-facing wording for the current repository/project context instead of leaning on raw `scope` terminology; added [how-memory-works.md](../user/how-memory-works.md) as a quick explainer for what mem does, what gets saved, when scope matters, and which commands are for normal users versus operators; linked the new explainer from the Go docs index and the root README.
- In progress: User-facing documentation clarity follow-up.
- Blockers: none new.
- Next step: Decide whether the quick explainer should be expanded into a full onboarding doc with screenshots or client-specific walkthroughs.

### 2026-03-13 Session Update

- Completed: Reorganized the `docs/go` documentation entry points by audience; updated [docs/go/README.md](../README.md) to route users, operators, and maintainers to different docs; clarified [client-examples.md](../operator/client-examples.md), [mcp-integration.md](./mcp-integration.md), [release-readiness.md](../operator/release-readiness.md), [troubleshooting.md](../operator/troubleshooting.md), [dev-kickoff.md](./dev-kickoff.md), [implementation-plan.md](./implementation-plan.md), [how-memory-works.md](../user/how-memory-works.md), and [prompt-examples.md](../user/prompt-examples.md) with explicit audience and use-case framing so source-tree maintainer docs are less likely to be mistaken for end-user setup guides.
- In progress: Go docs information-architecture polish follow-up.
- Blockers: none new.
- Next step: Decide whether the `docs/go` area should stay flat with audience-based navigation or be physically split into `user`, `operator`, and `maintainer` subdirectories.

### 2026-03-13 Session Update

- Completed: Physically split `docs/go` into [user](../user/README.md), [operator](../operator/README.md), and [maintainer](./README.md) directories; moved the existing docs into those directories; added per-audience index files; and updated cross-links plus root README references so the new layout is navigable without relying on one flat directory.
- In progress: Post-split information architecture polish.
- Blockers: none new.
- Next step: Decide whether the user and operator areas need more task-oriented onboarding guides now that the audience boundaries are enforced by path.

### 2026-03-14 Session Update

- Completed: Switched serve to the go-sdk stdio transport using newline-delimited JSON over sdkmcp.IOTransport; rewrote the stdio smoke test and unit coverage to exchange one JSON-RPC message per line instead of Content-Length frames; refreshed maintainer/operator docs to describe the SDK-backed stdio behavior.
- In progress: Removing or isolating the remaining obsolete custom MCP request-routing/runtime helpers that are no longer on the CLI path.
- Blockers: none beyond deciding how far to push the legacy cleanup in this slice.
- Next step: Extract shared tool/schema registration out of internal/mcp/server.go, then delete or quarantine the remaining unused custom transport code.
### 2026-03-14 Session Update

- Completed: Extracted the shared MCP tool catalog away from the legacy runtime wiring so doctor and the SDK-backed server share one definition source; removed the obsolete custom HTTP handler/tests; reduced internal/mcp/http.go to shared HTTP/origin helpers plus server startup; and verified the repository with go test ./..., go run ./scripts/mcp-smoke-test, and go run ./scripts/http-mcp-smoke-test.
- In progress: none.
- Blockers: none.
- Next step: move on from transport migration work unless a regression is found.
### 2026-03-14 Session Update

- Completed: Split the remaining MCP protocol constants/types and tool catalog out of `internal/mcp/server.go` into neutral shared files; deleted the last `server.go` / `server_test.go` naming leftovers from the old hand-written runtime path; kept the SDK-backed stdio and HTTP paths green with `go test ./internal/mcp ./internal/app`.
- In progress: none.
- Blockers: none.
- Next step: pick the next feature or operator-facing enhancement without reopening MCP runtime replacement work.
### 2026-03-14 Session Update

- Completed: Switched the SDK-backed HTTP transport from stateless JSON-only compatibility mode to session-aware streamable HTTP with standalone SSE support on `GET /mcp`; kept JSON POST compatibility for ordinary tool calls; rewrote both source-tree smoke tests to use the go-sdk client transports instead of hand-written JSON-RPC message structs; and re-verified with `go run ./scripts/mcp-smoke-test`, `go run ./scripts/http-mcp-smoke-test`, `go run ./scripts/readiness-check`, and `go test ./...`.
- In progress: none.
- Blockers: none.
- Next step: move back to product-facing work unless a concrete HTTP client compatibility issue appears.
### 2026-03-14 Session Update

- Completed: Fixed the remaining `golangci-lint` dead-code issue after the MCP cleanup; added `serve-http --session-timeout <duration>` so operators can automatically expire idle HTTP MCP sessions; wired that timeout through the SDK streamable HTTP handler; updated maintainer/operator docs for HTTP session headers, SSE behavior, and idle expiry; and kept `golangci-lint`, `go test ./...`, `go run ./scripts/mcp-smoke-test`, `go run ./scripts/http-mcp-smoke-test`, and `go run ./scripts/readiness-check` green.
- In progress: none.
- Blockers: none.
- Next step: choose a product-facing feature slice outside MCP transport and operator hardening.
### 2026-03-16 Session Update

- Completed: Refreshed root `AGENTS.md` to point at the moved `docs/go/maintainer/` planning documents; updated this tracker so the current target/session plan reflects the completed SDK-backed v1 baseline instead of an already-finished transport migration; and synced [release-readiness.md](../operator/release-readiness.md) with the current HTTP session, SSE, and session-timeout behavior.
- In progress: none.
- Blockers: none.
- Next step: choose a small product-facing or operator-facing follow-up slice and keep the regression harness in place for any code-bearing change.
### 2026-03-16 Session Update

- Completed: Added imports MVP plumbing with embedded `005_import_records.sql`, a new `internal/domain/imports` service, SQLite-backed import audit persistence in `internal/db/import_repository.go`, and `doctor`/runtime diagnostics coverage for import audit counts and readiness.
- In progress: none.
- Blockers: none.
- Next step: choose the first caller for the import plumbing, such as an MCP-exposed import flow, a CLI command, or a watcher-side ingestion path that can create durable memory from imported artifacts.
### 2026-03-16 Session Update

- Completed: Exposed the import audit flow as a new MCP tool `memory_save_import` instead of a CLI-specific workflow. The tool uses the existing imports service/repository, carries project-scoped dedupe and privacy suppression forward into the MCP surface, and increases the advertised tool count to ten; smoke/readiness/app tests and docs were updated accordingly.
- In progress: none.
- Blockers: none.
- Next step: decide whether imported artifacts should remain audit-only or whether a follow-up tool/workflow should materialize them into durable notes or handoffs.
## Recommended Next Step

Recommended next implementation slice:

1. Return to feature or product follow-up work outside the completed MCP transport migration.
2. Keep the stdio and HTTP smoke tests in the normal regression path.
3. Only revisit transport internals if a real client compatibility issue appears.
4. Prefer small, documented follow-ups instead of re-introducing parallel transport implementations.

Why this is the best next step now:

- the repository already has both stdio and HTTP MCP entrypoints on the SDK-backed path, so transport replacement is no longer the highest-value area
- the hand-written MCP runtime has been removed, which reduces maintenance cost and ambiguity about which path is authoritative
- the existing handler/domain boundary remains stable, so new work can focus on user-visible behavior instead of transport churn
- the readiness check plus stdio/HTTP smoke tests already provide a practical regression harness for future follow-up changes

## Session Handoff Template

When ending a coding session, append a short update in this format:

```md
### YYYY-MM-DD Session Update

- Completed:
- In progress:
- Blockers:
- Next step:
```

## Suggested Prompt For A New Session

```text
Read docs/go/maintainer/development-tracker.md, docs/go/maintainer/dev-kickoff.md, and docs/go/maintainer/implementation-plan.md, then continue the current Go implementation from the listed next tasks and update the tracker as you make progress.
```



