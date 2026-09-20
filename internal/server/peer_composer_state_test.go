package server

import (
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
