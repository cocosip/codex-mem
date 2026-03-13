# Tool Contracts

## Contract Rules

All tools in `codex-mem` v1:

- are scope-aware
- support success, warning, and error semantics
- must degrade gracefully when prior memory is absent
- must preserve privacy and scope boundaries

## memory_bootstrap_session

Purpose:

- start a new session and recover the most relevant prior context

Required input:

- `cwd`

Optional input:

- `task`
- `branch_name`
- `repo_remote`
- `include_related_projects`
- `related_reason`
- `max_notes`
- `max_handoffs`

Required output:

- `scope`
- `session`
- `latest_handoff`
- `recent_notes`
- `related_notes`
- `startup_brief`
- `warnings`

Behavior:

- resolve scope
- create a fresh active session
- retrieve latest relevant handoff
- retrieve recent high-value notes
- optionally retrieve related-project notes
- synthesize startup brief

Rules:

- Must succeed in an empty store.
- Must not reactivate an ended session.

## memory_resolve_scope

Purpose:

- resolve system, project, and workspace identity from local context

Required input:

- `cwd`

Optional input:

- `branch_name`
- `repo_remote`
- `project_name_hint`
- `system_name_hint`

Required output:

- `scope`
- `resolved_by`
- `warnings`

Rules:

- Prefer stable identity over local naming.
- Warn when only weak inference is available.

## memory_start_session

Purpose:

- create a new active session without full bootstrap

Required input:

- `scope`

Optional input:

- `task`
- `branch_name`

Required output:

- `session`
- `warnings`

Rules:

- Always create a fresh session.

## memory_save_note

Purpose:

- persist a high-value structured memory note

Required input:

- `scope`
- `session_id`
- `type`
- `title`
- `content`
- `importance`

Optional input:

- `tags`
- `file_paths`
- `related_project_ids`
- `status`
- `source`

Required output:

- `note`
- `stored_at`
- `deduplicated`
- `warnings`

Rules:

- Reject invalid type or importance.
- Reject scope/session inconsistencies.
- Preserve provenance.

## memory_save_handoff

Purpose:

- persist a checkpoint or end-of-session continuation record

Required input:

- `scope`
- `session_id`
- `kind`
- `task`
- `summary`
- `next_steps`
- `status`

Optional input:

- `completed`
- `open_questions`
- `risks`
- `files_touched`
- `related_note_ids`

Required output:

- `handoff`
- `stored_at`
- `eligible_for_bootstrap`
- `warnings`

Rules:

- Require actionable `next_steps`.
- Validate kind and status.
- Warn instead of silently overwriting same-task open handoffs.

## memory_search

Purpose:

- search durable memory within a scoped boundary

Required input:

- `query`
- `scope`

Optional input:

- `types`
- `states`
- `min_importance`
- `limit`
- `include_handoffs`
- `include_related_projects`
- `intent`

Required output:

- `results`
- `warnings`

Rules:

- Default to current-project retrieval.
- Zero results is success.
- Cross-project results must be labeled.

## memory_get_recent

Purpose:

- fetch recent scoped activity without a free-text query

Required input:

- `scope`

Optional input:

- `limit`
- `include_handoffs`
- `include_notes`
- `include_related_projects`

Required output:

- `handoffs`
- `notes`
- `warnings`

## memory_get_note

Purpose:

- fetch one note or handoff in full detail by id

Required input:

- `id`
- `kind`

Required output:

- `record`
- `warnings`

Rules:

- Missing object by id is an error.

## memory_install_agents

Purpose:

- install or update `AGENTS.md` templates

Required input:

- `target`
- `mode`

Optional input:

- `cwd`
- `project_name`
- `system_name`
- `related_repositories`
- `preferred_tags`
- `allow_related_project_memory`

Required output:

- `written_files`
- `skipped_files`
- `warnings`

Rules:

- Default to non-destructive behavior.
- Must report exactly what changed or was skipped.
