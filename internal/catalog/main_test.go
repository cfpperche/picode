package catalog

import (
	"os"
	"strings"
	"testing"
)

// The vault follows PICODE_DATA before HOME (internal/credentials defaultDir)
// and the agent runtime hands it to every PiCode terminal, so a bare
// `go test ./internal/catalog` from inside PiCode read and wrote the live
// vault: this package syncs pi's auth.json into it (accounts.go). The suite
// gets a throwaway data dir for the whole process, and the guardrail below
// keeps it there.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "picode-catalog-data")
	if err != nil {
		panic(err)
	}
	os.Setenv("PICODE_DATA", dir)
	os.Exit(m.Run())
}

func TestSuiteIsVaultIsolated(t *testing.T) {
	dir := os.Getenv("PICODE_DATA")
	if dir == "" {
		t.Fatal("PICODE_DATA is unset — the suite would resolve the live vault; keep the TestMain sandbox")
	}
	if !strings.HasPrefix(dir, os.TempDir()) {
		t.Fatalf("PICODE_DATA = %s — outside the temp sandbox; keep the TestMain sandbox", dir)
	}
}
