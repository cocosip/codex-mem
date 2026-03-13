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

## Safe Defaults

Recommended onboarding defaults:

- use non-destructive AGENTS installation
- prefer explicit system/project metadata
- do not enable broad related-project retrieval globally
- keep storage local-first
- do not rely on transcript import during first setup
