# Codex Global Workflow

## Memory Workflow

- At the beginning of each fresh session, call `memory_bootstrap_session` with the current working directory and the task if it is already known.
- Use the returned `startup_brief` as the primary continuation context instead of trying to reconstruct prior state from raw history.
- During work, save a memory note only after important decisions, confirmed bugfixes, meaningful discoveries, durable constraints, or lasting user preferences that are likely to matter beyond the current task checkpoint.
- Before ending or pausing a session, save a handoff with the current task state, next steps, touched files, open questions, and risks, or when the user explicitly asks for a checkpoint/resume record.
- Do not save both in the same turn by default. Write both only when one artifact captures reusable long-term knowledge and the other captures task-specific continuation state.

## Memory Scope Safety

- Default all memory reads to the current project scope.
- Only use related-project memory when the task clearly requires cross-repository context.
- When related-project memory is used, label it clearly and do not treat it as if it came from the current repository.

## Memory Quality Rules

- Prefer concise, structured memory over verbose transcript-style notes.
- Do not save routine shell output, repetitive searches, or low-signal exploration as high-value memory.
- High-value memory includes decisions, bug root causes, confirmed fixes, architecture discoveries, unresolved todos, and important constraints.
