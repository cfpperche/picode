package pimission

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureUsesPrivateHomeFallbackAndNoPackageImports(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := Ensure("")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".picode", "intercept", "pi-mission.ts"); path != want {
		t.Fatalf("extension path = %q, want %q", path, want)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, external := range []string{"@earendil-works/pi-coding-agent", "typebox"} {
		if strings.Contains(string(source), external) {
			t.Fatalf("native extension imports external package %q", external)
		}
	}
}
