package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetName(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.jsonl")
	if err := os.WriteFile(p, []byte(`{"type":"session","id":"abc","timestamp":"2026-08-24T01:00:00Z"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetName(p, "Refactor auth"); err != nil {
		t.Fatal(err)
	}
	s, err := Summarize(p)
	if err != nil || s.Name != "Refactor auth" {
		t.Fatalf("%v %+v", err, s)
	}
}
