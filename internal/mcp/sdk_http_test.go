package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
	}, httpTestRequestOptions{})
	if got, want := initResponse.Code, http.StatusOK; got != want {
		t.Fatalf("initialize status mismatch: got %d want %d", got, want)
	}
	sessionID := initResponse.Header().Get(mcpSessionIDHeader)
	if sessionID == "" {
		t.Fatal("initialize missing session header")
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
	}, httpTestRequestOptions{SessionID: sessionID})
	if got, want := notification.Code, http.StatusAccepted; got != want {
		t.Fatalf("notification status mismatch: got %d want %d", got, want)
	}

	listResponse := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	}, httpTestRequestOptions{SessionID: sessionID})
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
	if got, want := len(listResult.Tools), 10; got != want {
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
	}, httpTestRequestOptions{SessionID: sessionID})
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

func TestSDKHTTPHandlerGETOpensSSEStreamForInitializedSession(t *testing.T) {
	root := t.TempDir()
	handler := NewSDKHTTPHandler(NewSDKServer(&Handlers{
		agentsService: agents.NewService(agents.Options{HomeDir: root}),
	}), HTTPOptions{})
	server := httptest.NewServer(handler)
	defer server.Close()

	initResponse := doHTTPRPC(t, server.URL+"/mcp", rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  mustMarshalRaw(t, initializeParams{ProtocolVersion: "2025-03-26"}),
	}, "")
	if got, want := initResponse.StatusCode, http.StatusOK; got != want {
		t.Fatalf("initialize status mismatch: got %d want %d", got, want)
	}
	sessionID := initResponse.Header.Get(mcpSessionIDHeader)
	if sessionID == "" {
		t.Fatal("initialize missing session header")
	}
	_ = initResponse.Body.Close()

	initialized := doHTTPRPC(t, server.URL+"/mcp", rpcRequest{
		JSONRPC: jsonRPCVersion,
		Method:  "notifications/initialized",
	}, sessionID)
	if got, want := initialized.StatusCode, http.StatusAccepted; got != want {
		t.Fatalf("notification status mismatch: got %d want %d", got, want)
	}
	_ = initialized.Body.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/mcp", nil)
	if err != nil {
		t.Fatalf("build GET request: %v", err)
	}
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set(mcpSessionIDHeader, sessionID)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET SSE request failed: %v", err)
	}
	defer func() {
		cancel()
		_ = response.Body.Close()
	}()

	if got, want := response.StatusCode, http.StatusOK; got != want {
		t.Fatalf("GET status mismatch: got %d want %d", got, want)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("GET content-type mismatch: got %q", contentType)
	}
}

func TestSDKHTTPHandlerRejectsUntrustedOrigin(t *testing.T) {
	handler := NewSDKHTTPHandler(NewSDKServer(&Handlers{}), HTTPOptions{})
	response := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "ping",
		Params:  json.RawMessage(`{}`),
	}, httpTestRequestOptions{
		Origin: "http://evil.example.com",
		Host:   "localhost:8080",
	})
	if got, want := response.Code, http.StatusForbidden; got != want {
		t.Fatalf("status mismatch: got %d want %d", got, want)
	}
}

func TestSDKHTTPHandlerExpiresIdleSession(t *testing.T) {
	root := t.TempDir()
	handler := NewSDKHTTPHandler(NewSDKServer(&Handlers{
		agentsService: agents.NewService(agents.Options{HomeDir: root}),
	}), HTTPOptions{SessionTimeout: 50 * time.Millisecond})

	initResponse := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  mustMarshalRaw(t, initializeParams{ProtocolVersion: "2025-03-26"}),
	}, httpTestRequestOptions{})
	if got, want := initResponse.Code, http.StatusOK; got != want {
		t.Fatalf("initialize status mismatch: got %d want %d", got, want)
	}
	sessionID := initResponse.Header().Get(mcpSessionIDHeader)
	if sessionID == "" {
		t.Fatal("initialize missing session header")
	}

	time.Sleep(120 * time.Millisecond)

	listResponse := performHTTPRPCRequest(t, handler, rpcRequest{
		JSONRPC: jsonRPCVersion,
		ID:      json.RawMessage(`2`),
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	}, httpTestRequestOptions{SessionID: sessionID})
	if got, want := listResponse.Code, http.StatusNotFound; got != want {
		t.Fatalf("expired session status mismatch: got %d want %d", got, want)
	}
}

func doHTTPRPC(t *testing.T, endpointURL string, request rpcRequest, sessionID string) *http.Response {
	t.Helper()

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	httpRequest, err := http.NewRequest(http.MethodPost, endpointURL, strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		httpRequest.Header.Set(mcpSessionIDHeader, sessionID)
	}

	response, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return response
}
