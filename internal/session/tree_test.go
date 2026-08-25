package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadTreeBranches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	body := `{"type":"session","id":"s1","timestamp":"2024-01-01T00:00:00.000Z"}
{"type":"message","id":"u1","parentId":null,"message":{"role":"user","content":"hello"}}
{"type":"message","id":"a1","parentId":"u1","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}
{"type":"message","id":"u2","parentId":"u1","message":{"role":"user","content":"other"}}
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	tr, err := ReadTree(path)
	if err != nil {
		t.Fatal(err)
	}
	if tr.LeafID != "u2" {
		t.Fatalf("leaf %s", tr.LeafID)
	}
	if len(tr.Tree) != 1 || tr.Tree[0].ID != "u1" || len(tr.Tree[0].Children) != 2 {
		t.Fatalf("%+v", tr.Tree)
	}
	if tr.Tree[0].Children[1].Text != "other" {
		t.Fatalf("branch text %q", tr.Tree[0].Children[1].Text)
	}
}

func TestReadTreeMissing(t *testing.T) {
	tr, err := ReadTree(filepath.Join(t.TempDir(), "nope.jsonl"))
	if err != nil || len(tr.Tree) != 0 {
		t.Fatalf("%+v %v", tr, err)
	}
}
