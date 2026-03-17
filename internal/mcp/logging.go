package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func newServerLogger() *slog.Logger {
	return slog.Default().With("component", "mcp")
}

func requestLoggingMiddleware(logger *slog.Logger) sdkmcp.Middleware {
	return func(next sdkmcp.MethodHandler) sdkmcp.MethodHandler {
		return func(ctx context.Context, method string, req sdkmcp.Request) (sdkmcp.Result, error) {
			started := time.Now()
			result, err := next(ctx, method, req)
			if method == "tools/call" {
				return result, err
			}

			attrs := append(requestLogAttrs(req), "method", method, "duration_ms", time.Since(started).Milliseconds())
			if err != nil {
				logger.Error("MCP request failed", appendAttrs(attrs, "err", err)...)
				return nil, err
			}
			if result == nil {
				return nil, nil
			}

			logger.Info("MCP request completed", appendAttrs(attrs, summarizeMethodParams(method, req.GetParams())...)...)
			return result, nil
		}
	}
}

func requestLogAttrs(req sdkmcp.Request) []any {
	attrs := make([]any, 0, 10)
	if req == nil {
		return attrs
	}

	if session := req.GetSession(); session != nil && session.ID() != "" {
		attrs = append(attrs, "session_id", session.ID())
	}

	return attrs
}

func summarizeMethodParams(method string, params sdkmcp.Params) []any {
	if method != "initialize" {
		return nil
	}

	initParams, ok := params.(*sdkmcp.InitializeParams)
	if !ok || initParams == nil {
		return nil
	}

	attrs := make([]any, 0, 6)
	if initParams.ClientInfo != nil {
		if initParams.ClientInfo.Name != "" {
			attrs = append(attrs, "client_name", initParams.ClientInfo.Name)
		}
		if initParams.ClientInfo.Version != "" {
			attrs = append(attrs, "client_version", initParams.ClientInfo.Version)
		}
	}
	if initParams.ProtocolVersion != "" {
		attrs = append(attrs, "protocol_version", initParams.ProtocolVersion)
	}
	return attrs
}

func toolLogAttrs(request *sdkmcp.CallToolRequest, toolName string, raw json.RawMessage) []any {
	attrs := make([]any, 0, 12)
	if request != nil && request.Session != nil {
		if sessionID := request.Session.ID(); sessionID != "" {
			attrs = append(attrs, "session_id", sessionID)
		}
		if initParams := request.Session.InitializeParams(); initParams != nil {
			if initParams.ClientInfo != nil {
				if initParams.ClientInfo.Name != "" {
					attrs = append(attrs, "client_name", initParams.ClientInfo.Name)
				}
				if initParams.ClientInfo.Version != "" {
					attrs = append(attrs, "client_version", initParams.ClientInfo.Version)
				}
			}
			if initParams.ProtocolVersion != "" {
				attrs = append(attrs, "protocol_version", initParams.ProtocolVersion)
			}
		}
	}

	attrs = append(attrs, "tool_name", toolName)
	if summary := summarizeToolArguments(raw); len(summary) > 0 {
		attrs = append(attrs, "args_summary", summary)
	}
	return attrs
}

func summarizeToolArguments(raw json.RawMessage) map[string]any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]any{"raw": compactString(string(raw), 160)}
	}

	summary := map[string]any{
		"keys": sortedKeys(payload),
	}
	addStringSummary(summary, "cwd", payload, "cwd")
	addStringSummary(summary, "task", payload, "task")
	addStringSummary(summary, "title", payload, "title")
	addStringSummary(summary, "query", payload, "query")
	addStringSummary(summary, "id", payload, "id")
	addStringSummary(summary, "kind", payload, "kind")
	addStringSummary(summary, "session_id", payload, "session_id")
	addStringSummary(summary, "source", payload, "source")
	addStringSummary(summary, "target", payload, "target")
	addStringSummary(summary, "mode", payload, "mode")
	addStringSummary(summary, "external_id", payload, "external_id")
	addStringSummary(summary, "payload_hash", payload, "payload_hash")

	if value, ok := payload["content"].(string); ok {
		summary["content_chars"] = len(value)
	}
	if value, ok := payload["summary"].(string); ok {
		summary["summary_chars"] = len(value)
	}
	if value, ok := payload["importance"]; ok {
		summary["importance"] = value
	}
	if value, ok := payload["limit"]; ok {
		summary["limit"] = value
	}

	addArrayCount(summary, "tags_count", payload, "tags")
	addArrayCount(summary, "file_paths_count", payload, "file_paths")
	addArrayCount(summary, "completed_count", payload, "completed")
	addArrayCount(summary, "next_steps_count", payload, "next_steps")
	addArrayCount(summary, "open_questions_count", payload, "open_questions")
	addArrayCount(summary, "risks_count", payload, "risks")
	addArrayCount(summary, "types_count", payload, "types")
	addArrayCount(summary, "states_count", payload, "states")

	if scopeSummary := summarizeScope(payload["scope"]); len(scopeSummary) > 0 {
		summary["scope"] = scopeSummary
	}

	return summary
}

func summarizeScope(value any) map[string]any {
	scopeMap, ok := value.(map[string]any)
	if !ok || len(scopeMap) == 0 {
		return nil
	}

	summary := map[string]any{}
	addStringSummary(summary, "system_id", scopeMap, "system_id")
	addStringSummary(summary, "project_id", scopeMap, "project_id")
	addStringSummary(summary, "workspace_id", scopeMap, "workspace_id")
	addStringSummary(summary, "branch_name", scopeMap, "branch_name")
	addStringSummary(summary, "resolved_by", scopeMap, "resolved_by")
	addStringSummary(summary, "workspace_root", scopeMap, "workspace_root")
	return summary
}

func summarizeToolResult(payload any) []any {
	if payload == nil {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil
	}

	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		return nil
	}

	attrs := make([]any, 0, 10)
	if okValue, exists := object["ok"].(bool); exists {
		attrs = append(attrs, "response_ok", okValue)
	}
	if warnings, ok := object["warnings"].([]any); ok && len(warnings) > 0 {
		attrs = append(attrs, "warning_count", len(warnings), "warning_codes", warningCodes(warnings))
	}
	if errValue, ok := object["error"].(map[string]any); ok {
		if code, ok := errValue["code"].(string); ok && code != "" {
			attrs = append(attrs, "error_code", code)
		}
	}
	if dataSummary := summarizeResultData(object["data"]); len(dataSummary) > 0 {
		attrs = append(attrs, "data_summary", dataSummary)
	}
	return attrs
}

func summarizeResultData(value any) map[string]any {
	data, ok := value.(map[string]any)
	if !ok || len(data) == 0 {
		return nil
	}

	summary := map[string]any{
		"keys": sortedKeys(data),
	}

	if sessionValue, ok := data["session"].(map[string]any); ok {
		addStringSummary(summary, "session_id", sessionValue, "id")
	}
	if noteValue, ok := data["note"].(map[string]any); ok {
		addStringSummary(summary, "note_id", noteValue, "id")
	}
	if handoffValue, ok := data["handoff"].(map[string]any); ok {
		addStringSummary(summary, "handoff_id", handoffValue, "id")
	}
	if importValue, ok := data["import"].(map[string]any); ok {
		addStringSummary(summary, "import_id", importValue, "id")
	}
	if scopeSummary := summarizeScope(data["scope"]); len(scopeSummary) > 0 {
		summary["scope"] = scopeSummary
	}

	addArrayCount(summary, "results_count", data, "results")
	addArrayCount(summary, "notes_count", data, "notes")
	addArrayCount(summary, "handoffs_count", data, "handoffs")
	addArrayCount(summary, "written_files_count", data, "written_files")
	addArrayCount(summary, "skipped_files_count", data, "skipped_files")
	addArrayCount(summary, "recent_notes_count", data, "recent_notes")
	addArrayCount(summary, "related_notes_count", data, "related_notes")

	if value, ok := data["deduplicated"]; ok {
		summary["deduplicated"] = value
	}
	if value, ok := data["suppressed"]; ok {
		summary["suppressed"] = value
	}
	if value, ok := data["materialized"]; ok {
		summary["materialized"] = value
	}

	return summary
}

func warningCodes(warnings []any) []string {
	codes := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		warningMap, ok := warning.(map[string]any)
		if !ok {
			continue
		}
		code, _ := warningMap["code"].(string)
		if code == "" {
			continue
		}
		codes = append(codes, code)
	}
	return codes
}

func addStringSummary(summary map[string]any, outKey string, payload map[string]any, inKey string) {
	value, ok := payload[inKey].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return
	}
	summary[outKey] = compactString(value, 120)
}

func addArrayCount(summary map[string]any, outKey string, payload map[string]any, inKey string) {
	items, ok := payload[inKey].([]any)
	if !ok || len(items) == 0 {
		return
	}
	summary[outKey] = len(items)
}

func sortedKeys(payload map[string]any) []string {
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func compactString(value string, maxLen int) string {
	value = strings.Join(strings.Fields(value), " ")
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	if maxLen <= 3 {
		return value[:maxLen]
	}
	return value[:maxLen-3] + "..."
}

func appendAttrs(base []any, extra ...any) []any {
	attrs := make([]any, 0, len(base)+len(extra))
	attrs = append(attrs, base...)
	attrs = append(attrs, extra...)
	return attrs
}
