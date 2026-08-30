//go:build embedui

package web

import (
	"embed"
	"io/fs"
)

// all: so the directory is embeddable even when Vite writes a dot-prefixed
// file — a plain `go:embed public` skips those, and an all-dot directory
// fails to compile at all.
//
//go:embed all:public
var public embed.FS

// UI is the build output sealed into the binary at compile time.
func UI() fs.FS {
	sub, err := fs.Sub(public, "public")
	if err != nil {
		// Cannot happen: go:embed refuses to compile without public/.
		panic("picode: embedded UI missing: " + err.Error())
	}
	return sub
}

// Dir is empty in an embedded build: nothing is read from disk.
func Dir() string { return "" }

// Embedded reports how this binary was built.
func Embedded() bool { return true }
