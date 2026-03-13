package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"codex-mem/internal/buildinfo"
	"codex-mem/internal/domain/agents"
	"codex-mem/internal/domain/handoff"
	"codex-mem/internal/domain/memory"
	"codex-mem/internal/domain/retrieval"
	"codex-mem/internal/domain/scope"
	"codex-mem/internal/domain/session"
)

const (
	defaultProtocolVersion = "2025-03-26"
	jsonRPCVersion         = "2.0"
)

type Server struct {
	handlers *Handlers
	tools    map[string]toolDefinition
}

type toolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	call        func(context.Context, json.RawMessage) (toolCallResult, error)
}

type toolCallResult struct {
	Content           []toolContent `json:"content"`
	StructuredContent any           `json:"structuredContent,omitempty"`
	IsError           bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type initializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      serverInfo     `json:"serverInfo"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type listToolsResult struct {
	Tools []toolDefinition `json:"tools"`
}

type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
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

func NewServer(handlers *Handlers) *Server {
	server := &Server{handlers: handlers}
	server.tools = map[string]toolDefinition{}
	for _, tool := range server.buildTools() {
		server.tools[tool.Name] = tool
	}
	return server
}

func (s *Server) Serve(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	reader := bufio.NewReader(stdin)
	writer := bufio.NewWriter(stdout)

	for {
		payload, err := readFrame(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var request rpcRequest
		if err := json.Unmarshal(payload, &request); err != nil {
			if err := writeFrame(writer, rpcResponse{
				JSONRPC: jsonRPCVersion,
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			}); err != nil {
				return err
			}
			continue
		}

		response, shouldRespond := s.handleRequest(ctx, request)
		if !shouldRespond {
			continue
		}
		if err := writeFrame(writer, response); err != nil {
			return err
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, request rpcRequest) (rpcResponse, bool) {
	response := rpcResponse{
		JSONRPC: jsonRPCVersion,
		ID:      request.ID,
	}

	if request.JSONRPC != "" && request.JSONRPC != jsonRPCVersion {
		response.Error = &rpcError{Code: -32600, Message: "jsonrpc must be 2.0"}
		return response, hasResponseID(request)
	}
	if strings.TrimSpace(request.Method) == "" {
		response.Error = &rpcError{Code: -32600, Message: "method is required"}
		return response, hasResponseID(request)
	}

	switch request.Method {
	case "initialize":
		var params initializeParams
		if err := decodeArgs(request.Params, &params); err != nil {
			response.Error = &rpcError{Code: -32602, Message: err.Error()}
			return response, true
		}
		response.Result = initializeResult{
			ProtocolVersion: chooseProtocolVersion(params.ProtocolVersion),
			Capabilities: map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			ServerInfo: serverInfo{Name: "codex-mem", Version: buildinfo.Summary()},
		}
		return response, true
	case "notifications/initialized":
		return rpcResponse{}, false
	case "ping":
		response.Result = map[string]any{}
		return response, true
	case "tools/list":
		response.Result = listToolsResult{Tools: s.listTools()}
		return response, true
	case "tools/call":
		var params callToolParams
		if err := decodeArgs(request.Params, &params); err != nil {
			response.Error = &rpcError{Code: -32602, Message: err.Error()}
			return response, true
		}
		tool, ok := s.tools[strings.TrimSpace(params.Name)]
		if !ok {
			response.Error = &rpcError{Code: -32602, Message: fmt.Sprintf("unknown tool %q", params.Name)}
			return response, true
		}
		result, err := tool.call(ctx, params.Arguments)
		if err != nil {
			response.Error = &rpcError{Code: -32602, Message: err.Error()}
			return response, true
		}
		response.Result = result
		return response, true
	default:
		response.Error = &rpcError{Code: -32601, Message: fmt.Sprintf("method %q not found", request.Method)}
		return response, hasResponseID(request)
	}
}

func (s *Server) listTools() []toolDefinition {
	tools := s.buildTools()
	result := make([]toolDefinition, 0, len(tools))
	for _, tool := range tools {
		result = append(result, toolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	return result
}

func (s *Server) ToolCount() int {
	return len(s.listTools())
}

func (s *Server) buildTools() []toolDefinition {
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
				return structuredToolResult(s.handlers.HandleMemoryBootstrapSession(ctx, retrieval.BootstrapInput{
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
				return structuredToolResult(s.handlers.HandleMemoryResolveScope(ctx, scope.ResolveInput{
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
				return structuredToolResult(s.handlers.HandleMemoryStartSession(ctx, session.StartInput{
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
				return structuredToolResult(s.handlers.HandleMemorySaveNote(ctx, memory.SaveInput{
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
				return structuredToolResult(s.handlers.HandleMemorySaveHandoff(ctx, handoff.SaveInput{
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
				return structuredToolResult(s.handlers.HandleMemorySearch(ctx, retrieval.SearchInput{
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
				return structuredToolResult(s.handlers.HandleMemoryGetRecent(ctx, retrieval.GetRecentInput{
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
				return structuredToolResult(s.handlers.HandleMemoryGetNote(ctx, retrieval.GetRecordInput{
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
				return structuredToolResult(s.handlers.HandleMemoryInstallAgents(ctx, agents.InstallInput{
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

func chooseProtocolVersion(requested string) string {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return requested
	}
	return defaultProtocolVersion
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

func hasResponseID(request rpcRequest) bool {
	return len(request.ID) > 0
}

func readFrame(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && line == "" && contentLength == -1 {
				return nil, io.EOF
			}
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if contentLength < 0 {
				continue
			}
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		if key != "content-length" {
			continue
		}

		value := strings.TrimSpace(parts[1])
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return nil, fmt.Errorf("invalid content-length %q", value)
		}
		contentLength = parsed
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("missing content-length header")
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeFrame(writer *bufio.Writer, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal frame: %w", err)
	}

	if _, err := writer.WriteString(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))); err != nil {
		return err
	}
	if _, err := writer.Write(body); err != nil {
		return err
	}
	return writer.Flush()
}
