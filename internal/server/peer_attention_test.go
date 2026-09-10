package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/tmux"
)

func TestPeerInputDecisionTable(t *testing.T) {
	for _, tc := range []struct {
		name, cli, line, expected string
		x                         int
		mode, want                bool
	}{
		{"empty Grok", "grok", "❯", "", 2, false, true},
		{"Grok draft", "grok", "❯ keep my draft", "", 2, false, false},
		{"cursor in draft", "grok", "❯ draft", "", 7, false, false},
		{"permission picker", "grok", "❯ Allow once", "", 2, false, false},
		{"copy mode", "grok", "❯", "", 2, true, false},
		{"shell", "grok", "$", "", 2, false, false},
		{"Hermes ghost", "hermes", "❯ \x1b[3mAsk anything\x1b[0m", "", 2, false, true},
		{"Hermes typed ghost text", "hermes", "❯ Ask anything", "", 2, false, false},
		{"Claude blank", "claude-code", "❯ ", "", 2, false, true},
		{"Claude native nonbreaking prompt space", "claude-code", "❯\u00a0" + peerPointer, peerPointer, 2 + utf8.RuneCountInString(peerPointer), false, true},
		{"unknown OpenCode input", "opencode", "", "", 2, false, false},
		{"exact pointer", "grok", "❯ " + peerPointer, peerPointer, 2 + utf8.RuneCountInString(peerPointer), false, true},
		{"typed between paste and Enter", "grok", "❯ " + peerPointer + " my draft", peerPointer, 2 + utf8.RuneCountInString(peerPointer), false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := tmux.InputSnapshot{CursorY: 1, CursorX: tc.x, Width: 100, InMode: tc.mode, Lines: []string{"────────────────────", tc.line, "────────────────────"}}
			if tc.cli == "grok" {
				s.Lines[2] = "Grok 4.6 · ctrl+o transcript"
			}
			if got := peerInputMatches(tc.cli, s, tc.expected); got != tc.want {
				t.Fatal(got, tc.want)
			}
			s.Width = 40
			if peerInputMatches(tc.cli, s, tc.expected) {
				t.Fatal("narrow wrapped input accepted")
			}
		})
	}
	// The pointer has no sender-controlled text or message body.
	if strings.ContainsAny(peerPointer, "\r\n\x1b") {
		t.Fatal("unsafe pointer")
	}
}

func TestPeerInputRejectsMultilineComposer(t *testing.T) {
	for _, cli := range []string{"grok", "hermes", "claude-code"} {
		footer := "────────────────────"
		if cli == "grok" {
			footer = "Grok · ctrl+o transcript"
		}
		for _, lines := range [][]string{
			{"────────────────────", "❯", "  keep the second draft line", footer},
			{"────────────────────", "❯ draft first line", "  ❯", footer},
			{"────────────────────", "❯", footer, "other active prompt"},
			{"────────────────────", "❯", "  ────────────────────", "  private draft", footer},
		} {
			s := tmux.InputSnapshot{CursorX: 2, CursorY: 1, Width: 120, Lines: lines}
			// Anything resembling a second composer must remain untouched.

			if peerInputMatches(cli, s, "") {
				t.Fatalf("accepted draft %s: %q", cli, lines)
			}
			s.CursorY = 2
			if peerInputMatches(cli, s, "") {
				t.Fatalf("accepted continuation %s: %q", cli, lines)
			}
		}
	}
}

func TestPeerPointerKeepsExpectedNativeSession(t *testing.T) {
	for _, key := range []string{"managed-fixture", "term-fixture"} {
		t.Run(key, func(t *testing.T) {
			deps := Deps{DataDir: t.TempDir(), Replies: NewTuiReplies()}
			deps.Replies.HelloSession(key, "original.jsonl")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- deliverPeerPointer(ctx, deps, key, "original.jsonl") }()
			var doc replyFile
			deadline := time.Now().Add(time.Second)
			for time.Now().Before(deadline) {
				files, _ := filepath.Glob(filepath.Join(deps.DataDir, "tui-inbox", key, "*.json"))
				if len(files) > 0 {
					raw, _ := os.ReadFile(files[0])
					if json.Unmarshal(raw, &doc) == nil {
						break
					}
				}
				time.Sleep(time.Millisecond)
			}
			if doc.Nonce == "" {
				t.Fatal("no receiver document")
			}
			deps.Replies.HelloSession(key, "new.jsonl")
			if doc.SessionPath != "original.jsonl" {
				t.Fatal("redirected to a new conversation")
			}
			// Native receiver rejects the stale exact-session document, never submits it.
			deps.Replies.resolveAck(doc.Nonce, replyAck{OK: false, Reason: "different session"})
			if err := <-result; err == nil {
				t.Fatal("rejected native delivery reported success")
			}
			if err := deliverPeerPointer(ctx, deps, key, "original.jsonl"); err == nil {
				t.Fatal("changed receiver accepted")
			}
		})
	}
}

func TestNativeCodexAndOpenCodeComposers(t *testing.T) {
	codex := tmux.InputSnapshot{CursorX: 2, CursorY: 2, Width: 120, Lines: []string{"history", "", "\x1b[1m›\x1b[0m \x1b[2mAsk Codex to do anything\x1b[0m", "", "  model low · /fixture", ""}}
	if !peerInputMatches("codex", codex, "") {
		t.Fatal("native Codex ghost refused")
	}
	codex.Lines[2] = "› Ask Codex to do anything"
	if peerInputMatches("codex", codex, "") {
		t.Fatal("typed placeholder accepted")
	}
	oc := tmux.InputSnapshot{CursorX: 5, CursorY: 3, Width: 120, Lines: []string{"history", "", "  ┃", "  ┃", "  ┃", "  ┃  Build · GLM-5.3-Flash Z.AI · max", "  ╹▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀", "   /fixture · ctrl+p commands", ""}}
	if !peerInputMatches("opencode", oc, "") {
		t.Fatal("native OpenCode composer refused")
	}
	oc.Lines[3] = "  ┃  " + peerPointer
	oc.CursorX += len(peerPointer)
	if !peerInputMatches("opencode", oc, peerPointer) {
		t.Fatal("exact OpenCode pointer refused")
	}
	oc.Lines[3] += " draft"
	if peerInputMatches("opencode", oc, peerPointer) {
		t.Fatal("concurrent draft accepted")
	}
	oc.Lines[3] = "  ┃"
	oc.CursorX = 5
	oc.Lines[2] = "  ┃  draft above cursor"
	if peerInputMatches("opencode", oc, "") {
		t.Fatal("multiline OpenCode draft accepted")
	}
}

func TestNativePiResumeComposer(t *testing.T) {
	rule := strings.Repeat("─", 80)
	s := tmux.InputSnapshot{Width: 80, CursorX: 0, CursorY: 1, Lines: []string{rule, "", rule, "/workspace (main)", "0.0%/1.0M (auto) (zai) glm-5.3-flash • low"}}
	if !peerInputMatches("pi", s, "") {
		t.Fatal("native empty Pi editor refused")
	}
	for _, lines := range [][]string{
		{rule, "draft", rule, "/workspace", "0.0%/1.0M"},
		{rule, "", "draft", rule, "/workspace", "0.0%/1.0M"},
		{rule, "", rule, "Choose a permission", "0.0%/1.0M"},
		{rule, "", rule, "/workspace", "unknown footer"},
	} {
		s.Lines = lines
		if peerInputMatches("pi", s, "") {
			t.Fatal("unsafe Pi composer", lines)
		}
	}
}
