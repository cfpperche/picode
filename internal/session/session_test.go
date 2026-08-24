package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirName(t *testing.T) {
	if got := DirName("/home/goat/picode"); got != "--home-goat-picode--" {
		t.Fatalf("DirName = %q", got)
	}
}

func TestSummarize(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "s.jsonl")
	body := `{"type":"session","version":3,"id":"abc","timestamp":"2026-08-24T01:00:00.000Z","cwd":"/tmp"}
{"type":"session_info","id":"n1","parentId":"abc","timestamp":"2026-08-24T01:00:01.000Z","name":"Refactor auth"}
{"type":"message","message":{"role":"user","content":[{"type":"text","text":"List the files"}],"timestamp":1}}
{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"ok"}],"usage":{"cost":{"total":0.12}},"timestamp":2}}
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Summarize(p)
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "abc" || s.Name != "Refactor auth" || s.Preview != "List the files" || s.Messages != 2 {
		t.Fatalf("summary = %+v", s)
	}
	if s.Cost < 0.11 || s.Cost > 0.13 {
		t.Fatalf("cost = %v", s.Cost)
	}
}

func TestListMissingDir(t *testing.T) {
	got, err := List(filepath.Join(t.TempDir(), "nope"))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v %v", got, err)
	}
}
