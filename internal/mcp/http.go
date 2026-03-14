package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultHTTPEndpointPath = "/mcp"

// HTTPOptions configures the HTTP transport wrapper around the MCP server.
type HTTPOptions struct {
	EndpointPath   string
	AllowedOrigins []string
	SessionTimeout time.Duration
}

type originValidator struct {
	allowedOrigins []string
}

func newOriginValidator(allowedOrigins []string) *originValidator {
	normalized := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if value := normalizeOrigin(origin); value != "" {
			normalized = append(normalized, value)
		}
	}
	return &originValidator{allowedOrigins: normalized}
}

func (v *originValidator) validateOrigin(r *http.Request) error {
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
	for _, allowed := range v.allowedOrigins {
		if normalizedOrigin == allowed || normalizeHost(parsed.Host) == allowed {
			return nil
		}
	}
	return fmt.Errorf("origin not allowed")
}

func writeHTTPRPCError(w http.ResponseWriter, status int, response rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
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

// ServeHTTP runs the HTTP MCP server until shutdown or listener failure.
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
