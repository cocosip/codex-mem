// Command http-mcp-smoke-test runs an end-to-end HTTP MCP smoke test against the local source tree.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	internalmcp "codex-mem/internal/mcp"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	repoRoot, err := os.Getwd()
	if err != nil {
		failf("resolve working directory: %v", err)
	}
	tempRoot, err := os.MkdirTemp("", "codex-mem-http-mcp-smoke-*")
	if err != nil {
		failf("create temp root: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempRoot)
	}()

	tempProject := filepath.Join(tempRoot, "project")
	if err := os.MkdirAll(tempProject, 0o755); err != nil {
		failf("create temp project: %v", err)
	}
	binaryPath := filepath.Join(tempRoot, "codex-mem-http-smoke.exe")
	buildBinary(ctx, repoRoot, binaryPath)

	port, err := reservePort()
	if err != nil {
		failf("reserve port: %v", err)
	}
	listenAddr := "127.0.0.1:" + strconv.Itoa(port)
	endpointURL := "http://" + listenAddr + "/mcp"

	cmd := exec.CommandContext(ctx, binaryPath, "serve-http", "--listen", listenAddr, "--path", "/mcp")
	cmd.Dir = repoRoot
	cmd.Stdout = new(bytes.Buffer)
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Start(); err != nil {
		failf("start MCP HTTP server: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_, _ = cmd.Stdout.(*bytes.Buffer).WriteTo(os.Stderr)
		_, _ = cmd.Stderr.(*bytes.Buffer).WriteTo(os.Stderr)
		_ = cmd.Wait()
	}()

	waitForReady(ctx, endpointURL)

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "codex-mem-http-smoke",
		Version: "0.0.1",
	}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint:   endpointURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		MaxRetries: 1,
	}, nil)
	if err != nil {
		failf("connect SDK HTTP client: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	listResult, err := session.ListTools(ctx, &sdkmcp.ListToolsParams{})
	if err != nil {
		failf("tools/list failed: %v", err)
	}
	if len(listResult.Tools) != 9 {
		failf("tools/list count mismatch: got %d want %d", len(listResult.Tools), 9)
	}

	callResult, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name: "memory_install_agents",
		Arguments: map[string]any{
			"target":       "project",
			"mode":         "safe",
			"cwd":          tempProject,
			"project_name": "http-mcp-smoke-test",
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
	if !installResponse.Ok || installResponse.Data == nil || len(installResponse.Data.WrittenFiles) != 1 {
		failf("unexpected install agents response: %+v", installResponse)
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

	fmt.Printf("http mcp smoke test passed\n")
	fmt.Printf("endpoint=%s\n", endpointURL)
	fmt.Printf("session_id=%s\n", session.ID())
	fmt.Printf("protocol_version=%s\n", protocolVersion)
	fmt.Printf("tool_count=%d\n", len(listResult.Tools))
	fmt.Printf("tool_call=memory_install_agents\n")
	fmt.Printf("written_file=%s\n", agentsPath)
}

func buildBinary(ctx context.Context, repoRoot string, binaryPath string) {
	cmd := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "./cmd/codex-mem")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		failf("build temporary binary: %v\n%s", err, string(output))
	}
}

func reservePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = listener.Close()
	}()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address %T", listener.Addr())
	}
	return addr.Port, nil
}

func waitForReady(ctx context.Context, endpointURL string) {
	client := &http.Client{Timeout: 2 * time.Second}
	for {
		select {
		case <-ctx.Done():
			failf("timed out waiting for HTTP MCP server readiness")
		default:
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
		if err == nil {
			request.Header.Set("Accept", "text/event-stream")
		}
		response, doErr := client.Do(request)
		if doErr == nil {
			_ = response.Body.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
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
