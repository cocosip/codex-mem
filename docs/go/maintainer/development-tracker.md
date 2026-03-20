# codex-mem Go Development Tracker

Last updated: 2026-03-20
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

Current target: Maintain the completed v1 implementation and named v1 conformance coverage, keep `modelcontextprotocol/go-sdk` as the only MCP runtime, and choose small post-v1 feature or operator-facing follow-up slices without reopening baseline transport or continuity work.

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

- Keep the imports workflow aligned across audit-only and imported-note materialization paths, including explicit-memory precedence.

Immediate next tasks:

1. Keep `go test ./...` plus the readiness/smoke checks in the normal regression path for code-bearing changes.
2. Keep the import MCP workflows aligned with the underlying import audit schema and project-scoped dedupe/precedence rules.
3. Preserve retrieval preference for explicit memory when imported artifacts overlap with the same project-level durable note.
4. Do not reintroduce a parallel in-tree MCP runtime, and only revisit transport internals if a real client compatibility issue appears.

## Decisions Log

### 2026-03-20 Session Update

- Completed: Continued the exploratory v2 documentation set by adding a maintainer-facing semantic-interfaces draft plus a migration-sequencing draft. The interface doc maps the v2 design into the current Go package boundaries, proposes additive retrieval, memory-domain, semantic-index, and runtime-resurfacing interfaces, and keeps `retrieval.Service` on a nil-safe lexical-first path. The migration doc then turns the same design into a rollout order: interfaces first, additive note-embedding metadata migration next, sidecar/backfill tooling after that, notes-only hybrid retrieval later, and implicit resurfacing only after hybrid retrieval itself is stable.
- In progress: none.
- Blockers: none.
- Next step: decide whether to keep extending docs with an implementation-spike plan, or start the first Go slice for additive metadata fields and nil-safe semantic interfaces.

### 2026-03-20 Session Update

- Completed: Continued the exploratory v2 documentation set with two more design artifacts: a v2 conformance-scenarios draft and an operator-facing embedding backfill/health draft. The conformance doc defines the minimum scenario matrix for lexical-only baseline behavior, happy-path hybrid retrieval, degraded semantic states, runtime resurfacing controls, privacy/lifecycle guardrails, and sidecar rebuild recovery. The operator doc turns the storage draft into a rollout sequence, health-state model, expected summary fields, and repair-oriented workflows, while the roadmap and readmes now link both drafts.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next v2 design pass should sketch concrete Go interfaces and migration ordering, or stop design expansion and begin implementation spikes behind config gates.

### 2026-03-20 Session Update

- Completed: Continued the exploratory v2 documentation set by adding a dedicated embedding-storage and backfill draft. The new doc compares direct-in-SQLite, local sidecar, and external-backend shapes; recommends SQLite metadata plus a pluggable local sidecar index for the first rollout; defines minimal embedding lifecycle metadata, explicit one-shot backfill flow, repair semantics, and degraded retrieval rules; and links that design back into the v2 outline, roadmap, and doc indexes.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next v2 draft follow-up should add operator-facing backfill/health guidance or a conformance-scenarios draft for lexical-only, hybrid, and degraded retrieval modes.

### 2026-03-20 Session Update

- Completed: Continued the exploratory v2 documentation set by adding a dedicated runtime resurfacing draft plus a companion v2 config draft. The new docs define a suggested consult-scoring pipeline, prototype confidence thresholds, session-local suppression-cache cooling rules, a shaped working-context payload for implicit memory injection, and a notes-first early-v2 boundary that keeps handoffs lexical-only at first. The v2 outline, roadmap, and doc indexes now link to the new drafts so the design direction is easier to re-enter.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next v2 draft follow-up should focus on embedding storage/backfill design, conformance scenarios for hybrid retrieval, or pause until implementation work begins.

### 2026-03-19 Session Update

- Completed: Added `--fail-on-partial` to `ingest-imports` so operators can keep `--continue-on-error` partial-success behavior, failed-output exports, and failed-manifest exports while still receiving a non-zero exit when any line in the batch failed. The ingest report now surfaces `fail_on_partial`, CLI coverage verifies that the partial report is still written before the command exits with an error, and the reusable `App.IngestImports` path now honors the same flag so embedded callers can receive the same report-plus-error behavior instead of only the CLI getting that control. Operator docs plus the README describe the automation-oriented workflow.
- In progress: none.
- Blockers: none.
- Next step: decide whether the next import/follow ergonomics slice should extend similar automation-gating controls into more import workflows, or move to another bounded operator-facing improvement.

- Completed: Added `--example` filtering to `list-command-examples`, including repeated and comma-separated example-name selectors. Text and JSON output now share the same embedded-manifest filter pipeline across command, example, format, and tag filters, runtime coverage rejects unknown example names and missing values, and operator-facing docs now show how to jump straight to one or more named fixtures.
- In progress: none.
- Blockers: none.
- Next step: decide whether the example catalog now has enough discovery power, or whether the next bounded operator-facing slice should move back to import/follow workflow ergonomics instead of further catalog expansion.

- Completed: Extended multi-input `follow-imports` aggregate reports with a top-level `batch_summary` rollup so operators can see aggregate attempted, processed, failed, materialized, suppressed, dedupe, suppression-reason, and warning-by-code counts without manually summing each nested input batch. Added a companion `retry_summary` block that totals failed-output and failed-manifest activity across the consumed inputs in the same pass and now also surfaces aggregate retry artifact paths, added a compact `batch_error_summary` block so the aggregate report also shows how many inputs surfaced follow-level `batch_error` payloads and which error codes appeared, added a `state_summary` block that rolls up truncation and checkpoint-reset visibility plus reset-reason counts across inputs, added a `pending_summary` block that counts inputs with trailing pending bytes and highlights the largest pending backlog, and then added a compact `watch_summary` block that rolls up watch-event kind counts plus mode-transition pairs for the current emitted watch event batch. Idle aggregate reports with pending-summary-only or state-summary-only changes now still emit so backlog and reset/truncation events are not silently hidden. JSON fixtures, format coverage, and operator docs now describe all six summary blocks alongside the existing per-input reports.
- In progress: none.
- Blockers: none.
- Next step: decide whether this aggregate reporting slice is now complete, or whether the better next maintenance step is to consolidate the aggregate-report fixture/test setup so future shape changes touch fewer duplicated literals.

### 2026-03-18 Session Update

- Completed: Tightened command-example manifest parsing to reject duplicate fields inside one entry instead of silently letting a later value overwrite an earlier one. Parser tests now cover duplicate-key rejection alongside missing metadata and malformed tag lists, which makes hand-edited catalog drift easier to catch before it reaches the embedded binary.
- In progress: none.
- Blockers: none.
- Next step: keep chipping away at ambiguous or silent-fallback cases in the operator catalog path rather than adding more surface area.

### 2026-03-18 Session Update

- Completed: Tightened command-example manifest parsing so each entry must now carry non-empty `tags` and `summary` metadata, not just the required command/name/format/path fields. Parser coverage now explicitly rejects missing metadata alongside malformed or empty tag lists, so the embedded example catalog contract is enforced at parse time instead of only by higher-level tests.
- In progress: none.
- Blockers: none.
- Next step: keep this operator-catalog line focused on correctness and low-maintenance guarantees unless a clearly necessary usability gap shows up.

### 2026-03-18 Session Update

- Completed: Hardened the embedded command-example catalog tests so every JSON-visible example entry must carry non-empty `tags` and `summary` metadata, and tightened manifest tag parsing to trim whitespace, dedupe repeated tags, and reject empty tag slots in hand-edited `tags=` values.
- In progress: none.
- Blockers: none.
- Next step: keep future operator-catalog changes focused on data quality and maintenance safety unless a clearly useful discovery feature is needed.

### 2026-03-18 Session Update

- Completed: Added `--tag` filtering plus embedded `tags` and `summary` metadata to `list-command-examples`. The manifest now classifies fixtures with stable labels such as `audit-only`, `target-profile`, `filtered`, `dry-run`, and input-shape tags, while also carrying a short human-readable purpose string for each example. Text and JSON output share the same command/format/tag filter pipeline, and sync/runtime coverage verifies the richer manifest format plus tag-based filtering.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current tags-plus-summary catalog is enough, or whether operators would benefit from one more discovery aid such as sorting or another filter dimension.

### 2026-03-18 Session Update

- Completed: Recorded a maintainer workflow constraint that `GOCACHE` must not point into the repository workspace. If a local cache override is needed for Go verification on this machine, it should use a temporary directory outside the project tree so the working tree does not accumulate cache artifacts.
- In progress: none.
- Blockers: none.
- Next step: continue the current operator-facing extension work without reintroducing repo-local Go cache state.

### 2026-03-18 Session Update

- Completed: Added `--format` filtering to `list-command-examples`, including repeated and comma-separated format selectors. Text and JSON output now share the same embedded-manifest filtering path across command and format filters, runtime coverage verifies mixed command+format filtering plus argument rejection, and operator docs now show how to browse only text or JSON fixtures.
- In progress: none.
- Blockers: none.
- Next step: decide whether operators now need richer catalog metadata such as fixture purpose/category tags, or whether command plus format filtering is enough.

### 2026-03-18 Session Update

- Completed: Added `--command` filtering to `list-command-examples`, including repeated and comma-separated command selectors. Text and JSON output now share the same parsed embedded manifest path, so filtered operator lookup stays consistent across both formats without introducing another source of truth.
- In progress: none.
- Blockers: none.
- Next step: decide whether list-command-examples now has enough discovery power, or whether operators also need format-level filtering such as `--format json`.

### 2026-03-18 Session Update

- Completed: Extended `list-command-examples` with a `--json` mode that parses the embedded text manifest into a stable structured report. Runtime coverage now verifies both text and JSON output, so the packaged-binary example catalog can serve human lookup and simple automation without introducing a second checked-in source of truth.
- In progress: none.
- Blockers: none.
- Next step: decide whether operators need filtering options such as `--command follow-imports`, or whether keeping the catalog intentionally small and whole is the better interface.

### 2026-03-18 Session Update

- Completed: Added a packaged-binary `list-command-examples` command that prints the embedded import/follow example manifest. The manifest remains checked in under `internal/app/testdata`, runtime code now embeds it for operator discovery, and CLI coverage verifies both the happy path and argument rejection.
- In progress: none.
- Blockers: none.
- Next step: decide whether this manifest should eventually gain a machine-readable JSON form, or whether the current text format is the better stable operator-facing surface.

### 2026-03-18 Session Update

- Completed: Added a checked-in import/follow command example manifest under `internal/app/testdata` together with generator/sync tests and an env-gated refresh helper. The operator import-ingestion guide now points maintainers at one catalog file instead of repeating every sample-output path inline, which should reduce doc drift as the fixture set grows.
- In progress: none.
- Blockers: none.
- Next step: decide whether the example catalog should stay testdata-backed only, or whether operators would benefit from a small packaged-binary command that prints the same manifest on demand.

### 2026-03-18 Session Update

- Completed: Added checked-in `follow-imports` audit-only sample outputs under `internal/app/testdata` together with sync and env-gated refresh tests. The new fixtures cover both single-input text output and multi-input aggregate JSON output, so nested batch counters plus suppression-reason summaries now have checked-in examples beyond ordinary assertions. The operator import-ingestion guide and README now point at those follow-mode examples and document the refresh helper.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current import/follow sample catalog is complete enough, or whether maintainers would benefit from a dedicated example-manifest helper if this operator-facing fixture set keeps growing.

### 2026-03-18 Session Update

- Completed: Added checked-in `ingest-imports` audit-only sample outputs under `internal/app/testdata` together with sync and env-gated refresh tests. The repository now carries one text fixture that shows aggregate suppression-reason counts plus `would_materialize`, and one JSON fixture that shows audit-only linking plus the `import_policy` suppression bucket. The root README and operator import-ingestion guide now point operators and maintainers at those richer examples.
- In progress: none.
- Blockers: none.
- Next step: decide whether `follow-imports` should get its own checked-in audit-only sample outputs, or whether the current nested batch assertions plus ingest-only fixtures are enough for now.

### 2026-03-18 Session Update

- Completed: Added aggregate `suppression_reasons` counts to import ingestion reports so audit/materialization summaries can show why events were suppressed without scanning every per-line result. JSON output now exposes a normalized reason-count map, text output flattens the same counts as `suppression_reason_<reason>=<count>`, and the fallback `import_policy` bucket keeps generic duplicate/import-policy suppression visible even when no explicit reason string is attached. Follow-mode text output forwards the nested batch counts, and app/report coverage now verifies both privacy and explicit-memory suppression buckets.
- In progress: none.
- Blockers: none.
- Next step: decide whether operators now want the root README to advertise the richer audit report fields, or whether keeping that detail in the operator guide is the better boundary.

### 2026-03-18 Session Update

- Completed: Added richer audit-only import summary counters to the ingest/follow reporting path. `ingest-imports` JSON reports now surface `would_materialize` for unsuppressed artifacts that would have created a new imported note and `linked_existing_note` for unsuppressed artifacts that only linked an existing imported note, while follow-mode text output forwards the same nested batch counters. App and CLI coverage now verify both the fresh audit-only path and the duplicate-link audit-only path.
- In progress: none.
- Blockers: none.
- Next step: decide whether audit-oriented summaries also need top-level suppression-reason buckets, or whether `suppressed` plus per-result `suppression_reason` remains the better balance.

### 2026-03-18 Session Update

- Completed: Added `--audit-only` to `ingest-imports` and `follow-imports` so operators can store only import-audit provenance while reusing the same scope/session resolution, retry exports, checkpoint handling, privacy suppression, imported-note dedupe, and explicit-memory precedence as the materializing path. The import domain service now exposes an audit-only evaluation path, app and CLI reports surface `audit_only` plus per-result `suppression_reason`, and domain/app/run tests cover the new mode. Operator docs and the root README now describe when to use the audit-only path.
- In progress: none.
- Blockers: none.
- Next step: decide whether the audit-only summaries now need richer top-level counters such as distinguishing newly auditable artifacts from audit records that linked an existing imported note, or whether the current `audit_only` plus per-result metadata is sufficient.


## Recommended Next Step

Recommended next implementation slice:

1. Treat the v1 baseline plus conformance matrix as complete and move to post-v1 feature or operator follow-up work.
2. Keep the stdio and HTTP smoke tests plus the named conformance cases in the normal regression path.
3. Only revisit transport internals or baseline continuity semantics if a real compatibility or conformance regression appears.
4. Prefer small, documented follow-ups such as readiness/operator example output, import-operator ergonomics, or another clearly bounded product-facing enhancement.

Why this is the best next step now:

- the repository already has both stdio and HTTP MCP entrypoints on the SDK-backed path, so transport replacement is no longer the highest-value area
- the hand-written MCP runtime has been removed, which reduces maintenance cost and ambiguity about which path is authoritative
- the v1 conformance matrix now has direct named test coverage across retrieval, DB, handoff, imports, and agents layers, so baseline verification is no longer the main missing investment
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
