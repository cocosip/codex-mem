# AGENTS Policy

## Purpose

`AGENTS.md` provides stable workflow guidance for Codex. It is not the primary dynamic memory store.

## What AGENTS.md Is For

Use `AGENTS.md` for:

- when Codex should call memory tools
- what categories of events should be saved as notes
- when handoff must be written
- related-project retrieval guardrails
- project-specific workflow expectations

## What AGENTS.md Is Not For

Do not use `AGENTS.md` for:

- current task progress
- latest handoff contents
- dynamic session summaries
- volatile transcript-derived state

Those belong in durable memory storage and MCP retrieval.

## Layering

Recommended layering:

- global `AGENTS.md` for cross-project workflow rules
- project `AGENTS.md` for repo-specific rules

## Global Template Role

The global template should define:

- bootstrap at session start
- save note at important milestones
- save handoff before pausing or ending
- avoid unrelated cross-project retrieval

## Project Template Role

The project template should define:

- project identity
- system identity
- preferred tags
- related-project policy
- repository-specific guidance

## Installation Policy

`memory_install_agents` should support:

- global installation
- project installation
- both

Default mode:

- non-destructive creation if missing

Safe optional modes:

- append block
- explicit overwrite

Rules:

- Do not silently overwrite existing `AGENTS.md` in safe/default mode.
- Preserve unresolved placeholders rather than inventing false values.

## Hard Rule

`AGENTS.md` may shape behavior, but it must not override privacy exclusions or durable memory integrity rules.

## Cache-Friendly Guidance

`AGENTS.md` should be treated as a stable instruction layer, not a frequently changing runtime state channel.

Reason:

- prompt caching depends on stable prompt prefixes
- frequent changes to instruction-like content can reduce cache reuse

Recommended rules:

- keep `AGENTS.md` content stable once installed
- use AGENTS updates mainly for setup, explicit policy changes, or rare project-level workflow changes
- do not write fast-changing handoff data, startup briefs, or session summaries into `AGENTS.md`
- keep dynamic continuity data in durable memory storage and expose it through MCP tools instead

Implication for `memory_install_agents` or any AGENTS-writing automation:

- default behavior should be low-frequency and non-destructive
- AGENTS-writing features should not be used as a normal per-session update mechanism
