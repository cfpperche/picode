//go:build embedui

package web

import (
	"io/fs"
	"testing"
)

// The shipped binary carries the UI. If this fails, `make build` produced
// something that would serve nothing to a user (ADR-0001).
func TestEmbeddedBuildCarriesTheUI(t *testing.T) {
	if !Embedded() {
		t.Fatal("built with -tags embedui but Embedded() is false")
	}
	if !Built() {
		t.Fatal("no index.html inside the binary")
	}
	for _, path := range []string{"desktop/index.html", "mobile/index.html", "manifest.json", "sw.js"} {
		if _, err := fs.Stat(UI(), path); err != nil {
			t.Fatalf("incomplete frontend release: %s: %v", path, err)
		}
	}
	if Dir() != "" {
		t.Fatalf("an embedded build reads nothing from disk, got Dir() = %q", Dir())
	}
}
