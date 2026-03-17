# codex-mem Go Development Tracker

Last updated: 2026-03-17
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

### 2026-03-17 Session Update

- Completed: Added `--summary-only` to `cleanup-follow-imports` and `audit-follow-imports` for automation-friendly compact reports. The hygiene commands still compute the same target counts, fail-if-matched behavior, and follow-health status, but they now omit detailed checkpoint and retry-artifact path lists from text and JSON output when summary-only mode is requested. Parser coverage plus end-to-end run tests now verify both flag parsing and the suppression of per-path details, and the operator docs/README now show when to use the compact mode for scheduled hygiene runs.
- In progress: none.
- Blockers: none.
- Next step: decide whether follow/import operators would benefit more from another report-shaping control such as capped path samples, or whether the next slice should move back to a new ingestion/follow capability.

### 2026-03-17 Session Update

- Completed: Added a fifth follow-import hygiene target preset: `--target-profile artifacts`. It expands to checkpoint-sidecar plus retry-artifact targets without touching follow-health snapshots, which fills the gap between `all` and the narrower `state` / `retry` profiles. Parser and end-to-end CLI tests now verify that `artifacts` selects state plus retry work while leaving follow-health out of both cleanup and audit reports, and the operator docs/README now advertise the new preset.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current target-profile catalog is now complete enough, or whether future operator feedback should drive any further preset additions.

### 2026-03-17 Session Update

- Completed: Added checked-in sample outputs that explicitly exercise the new follow-import hygiene target presets. The repository now includes a `cleanup-follow-imports` text fixture for `--target-profile all` plus an `audit-follow-imports` JSON fixture for `--target-profile retry`, and the example-list tests plus operator docs now advertise those preset-focused fixtures alongside the older age/pattern examples.
- In progress: none.
- Blockers: none.
- Next step: decide whether the preset catalog is complete enough now, or whether operators would benefit more from another profile such as "artifacts-only" that excludes follow-health.

### 2026-03-17 Session Update

- Completed: Added `--target-profile` presets to `cleanup-follow-imports` and `audit-follow-imports` with initial `all`, `state`, `retry`, and `health` values. The profile layer expands to the existing prune/check booleans instead of introducing a second hygiene engine, so explicit `--prune-*` and `--check-*` flags still work and can be combined with a profile. App and parser coverage now verifies target-profile parsing plus end-to-end cleanup/audit runs, and the import-ingestion/README docs now show the new shorthand for common hygiene slices.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current target-profile catalog is enough, or whether follow/import operators would benefit more from checked-in sample outputs that exercise the new presets explicitly.

### 2026-03-17 Session Update

- Completed: Fixed Linux CI drift in unit tests and pinned the workflow lint toolchain to the same `golangci-lint` version used locally. The follow-import and readiness example-list tests now build expected fixture paths with `filepath.Join(...)` instead of hard-coding Windows separators, the duplicate follow-input test now uses a platform-neutral relative path, and `.github/workflows/build-release.yml` now pins `golangci-lint` to `v2.10.1` with the same `./...` target as the local command.
- In progress: none.
- Blockers: none.
- Next step: keep CI green by committing the current cross-platform test fixes plus the pinned lint version, then resume the next operator-facing slice.

### 2026-03-17 Session Update

- Completed: Deduplicated the shared cleanup/audit hygiene option layer in `internal/app/import_follow.go`. `cleanup-follow-imports` and `audit-follow-imports` now embed one common option struct for shared paths, filters, retention profiles, fail-if-matched, and age-gating; they reuse the same common flag parser, retention-profile application, and pattern/age/target-binding validation while keeping command-specific prune/check flags separate. App tests now also cover audit-side invalid pattern and retention-profile parsing so both command families stay aligned under one helper path.
- In progress: none.
- Blockers: none.
- Next step: pivot from this follow/import parser cleanup back to a new operator-facing capability, unless another small cleanup slice around report/renderer sharing is clearly higher value.

### 2026-03-17 Session Update

- Completed: Removed the maintainer-only `--list-examples` / `--refresh-examples` execution path from the production `cleanup-follow-imports` and `audit-follow-imports` commands. Hygiene example catalogs now live only in `internal/app/follow_import_example_fixtures_test.go`, checked-in fixture rewrites happen through env-gated test helpers, and the operator/maintainer docs now describe that test-driven maintenance flow instead of advertising unsupported runtime flags.
- In progress: none.
- Blockers: none.
- Next step: decide whether the remaining cleanup/audit flag parsing should also move behind a smaller shared helper, or whether the next slice should return to a new operator-facing capability.

### 2026-03-17 Session Update

- Completed: Extended the shared follow/import example helper to cover example-mode flag parsing and validation as well as fixture IO. `cleanup-follow-imports` and `audit-follow-imports` now reuse the same `--list-examples` / `--refresh-examples[=<name[,name...]>]` parsing, duplicate-name normalization, incompatibility checks, and named-fixture validation, leaving only command-specific operational-target validation in `internal/app/import_follow.go`.
- In progress: none.
- Blockers: none.
- Next step: decide whether the remaining cleanup/audit option-parsing duplication is worth another shared helper, or whether follow/import work should pivot back to new operator-facing functionality.

### 2026-03-17 Session Update

- Completed: Refactored the follow/import example-fixture workflow behind a shared helper in `internal/app/follow_examples.go`. `cleanup-follow-imports` and `audit-follow-imports` now reuse the same base-dir resolution, example-name parsing, named fixture selection, list output, and refresh-writing logic instead of carrying two near-identical helper stacks, while keeping their command-specific example catalogs and renderers unchanged.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current shared helper plus hard-coded example catalogs is the right steady state, or whether a later slice should move those example catalogs into a small manifest once the operator sample set grows further.

### 2026-03-17 Session Update

- Completed: Added checked-in `audit-follow-imports` sample outputs under `internal/app/testdata` together with `--list-examples` and `--refresh-examples[=<name[,name...]>]` maintenance helpers. The audit command now matches the cleanup command's fixture workflow, app and parser coverage verify example-mode parsing plus refresh/list wiring, and `TestAuditFollowImportsExampleOutputsStayInSync` fails if future audit renderer changes are not reflected in the checked-in fixtures and docs.
- In progress: none.
- Blockers: none.
- Next step: decide whether the audit example catalog is complete enough now, or whether future follow/import operator samples should move into a small manifest once the hard-coded fixture set grows further.

### 2026-03-17 Session Update

- Completed: Added `--policy-profile` presets to `scripts/readiness-check` with initial `ci` and `release` profiles. Profiles expand to the existing threshold and warning-policy flags instead of introducing a second policy engine: `ci` sets the current slow-run thresholds, while `release` adds the stale follow-health failure policy on top. Explicit flags still override profile thresholds and append extra warning codes, and the final text/JSON outputs now expose `policy_profile` alongside the already-expanded thresholds and warning-policy fields. Tests now cover profile parsing, override behavior, lint is clean again, and `go test ./...` remains green.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current preset names and values are enough for automation consumers, or whether release/CI docs should publish concrete recommended invocation examples for different operator goals.
### 2026-03-17 Session Update

- Completed: Published concrete `readiness-check` invocation examples for the current `ci` and `release` policy profiles in the root README plus maintainer/operator docs. The docs now distinguish between quick local checks, CI JSON capture, release gating, and failure-investigation runs that combine `--keep-going` with the release profile.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current `ci` and `release` preset values should stay fixed, or whether operators need a third preset or different threshold defaults for slower environments.
### 2026-03-17 Session Update

- Completed: Added a third `readiness-check` preset for slower environments: `--policy-profile slow-ci`. It expands to more forgiving timing thresholds (`slow_run_ms=20000`, `slow_phase_ms=4000`) without changing warning-failure policy, so slower or more contended CI runners can stay on a named preset instead of copying custom threshold flags. Tests now cover the new preset and summary output, and the README plus maintainer/operator docs now show when to use `ci`, `slow-ci`, and `release`.
- In progress: none.
- Blockers: none.
- Next step: decide whether the preset catalog is complete enough now, or whether operators would benefit more from real-world sample output snippets than from additional preset names.

### 2026-03-17 Session Update

- Completed: Added explicit conformance coverage for `C10 Import suppression` under `internal/domain/imports/conformance_test.go`. The new coverage exercises both privacy-blocked imported artifacts and imported duplicates that are suppressed because stronger explicit memory already exists, and it verifies warning visibility plus import-audit suppression metadata instead of relying only on lower-level service tests.
- In progress: none.
- Blockers: none.
- Next step: decide whether to continue promoting the remaining conformance-matrix scenarios into explicitly named `TestConformance...` cases where coverage still mostly lives in service-level tests.

### 2026-03-17 Session Update

- Completed: Promoted more v1 matrix scenarios into explicit conformance tests. `internal/domain/retrieval/conformance_test.go` now includes `C04 No search hits`, `internal/db/conformance_test.go` now includes `C07 Save note scope validation`, and `internal/domain/handoff/conformance_test.go` now includes `C08 Save handoff validity`. This makes zero-hit search, session/scope mismatch rejection, and mandatory actionable handoff next steps part of the named conformance layer instead of only ordinary unit/service coverage.
- In progress: none.
- Blockers: none.
- Next step: decide whether to keep promoting remaining service-level coverage into explicit conformance names, or stop once the current v1 matrix entries all have direct named tests in either domain or DB layers.

### 2026-03-17 Session Update

- Completed: Added checked-in `scripts/readiness-check/testdata` sample outputs for common operator workflows and a test that keeps them synchronized with the renderer. The repository now carries a clean `slow-ci` text success example, a `ci` JSON success example, and a `release` warning-policy failure example, and the README plus maintainer/operator docs now point readers at those fixtures instead of describing the output shape only in prose.
- In progress: none.
- Blockers: none.
- Next step: decide whether to stop here with fixed sample outputs, or extend the readiness helper with a dedicated example-generation/update workflow if fixture churn becomes noticeable.

### 2026-03-17 Session Update

- Completed: Added an explicit `go run ./scripts/readiness-check --refresh-examples` helper for checked-in readiness sample outputs. The fixed example-report builders now live in shared package code instead of only in tests, `TestReadinessExampleOutputsStayInSync` stays read-only, and the maintainer MCP integration guide now documents the helper command.
- In progress: none.
- Blockers: none.
- Next step: decide whether the refresh helper should remain a mode on `readiness-check`, or move to a separate maintainer script if the checked-in example catalog grows further.

### 2026-03-17 Session Update

- Completed: Made the readiness example helper scale a bit better as the fixture catalog grows. `scripts/readiness-check` now supports `--list-examples` plus `--refresh-examples=<name[,name...]>`, validates requested fixture names up front, and test coverage now checks list output plus subset-only rewrites instead of assuming every refresh touches the full catalog.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current list-and-subset workflow is enough for maintainers, or whether the example catalog should eventually move into a small manifest file with richer metadata.

### 2026-03-17 Session Update

- Completed: Added explicit stale follow-health pruning to `doctor`. Operators can now run `doctor --prune-stale-follow-health` to remove only stale `follow-imports.health.json` sidecars, and the doctor text/JSON report now surfaces whether a snapshot was pruned plus the prune reason. App coverage now verifies stale snapshots are removed only when requested and that fresh snapshots are preserved.
- In progress: none.
- Blockers: none.
- Next step: decide whether explicit stale pruning is enough operator hygiene, or whether a later slice should add a broader import/follow cleanup surface for old checkpoints and retry artifacts too.

### 2026-03-17 Session Update

- Completed: Added `cleanup-follow-imports` as an explicit operator cleanup command for follow-mode artifacts. Operators can now prune checkpoint sidecars, range-suffixed failed-output JSONL files, range-suffixed failed-manifest JSON sidecars, and stale follow-health snapshots from one command while preserving the unsuffixed retry-artifact base paths. The cleanup flow reuses the same default checkpoint derivation and multi-input per-file naming rules as `follow-imports`, so cleanup targets stay aligned with the ingestion path instead of maintaining a second naming scheme.
- In progress: none.
- Blockers: none.
- Next step: decide whether `cleanup-follow-imports` should stay as an explicit path-targeted cleanup tool, or whether operators would benefit from an additional dry-run/age-filter layer before expanding the cleanup surface further.

### 2026-03-17 Session Update

- Completed: Added safer operator controls to `cleanup-follow-imports` with `--dry-run` and `--older-than <duration>`. Cleanup reports now distinguish matched files from actually removed files, surface which checkpoint/retry artifacts were skipped because they were too new, and preview stale follow-health pruning through `follow_health_would_prune` when running in dry-run mode. App coverage now verifies both actual cleanup and dry-run age-gated preview behavior.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current path-targeted cleanup surface is complete enough, or whether operators need broader selection controls such as explicit include/exclude patterns or retention presets before expanding cleanup any further.

### 2026-03-17 Session Update

- Completed: Added explicit include/exclude glob filters to `cleanup-follow-imports`. Operators can now repeat `--include` and `--exclude` (or pass comma-separated patterns) to narrow checkpoint and retry-artifact cleanup to specific path families, with excludes winning over includes. Cleanup reports now surface the active pattern lists plus which files were skipped by pattern, and app coverage verifies pattern parsing plus filtered cleanup/report behavior.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current cleanup surface is now sufficient, or whether a later slice should add higher-level retention presets or other operator presets on top of the explicit pattern/age controls.

### 2026-03-17 Session Update

- Completed: Added `cleanup-follow-imports --retention-profile` with initial `stale`, `daily`, and `reset` presets. The profile layer expands only to default `--older-than` values (`1h`, `24h`, and `0s` respectively) so the command keeps one cleanup engine, while explicit `--older-than` still overrides the profile. Cleanup reports now surface `retention_profile` alongside the effective age gate, and app coverage verifies preset parsing, unknown-profile rejection, and explicit age overrides.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current cleanup preset catalog is sufficient, or whether operators would benefit more from a couple of documented real-world cleanup invocations or checked-in sample outputs before adding more preset names.

### 2026-03-17 Session Update

- Completed: Added checked-in `cleanup-follow-imports` sample outputs under `internal/app/testdata` plus a sync test in `internal/app/run_test.go`. The repository now carries one text example for a `daily` dry-run preview and one JSON example for an include/exclude-filtered cleanup pass, and `TestCleanupFollowImportsExampleOutputsStayInSync` fails if future renderer changes are not reflected in those fixtures and docs.
- In progress: none.
- Blockers: none.
- Next step: decide whether these fixed cleanup samples are enough, or whether maintainers would benefit from a dedicated fixture-refresh helper if cleanup output keeps evolving.

### 2026-03-17 Session Update

- Completed: Added a dedicated `cleanup-follow-imports` fixture-maintenance workflow with `--list-examples` plus `--refresh-examples[=<name[,name...]>]`. Cleanup example definitions now live in shared package code instead of only in tests, maintainers can refresh the whole catalog or a named subset, and app coverage now verifies example listing, subset refresh, command wiring, and read-only fixture sync against the checked-in outputs.
- In progress: none.
- Blockers: none.
- Next step: decide whether the current cleanup example catalog can stay hard-coded in package code, or whether it should eventually move into a small manifest as the number of operator sample flows grows.

### 2026-03-17 Session Update

- Completed: Added `cleanup-follow-imports --fail-if-matched` as an automation-friendly hygiene gate. The cleanup report now surfaces whether the gate was enabled plus whether the selected target set matched any checkpoint, retry-artifact, or stale follow-health cleanup work, and the command now returns a non-zero error after writing the report when matches are found. App coverage verifies both the failing dry-run gate path and the clean no-match path.
- In progress: none.
- Blockers: none.
- Next step: decide whether cleanup automation now has enough signaling, or whether follow/import operator workflows would benefit more from a dedicated scheduled-check/report command instead of adding more flags to cleanup itself.

### 2026-03-17 Session Update

- Completed: Added a dedicated `audit-follow-imports` read-only hygiene command for scheduled checks and operator review. It reuses the existing follow-artifact target derivation plus age/pattern filtering logic, reports pending checkpoint and retry-artifact matches without deleting them, includes detailed follow-health snapshot presence/staleness metadata, and supports `--fail-if-matched` for automation gating without requiring `cleanup-follow-imports --dry-run`.
- In progress: none.
- Blockers: none.
- Next step: decide whether the new audit command is enough operator-facing hygiene surface, or whether it should eventually grow checked-in sample outputs like the readiness and cleanup commands.

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
