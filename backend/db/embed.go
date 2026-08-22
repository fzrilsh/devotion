// Package db exposes the embedded SQL migrations so the binary carries its own
// schema and needs no external migrate tool in the image.
package db

import "embed"

// MigrationsFS holds every migration file. The runner in
// internal/platform/migrate reads it through an iofs source.
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
