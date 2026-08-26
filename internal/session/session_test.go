package session

import (
	"os"
	"path/filepath"
	"strings"
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

func TestTranscriptCompaction(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.jsonl")
	body := `{"type":"session","id":"x"}
{"type":"compaction","summary":"Old turns summarized."}
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ev, err := Transcript(p)
	if err != nil || len(ev) != 1 || ev[0].Text != "Old turns summarized." {
		t.Fatalf("%v %+v", err, ev)
	}
}

func TestTranscript(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.jsonl")
	body := `{"type":"session","id":"x","timestamp":"2026-08-24T01:00:00.000Z"}
{"type":"message","message":{"role":"user","content":[{"type":"text","text":"List files"}]}}
{"type":"message","message":{"role":"assistant","content":[{"type":"toolCall","id":"t1","name":"bash","arguments":{"command":"ls"}}]}}
{"type":"message","message":{"role":"toolResult","toolCallId":"t1","toolName":"bash","isError":false,"content":[{"type":"text","text":"AGENTS.md"}]}}
{"type":"message","message":{"role":"assistant","content":[{"type":"text","text":"AGENTS.md"}]}}
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ev, err := Transcript(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 3 {
		t.Fatalf("len=%d %+v", len(ev), ev)
	}
	if ev[0].Kind != "user" || ev[0].Text != "List files" {
		t.Fatalf("user = %+v", ev[0])
	}
	if ev[1].Kind != "tool" || ev[1].Name != "bash" || ev[1].Status != "ok" || ev[1].Detail != "AGENTS.md" {
		t.Fatalf("tool = %+v", ev[1])
	}
	if ev[2].Kind != "assistant" || ev[2].Text != "AGENTS.md" {
		t.Fatalf("asst = %+v", ev[2])
	}
}

func TestParseContextWindow(t *testing.T) {
	if ParseContextWindow("200K") != 200_000 {
		t.Fatal("200K")
	}
	if ParseContextWindow("1M") != 1_000_000 {
		t.Fatal("1M")
	}
	if ParseContextWindow("1.0M") != 1_000_000 {
		t.Fatal("1.0M")
	}
}

func TestTranscriptAssistantError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "e.jsonl")
	body := `{"type":"session"}
{"type":"message","message":{"role":"user","content":[{"type":"text","text":"teste"}]}}
{"type":"message","message":{"role":"assistant","content":[],"stopReason":"error","errorMessage":"You're out of extra usage."}}
`
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ev, err := Transcript(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(ev) != 2 || ev[1].Kind != "error" || !strings.Contains(ev[1].Text, "extra usage") {
		t.Fatalf("%+v", ev)
	}
}

func TestFormatCwd(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	if got := formatCwd(home); got != "~" {
		t.Fatalf("home = %q", got)
	}
	if got := formatCwd(filepath.Join(home, "picode")); got != "~/picode" {
		t.Fatalf("sub = %q", got)
	}
}

func TestListMissingDir(t *testing.T) {
	got, err := List(filepath.Join(t.TempDir(), "nope"))
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v %v", got, err)
	}
}
