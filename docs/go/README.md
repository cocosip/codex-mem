# Go Docs

This directory contains Go-specific documents for the `codex-mem` implementation.

The most useful way to read these docs is by audience, not by filename.

## If You Are Using `codex-mem` Day To Day

Read these first:

- [how-memory-works.md](./how-memory-works.md)
  What mem is for, what it stores, when scope matters, and how cross-project lookup works.
- [prompt-examples.md](./prompt-examples.md)
  Natural-language prompts you can send to Codex.
- [client-examples.md](./client-examples.md)
  How to register the packaged binary with a real MCP client such as Codex CLI.

These are the normal user-facing docs.

## If You Are Deploying Or Operating The MCP Server

Read these:

- [client-examples.md](./client-examples.md)
  Real client registration examples for local stdio and remote HTTP.
- [release-readiness.md](./release-readiness.md)
  Packaging, release, and readiness checklist.
- [troubleshooting.md](./troubleshooting.md)
  Setup and runtime failure diagnosis.

## If You Are Maintaining Or Testing The Go Implementation

Read these:

- [mcp-integration.md](./mcp-integration.md)
  Maintainer-oriented MCP transport and smoke-test guide from the source tree.
- [development-tracker.md](./development-tracker.md)
  Current execution tracker and recent implementation history.
- [dev-kickoff.md](./dev-kickoff.md)
  Fastest re-entry point for a new implementation session.
- [implementation-plan.md](./implementation-plan.md)
  Go-oriented architecture and implementation plan.

## Planning And Spec References

Language-neutral planning remains here:

- [Implementation Backlog](../implementation-backlog.md)

Normative product specification remains here:

- [Spec Index](../spec/README.md)
