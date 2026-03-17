# Go MCP Integration

## Purpose

This document describes the MCP transport behavior of the current Go server and the checked-in source-tree smoke tests.

It is intended for:

- maintainers validating that `codex-mem` can be launched as an MCP server over stdio or HTTP
- integrators checking a client's `initialize` and `tools/list` behavior
- CI or local source-tree smoke-testing of one real `tools/call` round trip end to end

Use this when:

- validating transport behavior from the source tree
- running checked-in smoke tests under `scripts/`
- comparing a custom client against a known-good MCP request flow

Do not use this for:

- first-time packaged-binary setup with Codex CLI
- learning how mem works as a user
- deciding how to implement the Go codebase

This document is mostly for maintainers, integrators, and CI.
It assumes you are working from the source tree, which is why it uses `go run` for the checked-in smoke-test programs.

If you are a normal user trying to connect a packaged `codex-mem` binary to Codex, start with [client-examples.md](../operator/client-examples.md) instead.

## Transport Summary

The current server exposes two transports:

### Local stdio

Start it with:

```powershell
codex-mem.exe serve
```

Characteristics:

- JSON-RPC version `2.0`
- stdio now uses the go-sdk server transport with newline-delimited JSON-RPC messages
- each stdin request and stdout response is encoded as one JSON value terminated by a newline
- server methods:
  `initialize`
  `ping`
  `tools/list`
  `tools/call`

### Remote HTTP

Start it with:

```powershell
codex-mem.exe serve-http --listen 127.0.0.1:8080 --path /mcp
```

Characteristics:

- same JSON-RPC method surface as stdio
- `POST /mcp` uses the SDK streamable HTTP transport with JSON responses for ordinary request/response calls
- `initialize` returns `Mcp-Session-Id`, and HTTP clients must reuse that header on subsequent requests
- `GET /mcp` opens a standalone `text/event-stream` channel for an active session
- `DELETE /mcp` closes an active session
- optional `--session-timeout <duration>` bounds idle HTTP session lifetime
- optional `--allow-origin <origin>` flags for browser-style origin checks

`doctor` should still report:

- `mcp_transport=stdio` in text mode
- `mcp.tool_count=11` in JSON mode

## Scope Of This Document

This is not the primary end-user setup guide.

Use this document when you need one of these:

- a source-tree protocol smoke test
- a transport-level request/response example
- a maintainer-oriented explanation of stdio versus HTTP MCP behavior

If you only want to register the packaged binary with Codex CLI, use [client-examples.md](../operator/client-examples.md).

## Fastest Source-Tree Smoke Test

Run the checked-in smoke test:

```powershell
go run ./scripts/mcp-smoke-test
```

What it does:

1. starts the stdio server process
2. sends `initialize`
3. sends `notifications/initialized`
4. sends `tools/list`
5. sends `tools/call` for `memory_install_agents`
6. verifies that a temporary `AGENTS.md` file was written

Expected output is shaped like:

```text
mcp smoke test passed
protocol_version=...
tool_count=11
tool_call=memory_install_agents
written_file=...
```

This uses `go run` because the smoke-test helper itself lives inside this repository under `scripts/`.

For the HTTP transport, run:

```powershell
go run ./scripts/http-mcp-smoke-test
```

Expected output is shaped like:

```text
http mcp smoke test passed
endpoint=http://127.0.0.1:...
session_id=...
protocol_version=2025-06-18
tool_count=11
tool_call=memory_install_agents
written_file=...
```

If you want one maintainer-oriented command that also validates `doctor --json`, run:

```powershell
go run ./scripts/readiness-check
```

For a structured readiness summary object, run:

```powershell
go run ./scripts/readiness-check --json
```

If you want the helper to keep running later phases after an earlier failure so the final summary includes every phase result it could still gather, run:

```powershell
go run ./scripts/readiness-check --keep-going
```

If you want the helper to classify slow overall runs or attempted phases into warnings for automation while keeping those warnings informational, run:

```powershell
go run ./scripts/readiness-check --slow-run-ms=8000 --slow-phase-ms=1000
```

If your automation wants to fail readiness only when specific warning codes are present in the final summary, run:

```powershell
go run ./scripts/readiness-check --fail-on-warning-code WARN_FOLLOW_IMPORTS_HEALTH_STALE
```

If you want a named preset instead of repeating the same flag bundle each time, run one of:

```powershell
go run ./scripts/readiness-check --policy-profile ci
go run ./scripts/readiness-check --policy-profile slow-ci
go run ./scripts/readiness-check --policy-profile release
```

Today those profiles expand to:

1. `ci`: `--slow-run-ms=8000 --slow-phase-ms=1000`
2. `slow-ci`: `--slow-run-ms=20000 --slow-phase-ms=4000`
3. `release`: the `ci` thresholds plus `--fail-on-warning-code WARN_FOLLOW_IMPORTS_HEALTH_STALE`

Recommended starting points:

1. Fast local maintainer sanity check:

```powershell
go run ./scripts/readiness-check
```

2. CI-oriented machine-readable summary with the current threshold preset:

```powershell
go run ./scripts/readiness-check --json --policy-profile ci
```

3. Slower or more contended CI runner with a more forgiving threshold preset:

```powershell
go run ./scripts/readiness-check --json --policy-profile slow-ci
```

4. Release-oriented run that keeps phase timing and fails on stale follow-health warnings:

```powershell
go run ./scripts/readiness-check --json --policy-profile release
```

5. Failure-investigation run that gathers every phase it can before exiting:

```powershell
go run ./scripts/readiness-check --json --keep-going --policy-profile release
```

Checked-in example outputs for those readiness flows live under [../../../scripts/readiness-check/testdata](../../../scripts/readiness-check/testdata/):

- [example-ci-success.json](../../../scripts/readiness-check/testdata/example-ci-success.json)
- [example-slow-ci-success.txt](../../../scripts/readiness-check/testdata/example-slow-ci-success.txt)
- [example-release-warning-failure.txt](../../../scripts/readiness-check/testdata/example-release-warning-failure.txt)

If a deliberate readiness-output change makes those fixtures drift, refresh them with:

```powershell
go run ./scripts/readiness-check --refresh-examples
```

If you only need one or two fixtures while iterating on a specific output shape, first list the available fixture names and then refresh just the subset you want:

```powershell
go run ./scripts/readiness-check --list-examples
go run ./scripts/readiness-check --refresh-examples=ci-json
```

Then rerun `go test ./scripts/readiness-check -run TestReadinessExampleOutputsStayInSync` plus the normal repo checks so the updated fixtures are verified in read-only mode again.

That combined check now covers:

1. `doctor --json`
2. stdio MCP smoke test
3. HTTP MCP smoke test

The checked-in `cleanup-follow-imports` example outputs under [../../../internal/app/testdata](../../../internal/app/testdata/) now support the same maintainer loop. When the cleanup text or JSON renderer changes on purpose, refresh every cleanup fixture from the repository root with:

```powershell
go run ./cmd/codex-mem cleanup-follow-imports --refresh-examples
```

If you only need a subset while iterating, list the current names first and then refresh the exact fixtures you touched:

```powershell
go run ./cmd/codex-mem cleanup-follow-imports --list-examples
go run ./cmd/codex-mem cleanup-follow-imports --refresh-examples=filtered-cleanup-json
```

Then rerun `go test ./internal/app -run TestCleanupFollowImportsExampleOutputsStayInSync` plus the normal repo checks so the updated cleanup fixtures are verified in read-only mode again.

The default text summary from `scripts/readiness-check` echoes the `doctor.follow_imports` fields as flat `doctor_follow_imports_*` lines so CI or local automation can inspect last-known runtime watch health from the existing sidecar without having to parse the full doctor JSON again. The helper also emits explicit overall `started_at`, `completed_at`, and `duration_ms` fields plus per-phase `phase_*_status`, `phase_*_summary`, and timing lines for the `doctor`, stdio smoke, and HTTP smoke phases, so a failed run still tells automation which phase stopped progress and how long both the whole run and each attempted phase took before the command exits non-zero. `--json` returns one structured summary object that embeds the parsed doctor report, compact stdio/HTTP smoke-test summaries, and the same overall plus per-phase timing results. `--keep-going` keeps attempting later phases after an earlier failure, which is useful when you want one failing run to still capture as much phase state as possible. `--slow-run-ms` and `--slow-phase-ms` add explicit readiness warnings to both text and JSON output when thresholds are exceeded, `--fail-on-warning-code` lets automation promote selected warning codes, including nested doctor follow-health warning codes, into a failed readiness result, and `--policy-profile` expands a small set of documented presets into those same underlying options while still exposing the final thresholds and warning-policy state in the output. In all modes the follow-health fields are informational by default unless your own automation explicitly chooses to gate on them.

## Manual Client Checklist

If you are wiring a real MCP client, confirm this order:

1. launch the `codex-mem` binary with `serve`
2. send `initialize`
3. send `notifications/initialized`
4. send `tools/list`
5. send `tools/call`

For HTTP clients, the equivalent order is:

1. `POST /mcp` with `initialize`
2. capture the `Mcp-Session-Id` response header
3. `POST /mcp` with `notifications/initialized` and the same session header
4. optionally open `GET /mcp` with `Accept: text/event-stream` and the same session header
5. `POST /mcp` for `tools/list` and `tools/call` with the same session header

Common client mistakes:

- sending `Content-Length`-framed stdio messages instead of newline-delimited JSON
- omitting `jsonrpc: "2.0"`
- skipping `initialize`
- forgetting to reuse `Mcp-Session-Id` on later HTTP requests
- opening `GET /mcp` without `Accept: text/event-stream`
- sending tool arguments with unknown fields

## Representative Request Flow

The smoke test uses this real tool call:

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "memory_install_agents",
    "arguments": {
      "target": "project",
      "mode": "safe",
      "cwd": "C:/temp/mcp-smoke-project",
      "project_name": "mcp-smoke-test",
      "system_name": "codex-mem"
    }
  }
}
```

This is a good integration check because it:

- exercises real schema validation
- proves tool registration is complete
- avoids depending on repository identity discovery
- leaves the repository itself untouched by writing into a temp directory

## When To Use Something Beyond The Smoke Test

Move past the smoke test when you need to validate:

- scope resolution and repository identity behavior
- continuity bootstrap with real session state
- search and recent retrieval against a populated database
- a specific external MCP client's process launch model

For packaged-binary client setup examples, use [client-examples.md](../operator/client-examples.md).
For environment and failure diagnosis, use [troubleshooting.md](../operator/troubleshooting.md).


