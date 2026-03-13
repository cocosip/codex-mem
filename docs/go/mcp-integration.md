# Go MCP Integration

## Purpose

This document gives a practical integration path for the current Go MCP server.

It is intended for:

- validating that `codex-mem` can be launched as an MCP server over stdio or HTTP
- checking a client's `initialize` and `tools/list` behavior
- smoke-testing one real `tools/call` round trip end to end

## Transport Summary

The current server exposes two transports:

### Local stdio

Start it with:

```powershell
go run ./cmd/codex-mem serve
```

Characteristics:

- JSON-RPC version `2.0`
- framed stdio messages using `Content-Length`
- server methods:
  `initialize`
  `ping`
  `tools/list`
  `tools/call`

### Remote HTTP

Start it with:

```powershell
go run ./cmd/codex-mem serve-http --listen 127.0.0.1:8080 --path /mcp
```

Characteristics:

- same JSON-RPC method surface as stdio
- `POST /mcp` request handling with JSON response mode
- `GET /mcp` returns `405` because SSE is not implemented yet
- optional `--allow-origin <origin>` flags for browser-style origin checks

`doctor` should still report:

- `mcp_transport=stdio` in text mode
- `mcp.tool_count=9` in JSON mode

## Fastest Smoke Test

Run the checked-in smoke test:

```powershell
go run ./scripts/mcp-smoke-test
```

What it does:

1. starts `go run ./cmd/codex-mem serve`
2. sends `initialize`
3. sends `notifications/initialized`
4. sends `tools/list`
5. sends `tools/call` for `memory_install_agents`
6. verifies that a temporary `AGENTS.md` file was written

Expected output is shaped like:

```text
mcp smoke test passed
protocol_version=2025-03-26
tool_count=9
tool_call=memory_install_agents
written_file=...
```

For the HTTP transport, run:

```powershell
go run ./scripts/http-mcp-smoke-test
```

Expected output is shaped like:

```text
http mcp smoke test passed
endpoint=http://127.0.0.1:...
protocol_version=2025-03-26
tool_count=9
tool_call=memory_install_agents
written_file=...
```

If you want one command that also validates `doctor --json`, run:

```powershell
go run ./scripts/readiness-check
```

That combined check now covers:

1. `doctor --json`
2. stdio MCP smoke test
3. HTTP MCP smoke test

## Manual Client Checklist

If you are wiring a real MCP client, confirm this order:

1. launch `go run ./cmd/codex-mem serve`
2. send `initialize`
3. send `notifications/initialized`
4. send `tools/list`
5. send `tools/call`

For HTTP clients, use the same method order over `POST /mcp`.

Common client mistakes:

- sending newline-delimited JSON instead of framed stdio
- omitting `jsonrpc: "2.0"`
- skipping `initialize`
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

For concrete client setup examples, use [client-examples.md](./client-examples.md).
For environment and failure diagnosis, use [troubleshooting.md](./troubleshooting.md).
