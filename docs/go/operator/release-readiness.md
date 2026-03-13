# Go Release Readiness

## Purpose

This document is a practical release/readiness checklist for the current Go implementation of `codex-mem`.

Audience:

- maintainers
- release engineers
- operators validating packaged builds before wider use

Use this when:

- preparing a release artifact
- validating that the built binary is ready for broader use
- checking packaging, readiness, and deployment gates

Do not use this for:

- first-time day-to-day user onboarding
- normal prompt usage inside Codex
- source-tree implementation planning

It is written for maintainers and release engineers.
End users should consume the produced binaries and should not need a local Go toolchain.

Use it before:

- handing the binary to another developer
- wiring it into a real Codex MCP client
- claiming that the Go implementation is ready for broader v1 usage

## Current Readiness Snapshot

As of 2026-03-13, the Go implementation includes:

- SQLite-backed durable storage with embedded migrations
- scoped continuity tools and retrieval tools
- AGENTS template installation
- MCP stdio and HTTP transports for all v1 tools
- `doctor` diagnostics for config, runtime readiness, and provenance/audit posture

The operational troubleshooting guide lives in [troubleshooting.md](./troubleshooting.md).
Client-specific MCP setup examples live in [client-examples.md](./client-examples.md).

Tagged releases should publish per-platform archives plus a SHA256SUMS manifest. Signature-based release verification is deferred until the project has stable signing-key management and distribution policy.

## Pre-Release Checklist

### 1. Runtime Health

Run:

```powershell
codex-mem doctor
```

For automation or CI, also run:

```powershell
codex-mem doctor --json
```

For a single combined local readiness check, run:

```powershell
go run ./scripts/readiness-check
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
codex-mem serve
```

If remote deployment is in scope, also run:

```powershell
codex-mem serve-http --listen 127.0.0.1:8080 --path /mcp
```

For an end-to-end client simulation, also run:

```powershell
go run ./scripts/mcp-smoke-test
```

For the HTTP transport, also run:

```powershell
go run ./scripts/http-mcp-smoke-test
```

Confirm that an MCP client can:

- call `initialize`
- list tools through `tools/list`
- call at least `memory_install_agents`
- call at least one continuity tool such as `memory_bootstrap_session`
- if using HTTP transport, reach the configured `/mcp` endpoint successfully

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
- repository-local behavior with a copied config file from [codex-mem.example.json](../../../configs/codex-mem.example.json)

Also verify environment overrides where relevant:

- `CODEX_MEM_DB_PATH`
- `CODEX_MEM_SYSTEM_NAME`
- `CODEX_MEM_CONFIG_FILE`

## Suggested Demo Flow

For a quick end-to-end demo:

1. Run `codex-mem doctor`
2. Start `codex-mem serve`
3. Call `memory_resolve_scope`
4. Call `memory_start_session`
5. Call `memory_save_note`
6. Call `memory_save_handoff`
7. Call `memory_search`
8. Call `memory_get_recent`

## Known Non-Release Blockers

These do not currently block internal usage:

- no dedicated README examples for a specific external MCP client
- `doctor` focuses on readiness and audit posture, not deep retrieval trace introspection

## Recommended Next Packaging Tasks

If the project is being prepared for wider use, the next packaging tasks are:

1. Decide whether release assets should also include detached signatures after signing-key management is established.
2. Add one or more external-client-specific integration examples if a concrete deployment target emerges.
3. Consider richer retrieval or audit traces only if integration troubleshooting shows a real need.

