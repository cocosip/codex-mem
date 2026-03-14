# Go Client Examples

## Purpose

This document gives client-specific examples for connecting `codex-mem` as an MCP server.

Audience:

- normal users registering `codex-mem` with a real MCP client
- operators choosing between local stdio and remote HTTP deployment

Use this when:

- you already have the packaged binary
- you want to connect Codex CLI or another real client to the server

Do not use this for:

- source-tree smoke tests from this repository
- CI validation of MCP protocol behavior
- Go implementation planning

Use it after:

- the release artifact for your platform has been unpacked
- the operator has chosen either local stdio or remote HTTP deployment

## Example 1: Codex CLI Local stdio Server

This is the most direct local integration path in the current repository environment.

### Start the local binary

Windows:

```powershell
codex-mem.exe serve
```

Unix-like:

```bash
./codex-mem serve
```

### Register the MCP server with Codex CLI

Minimal example:

```powershell
codex mcp add codex-mem -- C:\tools\codex-mem\codex-mem.exe serve
```

If you want to force a specific repository-local config file:

```powershell
codex mcp add codex-mem --env CODEX_MEM_CONFIG_FILE=D:\work\repo\configs\codex-mem.json -- C:\tools\codex-mem\codex-mem.exe serve
```

### Verify the registration

```powershell
codex mcp list
```

```powershell
codex mcp get codex-mem
```

What to confirm:

- the server is present in the Codex MCP list
- the configured command ends with `codex-mem.exe serve`
- any required environment variables are present

### Remove or replace the registration

```powershell
codex mcp remove codex-mem
```

Then re-add it with the updated command or environment.

## Example 2: Codex CLI Remote HTTP Server

If you want a private deployed MCP endpoint instead of a local process, use the HTTP transport.

### Start the HTTP transport

```powershell
codex-mem.exe serve-http --listen 127.0.0.1:8080 --path /mcp
```

For a broader private-network bind:

```powershell
codex-mem.exe serve-http --listen 0.0.0.0:8080 --path /mcp
```

Optional origin allowlist:

```powershell
codex-mem.exe serve-http --listen 0.0.0.0:8080 --path /mcp --allow-origin https://codex.example.com
```

Optional idle session timeout:

```powershell
codex-mem.exe serve-http --listen 0.0.0.0:8080 --path /mcp --session-timeout 30m
```

Use this when you want the server to automatically expire inactive HTTP MCP sessions instead of keeping them alive indefinitely.

### Register the HTTP MCP server with Codex CLI

```powershell
codex mcp add codex-mem-remote --url http://127.0.0.1:8080/mcp
```

For a private deployed host:

```powershell
codex mcp add codex-mem-remote --url https://mcp.example.internal/mcp
```

If your remote deployment is protected by a bearer token:

```powershell
codex mcp add codex-mem-remote --bearer-token-env-var CODEX_MEM_TOKEN --url https://mcp.example.internal/mcp
```

### Verify the registration

```powershell
codex mcp get codex-mem-remote
```

What to confirm:

- the configured URL ends with `/mcp`
- the server is reachable from the Codex client environment
- any required bearer token env var is present before the client starts

## Example 3: ChatGPT Connector Boundary

As of 2026-03-13, OpenAI's Apps SDK docs say ChatGPT supports only **remote** MCP servers, not local stdio MCP servers.

That means:

- local `codex-mem serve` is valid for stdio clients such as Codex CLI, but not for ChatGPT connectors
- the HTTP transport is the prerequisite starting point if ChatGPT connectivity is required
- production ChatGPT support would still need normal remote deployment concerns handled explicitly, such as reachable hosting, auth, and operational hardening

Source:

- [OpenAI Apps SDK: Connect from ChatGPT](https://developers.openai.com/apps-sdk/build/mcp-server/)

## Which Example To Use

Use the Codex CLI example when:

- you want a local MCP client on the same machine
- you want to use the existing stdio transport directly
- you want the shortest path from unpacked binary to real tool calls

Use the remote HTTP example when:

- you want private deployment instead of a local child process
- you want Codex to connect by URL
- you need a path toward shared or cross-machine access

Use the ChatGPT note when:

- you are deciding whether local stdio is enough
- you need to understand the boundary between local and remote MCP support
