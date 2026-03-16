package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"codex-mem/internal/domain/agents"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/imports"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/retrieval"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

type toolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	call        func(context.Context, json.RawMessage) (toolCallResult, error)
}

type scopePayload struct {
	SystemID      string `json:"system_id"`
	SystemName    string `json:"system_name,omitempty"`
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name,omitempty"`
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceRoot string `json:"workspace_root,omitempty"`
	BranchName    string `json:"branch_name,omitempty"`
	ResolvedBy    string `json:"resolved_by,omitempty"`
}

func (p scopePayload) ref() scope.Ref {
	return scope.Ref{
		SystemID:    p.SystemID,
		ProjectID:   p.ProjectID,
		WorkspaceID: p.WorkspaceID,
	}
}

func (p scopePayload) full() scope.Scope {
	return scope.Scope{
		SystemID:      p.SystemID,
		SystemName:    p.SystemName,
		ProjectID:     p.ProjectID,
		ProjectName:   p.ProjectName,
		WorkspaceID:   p.WorkspaceID,
		WorkspaceRoot: p.WorkspaceRoot,
		BranchName:    p.BranchName,
		ResolvedBy:    p.ResolvedBy,
	}
}

type resolveScopeRequest struct {
	CWD             string `json:"cwd"`
	BranchName      string `json:"branch_name,omitempty"`
	RepoRemote      string `json:"repo_remote,omitempty"`
	ProjectNameHint string `json:"project_name_hint,omitempty"`
	SystemNameHint  string `json:"system_name_hint,omitempty"`
}

type startSessionRequest struct {
	Scope      scopePayload `json:"scope"`
	Task       string       `json:"task,omitempty"`
	BranchName string       `json:"branch_name,omitempty"`
}

type saveNoteRequest struct {
	Scope             scopePayload `json:"scope"`
	SessionID         string       `json:"session_id"`
	Type              string       `json:"type"`
	Title             string       `json:"title"`
	Content           string       `json:"content"`
	Importance        int          `json:"importance"`
	Tags              []string     `json:"tags,omitempty"`
	FilePaths         []string     `json:"file_paths,omitempty"`
	RelatedProjectIDs []string     `json:"related_project_ids,omitempty"`
	Status            string       `json:"status,omitempty"`
	Source            string       `json:"source,omitempty"`
	PrivacyIntent     string       `json:"privacy_intent,omitempty"`
}

type saveHandoffRequest struct {
	Scope          scopePayload `json:"scope"`
	SessionID      string       `json:"session_id"`
	Kind           string       `json:"kind"`
	Task           string       `json:"task"`
	Summary        string       `json:"summary"`
	Completed      []string     `json:"completed,omitempty"`
	NextSteps      []string     `json:"next_steps"`
	OpenQuestions  []string     `json:"open_questions,omitempty"`
	Risks          []string     `json:"risks,omitempty"`
	FilesTouched   []string     `json:"files_touched,omitempty"`
	RelatedNoteIDs []string     `json:"related_note_ids,omitempty"`
	Status         string       `json:"status"`
	PrivacyIntent  string       `json:"privacy_intent,omitempty"`
}

type saveImportRequest struct {
	Scope           scopePayload `json:"scope"`
	SessionID       string       `json:"session_id"`
	Source          string       `json:"source"`
	ExternalID      string       `json:"external_id,omitempty"`
	PayloadHash     string       `json:"payload_hash,omitempty"`
	DurableMemoryID string       `json:"durable_memory_id,omitempty"`
	PrivacyIntent   string       `json:"privacy_intent,omitempty"`
}

type bootstrapRequest struct {
	CWD                    string `json:"cwd"`
	Task                   string `json:"task,omitempty"`
	BranchName             string `json:"branch_name,omitempty"`
	RepoRemote             string `json:"repo_remote,omitempty"`
	IncludeRelatedProjects bool   `json:"include_related_projects,omitempty"`
	RelatedReason          string `json:"related_reason,omitempty"`
	MaxNotes               int    `json:"max_notes,omitempty"`
	MaxHandoffs            int    `json:"max_handoffs,omitempty"`
}

type getRecentRequest struct {
	Scope                  scopePayload `json:"scope"`
	Limit                  int          `json:"limit,omitempty"`
	IncludeHandoffs        bool         `json:"include_handoffs,omitempty"`
	IncludeNotes           bool         `json:"include_notes,omitempty"`
	IncludeRelatedProjects bool         `json:"include_related_projects,omitempty"`
}

type getNoteRequest struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type searchRequest struct {
	Query                  string       `json:"query"`
	Scope                  scopePayload `json:"scope"`
	Types                  []string     `json:"types,omitempty"`
	States                 []string     `json:"states,omitempty"`
	MinImportance          int          `json:"min_importance,omitempty"`
	Limit                  int          `json:"limit,omitempty"`
	IncludeHandoffs        bool         `json:"include_handoffs,omitempty"`
	IncludeRelatedProjects bool         `json:"include_related_projects,omitempty"`
	Intent                 string       `json:"intent,omitempty"`
}

type installAgentsRequest struct {
	Target                    string   `json:"target"`
	Mode                      string   `json:"mode"`
	CWD                       string   `json:"cwd,omitempty"`
	ProjectName               string   `json:"project_name,omitempty"`
	SystemName                string   `json:"system_name,omitempty"`
	RelatedRepositories       []string `json:"related_repositories,omitempty"`
	PreferredTags             []string `json:"preferred_tags,omitempty"`
	AllowRelatedProjectMemory *bool    `json:"allow_related_project_memory,omitempty"`
}

// ToolCount returns the number of tools exposed by the codex-mem MCP surface.
func ToolCount() int {
	return len(buildTools(nil))
}

func buildTools(handlers *Handlers) []toolDefinition {
	return []toolDefinition{
		{
			Name:        "memory_bootstrap_session",
			Description: "Start a new session and recover the most relevant prior scoped context.",
			InputSchema: objectSchema(
				map[string]any{
					"cwd":                      stringSchema("Current working directory."),
					"task":                     stringSchema("Task summary for the new session."),
					"branch_name":              stringSchema("Current branch name."),
					"repo_remote":              stringSchema("Repository remote URL."),
					"include_related_projects": boolSchema("Whether to expand retrieval to related projects."),
					"related_reason":           stringSchema("Reason for related-project retrieval."),
					"max_notes":                intSchema("Maximum notes to include."),
					"max_handoffs":             intSchema("Reserved handoff limit input."),
				},
				"cwd",
			),
			call: func(ctx context.Context, raw json.RawMessage) (toolCallResult, error) {
				var req bootstrapRequest
				if err := decodeArgs(raw, &req); err != nil {
					return toolCallResult{}, err
				}
				return structuredToolResult(handlers.HandleMemoryBootstrapSession(ctx, retrieval.BootstrapInput{
					CWD:                    req.CWD,
					Task:                   req.Task,
					BranchName:             req.BranchName,
					RepoRemote:             req.RepoRemote,
					IncludeRelatedProjects: req.IncludeRelatedProjects,
					RelatedReason:          req.RelatedReason,
					MaxNotes:               req.MaxNotes,
					MaxHandoffs:            req.MaxHandoffs,
				}))
			},
		},
		{
			Name:        "memory_resolve_scope",
			Description: "Resolve system, project, and workspace identity from local context.",
			InputSchema: objectSchema(
				map[string]any{
					"cwd":               stringSchema("Current working directory."),
					"branch_name":       stringSchema("Current branch name."),
					"repo_remote":       stringSchema("Repository remote URL."),
					"project_name_hint": stringSchema("Preferred project name override."),
					"system_name_hint":  stringSchema("Preferred system name override."),
				},
				"cwd",
			),
			call: func(ctx context.Context, raw json.RawMessage) (toolCallResult, error) {
				var req resolveScopeRequest
				if err := decodeArgs(raw, &req); err != nil {
					return toolCallResult{}, err
				}
				return structuredToolResult(handlers.HandleMemoryResolveScope(ctx, scope.ResolveInput{
					CWD:             req.CWD,
					BranchName:      req.BranchName,
					RepoRemote:      req.RepoRemote,
					ProjectNameHint: req.ProjectNameHint,
					SystemNameHint:  req.SystemNameHint,
				}))
			},
		},
		{
			Name:        "memory_start_session",
			Description: "Create a fresh active session for a resolved scope.",
			InputSchema: objectSchema(
				map[string]any{
					"scope":       scopeSchema(),
					"task":        stringSchema("Task summary for the session."),
					"branch_name": stringSchema("Current branch name."),
				},
				"scope",
			),
			call: func(ctx context.Context, raw json.RawMessage) (toolCallResult, error) {
				var req startSessionRequest
				if err := decodeArgs(raw, &req); err != nil {
					return toolCallResult{}, err
				}
				return structuredToolResult(handlers.HandleMemoryStartSession(ctx, session.StartInput{
					Scope:      req.Scope.full(),
					Task:       req.Task,
					BranchName: req.BranchName,
				}))
			},
		},
		{
			Name:        "memory_save_note",
			Description: "Persist a high-value structured memory note.",
			InputSchema: objectSchema(
				map[string]any{
					"scope":               scopeRefLikeSchema(),
					"session_id":          stringSchema("Active session id."),
					"type":                stringEnumSchema("Canonical note type.", "decision", "bugfix", "discovery", "constraint", "preference", "todo"),
					"title":               stringSchema("Short note title."),
					"content":             stringSchema("Detailed durable note content."),
					"importance":          intSchema("Normalized importance from 1 to 5."),
					"tags":                stringArraySchema("Normalized note tags."),
					"file_paths":          stringArraySchema("Relevant file paths."),
					"related_project_ids": stringArraySchema("Explicit related-project links."),
					"status":              stringEnumSchema("Note lifecycle state.", "active", "resolved", "superseded"),
					"source":              stringEnumSchema("Memory provenance source.", "codex_explicit", "watcher_import", "relay_import", "recovery_generated"),
					"privacy_intent":      stringSchema("Explicit privacy handling intent."),
				},
				"scope",
				"session_id",
				"type",
				"title",
				"content",
				"importance",
			),
			call: func(ctx context.Context, raw json.RawMessage) (toolCallResult, error) {
				var req saveNoteRequest
				if err := decodeArgs(raw, &req); err != nil {
					return toolCallResult{}, err
				}
				return structuredToolResult(handlers.HandleMemorySaveNote(ctx, memory.SaveInput{
					Scope:             req.Scope.ref(),
					SessionID:         req.SessionID,
					Type:              memory.NoteType(req.Type),
					Title:             req.Title,
					Content:           req.Content,
					Importance:        req.Importance,
					Tags:              req.Tags,
					FilePaths:         req.FilePaths,
					RelatedProjectIDs: req.RelatedProjectIDs,
					Status:            memory.Status(req.Status),
					Source:            memory.Source(req.Source),
					PrivacyIntent:     req.PrivacyIntent,
				}))
			},
		},
		{
			Name:        "memory_save_handoff",
			Description: "Persist a checkpoint or end-of-session continuation record.",
			InputSchema: objectSchema(
				map[string]any{
					"scope":            scopeRefLikeSchema(),
					"session_id":       stringSchema("Active session id."),
					"kind":             stringEnumSchema("Handoff kind.", "final", "checkpoint", "recovery"),
					"task":             stringSchema("Task being handed off."),
					"summary":          stringSchema("Compact session summary."),
					"completed":        stringArraySchema("Completed items."),
					"next_steps":       stringArraySchema("Actionable next steps."),
					"open_questions":   stringArraySchema("Unresolved questions."),
					"risks":            stringArraySchema("Known risks."),
					"files_touched":    stringArraySchema("Touched file paths."),
					"related_note_ids": stringArraySchema("Linked note ids."),
					"status":           stringEnumSchema("Handoff status.", "open", "completed", "abandoned"),
					"privacy_intent":   stringSchema("Explicit privacy handling intent."),
				},
				"scope",
				"session_id",
				"kind",
				"task",
				"summary",
				"next_steps",
				"status",
			),
			call: func(ctx context.Context, raw json.RawMessage) (toolCallResult, error) {
				var req saveHandoffRequest
				if err := decodeArgs(raw, &req); err != nil {
					return toolCallResult{}, err
				}
				return structuredToolResult(handlers.HandleMemorySaveHandoff(ctx, handoff.SaveInput{
					Scope:          req.Scope.ref(),
					SessionID:      req.SessionID,
					Kind:           handoff.Kind(req.Kind),
					Task:           req.Task,
					Summary:        req.Summary,
					Completed:      req.Completed,
					NextSteps:      req.NextSteps,
					OpenQuestions:  req.OpenQuestions,
					Risks:          req.Risks,
					FilesTouched:   req.FilesTouched,
					RelatedNoteIDs: req.RelatedNoteIDs,
					Status:         handoff.Status(req.Status),
					PrivacyIntent:  req.PrivacyIntent,
				}))
			},
		},
		{
			Name:        "memory_save_import",
			Description: "Persist an import audit record for a secondary artifact.",
			InputSchema: objectSchema(
				map[string]any{
					"scope":             scopeRefLikeSchema(),
					"session_id":        stringSchema("Active session id."),
					"source":            stringEnumSchema("Import provenance source.", "watcher_import", "relay_import"),
					"external_id":       stringSchema("Stable external identifier for dedupe and audit."),
					"payload_hash":      stringSchema("Payload hash used for dedupe when no stable external id exists."),
					"durable_memory_id": stringSchema("Optional durable memory record linked to this import audit entry."),
					"privacy_intent":    stringSchema("Explicit privacy handling intent."),
				},
				"scope",
				"session_id",
				"source",
			),
			call: func(ctx context.Context, raw json.RawMessage) (toolCallResult, error) {
				var req saveImportRequest
				if err := decodeArgs(raw, &req); err != nil {
					return toolCallResult{}, err
				}
				return structuredToolResult(handlers.HandleMemorySaveImport(ctx, imports.SaveInput{
					Scope:           req.Scope.ref(),
					SessionID:       req.SessionID,
					Source:          imports.Source(req.Source),
					ExternalID:      req.ExternalID,
					PayloadHash:     req.PayloadHash,
					DurableMemoryID: req.DurableMemoryID,
					PrivacyIntent:   req.PrivacyIntent,
				}))
			},
		},
		{
			Name:        "memory_search",
			Description: "Search durable memory within a scoped boundary.",
			InputSchema: objectSchema(
				map[string]any{
					"query":                    stringSchema("Free-text search query."),
					"scope":                    scopeRefLikeSchema(),
					"types":                    stringArraySchema("Optional note type filters."),
					"states":                   stringArraySchema("Optional note or handoff state filters."),
					"min_importance":           intSchema("Minimum importance threshold."),
					"limit":                    intSchema("Maximum result count."),
					"include_handoffs":         boolSchema("Whether to search handoffs."),
					"include_related_projects": boolSchema("Whether to include related projects."),
					"intent":                   stringSchema("Optional search ranking intent."),
				},
				"query",
				"scope",
			),
			call: func(ctx context.Context, raw json.RawMessage) (toolCallResult, error) {
				var req searchRequest
				if err := decodeArgs(raw, &req); err != nil {
					return toolCallResult{}, err
				}
				return structuredToolResult(handlers.HandleMemorySearch(ctx, retrieval.SearchInput{
					Query:                  req.Query,
					Scope:                  req.Scope.ref(),
					Types:                  req.Types,
					States:                 req.States,
					MinImportance:          req.MinImportance,
					Limit:                  req.Limit,
					IncludeHandoffs:        req.IncludeHandoffs,
					IncludeRelatedProjects: req.IncludeRelatedProjects,
					Intent:                 req.Intent,
				}))
			},
		},
		{
			Name:        "memory_get_recent",
			Description: "Fetch recent scoped notes and handoffs without a query.",
			InputSchema: objectSchema(
				map[string]any{
					"scope":                    scopeRefLikeSchema(),
					"limit":                    intSchema("Maximum records to return."),
					"include_handoffs":         boolSchema("Whether to return handoffs."),
					"include_notes":            boolSchema("Whether to return notes."),
					"include_related_projects": boolSchema("Whether to include related-project notes."),
				},
				"scope",
			),
			call: func(ctx context.Context, raw json.RawMessage) (toolCallResult, error) {
				var req getRecentRequest
				if err := decodeArgs(raw, &req); err != nil {
					return toolCallResult{}, err
				}
				return structuredToolResult(handlers.HandleMemoryGetRecent(ctx, retrieval.GetRecentInput{
					Scope:                  req.Scope.ref(),
					Limit:                  req.Limit,
					IncludeHandoffs:        req.IncludeHandoffs,
					IncludeNotes:           req.IncludeNotes,
					IncludeRelatedProjects: req.IncludeRelatedProjects,
				}))
			},
		},
		{
			Name:        "memory_get_note",
			Description: "Fetch one note or handoff in full detail by id.",
			InputSchema: objectSchema(
				map[string]any{
					"id":   stringSchema("Record id."),
					"kind": stringEnumSchema("Record kind.", "note", "handoff"),
				},
				"id",
				"kind",
			),
			call: func(ctx context.Context, raw json.RawMessage) (toolCallResult, error) {
				var req getNoteRequest
				if err := decodeArgs(raw, &req); err != nil {
					return toolCallResult{}, err
				}
				return structuredToolResult(handlers.HandleMemoryGetNote(ctx, retrieval.GetRecordInput{
					ID:   req.ID,
					Kind: retrieval.RecordKind(req.Kind),
				}))
			},
		},
		{
			Name:        "memory_install_agents",
			Description: "Install or update AGENTS.md templates safely.",
			InputSchema: objectSchema(
				map[string]any{
					"target":                       stringEnumSchema("Installation target.", "global", "project", "both"),
					"mode":                         stringEnumSchema("Install mode.", "safe", "append", "overwrite"),
					"cwd":                          stringSchema("Project working directory."),
					"project_name":                 stringSchema("Explicit project name override."),
					"system_name":                  stringSchema("Explicit system name override."),
					"related_repositories":         stringArraySchema("Related repository names."),
					"preferred_tags":               stringArraySchema("Preferred memory tags."),
					"allow_related_project_memory": boolSchema("Whether AGENTS should permit related-project retrieval."),
				},
				"target",
				"mode",
			),
			call: func(ctx context.Context, raw json.RawMessage) (toolCallResult, error) {
				var req installAgentsRequest
				if err := decodeArgs(raw, &req); err != nil {
					return toolCallResult{}, err
				}
				return structuredToolResult(handlers.HandleMemoryInstallAgents(ctx, agents.InstallInput{
					Target:                    agents.Target(req.Target),
					Mode:                      agents.Mode(req.Mode),
					CWD:                       req.CWD,
					ProjectName:               req.ProjectName,
					SystemName:                req.SystemName,
					RelatedRepositories:       req.RelatedRepositories,
					PreferredTags:             req.PreferredTags,
					AllowRelatedProjectMemory: req.AllowRelatedProjectMemory,
				}))
			},
		},
	}
}

func decodeArgs(raw json.RawMessage, target any) error {
	payload := raw
	if len(payload) == 0 || string(payload) == "null" {
		payload = []byte("{}")
	}

	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode params: %w", err)
	}
	return nil
}

func structuredToolResult(payload any) (toolCallResult, error) {
	text, err := marshalCompact(payload)
	if err != nil {
		return toolCallResult{}, err
	}

	isError := false
	switch typed := payload.(type) {
	case Response[ResolveScopeData]:
		isError = !typed.Ok
	case Response[StartSessionData]:
		isError = !typed.Ok
	case Response[SaveNoteData]:
		isError = !typed.Ok
	case Response[SaveHandoffData]:
		isError = !typed.Ok
	case Response[SaveImportData]:
		isError = !typed.Ok
	case Response[BootstrapSessionData]:
		isError = !typed.Ok
	case Response[GetRecentData]:
		isError = !typed.Ok
	case Response[GetRecordData]:
		isError = !typed.Ok
	case Response[SearchData]:
		isError = !typed.Ok
	case Response[InstallAgentsData]:
		isError = !typed.Ok
	}

	return toolCallResult{
		Content:           []toolContent{{Type: "text", Text: text}},
		StructuredContent: payload,
		IsError:           isError,
	}, nil
}

func marshalCompact(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal tool response: %w", err)
	}
	return string(encoded), nil
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema(description string) map[string]any {
	schema := map[string]any{"type": "string"}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

func stringEnumSchema(description string, values ...string) map[string]any {
	schema := stringSchema(description)
	schema["enum"] = values
	return schema
}

func intSchema(description string) map[string]any {
	schema := map[string]any{"type": "integer"}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

func boolSchema(description string) map[string]any {
	schema := map[string]any{"type": "boolean"}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

func stringArraySchema(description string) map[string]any {
	schema := map[string]any{
		"type":  "array",
		"items": map[string]any{"type": "string"},
	}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

func scopeRefLikeSchema() map[string]any {
	return objectSchema(
		map[string]any{
			"system_id":      stringSchema("Canonical system id."),
			"system_name":    stringSchema("Canonical system name."),
			"project_id":     stringSchema("Canonical project id."),
			"project_name":   stringSchema("Canonical project name."),
			"workspace_id":   stringSchema("Canonical workspace id."),
			"workspace_root": stringSchema("Workspace root path."),
			"branch_name":    stringSchema("Workspace branch name."),
			"resolved_by":    stringSchema("Scope resolution evidence."),
		},
		"system_id",
		"project_id",
		"workspace_id",
	)
}

func scopeSchema() map[string]any {
	return objectSchema(
		map[string]any{
			"system_id":      stringSchema("Canonical system id."),
			"system_name":    stringSchema("Canonical system name."),
			"project_id":     stringSchema("Canonical project id."),
			"project_name":   stringSchema("Canonical project name."),
			"workspace_id":   stringSchema("Canonical workspace id."),
			"workspace_root": stringSchema("Workspace root path."),
			"branch_name":    stringSchema("Workspace branch name."),
			"resolved_by":    stringSchema("Scope resolution evidence."),
		},
		"system_id",
		"system_name",
		"project_id",
		"project_name",
		"workspace_id",
		"workspace_root",
		"resolved_by",
	)
}
