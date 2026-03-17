package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"codex-mem/internal/buildinfo"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewSDKServer constructs a go-sdk-backed MCP server that preserves the
// existing codex-mem tool surface and response envelopes.
func NewSDKServer(handlers *Handlers) *sdkmcp.Server {
	logger := newServerLogger()
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "codex-mem",
		Version: buildinfo.Summary(),
	}, &sdkmcp.ServerOptions{
		Capabilities: &sdkmcp.ServerCapabilities{
			Tools: &sdkmcp.ToolCapabilities{ListChanged: false},
		},
	})
	server.AddReceivingMiddleware(requestLoggingMiddleware(logger))

	for _, tool := range buildTools(handlers) {
		tool := tool
		server.AddTool(&sdkmcp.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}, func(ctx context.Context, request *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			started := time.Now()
			attrs := toolLogAttrs(request, tool.Name, request.Params.Arguments)
			result, err := tool.call(ctx, request.Params.Arguments)
			if err != nil {
				logger.Error("MCP tool call failed", appendAttrs(attrs, "duration_ms", time.Since(started).Milliseconds(), "err", err)...)
				return nil, err
			}

			sdkResult, err := sdkToolResult(result)
			if err != nil {
				logger.Error("MCP tool result encoding failed", appendAttrs(attrs, "duration_ms", time.Since(started).Milliseconds(), "err", err)...)
				return nil, err
			}

			attrs = append(attrs, "duration_ms", time.Since(started).Milliseconds(), "is_error", result.IsError)
			attrs = append(attrs, summarizeToolResult(result.StructuredContent)...)
			if result.IsError {
				logger.Warn("MCP tool call completed with application error", attrs...)
			} else {
				logger.Info("MCP tool call completed", attrs...)
			}
			return sdkResult, nil
		})
	}

	return server
}

func sdkToolResult(result toolCallResult) (*sdkmcp.CallToolResult, error) {
	content := make([]sdkmcp.Content, 0, len(result.Content))
	for _, item := range result.Content {
		switch item.Type {
		case "", "text":
			content = append(content, &sdkmcp.TextContent{Text: item.Text})
		default:
			return nil, fmt.Errorf("unsupported tool content type %q", item.Type)
		}
	}

	if result.StructuredContent != nil {
		body, err := json.Marshal(result.StructuredContent)
		if err != nil {
			return nil, fmt.Errorf("marshal structured content: %w", err)
		}

		var object map[string]any
		if err := json.Unmarshal(body, &object); err != nil {
			return nil, fmt.Errorf("decode structured content object: %w", err)
		}
		result.StructuredContent = object
	}

	return &sdkmcp.CallToolResult{
		Content:           content,
		StructuredContent: result.StructuredContent,
		IsError:           result.IsError,
	}, nil
}
