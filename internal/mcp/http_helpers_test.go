package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
