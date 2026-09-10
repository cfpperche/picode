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

func TestPeerClaudeNarrowFooter(t *testing.T) {
	for _, tc := range []struct {
		name, input, expected, shortcut string
		mode, want                      bool
	}{
		{name: "empty with remote control", shortcut: "/rc", want: true},
		{name: "pasted pointer", input: peerPointer, expected: peerPointer, shortcut: "/rc", want: true},
		{name: "draft", input: "keep this draft", shortcut: "/rc"},
		{name: "typed after paste", input: peerPointer + "x", expected: peerPointer, shortcut: "/rc"},
		{name: "unknown footer", shortcut: "Enter to approve"},
		{name: "copy mode", shortcut: "/rc", mode: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := tmux.InputSnapshot{Width: 80, CursorY: 1, CursorX: 2 + utf8.RuneCountInString(tc.input), InMode: tc.mode, Lines: []string{
				strings.Repeat("─", 80), "❯\u00a0" + tc.input, strings.Repeat("─", 80),
				"  Sonnet 5 · low | picode | main | Context n/a | cache n/a", "  ⏵⏵ auto mode on (shift+tab to cycle) · ← 2 agents", "                                                                           " + tc.shortcut,
			}}
			if got := peerInputMatches("claude-code", s, tc.expected); got != tc.want {
				t.Fatalf("accepted=%v, want %v", got, tc.want)
			}
			// Moving the cursor into a multiline draft cannot impersonate the frame.
			s.Lines = append([]string{"❯ keep first draft row"}, s.Lines...)
			s.CursorY++
			s.Lines[s.CursorY-1] = "  draft continuation"
			if peerInputMatches("claude-code", s, tc.expected) {
				t.Fatal("multiline draft accepted")
			}
		})
	}
}

func TestPeerGrokBorderedComposer(t *testing.T) {
	raw, err := os.ReadFile("testdata/grok-bordered-composer.json")
	if err != nil {
		t.Fatal(err)
	}
	var native tmux.InputSnapshot
	if err = json.Unmarshal(raw, &native); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, expected string
		change         func(*tmux.InputSnapshot)
		want           bool
	}{
		{name: "captured native empty input", want: true},
		{name: "draft", change: func(s *tmux.InputSnapshot) { s.Lines[1] = strings.Replace(s.Lines[1], "❯ ", "❯ keep this", 1) }},
		{name: "cursor moved", change: func(s *tmux.InputSnapshot) { s.CursorX++ }},
		{name: "copy mode", change: func(s *tmux.InputSnapshot) { s.InMode = true }},
		{name: "resized before capture", change: func(s *tmux.InputSnapshot) { s.Width-- }},
		{name: "multiline draft below", change: func(s *tmux.InputSnapshot) {
			s.Lines = append(s.Lines[:2], append([]string{"  │ keep this second line │"}, s.Lines[2:]...)...)
		}},
		{name: "continuation cursor", change: func(s *tmux.InputSnapshot) { s.Lines[0] = "  │ keep first line │" }},
		{name: "permission instead of model", change: func(s *tmux.InputSnapshot) { s.Lines[2] = strings.Replace(s.Lines[2], "Grok", "Allow", 1) }},
		{name: "unknown footer", change: func(s *tmux.InputSnapshot) { s.Lines[4] = "  Press Enter to allow" }},
		{name: "extra composer below", change: func(s *tmux.InputSnapshot) { s.Lines = append(s.Lines, "❯ new draft") }},
		{name: "exact pointer after paste", expected: peerCLIPointer, want: true, change: func(s *tmux.InputSnapshot) {
			s.Lines[1] = "  │ ❯ " + peerCLIPointer + strings.Repeat(" ", s.Width-9-len(peerCLIPointer)) + "│"
			s.Lines[4] = "  Enter:send  │  Shift+Tab:mode  │  Ctrl+x:shortcuts"
			s.CursorX = 6 + len(peerCLIPointer)
		}},
		{name: "typed after paste", expected: peerCLIPointer, change: func(s *tmux.InputSnapshot) {
			s.Lines[1] = "  │ ❯ " + peerCLIPointer + "x" + strings.Repeat(" ", s.Width-10-len(peerCLIPointer)) + "│"
			s.Lines[4] = "  Enter:send  │  Shift+Tab:mode  │  Ctrl+x:shortcuts"
			s.CursorX = 6 + len(peerCLIPointer)
		}},
		{name: "pointer wraps", expected: strings.Repeat("x", 200)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := native
			s.Lines = append([]string(nil), native.Lines...)
			if tc.change != nil {
				tc.change(&s)
			}
			if got := peerInputMatches("grok", s, tc.expected); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestPeerGrokNativeSuggestion(t *testing.T) {
	raw, err := os.ReadFile("testdata/grok-bordered-suggestion.json")
	if err != nil {
		t.Fatal(err)
	}
	var native tmux.InputSnapshot
	if err = json.Unmarshal(raw, &native); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*tmux.InputSnapshot)
		want   bool
	}{
		{name: "captured empty suggestion", want: true},
		{name: "typed draft with cursor at start", change: func(s *tmux.InputSnapshot) { s.Lines[1] = terminalSGR.ReplaceAllString(s.Lines[1], "") }},
		{name: "dim only", change: func(s *tmux.InputSnapshot) { s.Lines[1] = strings.Replace(s.Lines[1], "\x1b[2;3m", "\x1b[2m", 1) }},
		{name: "italic only", change: func(s *tmux.InputSnapshot) { s.Lines[1] = strings.Replace(s.Lines[1], "\x1b[2;3m", "\x1b[3m", 1) }},
		{name: "accepted prefix", change: func(s *tmux.InputSnapshot) { s.Lines[1] = strings.Replace(s.Lines[1], "❯ ", "❯ typed", 1) }},
		{name: "cursor moved", change: func(s *tmux.InputSnapshot) { s.CursorX++ }},
		{name: "wrong footer", change: func(s *tmux.InputSnapshot) { s.Lines[4] = "  Enter:send  │  Shift+Tab:mode  │  Ctrl+x:shortcuts" }},
		{name: "copy mode", change: func(s *tmux.InputSnapshot) { s.InMode = true }},
		{name: "resized", change: func(s *tmux.InputSnapshot) { s.Width-- }},
		{name: "multiline draft", change: func(s *tmux.InputSnapshot) { s.Lines[0] = "  │ previous draft line │" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := native
			s.Lines = append([]string(nil), native.Lines...)
			if tc.change != nil {
				tc.change(&s)
			}
			if got := peerInputMatches("grok", s, ""); got != tc.want {
				t.Fatalf("empty editor: got %v want %v", got, tc.want)
			}
			if peerInputMatches("grok", s, peerCLIPointer) {
				t.Fatal("unaccepted suggestion treated as submitted pointer")
			}
		})
	}
}

func TestPeerGrokCapturedPointer(t *testing.T) {
	raw, err := os.ReadFile("testdata/grok-bordered-pointer.json")
	if err != nil {
		t.Fatal(err)
	}
	var s tmux.InputSnapshot
	if err = json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	if !peerInputMatches("grok", s, peerCLIPointer) {
		t.Fatal("captured native post-paste frame rejected")
	}
	if peerInputMatches("grok", s, "") {
		t.Fatal("pasted text accepted as empty")
	}
	s.Lines[4] = "  Shift+Tab:mode  │  Ctrl+x:shortcuts"
	if peerInputMatches("grok", s, peerCLIPointer) {
		t.Fatal("incomplete post-paste redraw accepted")
	}
}

func TestPeerGrokPointerFitsBeforeClaim(t *testing.T) {
	for _, pointer := range []string{peerCLIPointer, "PiCode: run picode messages read for the connection test."} {
		minimum := 10 + utf8.RuneCountInString(pointer)
		for _, width := range []int{70, minimum - 1, minimum, minimum + 1, 162} {
			s := tmux.InputSnapshot{Width: width, CursorY: 0, Lines: []string{"  │ ❯       │"}}
			if got := peerPointerFits("grok", s, pointer); got != (width >= minimum) {
				t.Fatalf("width=%d pointer=%q got=%v", width, pointer, got)
			}
		}
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

func TestPeerClaudeSingleFooter(t *testing.T) {
	for _, tc := range []struct {
		footer string
		want   bool
	}{
		{"  ⏸ manual mode on · ? for shortcuts · ← for agents /rc active", true},
		{"  ⏸ manual mode on · gh auth login for PR status · ← for agents /rc active", true},
		{"  ⏵⏵ accept edits on (shift+tab to cycle) · /rc", true},
		{"  Enter to approve this tool", false},
		{"  Unknown native layout", false},
	} {
		s := tmux.InputSnapshot{Width: 80, CursorX: 2, CursorY: 1, Lines: []string{strings.Repeat("─", 80), "❯\u00a0", strings.Repeat("─", 80), tc.footer}}
		if got := peerInputMatches("claude-code", s, ""); got != tc.want {
			t.Fatalf("%q accepted=%v", tc.footer, got)
		}
		s.Lines[1] += "draft"
		s.CursorX += 5
		if peerInputMatches("claude-code", s, "") {
			t.Fatal("draft accepted")
		}
	}
}
