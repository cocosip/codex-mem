package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultHTTPEndpointPath = "/mcp"

type HTTPOptions struct {
	EndpointPath   string
	AllowedOrigins []string
}

type HTTPHandler struct {
	server         *Server
	endpointPath   string
	allowedOrigins []string
}

type rpcEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func NewHTTPHandler(server *Server, options HTTPOptions) http.Handler {
	endpointPath := strings.TrimSpace(options.EndpointPath)
	if endpointPath == "" {
		endpointPath = defaultHTTPEndpointPath
	}
	if !strings.HasPrefix(endpointPath, "/") {
		endpointPath = "/" + endpointPath
	}
	allowedOrigins := make([]string, 0, len(options.AllowedOrigins))
	for _, origin := range options.AllowedOrigins {
		if normalized := normalizeOrigin(origin); normalized != "" {
			allowedOrigins = append(allowedOrigins, normalized)
		}
	}
	return &HTTPHandler{
		server:         server,
		endpointPath:   endpointPath,
		allowedOrigins: allowedOrigins,
	}
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != h.endpointPath {
		http.NotFound(w, r)
		return
	}
	if err := h.validateOrigin(r); err != nil {
		writeHTTPRPCError(w, http.StatusForbidden, rpcResponse{
			JSONRPC: jsonRPCVersion,
			Error:   &rpcError{Code: -32600, Message: err.Error()},
		})
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.handlePOST(w, r)
	case http.MethodGet:
		w.Header().Set("Allow", "POST, GET")
		http.Error(w, "SSE stream not supported", http.StatusMethodNotAllowed)
	case http.MethodDelete:
		w.Header().Set("Allow", "POST, GET")
		http.Error(w, "session termination not supported", http.StatusMethodNotAllowed)
	default:
		w.Header().Set("Allow", "POST, GET, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPHandler) handlePOST(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeHTTPRPCError(w, http.StatusBadRequest, rpcResponse{
			JSONRPC: jsonRPCVersion,
			Error:   &rpcError{Code: -32700, Message: "read request body"},
		})
		return
	}
	payload := strings.TrimSpace(string(body))
	if payload == "" {
		writeHTTPRPCError(w, http.StatusBadRequest, rpcResponse{
			JSONRPC: jsonRPCVersion,
			Error:   &rpcError{Code: -32600, Message: "request body is required"},
		})
		return
	}

	isBatch := strings.HasPrefix(payload, "[")
	messages, err := decodeHTTPMessages([]byte(payload), isBatch)
	if err != nil {
		writeHTTPRPCError(w, http.StatusBadRequest, rpcResponse{
			JSONRPC: jsonRPCVersion,
			Error:   &rpcError{Code: -32700, Message: "parse error"},
		})
		return
	}

	responses := make([]rpcResponse, 0, len(messages))
	requestCount := 0
	for _, message := range messages {
		var envelope rpcEnvelope
		if err := json.Unmarshal(message, &envelope); err != nil {
			writeHTTPRPCError(w, http.StatusBadRequest, rpcResponse{
				JSONRPC: jsonRPCVersion,
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			})
			return
		}

		if !envelope.isRequestOrNotification() {
			continue
		}

		var request rpcRequest
		if err := json.Unmarshal(message, &request); err != nil {
			writeHTTPRPCError(w, http.StatusBadRequest, rpcResponse{
				JSONRPC: jsonRPCVersion,
				Error:   &rpcError{Code: -32700, Message: "parse error"},
			})
			return
		}

		if hasResponseID(request) {
			requestCount++
		}
		response, shouldRespond := h.server.handleRequest(r.Context(), request)
		if shouldRespond {
			responses = append(responses, response)
		}
	}

	if requestCount == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !isBatch && len(responses) == 1 {
		_ = json.NewEncoder(w).Encode(responses[0])
		return
	}
	_ = json.NewEncoder(w).Encode(responses)
}

func (h *HTTPHandler) validateOrigin(r *http.Request) error {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return nil
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid origin")
	}

	normalizedOrigin := normalizeOrigin(origin)
	requestHost := normalizeHost(r.Host)
	if normalizeHost(parsed.Host) == requestHost || normalizedOrigin == requestHost {
		return nil
	}
	for _, allowed := range h.allowedOrigins {
		if normalizedOrigin == allowed || normalizeHost(parsed.Host) == allowed {
			return nil
		}
	}
	return fmt.Errorf("origin not allowed")
}

func decodeHTTPMessages(payload []byte, isBatch bool) ([]json.RawMessage, error) {
	if isBatch {
		var messages []json.RawMessage
		if err := json.Unmarshal(payload, &messages); err != nil {
			return nil, err
		}
		return messages, nil
	}
	return []json.RawMessage{json.RawMessage(payload)}, nil
}

func writeHTTPRPCError(w http.ResponseWriter, status int, response rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func (e rpcEnvelope) isRequestOrNotification() bool {
	return strings.TrimSpace(e.Method) != ""
}

func normalizeOrigin(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.Host != "" {
		return strings.ToLower(parsed.Scheme + "://" + parsed.Host)
	}
	return normalizeHost(value)
}

func normalizeHost(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(value)
	if err == nil {
		if port == "80" || port == "443" {
			return strings.ToLower(host)
		}
		return strings.ToLower(net.JoinHostPort(host, port))
	}
	return strings.ToLower(value)
}

func ServeHTTP(ctx context.Context, addr string, handler http.Handler) error {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
