// Package buildinfo exposes version metadata embedded at build time.
package buildinfo

import "strings"

var (
	// Version is the semantic version or build identifier embedded at link time.
	Version = "dev"
	// Commit is the source commit embedded at link time.
	Commit = "unknown"
	// Date is the build timestamp embedded at link time.
	Date = "unknown"
)

// Summary returns the normalized version string for CLI and MCP metadata.
func Summary() string {
	version := strings.TrimSpace(Version)
	if version == "" {
		version = "dev"
	}
	return version
}
