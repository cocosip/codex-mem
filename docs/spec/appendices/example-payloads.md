# Example Payloads

## Purpose

This appendix provides language-neutral example request and response shapes for the current `codex-mem` v1 MCP tools.

These examples illustrate intent and semantics. They are not tied to one implementation language.

## `memory_bootstrap_session`

### Example request

```json
{
  "cwd": "D:/Code/go/order-web",
  "task": "Continue fixing order submission validation",
  "branch_name": "fix/order-validation",
  "repo_remote": "git@github.com:example/order-web.git",
  "include_related_projects": true,
  "related_reason": "Frontend validation depends on backend API contract",
  "max_notes": 5,
  "max_handoffs": 2
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "scope": {
      "system_id": "sys_order_platform",
      "system_name": "order-platform",
      "project_id": "proj_order_web",
      "project_name": "order-web",
      "workspace_id": "ws_order_web_main",
      "workspace_root": "D:/Code/go/order-web",
      "branch_name": "fix/order-validation",
      "resolved_by": "repo_remote"
    },
    "session": {
      "session_id": "sess_20260313_001",
      "scope": {
        "system_id": "sys_order_platform",
        "project_id": "proj_order_web",
        "workspace_id": "ws_order_web_main"
      },
      "status": "active",
      "started_at": "2026-03-13T10:30:00Z",
      "task": "Continue fixing order submission validation"
    },
    "latest_handoff": {
      "handoff_id": "handoff_104",
      "scope": {
        "system_id": "sys_order_platform",
        "project_id": "proj_order_web",
        "workspace_id": "ws_order_web_main"
      },
      "session_id": "sess_20260312_021",
      "kind": "final",
      "task": "Fix order validation mismatch",
      "summary": "Frontend validation still diverges from backend enum handling for payment method.",
      "next_steps": [
        "Confirm backend accepted values",
        "Update client-side enum validation",
        "Retest submit flow"
      ],
      "status": "open",
      "created_at": "2026-03-12T18:45:00Z"
    },
    "recent_notes": [
      {
        "note_id": "note_331",
        "scope": {
          "system_id": "sys_order_platform",
          "project_id": "proj_order_web",
          "workspace_id": "ws_order_web_main"
        },
        "session_id": "sess_20260312_021",
        "type": "decision",
        "title": "Keep validation rules aligned with backend enum source",
        "content": "Frontend should consume generated enum metadata instead of maintaining a separate hard-coded list.",
        "importance": 4,
        "status": "active",
        "source": "codex_explicit",
        "created_at": "2026-03-12T18:20:00Z"
      }
    ],
    "related_notes": [
      {
        "note_id": "note_912",
        "scope": {
          "system_id": "sys_order_platform",
          "project_id": "proj_order_api",
          "workspace_id": "ws_order_api_main"
        },
        "session_id": "sess_20260311_007",
        "type": "bugfix",
        "title": "Backend now rejects legacy payment aliases",
        "content": "The API accepts only canonical enum values after the March validation cleanup.",
        "importance": 4,
        "status": "active",
        "source": "codex_explicit",
        "created_at": "2026-03-11T09:15:00Z",
        "relation_type": "calls_api_of"
      }
    ],
    "startup_brief": {
      "current_task": "Continue fixing order submission validation",
      "last_known_state": "Frontend validation is still using stale enum assumptions.",
      "important_decisions": [
        "Validation should follow backend canonical enum definitions."
      ],
      "open_todos": [
        "Confirm accepted backend values",
        "Replace hard-coded validation list",
        "Retest submit flow"
      ],
      "risks": [
        "Backend rejects legacy aliases silently in older client flows."
      ],
      "touched_files": [
        "src/order/validation.ts",
        "src/order/submit.ts"
      ],
      "related_context": [
        "Backend API removed support for legacy payment aliases."
      ]
    }
  },
  "warnings": []
}
```

## `memory_resolve_scope`

### Example request

```json
{
  "cwd": "D:/Code/go/order-web",
  "branch_name": "fix/order-validation",
  "repo_remote": "git@github.com:example/order-web.git",
  "project_name_hint": "order-web",
  "system_name_hint": "order-platform"
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "scope": {
      "system_id": "sys_order_platform",
      "system_name": "order-platform",
      "project_id": "proj_order_web",
      "project_name": "order-web",
      "workspace_id": "ws_order_web_main",
      "workspace_root": "D:/Code/go/order-web",
      "branch_name": "fix/order-validation",
      "resolved_by": "repo_remote"
    },
    "resolved_by": "repo_remote"
  },
  "warnings": []
}
```

## `memory_start_session`

### Example request

```json
{
  "scope": {
    "system_id": "sys_order_platform",
    "system_name": "order-platform",
    "project_id": "proj_order_web",
    "project_name": "order-web",
    "workspace_id": "ws_order_web_main",
    "workspace_root": "D:/Code/go/order-web",
    "branch_name": "fix/order-validation",
    "resolved_by": "repo_remote"
  },
  "task": "Continue fixing order submission validation",
  "branch_name": "fix/order-validation"
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "session": {
      "session_id": "sess_20260313_001",
      "scope": {
        "system_id": "sys_order_platform",
        "project_id": "proj_order_web",
        "workspace_id": "ws_order_web_main"
      },
      "status": "active",
      "task": "Continue fixing order submission validation",
      "branch_name": "fix/order-validation",
      "started_at": "2026-03-13T10:30:00Z"
    }
  },
  "warnings": []
}
```

## `memory_save_note`

### Example request

```json
{
  "scope": {
    "system_id": "sys_order_platform",
    "project_id": "proj_order_web",
    "workspace_id": "ws_order_web_main"
  },
  "session_id": "sess_20260313_001",
  "type": "bugfix",
  "title": "Order validation now uses generated backend enum list",
  "content": "Client-side validation now reads generated enum metadata instead of maintaining a stale manual list.",
  "importance": 4,
  "tags": ["validation", "frontend", "api"],
  "file_paths": ["src/order/validation.ts"]
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "note": {
      "note_id": "note_402",
      "scope": {
        "system_id": "sys_order_platform",
        "project_id": "proj_order_web",
        "workspace_id": "ws_order_web_main"
      },
      "session_id": "sess_20260313_001",
      "type": "bugfix",
      "title": "Order validation now uses generated backend enum list",
      "content": "Client-side validation now reads generated enum metadata instead of maintaining a stale manual list.",
      "importance": 4,
      "status": "active",
      "source": "codex_explicit",
      "created_at": "2026-03-13T10:52:00Z"
    },
    "stored_at": "2026-03-13T10:52:00Z",
    "deduplicated": false
  },
  "warnings": []
}
```

## `memory_save_handoff`

### Example request

```json
{
  "scope": {
    "system_id": "sys_order_platform",
    "project_id": "proj_order_web",
    "workspace_id": "ws_order_web_main"
  },
  "session_id": "sess_20260313_001",
  "kind": "final",
  "task": "Fix order validation mismatch",
  "summary": "Frontend validation was updated to use generated enum metadata and submit flow now matches backend expectations.",
  "completed": [
    "Removed stale hard-coded payment enum list",
    "Rewired validation to generated metadata"
  ],
  "next_steps": [
    "Run regression test on saved draft checkout flow",
    "Confirm no legacy alias references remain"
  ],
  "open_questions": [
    "Do older cached clients still submit legacy aliases?"
  ],
  "risks": [
    "Draft restore flow may still inject old values."
  ],
  "files_touched": [
    "src/order/validation.ts",
    "src/order/submit.ts"
  ],
  "related_note_ids": ["note_402"],
  "status": "open"
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "handoff": {
      "handoff_id": "handoff_115",
      "scope": {
        "system_id": "sys_order_platform",
        "project_id": "proj_order_web",
        "workspace_id": "ws_order_web_main"
      },
      "session_id": "sess_20260313_001",
      "kind": "final",
      "task": "Fix order validation mismatch",
      "summary": "Frontend validation was updated to use generated enum metadata and submit flow now matches backend expectations.",
      "next_steps": [
        "Run regression test on saved draft checkout flow",
        "Confirm no legacy alias references remain"
      ],
      "status": "open",
      "created_at": "2026-03-13T11:10:00Z"
    },
    "stored_at": "2026-03-13T11:10:00Z",
    "eligible_for_bootstrap": true
  },
  "warnings": []
}
```

## `memory_save_import`

### Example request

```json
{
  "scope": {
    "system_id": "sys_order_platform",
    "project_id": "proj_order_web",
    "workspace_id": "ws_order_web_main"
  },
  "session_id": "sess_20260313_001",
  "source": "watcher_import",
  "external_id": "watcher:event:392",
  "payload_hash": "sha256:4db5b7c6f2a9",
  "privacy_intent": ""
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "import": {
      "import_id": "import_701",
      "scope": {
        "system_id": "sys_order_platform",
        "project_id": "proj_order_web",
        "workspace_id": "ws_order_web_main"
      },
      "session_id": "sess_20260313_001",
      "source": "watcher_import",
      "external_id": "watcher:event:392",
      "payload_hash": "sha256:4db5b7c6f2a9",
      "suppressed": false,
      "imported_at": "2026-03-13T11:18:00Z"
    },
    "stored_at": "2026-03-13T11:18:00Z",
    "suppressed": false,
    "deduplicated": false
  },
  "warnings": []
}
```

Replay of the same imported artifact or a privacy-blocked import still returns `ok: true`, but the payload flips to `suppressed: true` and includes warning visibility instead of silently writing another durable record.

## `memory_save_imported_note`

### Example request

```json
{
  "scope": {
    "system_id": "sys_order_platform",
    "project_id": "proj_order_web",
    "workspace_id": "ws_order_web_main"
  },
  "session_id": "sess_20260313_001",
  "source": "watcher_import",
  "external_id": "watcher:event:393",
  "payload_hash": "sha256:b7f650bfe12c",
  "type": "discovery",
  "title": "Watcher captured checkout retry regression",
  "content": "A local watcher run showed the checkout retry button still posts a legacy payment alias after draft restore.",
  "importance": 4,
  "tags": ["watcher", "checkout", "validation"],
  "file_paths": ["src/order/checkout.ts"],
  "status": "active"
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "note": {
      "note_id": "note_488",
      "scope": {
        "system_id": "sys_order_platform",
        "project_id": "proj_order_web",
        "workspace_id": "ws_order_web_main"
      },
      "session_id": "sess_20260313_001",
      "type": "discovery",
      "title": "Watcher captured checkout retry regression",
      "content": "A local watcher run showed the checkout retry button still posts a legacy payment alias after draft restore.",
      "importance": 4,
      "status": "active",
      "source": "watcher_import",
      "created_at": "2026-03-13T11:24:00Z"
    },
    "import": {
      "import_id": "import_702",
      "scope": {
        "system_id": "sys_order_platform",
        "project_id": "proj_order_web",
        "workspace_id": "ws_order_web_main"
      },
      "session_id": "sess_20260313_001",
      "source": "watcher_import",
      "external_id": "watcher:event:393",
      "payload_hash": "sha256:b7f650bfe12c",
      "durable_memory_id": "note_488",
      "suppressed": false,
      "imported_at": "2026-03-13T11:24:00Z"
    },
    "materialized": true,
    "note_deduplicated": false,
    "import_deduplicated": false,
    "suppressed": false
  },
  "warnings": []
}
```

When explicit project memory already covers the same artifact, or privacy rules exclude durable storage, the response still preserves the import audit while `materialized` becomes `false` and `suppressed` becomes `true`.

## `memory_search`

### Example request

```json
{
  "query": "payment enum validation",
  "scope": {
    "system_id": "sys_order_platform",
    "project_id": "proj_order_web",
    "workspace_id": "ws_order_web_main"
  },
  "types": ["decision", "bugfix"],
  "min_importance": 3,
  "limit": 5,
  "include_handoffs": true,
  "include_related_projects": false,
  "intent": "bugfix"
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "results": [
      {
        "kind": "note",
        "id": "note_402",
        "scope": {
          "system_id": "sys_order_platform",
          "project_id": "proj_order_web",
          "workspace_id": "ws_order_web_main"
        },
        "state": "active",
        "title": "Order validation now uses generated backend enum list",
        "summary": "Frontend validation now derives accepted values from generated backend metadata.",
        "importance": 4,
        "created_at": "2026-03-13T10:52:00Z"
      },
      {
        "kind": "handoff",
        "id": "handoff_115",
        "scope": {
          "system_id": "sys_order_platform",
          "project_id": "proj_order_web",
          "workspace_id": "ws_order_web_main"
        },
        "state": "open",
        "title": "Fix order validation mismatch",
        "summary": "Validation logic has been aligned, but draft checkout regression still needs confirmation.",
        "importance": 4,
        "created_at": "2026-03-13T11:10:00Z"
      }
    ]
  },
  "warnings": []
}
```

## `memory_get_recent`

### Example request

```json
{
  "scope": {
    "system_id": "sys_order_platform",
    "project_id": "proj_order_web",
    "workspace_id": "ws_order_web_main"
  },
  "limit": 3,
  "include_handoffs": true,
  "include_notes": true,
  "include_related_projects": false
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "handoffs": [
      {
        "handoff_id": "handoff_115",
        "scope": {
          "system_id": "sys_order_platform",
          "project_id": "proj_order_web",
          "workspace_id": "ws_order_web_main"
        },
        "session_id": "sess_20260313_001",
        "kind": "final",
        "task": "Fix order validation mismatch",
        "summary": "Validation logic has been aligned, but draft checkout regression still needs confirmation.",
        "next_steps": [
          "Run regression test on saved draft checkout flow",
          "Confirm no legacy alias references remain"
        ],
        "status": "open",
        "created_at": "2026-03-13T11:10:00Z"
      }
    ],
    "notes": [
      {
        "note_id": "note_488",
        "scope": {
          "system_id": "sys_order_platform",
          "project_id": "proj_order_web",
          "workspace_id": "ws_order_web_main"
        },
        "session_id": "sess_20260313_001",
        "type": "discovery",
        "title": "Watcher captured checkout retry regression",
        "content": "A local watcher run showed the checkout retry button still posts a legacy payment alias after draft restore.",
        "importance": 4,
        "status": "active",
        "source": "watcher_import",
        "created_at": "2026-03-13T11:24:00Z"
      },
      {
        "note_id": "note_402",
        "scope": {
          "system_id": "sys_order_platform",
          "project_id": "proj_order_web",
          "workspace_id": "ws_order_web_main"
        },
        "session_id": "sess_20260313_001",
        "type": "bugfix",
        "title": "Order validation now uses generated backend enum list",
        "content": "Client-side validation now reads generated enum metadata instead of maintaining a stale manual list.",
        "importance": 4,
        "status": "active",
        "source": "codex_explicit",
        "created_at": "2026-03-13T10:52:00Z"
      }
    ]
  },
  "warnings": []
}
```

## `memory_get_note`

### Example request

```json
{
  "id": "note_488",
  "kind": "note"
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "record": {
      "note_id": "note_488",
      "scope": {
        "system_id": "sys_order_platform",
        "project_id": "proj_order_web",
        "workspace_id": "ws_order_web_main"
      },
      "session_id": "sess_20260313_001",
      "type": "discovery",
      "title": "Watcher captured checkout retry regression",
      "content": "A local watcher run showed the checkout retry button still posts a legacy payment alias after draft restore.",
      "importance": 4,
      "status": "active",
      "source": "watcher_import",
      "created_at": "2026-03-13T11:24:00Z"
    }
  },
  "warnings": []
}
```

## `memory_install_agents`

### Example request

```json
{
  "target": "project",
  "mode": "safe",
  "cwd": "D:/Code/go/order-web",
  "project_name": "order-web",
  "system_name": "order-platform",
  "related_repositories": [
    "order-api",
    "order-worker"
  ],
  "preferred_tags": [
    "spec",
    "api",
    "go"
  ],
  "allow_related_project_memory": true
}
```

### Example response

```json
{
  "ok": true,
  "data": {
    "written_files": [
      {
        "path": "D:/Code/go/order-web/AGENTS.md",
        "target": "project",
        "mode": "safe"
      }
    ],
    "skipped_files": []
  },
  "warnings": []
}
```
