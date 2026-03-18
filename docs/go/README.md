# Go Docs

This directory contains Go-specific documents for the `codex-mem` implementation.

These docs are now physically grouped by audience, not just linked that way from a flat index.

## Audience Directories

- [user/](./user/README.md)
  End-user usage guidance, memory concepts, and prompt examples.
- [operator/](./operator/README.md)
  Client registration, deployment/readiness, and troubleshooting.
- [maintainer/](./maintainer/README.md)
  Source-tree integration, implementation planning, and development tracking.

## Suggested Starting Points

If you are using `codex-mem` day to day:

- [How Memory Works](./user/how-memory-works.md)
- [Prompt Examples](./user/prompt-examples.md)

If you are deploying or operating the MCP server:

- [Client Examples](./operator/client-examples.md)
- [Import Ingestion](./operator/import-ingestion.md)
  Batch import, long-lived follow mode, cleanup/audit hygiene commands, and packaged example-catalog lookup.
- [Release Readiness](./operator/release-readiness.md)
- [Troubleshooting](./operator/troubleshooting.md)

If you are maintaining or testing the Go implementation:

- [MCP Integration](./maintainer/mcp-integration.md)
  Stdio/HTTP transport behavior, current 11-tool MCP surface, and source-tree smoke tests.
- [Development Tracker](./maintainer/development-tracker.md)
- [Development Kickoff](./maintainer/dev-kickoff.md)
- [Implementation Plan](./maintainer/implementation-plan.md)

## Planning And Spec References

Go planning now lives under `maintainer/`:

- [Implementation Backlog](./maintainer/implementation-backlog.md)

Normative product specification remains here:

- [Spec Index](../spec/README.md)
