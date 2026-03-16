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

That combined check now covers:

1. `doctor --json`
2. stdio MCP smoke test
3. HTTP MCP smoke test

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


