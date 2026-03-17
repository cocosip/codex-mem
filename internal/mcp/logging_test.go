package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"codex-mem/internal/domain/agents"
	"codex-mem/internal/domain/imports"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewSDKServerLogsToolCalls(t *testing.T) {
	var logs bytes.Buffer
	oldDefault := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() {
		slog.SetDefault(oldDefault)
	})

	root := t.TempDir()
	server := NewSDKServer(&Handlers{
		agentsService: agents.NewService(agents.Options{HomeDir: root}),
		importService: imports.NewService(nilImportRepo{}, imports.Options{}),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverTransport, clientTransport := sdkmcp.NewInMemoryTransports()
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run(ctx, serverTransport)
	}()

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "codex-mem-test-client",
		Version: "0.0.1",
	}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	defer closeSDKSession(t, session, cancel, serverErr)

	if _, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "memory_install_agents",
		Arguments: map[string]any{
			"target":       "project",
			"mode":         "safe",
			"cwd":          root,
			"project_name": "codex-mem",
			"system_name":  "codex-mem",
		},
	}); err != nil {
		t.Fatalf("call tool: %v", err)
	}

	output := logs.String()
	if !strings.Contains(output, `msg="MCP tool call completed"`) {
		t.Fatalf("expected tool completion log, got %q", output)
	}
	if !strings.Contains(output, `tool_name=memory_install_agents`) {
		t.Fatalf("expected tool name in log, got %q", output)
	}
	if !strings.Contains(output, `client_name=codex-mem-test-client`) {
		t.Fatalf("expected client name in log, got %q", output)
	}
	if !strings.Contains(output, `written_files_count:1`) {
		t.Fatalf("expected written_files_count summary in log, got %q", output)
	}
}

func TestSummarizeToolArgumentsRedactsLargeContent(t *testing.T) {
	summary := summarizeToolArguments(json.RawMessage(`{
		"session_id":"sess_123",
		"title":"Example note",
		"content":"abcdefghijklmnopqrstuvwxyz",
		"scope":{"system_id":"sys_1","project_id":"proj_1","workspace_id":"ws_1"}
	}`))

	if got, want := summary["content_chars"], 26; got != want {
		t.Fatalf("content_chars mismatch: got %v want %v", got, want)
	}
	if _, exists := summary["content"]; exists {
		t.Fatalf("expected content body to stay out of logs, got %+v", summary)
	}
	scopeSummary, ok := summary["scope"].(map[string]any)
	if !ok || scopeSummary["project_id"] != "proj_1" {
		t.Fatalf("expected scope summary, got %+v", summary["scope"])
	}
}
