package mcp

import (
	"net/http"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewSDKHTTPHandler adapts the go-sdk streamable HTTP server behind the
// existing codex-mem HTTP endpoint and origin policy surface.
func NewSDKHTTPHandler(server *sdkmcp.Server, options HTTPOptions) http.Handler {
	endpointPath := strings.TrimSpace(options.EndpointPath)
	if endpointPath == "" {
		endpointPath = defaultHTTPEndpointPath
	}
	if !strings.HasPrefix(endpointPath, "/") {
		endpointPath = "/" + endpointPath
	}

	validator := newOriginValidator(options.AllowedOrigins)
	delegate := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server {
		return server
	}, &sdkmcp.StreamableHTTPOptions{
		JSONResponse:               true,
		Stateless:                  true,
		DisableLocalhostProtection: true,
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != endpointPath {
			http.NotFound(w, r)
			return
		}
		if err := validator.validateOrigin(r); err != nil {
			writeHTTPRPCError(w, http.StatusForbidden, rpcResponse{
				JSONRPC: jsonRPCVersion,
				Error:   &rpcError{Code: -32600, Message: err.Error()},
			})
			return
		}
		delegate.ServeHTTP(w, prepareSDKHTTPRequest(r))
	})
}

func prepareSDKHTTPRequest(r *http.Request) *http.Request {
	clone := r.Clone(r.Context())
	clone.Header = r.Header.Clone()

	if clone.Method != http.MethodPost {
		return clone
	}
	if strings.TrimSpace(clone.Header.Get("Content-Type")) == "" {
		clone.Header.Set("Content-Type", "application/json")
	}
	accept := strings.TrimSpace(clone.Header.Get("Accept"))
	switch {
	case accept == "":
		clone.Header.Set("Accept", "application/json, text/event-stream")
	case !containsMediaType(accept, "application/json") && !containsMediaType(accept, "application/*"):
		clone.Header.Set("Accept", accept+", application/json, text/event-stream")
	case !containsMediaType(accept, "text/event-stream"):
		clone.Header.Set("Accept", accept+", text/event-stream")
	}
	return clone
}

func containsMediaType(value string, want string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), want) {
			return true
		}
	}
	return false
}
