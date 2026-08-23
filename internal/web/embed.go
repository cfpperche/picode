// Package web embeds the frontend assets shipped inside the picode binary
// (ADR-0001: browser app served by a single Go binary).
package web

import "embed"

// Public holds the UI assets served at "/". Add hashed asset names when a
// build step lands (see docs/handoff.md "Known debts" re: vite).
//
//go:embed public
var Public embed.FS
