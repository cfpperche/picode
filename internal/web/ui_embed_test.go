//go:build embedui

package web

import "testing"

// The shipped binary carries the UI. If this fails, `make build` produced
// something that would serve nothing to a user (ADR-0001).
func TestEmbeddedBuildCarriesTheUI(t *testing.T) {
	if !Embedded() {
		t.Fatal("built with -tags embedui but Embedded() is false")
	}
	if !Built() {
		t.Fatal("no index.html inside the binary")
	}
	if Dir() != "" {
		t.Fatalf("an embedded build reads nothing from disk, got Dir() = %q", Dir())
	}
}
