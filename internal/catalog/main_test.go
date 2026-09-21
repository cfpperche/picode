package catalog

import (
	"os"
	"strings"
	"testing"
)

// The vault follows PICODE_DATA before HOME (internal/credentials defaultDir)
// and the agent runtime hands it to every PiCode terminal, so a bare
// `go test ./internal/catalog` from inside PiCode read and wrote the live
// vault — this package syncs pi's auth.json into it (accounts.go). 30 test
// fixtures landed in the owner's real vault that way on 2026-09-21.
//
// Two things, mirroring internal/server's sandbox: the ambient PICODE_DATA is
// removed (it outranks HOME, and tests here isolate themselves by HOME), and
// HOME gets a throwaway directory for tests that do not set their own.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "picode-catalog-home")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)
	os.Unsetenv("PICODE_DATA")
	os.Exit(m.Run())
}

func TestSuiteIsVaultIsolated(t *testing.T) {
	if dir := os.Getenv("PICODE_DATA"); dir != "" {
		t.Fatalf("PICODE_DATA = %s — it outranks HOME and would point the vault at the live one; keep the TestMain sandbox", dir)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(home, os.TempDir()) {
		t.Fatalf("UserHomeDir = %s — outside the temp sandbox; keep the TestMain sandbox", home)
	}
}
