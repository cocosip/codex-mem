package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const mcpSessionIDHeader = "Mcp-Session-Id"

type httpTestRequestOptions struct {
	Method    string
	Origin    string
	Host      string
	Accept    string
	SessionID string
}

func performHTTPRPCRequest(t *testing.T, handler http.Handler, request rpcRequest, options httpTestRequestOptions) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	method := options.Method
	if method == "" {
		method = http.MethodPost
	}
	httpRequest := httptest.NewRequest(method, "/mcp", bytes.NewReader(body))
	if options.Origin != "" {
		httpRequest.Header.Set("Origin", options.Origin)
	}
	if options.Host != "" {
		httpRequest.Host = options.Host
	}
	if options.Accept != "" {
		httpRequest.Header.Set("Accept", options.Accept)
	}
	if options.SessionID != "" {
		httpRequest.Header.Set(mcpSessionIDHeader, options.SessionID)
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
