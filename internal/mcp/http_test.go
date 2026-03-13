package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"codex-mem/internal/domain/agents"
)

func TestHTTPHandlerPOSTSupportsInitializeListAndToolCall(t *testing.T) {
	root := t.TempDir()
	handler := NewHTTPHandler(NewServer(&Handlers{
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

	listResponse := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	}, "", "")
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

func TestHTTPHandlerPOSTNotificationReturnsAccepted(t *testing.T) {
	handler := NewHTTPHandler(NewServer(&Handlers{}), HTTPOptions{})
	response := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		Method:  "notifications/initialized",
	}, "", "")
	if got, want := response.Code, http.StatusAccepted; got != want {
		t.Fatalf("status mismatch: got %d want %d", got, want)
	}
	if body := response.Body.String(); body != "" {
		t.Fatalf("expected empty body, got %q", body)
	}
}

func TestHTTPHandlerGETReturnsMethodNotAllowed(t *testing.T) {
	handler := NewHTTPHandler(NewServer(&Handlers{}), HTTPOptions{})
	request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if got, want := response.Code, http.StatusMethodNotAllowed; got != want {
		t.Fatalf("status mismatch: got %d want %d", got, want)
	}
}

func TestHTTPHandlerRejectsUntrustedOrigin(t *testing.T) {
	handler := NewHTTPHandler(NewServer(&Handlers{}), HTTPOptions{})
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

func TestHTTPHandlerAllowsConfiguredOrigin(t *testing.T) {
	handler := NewHTTPHandler(NewServer(&Handlers{}), HTTPOptions{
		AllowedOrigins: []string{"https://client.example.com"},
	})
	response := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "ping",
		Params:  json.RawMessage(`{}`),
	}, "https://client.example.com", "localhost:8080")
	if got, want := response.Code, http.StatusOK; got != want {
		t.Fatalf("status mismatch: got %d want %d", got, want)
	}
}

func TestHTTPHandlerBatchRequestReturnsBatchResponses(t *testing.T) {
	handler := NewHTTPHandler(NewServer(&Handlers{}), HTTPOptions{})
	body, err := json.Marshal([]rpcRequest{
		{
			JSONRPC: jsonRPCVersion,
			ID:      json.RawMessage(`1`),
			Method:  "initialize",
			Params:  mustMarshalRaw(t, initializeParams{ProtocolVersion: "2025-03-26"}),
		},
		{
			JSONRPC: jsonRPCVersion,
			ID:      json.RawMessage(`2`),
			Method:  "ping",
			Params:  json.RawMessage(`{}`),
		},
	})
	if err != nil {
		t.Fatalf("marshal batch: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got, want := response.Code, http.StatusOK; got != want {
		t.Fatalf("status mismatch: got %d want %d", got, want)
	}
	var batch []rpcResponse
	decodeHTTPBody(t, response.Body.Bytes(), &batch)
	if got, want := len(batch), 2; got != want {
		t.Fatalf("response count mismatch: got %d want %d", got, want)
	}
}

func performHTTPRPCRequest(t *testing.T, handler http.Handler, request rpcRequest, origin string, host string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	httpRequest := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	if origin != "" {
		httpRequest.Header.Set("Origin", origin)
	}
	if host != "" {
		httpRequest.Host = host
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httpRequest)
	return response
}

func decodeHTTPBody(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode body: %v\n%s", err, string(body))
	}
}
