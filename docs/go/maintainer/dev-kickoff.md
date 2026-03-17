# Go Development Kickoff

## Purpose

This file is the fastest entry point for the next Go implementation session now that the v1 feature set and MCP SDK migration are already in place for `codex-mem`.

Audience:

- maintainers continuing Go implementation work
- contributors starting a new coding session in this repository

Use this when:

- you are about to implement or extend the Go codebase
- you want the shortest path back into active development work
- you need a current restart point instead of the original Phase 1 bootstrap guidance

Do not use this for:

- normal user onboarding
- MCP client registration
- packaged-binary operations

Primary references:

- [Spec Index](../../spec/README.md)
- [Implementation Backlog](./implementation-backlog.md)
- [Go Implementation Plan](./implementation-plan.md)
- [Go Development Tracker](./development-tracker.md)

## Current Baseline

The repository already has:

- the v1 scope, session, note, handoff, retrieval, privacy, and AGENTS tool surface implemented
- SQLite storage, embedded migrations, and FTS-backed retrieval working
- direct named conformance coverage for the current v1 matrix scenarios (`C01` through `C12`)
- conformance and hardening coverage for the shipped v1 behavior
- `serve` and `serve-http` both wired through `modelcontextprotocol/go-sdk`
- `doctor --json`, stdio smoke test, HTTP smoke test, and readiness check in place
- the old hand-written MCP runtime removed from the active code path

## What The Next Session Does Not Need To Rebuild

Do not restart from early-project tasks such as:

- Go module initialization
- repository layout selection
- SQLite driver or migration-style selection
- MCP library selection
- basic MCP transport/runtime scaffolding

Those decisions are already made and implemented.

Current standing decisions worth preserving:

- repository-local configuration loads from `configs/`
- configuration loading uses `viper`
- SQLite uses `modernc.org/sqlite`
- `modelcontextprotocol/go-sdk` is the only MCP runtime path

## Recommended Next Coding Slice

Start from a new post-v1 feature or operator-facing follow-up slice, not from the original Phase 1 foundation plan.

Good default shape:

1. read the current tracker to confirm the active target and any open handoff notes
2. choose one small user-visible or operator-visible improvement now that the baseline and conformance matrix are already in place
3. implement it without changing the existing eleven-tool surface unless that change is intentional
4. keep `go test ./...`, the readiness/smoke checks, and the named conformance cases green
5. update the tracker and this kickoff doc if the restart guidance changes again

## MCP Constraint For Future Work

If the next session touches MCP behavior:

1. extend the SDK-backed path instead of adding a parallel custom runtime
2. preserve the existing handler/domain boundary in `internal/mcp/handlers.go`
3. preserve the `/mcp` HTTP entrypoint and origin-allowlist behavior unless the change explicitly targets them
4. verify compatibility with:
   - `go test ./...`
   - `go run ./scripts/mcp-smoke-test`
   - `go run ./scripts/http-mcp-smoke-test`
   - `go run ./scripts/readiness-check`

## Best Starting Documents For The Next Session

Read in this order:

1. [Go Development Tracker](./development-tracker.md)
2. [Go Implementation Plan](./implementation-plan.md)
3. the relevant spec file under [docs/spec/](../../spec/README.md) for the feature you are changing
4. [MCP Integration](./mcp-integration.md) if the work touches stdio or HTTP behavior

## Suggested First Prompt For The Next Session

Use a prompt like:

```text
Read docs/go/maintainer/dev-kickoff.md and docs/go/maintainer/development-tracker.md, confirm the next feature slice, implement it on top of the existing go-sdk-based MCP runtime, and update the tracker as you go.
```
