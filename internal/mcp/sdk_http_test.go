package mcp

import (
	"encoding/json"
	"net/http"
	"testing"

	"codex-mem/internal/domain/agents"
)

func TestSDKHTTPHandlerPOSTSupportsInitializeListAndToolCall(t *testing.T) {
	root := t.TempDir()
	handler := NewSDKHTTPHandler(NewSDKServer(&Handlers{
		agentsService: agents.NewService(agents.Options{HomeDir: root}),
	}), HTTPOptions{})

	initResponse := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  mustMarshalRaw(t, initializeParams{ProtocolVersion: "2025-03-26"}),
	}, "", "")
	if got, want := initResponse.Code, http.StatusOK; got != want {
		t.Fatalf("initialize status mismatch: got %d want %d", got, want)
	}
	var initRPC rpcResponse
	decodeHTTPBody(t, initResponse.Body.Bytes(), &initRPC)
	if initRPC.Error != nil {
		t.Fatalf("initialize rpc error: %+v", initRPC.Error)
	}
	var initResult initializeResult
	decodeJSONValue(t, initRPC.Result, &initResult)
	if got, want := initResult.ProtocolVersion, "2025-03-26"; got != want {
		t.Fatalf("protocol mismatch: got %q want %q", got, want)
	}

	notification := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		Method:  "notifications/initialized",
	}, "", "")
	if got, want := notification.Code, http.StatusAccepted; got != want {
		t.Fatalf("notification status mismatch: got %d want %d", got, want)
	}

	listResponse := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	}, "", "")
	if got, want := listResponse.Code, http.StatusOK; got != want {
		t.Fatalf("tools/list status mismatch: got %d want %d", got, want)
	}
	var listRPC rpcResponse
	decodeHTTPBody(t, listResponse.Body.Bytes(), &listRPC)
	if listRPC.Error != nil {
		t.Fatalf("tools/list rpc error: %+v", listRPC.Error)
	}
	var listResult listToolsResult
	decodeJSONValue(t, listRPC.Result, &listResult)
	if got, want := len(listResult.Tools), 9; got != want {
		t.Fatalf("tool count mismatch: got %d want %d", got, want)
	}

	callResponse := performHTTPRPCRequest(t, handler, rpcRequest{
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
	}, "", "")
	if got, want := callResponse.Code, http.StatusOK; got != want {
		t.Fatalf("tools/call status mismatch: got %d want %d", got, want)
	}
	var callRPC rpcResponse
	decodeHTTPBody(t, callResponse.Body.Bytes(), &callRPC)
	if callRPC.Error != nil {
		t.Fatalf("tools/call rpc error: %+v", callRPC.Error)
	}
	var callResult toolCallResult
	decodeJSONValue(t, callRPC.Result, &callResult)
	if callResult.IsError {
		t.Fatalf("expected successful tool call, got %+v", callResult)
	}
}

func TestSDKHTTPHandlerRejectsUntrustedOrigin(t *testing.T) {
	handler := NewSDKHTTPHandler(NewSDKServer(&Handlers{}), HTTPOptions{})
	response := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "ping",
		Params:  json.RawMessage(`{}`),
	}, "http://evil.example.com", "localhost:8080")
	if got, want := response.Code, http.StatusForbidden; got != want {
		t.Fatalf("status mismatch: got %d want %d", got, want)
	}
}
