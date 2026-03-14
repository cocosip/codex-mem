// Command mcp-smoke-test runs an end-to-end stdio MCP smoke test against the local source tree.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	SkippedFiles []fileChange `json:"skipped_files"`
}

type fileChange struct {
	Path string `json:"path"`
}

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
	cmd.Stderr = new(bytes.Buffer)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		failf("open stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		failf("open stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		failf("start MCP server: %v", err)
	}
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	encoder := json.NewEncoder(stdin)
	decoder := json.NewDecoder(stdout)

	writeRequest(encoder, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  mustMarshalRaw(initializeParams{ProtocolVersion: "2025-03-26"}),
	})
	initResponse := readResponse(decoder)
	mustNoRPCError("initialize", initResponse)
	var initResult initializeResult
	mustDecode(initResponse.Result, &initResult)
	if initResult.ProtocolVersion != "2025-03-26" {
		failf("initialize protocol mismatch: got %q want %q", initResult.ProtocolVersion, "2025-03-26")
	}

	writeRequest(encoder, rpcRequest{
		JSONRPC: jsonRPCVersion,
		Method:  "notifications/initialized",
	})

	writeRequest(encoder, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})
	listResponse := readResponse(decoder)
	mustNoRPCError("tools/list", listResponse)
	var listResult listToolsResult
	mustDecode(listResponse.Result, &listResult)
	if len(listResult.Tools) != 9 {
		failf("tools/list count mismatch: got %d want %d", len(listResult.Tools), 9)
	}

	writeRequest(encoder, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`3`),
		Method:  "tools/call",
		Params: mustMarshalRaw(callToolParams{
			Name: "memory_install_agents",
			Arguments: mustMarshalRaw(installAgentsRequest{
				Target:      "project",
				Mode:        "safe",
				CWD:         tempProject,
				ProjectName: "mcp-smoke-test",
				SystemName:  "codex-mem",
			}),
		}),
	})
	callResponse := readResponse(decoder)
	mustNoRPCError("tools/call", callResponse)
	var callResult toolCallResult
	mustDecode(callResponse.Result, &callResult)
	if callResult.IsError {
		failf("tool call returned isError=true")
	}
	var installResponse installAgentsResponse
	mustDecode(callResult.StructuredContent, &installResponse)
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

	fmt.Printf("mcp smoke test passed\n")
	fmt.Printf("protocol_version=%s\n", initResult.ProtocolVersion)
	fmt.Printf("tool_count=%d\n", len(listResult.Tools))
	fmt.Printf("tool_call=memory_install_agents\n")
	fmt.Printf("written_file=%s\n", agentsPath)
}

func failf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func writeRequest(encoder *json.Encoder, request rpcRequest) {
	if err := encoder.Encode(request); err != nil {
		failf("encode request: %v", err)
	}
}

func readResponse(decoder *json.Decoder) rpcResponse {
	var response rpcResponse
	if err := decoder.Decode(&response); err != nil {
		failf("decode response: %v", err)
	}
	return response
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
