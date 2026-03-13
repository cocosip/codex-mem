# Go Release Readiness

## Purpose

This document is a practical release/readiness checklist for the current Go implementation of `codex-mem`.

Use it before:

- handing the binary to another developer
- wiring it into a real Codex MCP client
- claiming that the Go implementation is ready for broader v1 usage

## Current Readiness Snapshot

As of 2026-03-13, the Go implementation includes:

- SQLite-backed durable storage with embedded migrations
- scoped continuity tools and retrieval tools
- AGENTS template installation
- MCP stdio transport for all v1 tools
- `doctor` diagnostics for config, runtime readiness, and provenance/audit posture

The operational troubleshooting guide lives in [troubleshooting.md](./troubleshooting.md).

## Pre-Release Checklist

### 1. Runtime Health

Run:

```powershell
go run ./cmd/codex-mem doctor
```

For automation or CI, also run:

```powershell
go run ./cmd/codex-mem doctor --json
```

Confirm:

- `required_schema_ok=true`
- `fts_ready=true`
- `migrations_pending=0`
- `foreign_keys=true`
- `note_provenance_ready=true`
- `exclusion_audit_ready=true`
- `mcp_tool_count=9`

### 2. Test Suite

Run:

```powershell
go test ./...
```

Expected:

- all packages pass
- no conformance or repository regression failures

### 3. MCP Smoke Check

Run:

```powershell
go run ./cmd/codex-mem serve
```

Confirm that an MCP client can:

- call `initialize`
- list tools through `tools/list`
- call at least `memory_install_agents`
- call at least one continuity tool such as `memory_bootstrap_session`

### 4. Onboarding Smoke Check

In a clean repository:

1. Run `memory_install_agents` in safe mode.
2. Start a session with `memory_bootstrap_session`.
3. Save one note with `memory_save_note`.
4. Save one handoff with `memory_save_handoff`.
5. Start a later bootstrap and confirm continuity is recovered.

### 5. Config Smoke Check

Verify both:

- default behavior with no `configs/codex-mem.json`
- repository-local behavior with a copied config file from [codex-mem.example.json](../../configs/codex-mem.example.json)

Also verify environment overrides where relevant:

- `CODEX_MEM_DB_PATH`
- `CODEX_MEM_SYSTEM_NAME`
- `CODEX_MEM_CONFIG_FILE`

## Suggested Demo Flow

For a quick end-to-end demo:

1. Run `go run ./cmd/codex-mem doctor`
2. Start `go run ./cmd/codex-mem serve`
3. Call `memory_resolve_scope`
4. Call `memory_start_session`
5. Call `memory_save_note`
6. Call `memory_save_handoff`
7. Call `memory_search`
8. Call `memory_get_recent`

## Known Non-Release Blockers

These do not currently block internal usage:

- no packaged release artifact workflow yet
- no dedicated README examples for a specific external MCP client
- `doctor` focuses on readiness and audit posture, not deep retrieval trace introspection

## Recommended Next Packaging Tasks

If the project is being prepared for wider use, the next packaging tasks are:

1. Add a binary build/release workflow and versioning guidance.
2. Add a client-facing MCP integration example.
3. Consider wiring `doctor --json` into scripted smoke checks or CI.
4. Consider richer retrieval or audit traces only if integration troubleshooting shows a real need.
