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

As of 2026-03-16, the Go implementation includes:

- SQLite-backed durable storage with embedded migrations
- scoped continuity tools and retrieval tools
- AGENTS template installation
- MCP stdio and HTTP transports for all v1 tools
- session-aware streamable HTTP with standalone SSE support on `GET /mcp`
- optional idle HTTP session expiry via `serve-http --session-timeout <duration>`
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

If your release automation prefers a single structured summary payload, run:

```powershell
go run ./scripts/readiness-check --json
```

If your release automation prefers one failing run to still capture every later phase it can, run:

```powershell
go run ./scripts/readiness-check --keep-going
```

If your release automation wants one summary object or text report to also flag slow overall runs or slow attempted phases, run:

```powershell
go run ./scripts/readiness-check --slow-run-ms=8000 --slow-phase-ms=1000
```

If your release gate wants specific warning codes to become release-blocking while leaving other warnings informational, run:

```powershell
go run ./scripts/readiness-check --fail-on-warning-code WARN_FOLLOW_IMPORTS_HEALTH_STALE
```

If you want a maintained preset instead of repeating the same release policy flags, run:

```powershell
go run ./scripts/readiness-check --policy-profile release
```

Recommended release-gate invocations:

1. Human-readable release gate:

```powershell
go run ./scripts/readiness-check --policy-profile release
```

2. JSON payload for automation or artifact capture:

```powershell
go run ./scripts/readiness-check --json --policy-profile release
```

3. Investigation run when you want one failing invocation to still collect later phases:

```powershell
go run ./scripts/readiness-check --json --keep-going --policy-profile release
```

Confirm:

- `required_schema_ok=true`
- `fts_ready=true`
- `migrations_pending=0`
- `foreign_keys=true`
- `note_provenance_ready=true`
- `exclusion_audit_ready=true`
- `mcp_tool_count=11`

If your deployment uses `follow-imports`, also inspect either the echoed `doctor_follow_imports_*` lines from `go run ./scripts/readiness-check` or the embedded `doctor.follow_imports` object from `go run ./scripts/readiness-check --json`. Those fields surface the last-known runtime watch-health snapshot from `doctor` for automation, but they remain informational unless your own release gate chooses to fail on stale or degraded follow state. The readiness helper now also emits overall run timing plus per-phase status and timing for `doctor`, stdio smoke, and HTTP smoke, which is useful when a release gate fails and you need to see which stage stopped and whether either the whole check or one phase is getting slower over time; `--keep-going` is available when you want later phases to continue running after an early failure, `--slow-run-ms` / `--slow-phase-ms` let the report surface slow-run warnings, `--fail-on-warning-code` lets the release gate opt into failing on selected warning codes without making every warning globally fatal, and `--policy-profile release` packages the current recommended release preset on top of those same underlying fields.

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
- if using HTTP transport, reuse `Mcp-Session-Id` on later requests and reach the configured `/mcp` endpoint successfully

### 4. Onboarding Smoke Check

In a clean repository:

1. Run `memory_install_agents` in safe mode.
2. Start a session with `memory_bootstrap_session`.
3. Save one note with `memory_save_note` in a step where you are testing durable note writes.
4. Save one handoff with `memory_save_handoff` in a separate step where you are testing continuation writes.
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
5. Call `memory_save_note` to verify durable note writes
6. Call `memory_save_handoff` to verify continuation writes
7. Call `memory_search`
8. Call `memory_get_recent`

## Known Non-Release Blockers

These do not currently block internal usage:

- no dedicated README examples for a specific external MCP client
- `doctor` focuses on readiness and audit posture, not deep retrieval trace introspection
- release artifacts publish checksums but not detached signatures yet

## Recommended Next Packaging Tasks

If the project is being prepared for wider use, the next packaging tasks are:

1. Decide whether release assets should also include detached signatures after signing-key management is established.
2. Add one or more external-client-specific integration examples if a concrete deployment target emerges.
3. Consider richer retrieval or audit traces only if integration troubleshooting shows a real need.

