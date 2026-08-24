// Package web embeds the frontend assets shipped inside the picode binary
// (ADR-0001: browser app served by a single Go binary).
package web

import "embed"

// Public holds the production UI (Vite build output from web/).
// Rebuild with `make web` (ADR-0008). go:embed is compile-time (ADR-0001).
//
//go:embed public
var Public embed.FS
