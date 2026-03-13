package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"codex-mem/internal/domain/agents"
)

func TestServerServeSupportsInitializeListAndToolCall(t *testing.T) {
	root := t.TempDir()
	server := NewServer(&Handlers{
		agentsService: agents.NewService(agents.Options{HomeDir: root}),
	})

	var input bytes.Buffer
	writeRequestFrame(t, &input, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  mustMarshalRaw(t, initializeParams{ProtocolVersion: "2025-03-26"}),
	})
	writeRequestFrame(t, &input, rpcRequest{
		JSONRPC: jsonRPCVersion,
		Method:  "notifications/initialized",
	})
	writeRequestFrame(t, &input, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})
	writeRequestFrame(t, &input, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`3`),
		Method:  "tools/call",
		Params: mustMarshalRaw(t, callToolParams{
			Name: "memory_install_agents",
			Arguments: mustMarshalRaw(t, installAgentsRequest{
				Target:      "project",
				Mode:        "safe",
				CWD:         root,
				ProjectName: "codex-mem",
				SystemName:  "codex-mem",
			}),
		}),
	})

	var output bytes.Buffer
	if err := server.Serve(context.Background(), &input, &output); err != nil {
		t.Fatalf("serve failed: %v", err)
	}

	reader := bufio.NewReader(&output)

	initResponse := readResponseFrame(t, reader)
	if initResponse.Error != nil {
		t.Fatalf("initialize returned error: %+v", initResponse.Error)
	}
	var initResult initializeResult
	decodeJSONValue(t, initResponse.Result, &initResult)
	if got, want := initResult.ProtocolVersion, "2025-03-26"; got != want {
		t.Fatalf("protocol version mismatch: got %q want %q", got, want)
	}

	listResponse := readResponseFrame(t, reader)
	if listResponse.Error != nil {
		t.Fatalf("tools/list returned error: %+v", listResponse.Error)
	}
	var listResult listToolsResult
	decodeJSONValue(t, listResponse.Result, &listResult)
	if got, want := len(listResult.Tools), 9; got != want {
		t.Fatalf("tool count mismatch: got %d want %d", got, want)
	}

	callResponse := readResponseFrame(t, reader)
	if callResponse.Error != nil {
		t.Fatalf("tools/call returned error: %+v", callResponse.Error)
	}
	var callResult toolCallResult
	decodeJSONValue(t, callResponse.Result, &callResult)
	if callResult.IsError {
		t.Fatalf("expected successful tool call, got %+v", callResult)
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

func TestServerServeReturnsInvalidParamsForUnknownTool(t *testing.T) {
	server := NewServer(&Handlers{})

	var input bytes.Buffer
	writeRequestFrame(t, &input, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params: mustMarshalRaw(t, callToolParams{
			Name:      "missing_tool",
			Arguments: json.RawMessage(`{}`),
		}),
	})

	var output bytes.Buffer
	if err := server.Serve(context.Background(), &input, &output); err != nil {
		t.Fatalf("serve failed: %v", err)
	}

	response := readResponseFrame(t, bufio.NewReader(&output))
	if response.Error == nil {
		t.Fatal("expected rpc error")
	}
	if got, want := response.Error.Code, -32602; got != want {
		t.Fatalf("error code mismatch: got %d want %d", got, want)
	}
}

func writeRequestFrame(t *testing.T, dst *bytes.Buffer, request rpcRequest) {
	t.Helper()

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if _, err := dst.WriteString("Content-Length: "); err != nil {
		t.Fatalf("write header label: %v", err)
	}
	if _, err := dst.WriteString(strconv.Itoa(len(body))); err != nil {
		t.Fatalf("write header length: %v", err)
	}
	if _, err := dst.WriteString("\r\n\r\n"); err != nil {
		t.Fatalf("write header separator: %v", err)
	}
	if _, err := dst.Write(body); err != nil {
		t.Fatalf("write body: %v", err)
	}
}

func readResponseFrame(t *testing.T, reader *bufio.Reader) rpcResponse {
	t.Helper()

	payload, err := readFrame(reader)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}

	var response rpcResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return response
}

func decodeJSONValue(t *testing.T, value any, target any) {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("unmarshal value: %v", err)
	}
}

func mustMarshalRaw(t *testing.T, value any) json.RawMessage {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal raw value: %v", err)
	}
	return json.RawMessage(body)
}
