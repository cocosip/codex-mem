// Command mcp-smoke-test runs an end-to-end stdio MCP smoke test against the local source tree.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	internalmcp "codex-mem/internal/mcp"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	repoRoot, err := os.Getwd()
	if err != nil {
		failf("resolve working directory: %v", err)
	}

	tempProject, err := os.MkdirTemp("", "codex-mem-mcp-smoke-*")
	if err != nil {
		failf("create temp project: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempProject)
	}()

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/codex-mem", "serve")
	cmd.Dir = repoRoot

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "codex-mem-stdio-smoke",
		Version: "0.0.1",
	}, nil)
	session, err := client.Connect(ctx, &sdkmcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		failf("connect SDK stdio client: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	listResult, err := session.ListTools(ctx, &sdkmcp.ListToolsParams{})
	if err != nil {
		failf("tools/list failed: %v", err)
	}
	if len(listResult.Tools) != 10 {
		failf("tools/list count mismatch: got %d want %d", len(listResult.Tools), 10)
	}

	callResult, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "memory_install_agents",
		Arguments: map[string]any{
			"target":       "project",
			"mode":         "safe",
			"cwd":          tempProject,
			"project_name": "mcp-smoke-test",
			"system_name":  "codex-mem",
		},
	})
	if err != nil {
		failf("tools/call failed: %v", err)
	}
	if callResult.IsError {
		failf("tool call returned isError=true")
	}

	var installResponse internalmcp.Response[internalmcp.InstallAgentsData]
	mustDecodeStructured(callResult.StructuredContent, &installResponse)
	if !installResponse.Ok || installResponse.Data == nil {
		failf("structured tool response not ok")
	}
	if len(installResponse.Data.WrittenFiles) != 1 {
		failf("expected one written file, got %+v", installResponse.Data)
	}

	agentsPath := filepath.Join(tempProject, "AGENTS.md")
	if _, err := os.Stat(agentsPath); err != nil {
		failf("expected AGENTS.md to be written at %s: %v", agentsPath, err)
	}
	if got := installResponse.Data.WrittenFiles[0].Path; got != agentsPath {
		failf("written file path mismatch: got %q want %q", got, agentsPath)
	}

	initializeResult := session.InitializeResult()
	protocolVersion := "unknown"
	if initializeResult != nil {
		protocolVersion = initializeResult.ProtocolVersion
	}

	fmt.Printf("mcp smoke test passed\n")
	fmt.Printf("protocol_version=%s\n", protocolVersion)
	fmt.Printf("tool_count=%d\n", len(listResult.Tools))
	fmt.Printf("tool_call=memory_install_agents\n")
	fmt.Printf("written_file=%s\n", agentsPath)
}

func mustDecodeStructured(value any, target any) {
	body, err := json.Marshal(value)
	if err != nil {
		failf("marshal structured content: %v", err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		failf("decode structured content: %v", err)
	}
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
