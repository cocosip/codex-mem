package mcp

import (
	"context"
	"errors"
	"testing"

	"codex-mem/internal/domain/agents"
	"codex-mem/internal/domain/imports"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewSDKServerListsToolsAndCallsTool(t *testing.T) {
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

	listResult, err := session.ListTools(ctx, &sdkmcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if got, want := len(listResult.Tools), 10; got != want {
		t.Fatalf("tool count mismatch: got %d want %d", got, want)
	}

	callResult, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "memory_install_agents",
		Arguments: map[string]any{
			"target":       "project",
			"mode":         "safe",
			"cwd":          root,
			"project_name": "codex-mem",
			"system_name":  "codex-mem",
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if callResult.IsError {
		t.Fatalf("expected successful tool call, got error result: %+v", callResult)
	}
	if got, want := len(callResult.Content), 1; got != want {
		t.Fatalf("content count mismatch: got %d want %d", got, want)
	}

	var structured Response[InstallAgentsData]
	decodeJSONValue(t, callResult.StructuredContent, &structured)
	if !structured.Ok {
		t.Fatalf("expected ok structured content, got %+v", structured.Error)
	}
	if structured.Data == nil || len(structured.Data.WrittenFiles) != 1 {
		t.Fatalf("expected one written file, got %+v", structured.Data)
	}
}

type nilImportRepo struct{}

func (nilImportRepo) FindDuplicate(imports.Record) (*imports.Record, error) { return nil, nil }

func (nilImportRepo) Create(imports.Record) error { return nil }

func closeSDKSession(t *testing.T, session *sdkmcp.ClientSession, cancel context.CancelFunc, serverErr <-chan error) {
	t.Helper()

	if err := session.Close(); err != nil {
		t.Fatalf("close session: %v", err)
	}
	cancel()

	if err := <-serverErr; err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("server run: %v", err)
	}
}
