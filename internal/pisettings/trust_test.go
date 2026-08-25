package pisettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTrust(t *testing.T, m map[string]bool) string {
	t.Helper()
	trust := filepath.Join(t.TempDir(), "trust.json")
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trust, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return trust
}

func TestTrustedSelfAndParent(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "app")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	trust := writeTrust(t, map[string]bool{root: true})
	if !trustedAt(child, trust) {
		t.Fatal("parent trust should cover child")
	}
	if !trustedAt(root, trust) {
		t.Fatal("self")
	}
	if trustedAt(t.TempDir(), trust) {
		t.Fatal("other dir")
	}
}

func TestTrustedFalseWins(t *testing.T) {
	dir := t.TempDir()
	trust := writeTrust(t, map[string]bool{dir: false})
	if trustedAt(dir, trust) {
		t.Fatal("explicit false")
	}
}

func TestTrustedMissingFile(t *testing.T) {
	if Trusted(t.TempDir()) {
		t.Fatal("no trust.json")
	}
}
