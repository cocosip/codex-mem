// Package migrations embeds the SQLite schema migration files.
package migrations

import "embed"

// FS contains the ordered SQL migrations for the local SQLite store.
//
//go:embed *.sql
var FS embed.FS
