package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"codex-mem/internal/buildinfo"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewSDKServer constructs a go-sdk-backed MCP server that preserves the
// existing codex-mem tool surface and response envelopes.
func NewSDKServer(handlers *Handlers) *sdkmcp.Server {
	server := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "codex-mem",
		Version: buildinfo.Summary(),
	}, &sdkmcp.ServerOptions{
		Capabilities: &sdkmcp.ServerCapabilities{
			Tools: &sdkmcp.ToolCapabilities{ListChanged: false},
		},
	})

	legacy := &Server{handlers: handlers}
	for _, tool := range legacy.buildTools() {
		tool := tool
		server.AddTool(&sdkmcp.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}, func(ctx context.Context, request *sdkmcp.CallToolRequest) (*sdkmcp.CallToolResult, error) {
			result, err := tool.call(ctx, request.Params.Arguments)
			if err != nil {
				return nil, err
			}
			return sdkToolResult(result)
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
