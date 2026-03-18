# Operator Docs

This directory is for deploying, registering, validating, and troubleshooting the Go server.

Start here:

- [Client Examples](./client-examples.md)
  Real MCP client registration examples for local stdio and remote HTTP.
- [Import Ingestion](./import-ingestion.md)
  JSONL batch ingestion through `ingest-imports`, checkpointed follow-mode ingestion through `follow-imports`, hygiene commands such as `cleanup-follow-imports` and `audit-follow-imports`, and `list-command-examples` discovery.
- [Release Readiness](./release-readiness.md)
  Packaging, readiness, and release checklist.
- [Troubleshooting](./troubleshooting.md)
  Setup, config, database, and MCP integration diagnosis.

If you only need normal day-to-day usage guidance, switch to [user/](../user/README.md).
If you are validating the source tree or changing implementation internals, switch to [maintainer/](../maintainer/README.md).
