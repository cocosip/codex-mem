# Domain Model

## Purpose

This document defines the core domain entities for `codex-mem` v1.

## Entity Hierarchy

Required scope hierarchy:

1. `System`
2. `Project`
3. `Workspace`
4. `Session`

Durable memory entities:

- `MemoryNote`
- `Handoff`
- `ProjectRelation`
- `ImportRecord`

## System

A `System` is the largest normal memory boundary below a global user layer.

Rules:

- Every project belongs to exactly one system.
- Cross-project retrieval is only meaningful within a system.
- Cross-system retrieval must be explicit.

## Project

A `Project` is one logical codebase or repository inside a system.

Rules:

- Default memory retrieval is limited to the current project.
- A project may have multiple workspaces.
- A project is not the same thing as a local folder name.

## Workspace

A `Workspace` is one concrete local working copy of a project.

Examples:

- one local clone
- one worktree
- one checkout in a specific directory

Rules:

- Every workspace belongs to exactly one project.
- Workspace-local retrieval outranks project-level retrieval.

## Session

A `Session` is one Codex interaction lifecycle bound to a workspace.

Rules:

- A new session always gets a new `session_id`.
- Every session belongs to one workspace, one project, and one system.
- Ended sessions remain historical records.

## MemoryNote

A `MemoryNote` is a high-value structured memory item.

Examples:

- decisions
- bug root causes
- confirmed bugfix summaries
- discoveries
- constraints
- preferences
- todos

Rules:

- Every note must be linked to one session and one scope chain.
- Only durable and reusable information belongs in the main memory index.
- Notes are not raw transcript events.

## Handoff

A `Handoff` is a structured continuation record.

Kinds:

- `final`
- `checkpoint`
- `recovery`

Rules:

- Every handoff must be linked to one session and one scope chain.
- Any handoff intended for future continuation must include actionable `next_steps`.
- Bootstrap prefers the latest open handoff before older notes.

## ProjectRelation

A `ProjectRelation` is an explicit relationship between two projects in the same system.

Examples:

- frontend depends on backend API
- SDK generated from service schema
- deployment tied to service repo

Rules:

- Related-project retrieval is guided by explicit relations when available.
- Relations do not permit default mixing of all project memory.

## ImportRecord

An `ImportRecord` tracks imported data from secondary sources.

Rules:

- Imports are secondary evidence, not the primary source of truth.
- Import records support dedupe, provenance, and future enrichment workflows.
