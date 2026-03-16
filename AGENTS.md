# Project Workflow

## Project Identity

- Project name: codex-mem
- System name: codex-mem
- Default memory scope: current project

## Memory Rules

- At the start of a fresh session in this repository, call `memory_bootstrap_session`.
- Save a memory note when work produces a lasting decision, bugfix insight, reusable discovery, or durable implementation constraint.
- Save a handoff before pausing, switching tasks, or ending the session.

## Related Project Policy

- Related-project memory is allowed only when the task clearly depends on another repository in the same system.
- Typical examples include API contracts, schema changes, generated clients, deployment coordination, and integration debugging.
- Do not pull memory from unrelated projects by default.

## Preferred Tags

- Use tags where useful, especially: spec, mcp, sqlite, go

## Project-Specific Notes

- Treat `docs/spec/` as the normative v1 reference.
- Treat `docs/go/maintainer/implementation-backlog.md` as the language-neutral execution plan.
- Treat `docs/go/maintainer/implementation-plan.md` as the Go-oriented engineering plan.
- Treat `docs/go/maintainer/development-tracker.md` as the current Go execution tracker.
- If the task is to begin implementation work, read `docs/go/maintainer/dev-kickoff.md` first.
- Keep durable memory focused on high-value notes and handoffs rather than transcript-style logs.

## System Relationships

- This repository belongs to system: codex-mem
- Related repositories may include: none currently declared
- Use related-project memory only when the current task depends on one of those repositories.

## Cross-Repo Memory Rules

- Prefer current-project memory first.
- Expand to related repositories only for integration-relevant work.
- When using related-project memory, mention the source repository explicitly in your reasoning and outputs.
