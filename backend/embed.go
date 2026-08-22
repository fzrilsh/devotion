// Package web embeds the built frontend so the Go binary serves it directly,
// keeping the deployment to a single artifact and a single origin (research
// R-06). CI overwrites webdist/ with the Vite build (frontend/dist) before the
// image is built; the committed index.html placeholder only exists so this
// embed directive compiles, since an empty directory is a compile error.
//
// The all: prefix is required: without it the embed tooling silently drops
// files whose names begin with "_" or ".", and Vite emits "_"-prefixed chunks.
package web

import "embed"

//go:embed all:webdist
var FS embed.FS
