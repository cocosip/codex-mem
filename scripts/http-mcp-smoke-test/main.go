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
)

const jsonRPCVersion = "2.0"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
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
	ProtocolVersion string `json:"protocolVersion"`
}

type listToolsResult struct {
	Tools []toolDefinition `json:"tools"`
}

type toolDefinition struct {
	Name string `json:"name"`
}

type callToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type installAgentsRequest struct {
	Target      string `json:"target"`
	Mode        string `json:"mode"`
	CWD         string `json:"cwd,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
	SystemName  string `json:"system_name,omitempty"`
}

type toolCallResult struct {
	Content           []toolContent   `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent,omitempty"`
	IsError           bool            `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type installAgentsResponse struct {
	Ok   bool               `json:"ok"`
	Data *installAgentsData `json:"data,omitempty"`
}

type installAgentsData struct {
	WrittenFiles []fileChange `json:"written_files"`
}

type fileChange struct {
	Path string `json:"path"`
}

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

	client := &http.Client{Timeout: 10 * time.Second}
	waitForReady(ctx, client, endpointURL)

	initResponse := doRPC(client, endpointURL, http.StatusOK, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  mustMarshalRaw(initializeParams{ProtocolVersion: "2025-03-26"}),
	})
	mustNoRPCError("initialize", initResponse)
	var initResult initializeResult
	mustDecode(initResponse.Result, &initResult)
	if initResult.ProtocolVersion != "2025-03-26" {
		failf("initialize protocol mismatch: got %q want %q", initResult.ProtocolVersion, "2025-03-26")
	}

	doNotification(client, endpointURL, rpcRequest{
		JSONRPC: jsonRPCVersion,
		Method:  "notifications/initialized",
	})

	listResponse := doRPC(client, endpointURL, http.StatusOK, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})
	mustNoRPCError("tools/list", listResponse)
	var listResult listToolsResult
	mustDecode(listResponse.Result, &listResult)
	if len(listResult.Tools) != 9 {
		failf("tools/list count mismatch: got %d want %d", len(listResult.Tools), 9)
	}

	callResponse := doRPC(client, endpointURL, http.StatusOK, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`3`),
		Method:  "tools/call",
		Params: mustMarshalRaw(callToolParams{
			Name: "memory_install_agents",
			Arguments: mustMarshalRaw(installAgentsRequest{
				Target:      "project",
				Mode:        "safe",
				CWD:         tempProject,
				ProjectName: "http-mcp-smoke-test",
				SystemName:  "codex-mem",
			}),
		}),
	})
	mustNoRPCError("tools/call", callResponse)
	var callResult toolCallResult
	mustDecode(callResponse.Result, &callResult)
	if callResult.IsError {
		failf("tool call returned isError=true")
	}
	var installResponse installAgentsResponse
	mustDecode(callResponse.Result, &callResult)
	mustDecode(callResult.StructuredContent, &installResponse)
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

	fmt.Printf("http mcp smoke test passed\n")
	fmt.Printf("endpoint=%s\n", endpointURL)
	fmt.Printf("protocol_version=%s\n", initResult.ProtocolVersion)
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

func waitForReady(ctx context.Context, client *http.Client, endpointURL string) {
	for {
		select {
		case <-ctx.Done():
			failf("timed out waiting for HTTP MCP server readiness")
		default:
		}
		response, err := doRPCRequest(client, endpointURL, rpcRequest{
			JSONRPC: jsonRPCVersion,
			ID:      json.RawMessage(`0`),
			Method:  "ping",
			Params:  json.RawMessage(`{}`),
		})
		if err == nil {
			_ = response.Body.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func doNotification(client *http.Client, endpointURL string, request rpcRequest) {
	response, err := doRPCRequest(client, endpointURL, request)
	if err != nil {
		failf("notification request failed: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if got, want := response.StatusCode, http.StatusAccepted; got != want {
		failf("notification status mismatch: got %d want %d", got, want)
	}
}

func doRPC(client *http.Client, endpointURL string, wantStatus int, request rpcRequest) rpcResponse {
	response, err := doRPCRequest(client, endpointURL, request)
	if err != nil {
		failf("rpc request failed: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()
	if got := response.StatusCode; got != wantStatus {
		failf("rpc status mismatch: got %d want %d", got, wantStatus)
	}
	var rpc rpcResponse
	if err := json.NewDecoder(response.Body).Decode(&rpc); err != nil {
		failf("decode rpc response: %v", err)
	}
	return rpc
}

func doRPCRequest(client *http.Client, endpointURL string, request rpcRequest) (*http.Response, error) {
	body, err := json.Marshal(request)
	if err != nil {
		failf("marshal request: %v", err)
	}
	httpRequest, err := http.NewRequest(http.MethodPost, endpointURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	return client.Do(httpRequest)
}

func mustMarshalRaw(value any) json.RawMessage {
	body, err := json.Marshal(value)
	if err != nil {
		failf("marshal raw value: %v", err)
	}
	return json.RawMessage(body)
}

func mustDecode(payload []byte, target any) {
	if err := json.Unmarshal(payload, target); err != nil {
		failf("decode payload: %v", err)
	}
}

func mustNoRPCError(method string, response rpcResponse) {
	if response.Error != nil {
		failf("%s failed: code=%d message=%s", method, response.Error.Code, response.Error.Message)
	}
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}




