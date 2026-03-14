package app

import (
	"testing"
	"time"
)

func TestParseServeHTTPOptionsDefaults(t *testing.T) {
	options, err := parseServeHTTPOptions(nil)
	if err != nil {
		t.Fatalf("parseServeHTTPOptions: %v", err)
	}
	if got, want := options.ListenAddr, "127.0.0.1:8080"; got != want {
		t.Fatalf("listen mismatch: got %q want %q", got, want)
	}
	if got, want := options.EndpointPath, "/mcp"; got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
	if len(options.AllowedOrigins) != 0 {
		t.Fatalf("expected no allowed origins, got %+v", options.AllowedOrigins)
	}
	if options.SessionTimeout != 0 {
		t.Fatalf("expected no session timeout, got %s", options.SessionTimeout)
	}
}

func TestParseServeHTTPOptionsOverridesValues(t *testing.T) {
	options, err := parseServeHTTPOptions([]string{
		"--listen", "0.0.0.0:9090",
		"--path", "remote",
		"--allow-origin", "https://client.example.com",
		"--allow-origin", "https://admin.example.com",
		"--session-timeout", "45s",
	})
	if err != nil {
		t.Fatalf("parseServeHTTPOptions: %v", err)
	}
	if got, want := options.ListenAddr, "0.0.0.0:9090"; got != want {
		t.Fatalf("listen mismatch: got %q want %q", got, want)
	}
	if got, want := options.EndpointPath, "/remote"; got != want {
		t.Fatalf("path mismatch: got %q want %q", got, want)
	}
	if got, want := len(options.AllowedOrigins), 2; got != want {
		t.Fatalf("allowed origin count mismatch: got %d want %d", got, want)
	}
	if got, want := options.SessionTimeout, 45*time.Second; got != want {
		t.Fatalf("session timeout mismatch: got %s want %s", got, want)
	}
}

func TestParseServeHTTPOptionsRejectsUnknownFlag(t *testing.T) {
	_, err := parseServeHTTPOptions([]string{"--unknown"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestParseServeHTTPOptionsRejectsInvalidSessionTimeout(t *testing.T) {
	_, err := parseServeHTTPOptions([]string{"--session-timeout", "later"})
	if err == nil {
		t.Fatal("expected error for invalid session timeout")
	}
}
