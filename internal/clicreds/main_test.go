package clicreds

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain points Omp's catalog reader at nothing: with the omp of a
// developer's PATH in play, every declaration test would see ninety extra
// providers that CI never does. The catalog's own tests set their file.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "clicreds-test")
	if err != nil {
		panic(err)
	}
	os.Setenv(ompRulesEnv, filepath.Join(dir, "no-omp-rules.json"))
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
