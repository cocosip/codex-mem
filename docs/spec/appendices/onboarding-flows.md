# Onboarding Flows

## Purpose

This appendix describes recommended first-run onboarding flows for `codex-mem` v1.

## Single-Repository Onboarding

### Goal

Enable `codex-mem` for one repository with safe defaults.

### Recommended flow

1. Install global AGENTS template in safe mode if missing.
2. Install project AGENTS template in the repository if missing.
3. Resolve or declare:
   - `system_name`
   - `project_name`
4. Keep related-project retrieval disabled unless the repo clearly belongs to a multi-repo system.
5. Start the first session with `memory_bootstrap_session`.
6. Save at least one note and one handoff during early usage to seed continuity.

### Expected outcome

- The project is identified correctly.
- Session bootstrap works even if there is no prior memory.
- Future sessions can recover continuity once notes and handoffs exist.

## Multi-Repository System Onboarding

### Goal

Enable `codex-mem` for a group of related repositories under one system.

### Recommended flow

1. Install global AGENTS template in safe mode if missing.
2. For each repository, install a project AGENTS template.
3. Declare the same `system_name` across those repositories.
4. Declare distinct `project_name` values per repository.
5. Register known related repositories and relation types where available.
6. Keep related-project retrieval opt-in by default.
7. Bootstrap and seed memory independently in each repository.

### Expected outcome

- Repositories remain isolated by default.
- Controlled cross-project retrieval becomes possible when explicitly requested.
- Shared system identity does not collapse project boundaries.

## Imported Artifact Onboarding

### Goal

Enable watcher or relay artifacts to enter the same scoped memory workflow without bypassing privacy, provenance, or explicit-memory precedence rules.

### Recommended flow

1. Start with ordinary repository onboarding and bootstrap first, so the destination scope is already stable.
2. Decide whether the upstream feed is best represented as `watcher_import` or `relay_import`.
3. Ensure each imported artifact carries at least one durable dedupe key such as `external_id` or `payload_hash`.
4. Use `memory_save_import` when you only need durable import-audit provenance for one artifact.
5. Use `memory_save_imported_note` when the imported artifact should also materialize into searchable durable note memory.
6. For operator-managed JSONL feeds, begin with `ingest-imports --audit-only` before enabling materialization or long-lived `follow-imports`.
7. Keep cleanup and audit of follow-mode sidecars explicit through `cleanup-follow-imports`, `audit-follow-imports`, and `doctor` rather than deleting artifacts implicitly.

### Expected outcome

- Imported artifacts are deduplicated and traceable.
- Privacy-blocked imports remain visible in audit history without re-entering durable searchable memory.
- Stronger explicit memory still wins over weaker imported duplicates in the same project.

## Safe Defaults

Recommended onboarding defaults:

- use non-destructive AGENTS installation
- prefer explicit system/project metadata
- do not enable broad related-project retrieval globally
- keep storage local-first
- do not rely on transcript import during first setup
