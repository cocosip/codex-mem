package mcp

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"codex-mem/internal/domain/agents"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServeStdioSupportsInitializeListAndToolCall(t *testing.T) {
	root := t.TempDir()
	server := NewSDKServer(&Handlers{
		agentsService: agents.NewService(agents.Options{HomeDir: root}),
	})

	client := startServeStdioClient(t, server)
	defer client.close(t)

	writeRequestMessage(t, client.encoder, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  mustMarshalRaw(t, initializeParams{ProtocolVersion: "2025-03-26"}),
	})
	initResponse := readResponseMessage(t, client.decoder)
	if initResponse.Error != nil {
		t.Fatalf("initialize returned error: %+v", initResponse.Error)
	}
	var initResult initializeResult
	mustDecodeValue(t, initResponse.Result, &initResult)
	if got, want := initResult.ProtocolVersion, "2025-03-26"; got != want {
		t.Fatalf("protocol version mismatch: got %q want %q", got, want)
	}

	writeRequestMessage(t, client.encoder, rpcRequest{
		JSONRPC: jsonRPCVersion,
		Method:  "notifications/initialized",
	})
	writeRequestMessage(t, client.encoder, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})
	listResponse := readResponseMessage(t, client.decoder)
	if listResponse.Error != nil {
		t.Fatalf("tools/list returned error: %+v", listResponse.Error)
	}
	var listResult listToolsResult
	mustDecodeValue(t, listResponse.Result, &listResult)
	if got, want := len(listResult.Tools), 9; got != want {
		t.Fatalf("tool count mismatch: got %d want %d", got, want)
	}

	writeRequestMessage(t, client.encoder, rpcRequest{
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
	callResponse := readResponseMessage(t, client.decoder)
	if callResponse.Error != nil {
		t.Fatalf("tools/call returned error: %+v", callResponse.Error)
	}
	var callResult toolCallResult
	mustDecodeValue(t, callResponse.Result, &callResult)
	if callResult.IsError {
		t.Fatalf("expected successful tool call, got %+v", callResult)
	}
	if got, want := len(callResult.Content), 1; got != want {
		t.Fatalf("content count mismatch: got %d want %d", got, want)
	}

	var structured Response[InstallAgentsData]
	mustDecodeValue(t, callResult.StructuredContent, &structured)
	if !structured.Ok {
		t.Fatalf("expected ok structured content, got %+v", structured.Error)
	}
	if structured.Data == nil || len(structured.Data.WrittenFiles) != 1 {
		t.Fatalf("expected one written file, got %+v", structured.Data)
	}
}

func TestServeStdioReturnsInvalidParamsForUnknownTool(t *testing.T) {
	server := NewSDKServer(&Handlers{})

	client := startServeStdioClient(t, server)
	defer client.close(t)

	writeRequestMessage(t, client.encoder, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  mustMarshalRaw(t, initializeParams{ProtocolVersion: "2025-03-26"}),
	})
	initResponse := readResponseMessage(t, client.decoder)
	if initResponse.Error != nil {
		t.Fatalf("initialize returned error: %+v", initResponse.Error)
	}

	writeRequestMessage(t, client.encoder, rpcRequest{
		JSONRPC: jsonRPCVersion,
		Method:  "notifications/initialized",
	})
	writeRequestMessage(t, client.encoder, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/call",
		Params: mustMarshalRaw(t, callToolParams{
			Name:      "missing_tool",
			Arguments: json.RawMessage(`{}`),
		}),
	})
	response := readResponseMessage(t, client.decoder)
	if response.Error == nil {
		t.Fatal("expected rpc error")
	}
	if got, want := response.Error.Code, -32602; got != want {
		t.Fatalf("error code mismatch: got %d want %d", got, want)
	}
}

type stdioTestClient struct {
	encoder   *json.Encoder
	decoder   *json.Decoder
	stdin     *io.PipeWriter
	stdout    *io.PipeReader
	serverErr <-chan error
}

func startServeStdioClient(t *testing.T, server *sdkmcp.Server) *stdioTestClient {
	t.Helper()

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	serverErr := make(chan error, 1)

	go func() {
		defer func() {
			_ = stdoutWriter.Close()
		}()
		serverErr <- ServeStdio(context.Background(), server, stdinReader, stdoutWriter)
	}()

	return &stdioTestClient{
		encoder:   json.NewEncoder(stdinWriter),
		decoder:   json.NewDecoder(stdoutReader),
		stdin:     stdinWriter,
		stdout:    stdoutReader,
		serverErr: serverErr,
	}
}

func (c *stdioTestClient) close(t *testing.T) {
	t.Helper()
	_ = c.stdin.Close()
	if err := <-c.serverErr; err != nil {
		t.Fatalf("server err: %v", err)
	}
	_ = c.stdout.Close()
}

func writeRequestMessage(t *testing.T, encoder *json.Encoder, request rpcRequest) {
	t.Helper()
	if err := encoder.Encode(request); err != nil {
		t.Fatalf("encode request: %v", err)
	}
}

func readResponseMessage(t *testing.T, decoder *json.Decoder) rpcResponse {
	t.Helper()

	var response rpcResponse
	if err := decoder.Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}

func decodeJSONValue(t *testing.T, value any, target any) {
	mustDecodeValue(t, value, target)
}

func mustDecodeValue(t *testing.T, value any, target any) {
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
