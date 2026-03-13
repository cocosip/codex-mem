// Package templates embeds AGENTS.md templates for runtime installation.
package templates

import _ "embed"

// GlobalAgentsTemplate contains the global AGENTS.md template shipped with the binary.
//
//go:embed AGENTS.global.template.md
var GlobalAgentsTemplate string

// ProjectAgentsTemplate contains the project AGENTS.md template shipped with the binary.
//
//go:embed AGENTS.project.template.md
var ProjectAgentsTemplate string

