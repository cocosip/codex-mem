package buildinfo

import "strings"

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func Summary() string {
	version := strings.TrimSpace(Version)
	if version == "" {
		version = "dev"
	}
	return version
}
