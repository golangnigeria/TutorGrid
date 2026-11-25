package migrations

import "embed"

// FS holds embedded SQL migration files for Goose.
//
//go:embed *.sql
var FS embed.FS
