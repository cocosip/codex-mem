# codex-mem

`codex-mem` is a local-first durable memory service for Codex workflows.
It stores structured notes and handoffs in SQLite, restores continuity across restarted sessions, and exposes the v1 tool surface over MCP stdio and HTTP transports.

## Current Status

- Go v1 implementation is feature-complete for the core spec slices.
- `serve` runs a native MCP stdio server.
- `serve-http` runs a native MCP HTTP server for remote or private deployment.
- `doctor` reports config, database readiness, migration status, provenance coverage, and MCP tool availability.
- AGENTS template installation is implemented for global and project workflows.

Normative product docs live in [docs/spec/README.md](docs/spec/README.md).
Go implementation planning and progress live in [docs/go/README.md](docs/go/README.md).

## User Model

`codex-mem` has two different surfaces:

- operator commands:
  used by whoever deploys or maintains the `codex-mem` process
- MCP tools:
  invoked by Codex after the MCP server is registered

The MCP tools are not shell commands.
After the server is registered, Codex triggers tools such as `memory_bootstrap_session` over MCP.

## User Quick Start

The normal user path should use a packaged binary, not `go run`.

1. Download the release artifact for your platform and unpack it.
2. Optionally place `codex-mem.example.json` next to your deployment config and adjust paths.
3. Start either:
   Windows local stdio:
   `codex-mem.exe serve`
   Windows remote HTTP:
   `codex-mem.exe serve-http --listen 127.0.0.1:8080 --path /mcp`
   Unix-like local stdio:
   `./codex-mem serve`
   Unix-like remote HTTP:
   `./codex-mem serve-http --listen 127.0.0.1:8080 --path /mcp`
4. Register the server with Codex:
   local stdio:
   `codex mcp add codex-mem -- /absolute/path/to/codex-mem serve`
   remote HTTP:
   `codex mcp add codex-mem --url http://127.0.0.1:8080/mcp`
5. Let Codex call the MCP tools during normal workflow.

## Operator Commands

These commands are for operators, packagers, and CI.
They are not MCP tools and are not the normal end-user interaction path.

- `codex-mem doctor`
  Prints effective config plus runtime readiness and audit diagnostics.
- `codex-mem doctor --json`
  Prints the same diagnostics in machine-readable JSON for automation or CI checks.
- `codex-mem migrate`
  Opens the configured SQLite database and applies embedded migrations.
- `codex-mem serve`
  Starts the MCP stdio transport and exposes the v1 tools.
- `codex-mem serve-http --listen 127.0.0.1:8080 --path /mcp`
  Starts the MCP HTTP transport with JSON response mode for remote clients.
- `codex-mem version`
  Prints the embedded build version, commit, and build date.

## Operator Quick Check

Before exposing the server to users, confirm:

- `doctor` reports:
   `required_schema_ok=true`
   `fts_ready=true`
   `migrations_pending=0`
   `mcp_tool_count=9`
- Codex can register the server successfully
- Codex can call at least one MCP tool successfully

## MCP Tool Surface

The current MCP server exposes:

- `memory_bootstrap_session`
- `memory_resolve_scope`
- `memory_start_session`
- `memory_save_note`
- `memory_save_handoff`
- `memory_search`
- `memory_get_recent`
- `memory_get_note`
- `memory_install_agents`

Request and response examples are documented in [example-payloads.md](docs/spec/appendices/example-payloads.md).
For a runnable client-side handshake and tool-call check, use [mcp-integration.md](docs/go/mcp-integration.md).
For concrete client setup examples, use [client-examples.md](docs/go/client-examples.md).
For end-user prompt templates that cause Codex to pick the memory tools automatically, use [prompt-examples.md](docs/go/prompt-examples.md).
For release packaging and operator guidance, use [release-readiness.md](docs/go/release-readiness.md).

## First-Run Workflow

For one repository:

1. Register `codex-mem` with Codex.
2. Let Codex run `memory_install_agents` in safe mode for project or both targets.
3. Let Codex start work with `memory_bootstrap_session`.
4. Let Codex save durable discoveries with `memory_save_note`.
5. Let Codex save a continuation record with `memory_save_handoff` before ending.

See [onboarding-flows.md](docs/spec/appendices/onboarding-flows.md) for the full onboarding guidance.

## Diagnostics

`doctor` now reports:

- config precedence and selected config file
- SQLite pragmas and schema readiness
- migration availability and applied status
- note and handoff audit counts
- note source-category coverage
- exclusion audit coverage
- MCP transport/tool availability

Use `codex-mem doctor --json` when the output needs to be consumed by scripts.
The combined readiness gate under `scripts/readiness-check` is for CI and maintainers, not end users.

For setup and integration failures, use the Go troubleshooting guide in [troubleshooting.md](docs/go/troubleshooting.md).

## Release Notes

The current release/readiness checklist lives in [release-readiness.md](docs/go/release-readiness.md).
