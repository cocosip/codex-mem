# codex-mem

`codex-mem` is a local-first durable memory service for Codex workflows.
It stores structured notes and handoffs in SQLite, restores continuity across restarted sessions, and exposes the v1 tool surface over MCP stdio and HTTP transports.

## Current Status

- Go v1 implementation is feature-complete for the core spec slices.
- `serve` runs a native MCP stdio server.
- `serve-http` runs a native MCP HTTP server for remote or private deployment.
- `doctor` reports config, database readiness, migration status, provenance coverage, and MCP tool availability.
- AGENTS template installation is implemented for global and project workflows.
- watcher/relay import ingestion is available as one-shot batches through `ingest-imports` and as a checkpointed long-lived adapter through `follow-imports`.

Normative product docs live in [docs/spec/README.md](docs/spec/README.md).
Go implementation docs now live under [docs/go/README.md](docs/go/README.md), grouped into user, operator, and maintainer directories.

## Documentation Map

Use the docs by audience:

- [User docs](docs/go/user/README.md)
  How memory works, what gets saved, and prompt patterns for normal Codex usage.
- [Operator docs](docs/go/operator/README.md)
  Client registration, deployment/readiness, packaging, and troubleshooting.
- [Import ingestion guide](docs/go/operator/import-ingestion.md)
  JSONL batch and checkpointed follow-mode ingestion for watcher and relay artifacts.
- [Maintainer docs](docs/go/maintainer/README.md)
  Source-tree MCP integration, implementation planning, and development tracking.

If you only want the full Go docs index, start at [docs/go/README.md](docs/go/README.md).

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

Detailed setup examples live in [docs/go/operator/client-examples.md](docs/go/operator/client-examples.md).
Prompt-level day-to-day usage examples live in [docs/go/user/prompt-examples.md](docs/go/user/prompt-examples.md).

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
  Prints effective config plus runtime readiness, audit diagnostics, and the last-known `follow-imports` watch-health snapshot when one has been written, including stale-snapshot detection for continuous follow mode.
- `codex-mem doctor --json`
  Prints the same diagnostics in machine-readable JSON for automation or CI checks.
- `codex-mem ingest-imports --source watcher_import [--input events.jsonl] [--json] [--continue-on-error] [--failed-output failed.jsonl] [--failed-manifest failed.json]`
  Imports newline-delimited watcher or relay note events into durable imported notes plus audit records, with optional partial-success handling plus retry-oriented failure exports.
- `codex-mem follow-imports --source watcher_import --input events-a.jsonl [--input events-b.jsonl ...] [--state-file events-a.offset.json --state-file events-b.offset.json ...] [--watch-mode auto|notify|poll] [--poll-interval 5s] [--once] [--json]`
  Follows one or more watcher or relay JSONL files incrementally, prefers filesystem notifications with polling fallback by default, keeps one checkpoint per input, automatically retries watcher recovery in `auto` mode, and reports command-level watch state plus poll-catchup/recovery events and warnings alongside per-input imported-note results.
- `codex-mem migrate`
  Opens the configured SQLite database and applies embedded migrations.
- `codex-mem serve`
  Starts the MCP stdio transport and exposes the v1 tools.
- `codex-mem serve-http --listen 127.0.0.1:8080 --path /mcp [--session-timeout 30m]`
  Starts the MCP HTTP transport for remote clients, with optional idle session expiry.
- `codex-mem version`
  Prints the embedded build version, commit, and build date.

## Operator Quick Check

Before exposing the server to users, confirm:

- `doctor` reports:
   `required_schema_ok=true`
   `fts_ready=true`
   `migrations_pending=0`
   `mcp_tool_count=11`
- Codex can register the server successfully
- Codex can call at least one MCP tool successfully

## MCP Tool Surface

The current MCP server exposes:

- `memory_bootstrap_session`
- `memory_resolve_scope`
- `memory_start_session`
- `memory_save_note`
- `memory_save_handoff`
- `memory_save_import`
- `memory_save_imported_note`
- `memory_search`
- `memory_get_recent`
- `memory_get_note`
- `memory_install_agents`

Request and response examples are documented in [example-payloads.md](docs/spec/appendices/example-payloads.md).
For concrete packaged-binary client setup examples, use [client-examples.md](docs/go/operator/client-examples.md).
For maintainer-oriented MCP transport and smoke-test guidance from the source tree, use [mcp-integration.md](docs/go/maintainer/mcp-integration.md).
For operator-facing JSONL batch ingestion details, use [import-ingestion.md](docs/go/operator/import-ingestion.md).
For a quick explanation of how memory works, what gets saved, and when scope matters, use [how-memory-works.md](docs/go/user/how-memory-works.md).
For end-user prompt templates that cause Codex to pick the memory tools automatically, use [prompt-examples.md](docs/go/user/prompt-examples.md).
For release packaging and operator guidance, use [release-readiness.md](docs/go/operator/release-readiness.md).

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
- import audit counts and suppression readiness
- note source-category coverage
- exclusion audit coverage
- MCP transport/tool availability

Use `codex-mem doctor --json` when the output needs to be consumed by scripts.
The combined readiness gate under `scripts/readiness-check` is for CI and maintainers, not end users. By default it echoes the last-known `follow-imports` doctor fields as informational runtime summary lines for automation, without turning stale or degraded follow health into a hard startup/readiness failure by itself. The helper now also reports overall run timing plus explicit per-phase status and timing lines for `doctor`, stdio smoke, and HTTP smoke so failures identify which phase stopped the check and how long both the whole run and each attempted phase took before the process exits non-zero. Use `go run ./scripts/readiness-check --json` when you want one structured readiness summary object instead of flat key/value text, `go run ./scripts/readiness-check --keep-going` when you want a failing run to still attempt later phases and report every phase result it can collect, `go run ./scripts/readiness-check --slow-run-ms=8000 --slow-phase-ms=1000` when you want the summary to add observational slow-run warnings, `go run ./scripts/readiness-check --fail-on-warning-code WARN_FOLLOW_IMPORTS_HEALTH_STALE` when your automation wants specific warning codes to upgrade an otherwise-successful readiness report into a non-zero exit, or `go run ./scripts/readiness-check --policy-profile ci` / `--policy-profile slow-ci` / `--policy-profile release` when you want a documented preset for common CI and release workflows.
Recommended presets:

```powershell
go run ./scripts/readiness-check --json --policy-profile ci
go run ./scripts/readiness-check --json --policy-profile slow-ci
go run ./scripts/readiness-check --json --policy-profile release
```

Checked-in sample outputs for those workflows live under [scripts/readiness-check/testdata](scripts/readiness-check/testdata/):

- [example-ci-success.json](scripts/readiness-check/testdata/example-ci-success.json)
- [example-slow-ci-success.txt](scripts/readiness-check/testdata/example-slow-ci-success.txt)
- [example-release-warning-failure.txt](scripts/readiness-check/testdata/example-release-warning-failure.txt)

For setup and integration failures, use the Go troubleshooting guide in [troubleshooting.md](docs/go/operator/troubleshooting.md).

## Release Notes

Tagged GitHub Actions releases now publish the per-platform archives plus a SHA256SUMS manifest for verification. Signature-based release verification is intentionally deferred until key management is in place.

The current release/readiness checklist lives in [release-readiness.md](docs/go/operator/release-readiness.md).

