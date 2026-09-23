package store

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestAddSessionHandoffRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, err := s.AddSessionHandoff(SessionHandoff{SourceCLI: "claude-code", SourceID: "cc-1", TargetCLI: "codex", Mode: "weird", Window: "all"}); err == nil {
		t.Fatal("invalid mode accepted")
	}
	if _, err := s.AddSessionHandoff(SessionHandoff{SourceCLI: "claude-code", TargetCLI: "codex", Mode: "native", Window: "all"}); err == nil {
		t.Fatal("missing source id accepted")
	}
	// A fork (Fork agent…) is a same-CLI row: the copy of a conversation.
	if _, err := s.AddSessionHandoff(SessionHandoff{SourceCLI: "codex", SourceID: "cx-1", TargetCLI: "codex", Mode: "fork", Window: "all"}); err != nil {
		t.Fatalf("fork mode refused: %v", err)
	}
	h, err := s.AddSessionHandoff(SessionHandoff{
		SourceCLI: "claude-code", SourceID: "cc-1", SourcePath: "/h/.claude/projects/-p/cc-1.jsonl",
		TargetCLI: "codex", TargetID: "cx-9", TargetPath: "/h/.codex/sessions/2026/09/06/rollout-x-cx-9.jsonl",
		Mode: "native", Window: "recent", Tools: "native",
		Manifest:   json.RawMessage(`{"dropped":{"thinking":3},"warnings":[]}`),
		TerminalID: "term-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if h.ID == "" || h.CreatedAt == "" {
		t.Fatalf("id/createdAt not assigned: %+v", h)
	}
	got, err := s.SessionHandoff(h.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SourceID != "cc-1" || got.TargetID != "cx-9" || got.Mode != "native" || got.Window != "recent" || got.TerminalID != "term-1" || string(got.Manifest) != `{"dropped":{"thinking":3},"warnings":[]}` {
		t.Fatalf("round trip = %+v", got)
	}
	if _, err := s.SessionHandoff("nope"); err == nil {
		t.Fatal("unknown id must error")
	}
	// An invalid manifest is stored as {} rather than refused.
	h2, err := s.AddSessionHandoff(SessionHandoff{SourceCLI: "pi", SourceID: "p-1", TargetCLI: "claude-code", Mode: "brief", Window: "all", Manifest: json.RawMessage(`{not json`)})
	if err != nil {
		t.Fatal(err)
	}
	if string(h2.Manifest) != "{}" {
		t.Fatalf("manifest = %s", h2.Manifest)
	}
}

func TestSessionHandoffsNewestFirstAndLimited(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for i, at := range []string{"2026-09-06T10:00:00Z", "2026-09-06T12:00:00Z", "2026-09-06T11:00:00Z"} {
		if _, err := s.AddSessionHandoff(SessionHandoff{ID: "h" + string(rune('a'+i)), SourceCLI: "pi", SourceID: "p", TargetCLI: "codex", Mode: "native", Window: "all", CreatedAt: at}); err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.SessionHandoffs(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].ID != "hb" || all[1].ID != "hc" || all[2].ID != "ha" {
		t.Fatalf("order = %+v", all)
	}
	two, _ := s.SessionHandoffs(2)
	if len(two) != 2 || two[1].ID != "hc" {
		t.Fatalf("limit = %+v", two)
	}
}
