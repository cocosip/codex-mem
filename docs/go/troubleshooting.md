# Go Troubleshooting

## Purpose

This document is the first-stop troubleshooting guide for the current Go implementation of `codex-mem`.

Audience:

- operators
- maintainers
- integrators debugging runtime or MCP setup issues

Use this when:

- `doctor` fails or reports unexpected readiness values
- config changes do not appear to take effect
- SQLite cannot be opened or migrated
- `serve` starts but an MCP client cannot initialize or call tools

Do not use this for:

- learning what mem is for
- normal day-to-day prompt usage
- release packaging policy decisions

## Fast Triage

Start with:

```powershell
go run ./cmd/codex-mem doctor
```

For scripts or CI:

```powershell
go run ./cmd/codex-mem doctor --json
```

Check these first:

- `config_file_used`
- `database`
- `journal_mode`
- `foreign_keys`
- `required_schema_ok`
- `fts_ready`
- `migrations_pending`
- `mcp_tool_count`

If `doctor` itself fails, move to the matching section below.

## Config Resolution Problems

### Symptom: config changes do not apply

Expected precedence is:

```text
defaults < config file < environment
```

What to check:

- `doctor` should show the effective `config_file_used` path.
- If `config_file_used=none`, the default file was not loaded.
- The default config location is `configs/codex-mem.json` under the current working directory.
- `CODEX_MEM_CONFIG_FILE` overrides the default discovery path.

Important behavior:

- If `CODEX_MEM_CONFIG_FILE` is relative, it is resolved relative to `configs/`, not the repository root.
- If `CODEX_MEM_CONFIG_FILE` is absolute, that exact file is used.

Examples:

```powershell
go run ./cmd/codex-mem doctor
```

```powershell
$env:CODEX_MEM_CONFIG_FILE="custom.toml"
go run ./cmd/codex-mem doctor
```

```powershell
$env:CODEX_MEM_CONFIG_FILE="D:\shared\codex-mem.toml"
go run ./cmd/codex-mem doctor
```

### Symptom: a config file is present but startup fails with `read config file`

Likely causes:

- the path in `CODEX_MEM_CONFIG_FILE` does not exist
- the file format is not valid for its extension
- the file contents are syntactically invalid

What to do:

1. Confirm the exact path from `doctor` or from the environment variable.
2. Remove `CODEX_MEM_CONFIG_FILE` temporarily and verify the default path works.
3. Start from [codex-mem.example.json](../../configs/codex-mem.example.json) and reapply changes incrementally.

### Symptom: startup fails with `invalid log_level`, `invalid busy_timeout_ms`, or another `invalid ...` message

These are validation failures in config parsing.

What to check:

- `log_level` must be one of `debug`, `info`, `warn`, or `error`
- numeric settings such as `busy_timeout_ms`, `log_max_size_mb`, `log_max_backups`, and `log_max_age_days` must be positive integers
- boolean settings such as `log_compress` and `log_stderr` must be valid boolean-like strings

Recommended recovery:

1. Remove environment overrides one by one.
2. Run `doctor` after each change.
3. Prefer a minimal config file first, then add optional settings back.

### Symptom: the wrong database path or system name is in use

The most common cause is an environment override still being set.

Check these variables:

- `CODEX_MEM_DB_PATH`
- `CODEX_MEM_SYSTEM_NAME`
- `CODEX_MEM_CONFIG_FILE`
- `CODEX_MEM_BUSY_TIMEOUT_MS`
- `CODEX_MEM_JOURNAL_MODE`
- `CODEX_MEM_LOG_LEVEL`
- `CODEX_MEM_LOG_FILE`

On PowerShell, inspect current overrides with:

```powershell
Get-ChildItem Env:CODEX_MEM_*
```

## Database Path and SQLite Problems

### Symptom: startup fails while creating the database directory

The database layer creates the parent directory automatically before opening SQLite.

Likely causes:

- the configured directory is not writable
- the path points into a protected location
- a parent segment is invalid for the current platform

What to do:

1. Check the effective `database` path from `doctor`.
2. Move the database under a writable project-local location such as `data/codex-mem.db`.
3. Re-run `go run ./cmd/codex-mem doctor`.

### Symptom: startup fails with `open sqlite database`, `ping sqlite database`, or `apply sqlite pragma`

These indicate the SQLite handle opened incorrectly or could not become usable.

Likely causes:

- the database file location is invalid or unwritable
- another process is holding the database in a conflicting state
- the configured driver name was changed away from `sqlite`

What to check:

- `sqlite_driver` should normally remain `sqlite`
- `database` should point to a valid local path or `:memory:`
- `journal_mode` should normally be `WAL`

Recommended recovery:

1. Reset to the default `sqlite` driver unless there is a very specific reason not to.
2. Point `CODEX_MEM_DB_PATH` to a fresh local file.
3. Run `doctor` again and confirm:
   `foreign_keys=true`
   `required_schema_ok=true`
   `fts_ready=true`
   `migrations_pending=0`

### Symptom: the database opens but readiness fields are wrong

Use the `doctor` fields as the decision table:

- `required_schema_ok=false`
  The database opened, but the expected schema objects are missing. Run `go run ./cmd/codex-mem migrate` or point back to the intended database file.
- `fts_ready=false`
  Migrations are incomplete or a different database file is being inspected.
- `migrations_pending>0`
  The current database has not applied all embedded migrations yet.
- `foreign_keys=false`
  SQLite pragmas did not apply as expected and the runtime should be treated as unhealthy.

## Logging and Runtime Visibility

### Symptom: there is no obvious runtime output

By default:

- structured logs go to the configured log file
- logs also go to stderr unless `log_stderr=false`

Check:

- `log_file`
- `log_stderr`
- `log_level`

Typical default log location:

```text
logs/codex-mem.log
```

If `serve` appears silent, that can be normal on stdout because stdout is reserved for MCP frames.

## MCP Server Startup and Client Integration Problems

### Symptom: `serve` runs, but the client cannot connect or initialize

The Go server uses MCP over stdio with framed JSON-RPC messages.

What the client must support:

- stdio transport
- `Content-Length` framed messages
- JSON-RPC `2.0`
- `initialize`
- `tools/list`
- `tools/call`

If a client expects newline-delimited JSON instead of framed stdio, initialization will fail.

### Symptom: the client reports parse errors or protocol errors

Likely causes:

- the client is not using stdio framing correctly
- the client is not sending `jsonrpc: "2.0"`
- the client is calling unsupported methods
- tool arguments contain unknown fields and are rejected during decode

What to do:

1. Confirm the client sends `initialize` first.
2. Confirm the request body is framed with `Content-Length`.
3. Confirm tool calls use the schemas exposed by `tools/list`.
4. If the client is custom, compare it against a known-good initialize and ping flow.

### Symptom: `tools/list` works, but a tool call fails with a decode error

Tool input decoding uses strict unknown-field rejection.

That means:

- misspelled property names fail
- extra properties fail
- wrong JSON types fail

Best recovery path:

1. Fetch the live schema from `tools/list`.
2. Trim the request down to only required fields.
3. Add optional fields back one at a time.

### Symptom: MCP capability checks look incomplete

Use `doctor` to confirm the server-side registration is healthy:

- `mcp_transport=stdio` in text mode
- `mcp.tool_count=9` in JSON mode

If tool count is lower than expected, treat it as a server construction regression instead of a client problem.

## Minimal Recovery Recipes

### Reset to default local config behavior

```powershell
Remove-Item Env:CODEX_MEM_CONFIG_FILE -ErrorAction SilentlyContinue
Remove-Item Env:CODEX_MEM_DB_PATH -ErrorAction SilentlyContinue
Remove-Item Env:CODEX_MEM_SYSTEM_NAME -ErrorAction SilentlyContinue
go run ./cmd/codex-mem doctor
```

### Test against a fresh database file

```powershell
$env:CODEX_MEM_DB_PATH="data\troubleshooting.db"
go run ./cmd/codex-mem doctor
```

### Verify MCP server startup in isolation

```powershell
go run ./cmd/codex-mem serve
```

Expected result:

- the process stays running
- no human-readable protocol output is printed to stdout
- logs, if enabled, appear in stderr and the configured log file

## When To Escalate Beyond This Guide

Move beyond this guide when:

- `doctor` succeeds but retrieval behavior still looks wrong
- the database is healthy but search or bootstrap ranking looks suspicious
- a specific MCP client still fails after stdio framing and schema checks pass

Those cases are more likely to need client-specific examples or richer retrieval and audit traces.
