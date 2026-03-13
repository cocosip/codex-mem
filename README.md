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

## Commands

- `go run ./cmd/codex-mem doctor`
  Prints effective config plus runtime readiness and audit diagnostics.
- `go run ./cmd/codex-mem doctor --json`
  Prints the same diagnostics in machine-readable JSON for automation or CI checks.
- `go run ./cmd/codex-mem migrate`
  Opens the configured SQLite database and applies embedded migrations.
- `go run ./cmd/codex-mem serve`
  Starts the MCP stdio transport and exposes the v1 tools.
- `go run ./cmd/codex-mem serve-http --listen 127.0.0.1:8080 --path /mcp`
  Starts the MCP HTTP transport with JSON response mode for remote clients.

## Quick Start

1. Copy [codex-mem.example.json](configs/codex-mem.example.json) to `configs/codex-mem.json` if you want repository-local config.
2. Run `go run ./cmd/codex-mem doctor`.
3. Confirm:
   `required_schema_ok=true`
   `fts_ready=true`
   `migrations_pending=0`
   `mcp_tool_count=9`
4. Start either:
   `go run ./cmd/codex-mem serve`
   or
   `go run ./cmd/codex-mem serve-http --listen 127.0.0.1:8080 --path /mcp`

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

## First-Run Workflow

For one repository:

1. Run `memory_install_agents` in safe mode for project or both targets.
2. Start work with `memory_bootstrap_session`.
3. Save durable discoveries with `memory_save_note`.
4. Save a continuation record with `memory_save_handoff` before ending.

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

Use `go run ./cmd/codex-mem doctor --json` when the output needs to be consumed by scripts.
For a single local readiness gate that combines diagnostics plus stdio and HTTP MCP handshake validation, run `go run ./scripts/readiness-check`.

For setup and integration failures, use the Go troubleshooting guide in [troubleshooting.md](docs/go/troubleshooting.md).

## Release Notes

The current release/readiness checklist lives in [release-readiness.md](docs/go/release-readiness.md).
