package server

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/tmux"
)

func snap(cli string, cursorX, cursorY int, rows []string) tmux.InputSnapshot {
	_ = cli
	return tmux.InputSnapshot{Width: 200, CursorX: cursorX, CursorY: cursorY, Lines: rows}
}

// The door's gate is three-state: a recognized empty row delivers and
// verifies; positive draft evidence refuses; an unclassifiable row
// DELIVERS with a demoted receipt (owner report 2026-09-20 — clean
// composers were refused on every agent terminal).

func claudeGhostPane() []string {
	rule := strings.Repeat("\u2500", 200)
	rows := paneRows("", 11)
	return append(rows, rule, ghostRow, rule)
}

func claudeDraftPane() []string {
	rule := strings.Repeat("\u2500", 200)
	rows := paneRows("", 11)
	return append(rows, rule, draftRow, rule)
}

var ghostRow = "\x1b[39m\u276f\u00a0\x1b[2mTry \"fix typecheck errors\"\x1b[0m"
var draftRow = "\x1b[39m\u276f \x1b[0mtype this"

func TestPeerComposerState(t *testing.T) {
	// Claude, ghost suggestion: known empty.
	st, known := peerComposerState("claude-code", snap("claude-code", 2, 12, claudeGhostPane()))
	if !known || st != "empty" {
		t.Fatalf("claude ghost = %q known=%v", st, known)
	}

	// Claude, typed draft: known occupied.
	st, known = peerComposerState("claude-code", snap("claude-code", 12, 12, claudeDraftPane()))
	if !known || st != "occupied" {
		t.Fatalf("claude draft = %q known=%v", st, known)
	}

	// Codex, italic suggestion without the native frame around it: unknown
	// — it must deliver, not block.
	st, known = peerComposerState("codex", snap("codex", 2, 10, paneRows("\x1b[39m› \x1b[2mplan first\x1b[0m", 11)))
	if known {
		t.Fatalf("codex ghost without frame = %q known=%v; want unknown", st, known)
	}

	// Codex, typed draft: known occupied.
	st, known = peerComposerState("codex", snap("codex", 9, 10, paneRows("\x1b[39m› \x1b[0mfix the bug", 11)))
	if !known || st != "occupied" {
		t.Fatalf("codex draft = %q known=%v", st, known)
	}

	// Grok bordered composer, draft inside the border: known occupied.
	st, known = peerComposerState("grok", snap("grok", 9, 10, paneRows("│ ❯ ship it │", 11)))
	if !known || st != "occupied" {
		t.Fatalf("grok draft = %q known=%v", st, known)
	}

	// An unclassifiable row is UNKNOWN — it must deliver, not refuse.
	st, known = peerComposerState("claude-code", snap("claude-code", 1, 8, paneRows("plain screen text", 11)))
	if known {
		t.Fatalf("unclassifiable claude row claimed known (%q)", st)
	}
	st, known = peerComposerState("opencode", snap("opencode", 4, 3, paneRows("▌bar", 6)))
	if known && st == "occupied" {
		t.Fatalf("opencode unknown row claimed occupied")
	}

	// Claude Code 2.1.282 renders the ghost word by word (measured
	// 2026-09-25): still a known empty composer, so an unattended run can
	// deliver (ADR-0217) and the check after Enter confirms it.
	perWord := "\x1b[39m\u276f\u00a0\x1b[2mTry\x1b[0m \x1b[2m\"edit\x1b[0m \x1b[2mcli_launch.go\x1b[0m \x1b[2mto...\"\x1b[0m"
	rule := strings.Repeat("\u2500", 200)
	st, known = peerComposerState("claude-code", snap("claude-code", 2, 12, append(paneRows("", 11), rule, perWord, rule)))
	if !known || st != "empty" {
		t.Fatalf("claude per-word ghost = %q known=%v", st, known)
	}
	// Cut at the pane's edge, the last dim run has no reset (measured).
	edge := "\x1b[39m\u276f\u00a0\x1b[2mTry\x1b[0m \x1b[2m\"create\x1b[0m \x1b[2mthat...\""
	if !peerInputMatches("claude-code", snap("claude-code", 2, 12, append(paneRows("", 11), rule, edge, rule)), "") {
		t.Fatal("a ghost cut at the edge was not read as empty")
	}
	// While it works, Claude shows the mode row and an effort row under an
	// empty composer (measured on 2.1.282): still an empty composer, which is
	// what the check after Enter needs.
	working := append(paneRows("", 11), rule, "\x1b[39m\u276f\u00a0", rule,
		"  ⏵⏵ auto mode on (shift+tab to cycle) · esc to interrupt · ← for agents",
		"                                                            ◐ medium · /effort")
	if !peerInputMatches("claude-code", snap("claude-code", 2, 12, working), "") {
		t.Fatal("the working footer with the effort row was not read as an empty composer")
	}
	// Codex 0.157: its shortcuts hint sits under the model row (measured).
	codex := append(paneRows("", 20), "\x1b[1m›\x1b[0m \x1b[2mAsk Codex to do anything\x1b[0m", "",
		"  GPT-6-Astra default · /home/goat/picode",
		"  ? for shortcuts                                      ⚠ 1 warning · f2 to view")
	if !peerInputMatches("codex", snap("codex", 2, 20, codex), "") {
		t.Fatal("codex 0.157 composer with its shortcuts row was not read as empty")
	}
	// A bright word among dim ones is typed text, not the ghost.
	mixed := "\x1b[39m\u276f\u00a0\x1b[2mTry\x1b[0m typed"
	if peerInputMatches("claude-code", snap("claude-code", 2, 12, append(paneRows("", 11), rule, mixed, rule)), "") {
		t.Fatal("a bright word read as the ghost")
	}

	// Rendering drift (a ghost in a format the empty-matcher does not know)
	// is dim, so it must read UNKNOWN — deliver, never block.
	st, known = peerComposerState("claude-code", snap("claude-code", 2, 10,
		paneRows("\x1b[39m❯ \x1b[0m\x1b[2mbrand new ghost format\x1b[0m", 11)))
	if known {
		t.Fatalf("drifted ghost = %q known=%v; want unknown so delivery proceeds", st, known)
	}

	// Pi's frame region: content between the rules is positive draft.
	piRows := piPaneRows("write the tests")
	st, known = peerComposerState("pi", snap("pi", 0, 12, piRows))
	if !known || st != "occupied" {
		t.Fatalf("pi draft = %q known=%v", st, known)
	}
	emptyPi := piPaneRows("")
	st, known = peerComposerState("pi", snap("pi", 0, 12, emptyPi))
	if !known || st != "empty" {
		t.Fatalf("pi empty = %q known=%v", st, known)
	}
}

// make fills rows into a pane of height h with the payload on the last row.
func paneRows(last string, h int) []string {
	rows := make([]string, h)
	for i := range rows {
		rows[i] = ""
	}
	rows[h-1] = last
	return rows
}

// makePiRows builds a pi composer pane: cursor row between two rules, the
// pi status footer ("/" hint and "%/") under it.
func piPaneRows(draft string) []string {
	rows := make([]string, 24)
	rows[11] = strings.Repeat("─", 200)
	rows[12] = draft
	rows[13] = strings.Repeat("─", 200)
	rows[14] = "/ for commands"
	rows[15] = "42%/ context left"
	return rows
}

// ADR-0217: Omp 18.2 and OpenCode 1.18's home screen, captured live
// 2026-09-25 — the empty field reads empty (OpenCode's grey placeholder
// included), a typed draft reads occupied.
func TestOmpAndOpenCodeComposers(t *testing.T) {
	load := func(name string) tmux.InputSnapshot {
		t.Helper()
		b, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var s tmux.InputSnapshot
		if err := json.Unmarshal(b, &s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	for _, cli := range []string{"omp", "opencode"} {
		if st, known := peerComposerState(cli, load("composer-empty-"+cli+".json")); !known || st != "empty" {
			t.Errorf("%s empty = %q known=%v", cli, st, known)
		}
		draft := load("composer-draft-" + cli + ".json")
		if st, known := peerComposerState(cli, draft); !known || st != "occupied" {
			t.Errorf("%s draft = %q known=%v", cli, st, known)
		}
		if !peerInputMatches(cli, draft, "hello world") {
			t.Errorf("%s draft does not match its own text", cli)
		}
	}
}

// Grok 1.0.41 (captured live 2026-09-25): the welcome row names the build,
// and the indented box must not read as a draft when it is empty.
func TestGrok1041Composer(t *testing.T) {
	load := func(name string) tmux.InputSnapshot {
		t.Helper()
		b, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var s tmux.InputSnapshot
		if err := json.Unmarshal(b, &s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if st, known := peerComposerState("grok", load("grok-1041-empty.json")); !known || st != "empty" {
		t.Fatalf("grok 1.0.41 empty = %q known=%v", st, known)
	}
	if st, known := peerComposerState("grok", load("grok-1041-draft.json")); !known || st != "occupied" {
		t.Fatalf("grok 1.0.41 draft = %q known=%v", st, known)
	}
}
