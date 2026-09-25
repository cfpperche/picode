package server

import (
	"context"
	"encoding/json"
	"errors"
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

// Claude Code's predicted follow-up renders dim after an empty cursor.
// Typed (bright) text, a cursor inside the text and a changed frame still
// refuse; the ghost is never treated as submitted text.
func TestPeerClaudeGhostSuggestion(t *testing.T) {
	raw, err := os.ReadFile("testdata/claude-ghost-suggestion.json")
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
		{name: "captured ghost suggestion", want: true},
		{name: "typed bright text", change: func(s *tmux.InputSnapshot) { s.Lines[s.CursorY] = terminalSGR.ReplaceAllString(s.Lines[s.CursorY], "") }},
		{name: "italic instead of dim", change: func(s *tmux.InputSnapshot) {
			s.Lines[s.CursorY] = strings.Replace(s.Lines[s.CursorY], "\x1b[2m", "\x1b[3m", 1)
		}},
		{name: "cursor moved into the text", change: func(s *tmux.InputSnapshot) { s.CursorX = 8 }},
		{name: "copy mode", change: func(s *tmux.InputSnapshot) { s.InMode = true }},
		{name: "status footer changed", change: func(s *tmux.InputSnapshot) { s.Lines[s.CursorY+2] = "  something else" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := native
			s.Lines = append([]string(nil), native.Lines...)
			if tc.change != nil {
				tc.change(&s)
			}
			if got := peerInputMatches("claude-code", s, ""); got != tc.want {
				t.Fatalf("empty editor: got %v want %v", got, tc.want)
			}
			if peerInputMatches("claude-code", s, peerCLIPointer) {
				t.Fatal("unaccepted suggestion treated as submitted pointer")
			}
		})
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
		{name: "1.0.34 Enter footer after paste", expected: peerCLIPointer, want: true, change: func(s *tmux.InputSnapshot) {
			s.Lines[1] = "  │ ❯ " + peerCLIPointer + strings.Repeat(" ", s.Width-9-len(peerCLIPointer)) + "│"
			s.Lines[4] = "  Enter:send  │  Shift+Enter/Alt+Enter:newline  │  Shift+Tab:mode  │  Ctrl+x:shortcuts"
			s.CursorX = 6 + len(peerCLIPointer)
		}},
		{name: "wrong Enter footer variant after paste", expected: peerCLIPointer, change: func(s *tmux.InputSnapshot) {
			s.Lines[1] = "  │ ❯ " + peerCLIPointer + strings.Repeat(" ", s.Width-9-len(peerCLIPointer)) + "│"
			s.Lines[4] = "  Enter:send  │  Ctrl+x:shortcuts"
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
	testPeerGrokSuggestionFixture(t, "testdata/grok-bordered-suggestion.json",
		[]suggestionMutation{
			{name: "dim only", change: func(s *tmux.InputSnapshot) {
				s.Lines[s.CursorY] = strings.Replace(s.Lines[s.CursorY], "\x1b[2;3m", "\x1b[2m", 1)
			}},
			{name: "italic only", change: func(s *tmux.InputSnapshot) {
				s.Lines[s.CursorY] = strings.Replace(s.Lines[s.CursorY], "\x1b[2;3m", "\x1b[3m", 1)
			}},
		})
}

// Grok 1.0.30 restyled the unaccepted suggestion (styled border and gutter,
// italic plus separate dim-gray text, styled closing border) — same captured
// shape, cursor, frame and footer requirements, verified against a live frame.
func TestPeerGrokNativeSuggestion1030(t *testing.T) {
	testPeerGrokSuggestionFixture(t, "testdata/grok-bordered-suggestion-1030.json",
		[]suggestionMutation{
			{name: "italic dropped", change: func(s *tmux.InputSnapshot) {
				s.Lines[s.CursorY] = strings.Replace(s.Lines[s.CursorY], "\x1b[3m", "", 1)
			}},
			{name: "dim gray recolored", change: func(s *tmux.InputSnapshot) {
				s.Lines[s.CursorY] = strings.Replace(s.Lines[s.CursorY], "\x1b[38;2;88;88;88m", "\x1b[38;2;200;200;200m", 1)
			}},
			{name: "plain padding recolored", change: func(s *tmux.InputSnapshot) {
				s.Lines[s.CursorY] = strings.Replace(s.Lines[s.CursorY], "\x1b[48;2;20;20;20m", "", 1)
			}},
		})
}

type suggestionMutation struct {
	name   string
	change func(*tmux.InputSnapshot)
	want   bool
}

func testPeerGrokSuggestionFixture(t *testing.T, path string, styles []suggestionMutation) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var native tmux.InputSnapshot
	if err = json.Unmarshal(raw, &native); err != nil {
		t.Fatal(err)
	}
	mutations := append([]suggestionMutation{
		{name: "captured empty suggestion", want: true},
		{name: "typed draft with cursor at start", change: func(s *tmux.InputSnapshot) { s.Lines[s.CursorY] = terminalSGR.ReplaceAllString(s.Lines[s.CursorY], "") }},
		{name: "accepted prefix", change: func(s *tmux.InputSnapshot) {
			s.Lines[s.CursorY] = strings.Replace(s.Lines[s.CursorY], "❯ ", "❯ typed", 1)
		}},
		{name: "cursor moved", change: func(s *tmux.InputSnapshot) { s.CursorX++ }},
		{name: "wrong footer", change: func(s *tmux.InputSnapshot) {
			s.Lines[s.CursorY+3] = "  Enter:send  │  Shift+Tab:mode  │  Ctrl+x:shortcuts"
		}},
		{name: "copy mode", change: func(s *tmux.InputSnapshot) { s.InMode = true }},
		{name: "resized", change: func(s *tmux.InputSnapshot) { s.Width-- }},
		{name: "multiline draft above", change: func(s *tmux.InputSnapshot) { s.Lines[s.CursorY-1] = "  │ previous draft line │" }},
	}, styles...)
	for _, tc := range mutations {
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
			deps.Replies.helloReceiver(key, "original.jsonl", "", "", 4242)
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
			if doc.PID != 4242 {
				t.Fatalf("receiver document pid = %d; want the hello's 4242", doc.PID)
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
	oc := tmux.InputSnapshot{CursorX: 5, CursorY: 3, Width: 120, Lines: []string{"history", "", "  ┃", "  ┃", "  ┃", "  ┃  Build · GLM-5.3-Flash Z.AI · max", "  ╹" + strings.Repeat("▀", 113), "   /fixture · ctrl+p commands", ""}}
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

// A phone attached to the terminal narrows the pane to its width; pi then
// cuts the footer before "%/". The rows below are a real 44-column capture
// (2026-09-25): still Pi's empty editor. Other CLIs keep the 70-column gate.
func TestNativePiComposerOnANarrowPane(t *testing.T) {
	rule := "\x1b[38;2;178;148;187m" + strings.Repeat("─", 44)
	lines := []string{
		rule,
		"\x1b[7m\x1b[39m \x1b[0m",
		rule,
		"\x1b[38;2;102;102;102m/home/goat/picode/.worktrees/mobile-fork ...",
		"↑23k ↓3.8k R132k CH99.0% $0.134 (sub) 4.6\x1b[39m...",
		"compact on · 22,769 / 250,000 · zai/glm-5\x1b[38;2;102;102;102m...",
	}
	s := tmux.InputSnapshot{Width: 44, CursorX: 0, CursorY: 1, Lines: lines}
	if !peerInputMatches("pi", s, "") {
		t.Fatal("Pi's empty editor at 44 columns refused")
	}
	if state, known := peerComposerState("pi", s); !known || state != "empty" {
		t.Fatalf("composer = %q %v", state, known)
	}
	for _, bad := range [][]string{
		{rule, "draft", rule, lines[3], lines[4]},
		{rule, "", strings.Repeat("─", 30), lines[3], lines[4]},
		{rule, "", rule, "Choose a permission", lines[4]},
		{rule, "", rule, lines[3], ""},
	} {
		s.Lines = bad
		if peerInputMatches("pi", s, "") {
			t.Fatal("unsafe narrow Pi composer", bad)
		}
	}
	tiny := tmux.InputSnapshot{Width: 20, CursorX: 0, CursorY: 1, Lines: []string{strings.Repeat("─", 20), "", strings.Repeat("─", 20), "/w", "x"}}
	if peerInputMatches("pi", tiny, "") {
		t.Fatal("a pane too narrow to trust was accepted")
	}
	claude := tmux.InputSnapshot{Width: 44, CursorX: 2, CursorY: 0, Lines: []string{"❯ ", "", ""}}
	if peerInputMatches("claude-code", claude, "") {
		t.Fatal("other CLIs keep the 70-column gate")
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

func TestPeerAttentionFailureReasons(t *testing.T) {
	raw, err := os.ReadFile("testdata/grok-bordered-composer.json")
	if err != nil {
		t.Fatal(err)
	}
	var base tmux.InputSnapshot
	if err = json.Unmarshal(raw, &base); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, want, expected string
		change               func(*tmux.InputSnapshot)
		snapshotErr          error
		pointer              string
	}{
		{name: "ready"},
		{name: "snapshot unavailable", snapshotErr: errors.New("secret native output"), want: "pane snapshot unavailable"},
		{name: "pane replaced", change: func(s *tmux.InputSnapshot) { s.PanePID++ }, want: "pane changed"},
		{name: "draft", change: func(s *tmux.InputSnapshot) { s.Lines[1] = "secret draft" }, want: "composer changed or is not ready"},
		{name: "too narrow", pointer: strings.Repeat("x", 500), want: "pointer does not fit"},
		{name: "paste not rendered yet", expected: peerPointer, pointer: peerPointer, want: "paste not rendered yet"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := base
			current.Lines = append([]string(nil), base.Lines...)
			if tc.change != nil {
				tc.change(&current)
			}
			got := peerInputRecheck("grok", base, current, tc.expected, tc.pointer, tc.snapshotErr)
			if tc.want == "" {
				if got != nil {
					t.Fatal(got)
				}
				return
			}
			if got == nil || got.Error() != tc.want {
				t.Fatalf("got %v want %s", got, tc.want)
			}
		})
	}
	if got := peerAttentionReason(errors.New("secret receiver body")); got != "native receiver or control rejected delivery" {
		t.Fatal(got)
	}
	if got := peerAttentionReason(peerAttentionFailure("after paste: pane changed")); got != "after paste: pane changed" {
		t.Fatal(got)
	}
}

func TestPeerPasteSettleWindow(t *testing.T) {
	oldWindow, oldInterval := peerPasteSettleWindow, peerPasteSettleInterval
	peerPasteSettleWindow, peerPasteSettleInterval = 90*time.Millisecond, 20*time.Millisecond
	defer func() { peerPasteSettleWindow, peerPasteSettleInterval = oldWindow, oldInterval }()

	for _, tc := range []struct {
		name       string
		samples    []error
		want       error
		wantCalls  int
		wantAtMost int
	}{
		{name: "first sample passes", samples: []error{nil}, wantCalls: 1, wantAtMost: 1},
		{
			name:      "not rendered then rendered",
			samples:   []error{errPeerNotRendered, errPeerNotRendered, nil},
			wantCalls: 3, wantAtMost: 3,
		},
		{
			name:       "expiry refuses unsettled",
			samples:    []error{errPeerNotRendered},
			want:       errPeerNotRendered,
			wantAtMost: 8,
		},
		{
			name:       "foreign composer expires unsettled",
			samples:    []error{errPeerComposerChanged},
			want:       errPeerComposerChanged,
			wantAtMost: 8,
		},
		{name: "structural refuses immediately", samples: []error{errPeerPaneChanged}, want: errPeerPaneChanged, wantCalls: 1, wantAtMost: 1},
		{name: "fit refuses immediately", samples: []error{errPeerPointerFit}, want: errPeerPointerFit, wantCalls: 1, wantAtMost: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			check := func(expected string) error {
				if expected != peerPointer {
					t.Fatalf("expected the pasted pointer, got %q", expected)
				}
				if calls < len(tc.samples) {
					err := tc.samples[calls]
					calls++
					return err
				}
				calls++
				return tc.samples[len(tc.samples)-1]
			}
			got := peerAwaitComposer(context.Background(), check, peerPointer)
			if tc.want == nil {
				if got != nil {
					t.Fatal(got)
				}
			} else if !errors.Is(got, tc.want) {
				t.Fatalf("got %v want %v", got, tc.want)
			}
			if tc.wantCalls != 0 && calls != tc.wantCalls {
				t.Fatalf("calls=%d want %d", calls, tc.wantCalls)
			}
			if calls > tc.wantAtMost {
				t.Fatalf("calls=%d exceeds %d", calls, tc.wantAtMost)
			}
		})
	}

	// A cancelled attempt never keeps polling.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	got := peerAwaitComposer(ctx, func(string) error { calls++; return errPeerNotRendered }, peerPointer)
	if !errors.Is(got, context.Canceled) || calls != 1 {
		t.Fatalf("got %v calls=%d", got, calls)
	}
}

func TestPeerOpenCodeSidebarAndWrappedFooter(t *testing.T) {
	raw, err := os.ReadFile("testdata/opencode-sidebar-wrapped.json")
	if err != nil {
		t.Fatal(err)
	}
	var original tmux.InputSnapshot
	if err = json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		edit func(*tmux.InputSnapshot)
		want bool
	}{
		{"native empty", func(s *tmux.InputSnapshot) {}, true},
		{"draft at cursor", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY] = "  ┃  draft" }, false},
		{"draft above", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY-1] = "  ┃  draft" }, false},
		{"draft below", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY+1] = "  ┃  draft" }, false},
		{"broken border", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY+3] = "  ╹" + strings.Repeat("▀", 100) + "X" }, false},
		{"bad gutter", func(s *tmux.InputSnapshot) {
			s.Lines[s.CursorY+1] = "  ┃" + strings.Repeat(" ", peerOpenCodeWidth(*s)-3) + "X"
		}, false},
		{"dialog footer", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY+4] = "  enter confirm esc cancel" }, false},
		{"unknown footer", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY+5] = "   permission required" }, false},
		{"path wraps twice", func(s *tmux.InputSnapshot) {
			s.Lines = append(s.Lines, "   onboarding-opencode-zai/var/qa/oc-zai")
		}, true},
		{"path wrap then footer text", func(s *tmux.InputSnapshot) {
			s.Lines = append(s.Lines, "   14.0K (1% ctx)")
		}, false},
		{"path wrap then bare text", func(s *tmux.InputSnapshot) {
			s.Lines = append(s.Lines, "junk")
		}, false},
		{"copy mode", func(s *tmux.InputSnapshot) { s.InMode = true }, false},
		{"malformed escape", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY] = "  ┃\x1b[broken" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := original
			s.Lines = append([]string(nil), original.Lines...)
			tc.edit(&s)
			if got := peerInputMatches("opencode", s, ""); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
	width := peerOpenCodeWidth(original)
	for _, n := range []int{width - 6, width - 5, width} {
		if got := peerPointerFits("opencode", original, strings.Repeat("x", n)); got != (n < width-5) {
			t.Fatalf("fit %d=%v", n, got)
		}
	}
	original.Lines[original.CursorY] = "  ┃  " + peerPointer
	original.CursorX = 5 + utf8.RuneCountInString(peerPointer)
	if !peerInputMatches("opencode", original, peerPointer) {
		t.Fatal("exact pasted pointer refused")
	}
	original.Lines[original.CursorY] += "draft"
	if peerInputMatches("opencode", original, peerPointer) {
		t.Fatal("post-paste edit accepted")
	}
}

func TestPeerGrokWelcomeComposer(t *testing.T) {
	raw, err := os.ReadFile("testdata/grok-welcome-composer.json")
	if err != nil {
		t.Fatal(err)
	}
	var original tmux.InputSnapshot
	if err = json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		edit func(*tmux.InputSnapshot)
		want bool
	}{
		{"untouched welcome", func(s *tmux.InputSnapshot) {}, true},
		{"draft", func(s *tmux.InputSnapshot) {
			s.Lines[s.CursorY] = strings.Replace(s.Lines[s.CursorY], "❯ ", "❯ draft", 1)
		}, false},
		{"moved cursor", func(s *tmux.InputSnapshot) { s.CursorX++ }, false},
		{"copy mode", func(s *tmux.InputSnapshot) { s.InMode = true }, false},
		{"changed width", func(s *tmux.InputSnapshot) { s.Width-- }, false},
		{"unknown footer", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY+3] = strings.Repeat(" ", s.Width-10) + "[other]" }, false},
		{"shifted footer", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY+3] = "[stable]" }, false},
		{"broken frame", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY-1] = "" }, false},
		{"extra editor row", func(s *tmux.InputSnapshot) { s.Lines[s.CursorY+2] = "draft" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := original
			s.Lines = append([]string(nil), original.Lines...)
			tc.edit(&s)
			if got := peerInputMatches("grok", s, ""); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
	raw, err = os.ReadFile("testdata/grok-welcome-pasted.json")
	if err != nil {
		t.Fatal(err)
	}
	var pasted tmux.InputSnapshot
	if err = json.Unmarshal(raw, &pasted); err != nil {
		t.Fatal(err)
	}
	if !peerInputMatches("grok", pasted, peerCLIPointer) {
		t.Fatal("native welcome-to-editor transition refused")
	}
	pasted.Lines[pasted.CursorY+3] = strings.Repeat(" ", pasted.Width-10) + "[stable]"
	if peerInputMatches("grok", pasted, peerCLIPointer) {
		t.Fatal("post-paste stale welcome footer accepted")
	}
}

// Real native frames capture supported empty editors and user drafts for all
// six CLIs. These are parser checks, not a substitute for native exchange QA.
func TestNativeCLIAttentionMatrix(t *testing.T) {
	raw, err := os.ReadFile("testdata/cli-attention-matrix.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Name, CLI string
		Snapshot  tmux.InputSnapshot
		Want      bool
	}
	if err = json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(row.Name, func(t *testing.T) {
			if got := peerInputMatches(row.CLI, row.Snapshot, ""); got != row.Want {
				t.Fatalf("got %v want %v", got, row.Want)
			}
		})
	}
}

// Omp at a phone's width (a real 44-column capture, 2026-09-25): the
// status bar keeps its " > " and the input row its "╰─", so the empty
// editor is recognized; a draft or a missing status bar is not.
func TestOmpComposerOnANarrowPane(t *testing.T) {
	lines := make([]string, 40)
	lines[28] = "\x1b[48;2;15;18;22m \x1b[38;2;107;114;128mpi\x1b[39m \x1b[38;2;42;48;56m>\x1b[39m \x1b[38;2;232;236;244m[D] …ode\x1b[39m \x1b[38;2;15;18;22m\x1b[49m>\x1b[38;2;0;180;255m-1.0%"
	lines[29] = "\x1b[38;2;212;192;144m╰─ \x1b[39m"
	s := tmux.InputSnapshot{Width: 44, CursorX: 3, CursorY: 29, Lines: lines}
	if !peerInputMatches("omp", s, "") {
		t.Fatal("Omp's empty editor at 44 columns refused")
	}
	draft := append([]string{}, lines...)
	draft[29] = "╰─ half a thought"
	if peerInputMatches("omp", tmux.InputSnapshot{Width: 44, CursorX: 3, CursorY: 29, Lines: draft}, "") {
		t.Fatal("a draft was taken for an empty editor")
	}
	bare := append([]string{}, lines...)
	bare[28] = "Tip: something"
	if peerInputMatches("omp", tmux.InputSnapshot{Width: 44, CursorX: 3, CursorY: 29, Lines: bare}, "") {
		t.Fatal("no status bar, yet accepted")
	}
	if peerInputMatches("omp", tmux.InputSnapshot{Width: 20, CursorX: 3, CursorY: 29, Lines: lines}, "") {
		t.Fatal("a pane too narrow to trust was accepted")
	}
}
