# Project Workflow

## Project Identity

- Project name: <project-name>
- System name: <system-name>
- Default memory scope: current project

## Memory Rules

- At the start of a fresh session in this repository, call `memory_bootstrap_session`.
- Save a memory note only when work produces a lasting decision, bugfix insight, reusable discovery, or durable implementation constraint that is likely to matter beyond the current task checkpoint.
- Save a handoff only before pausing, switching tasks, ending the session, or when the user explicitly asks for a checkpoint or resume record.
- Do not save both in the same turn by default. Write both only when one artifact captures reusable long-term knowledge and the other captures task-specific continuation state.

## Related Project Policy

- Related-project memory is allowed only when the task clearly depends on another repository in the same system.
- Typical examples include API contracts, schema changes, generated clients, deployment coordination, and integration debugging.
- Do not pull memory from unrelated projects by default.

## Preferred Tags

- Use tags where useful, especially: <tag-1>, <tag-2>, <tag-3>

## Project-Specific Notes

- Add any repository-specific workflow expectations here.
- Add naming conventions for modules or domains here if they improve memory quality.

## System Relationships

- This repository belongs to system: <system-name>
- Related repositories may include: <repo-a>, <repo-b>, <repo-c>
- Use related-project memory only when the current task depends on one of those repositories.

## Cross-Repo Memory Rules

- Prefer current-project memory first.
- Expand to related repositories only for integration-relevant work.
- When using related-project memory, mention the source repository explicitly in your reasoning and outputs.
