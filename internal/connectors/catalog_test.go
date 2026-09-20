package connectors

import (
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

// The claude-code lesson (ADR-0150 phase 1): a driver id must be the CLI
// catalog id so deep links and the roster resolve, and the binary the
// driver names must be the catalog's executable.
func TestDriverIDsMatchCatalog(t *testing.T) {
	for _, id := range []string{"claude-code", "codex", "omp", "agy", "opencode", "grok", "muse", "hermes"} {
		d := For(id)
		if d == nil {
			t.Fatalf("no driver registered for catalog id %q", id)
		}
		if d.ID() != id {
			t.Fatalf("driver id = %q, want %q", d.ID(), id)
		}
		cli, ok := clilaunch.Find(id)
		if !ok {
			t.Fatalf("%q is not in the CLI catalog", id)
		}
		if d.Bin() != cli.Command {
			t.Fatalf("driver %s bin = %q, catalog executable = %q", id, d.Bin(), cli.Command)
		}
	}
}

// CLI agent sign-in hints: terminal commands for claude-code, codex, opencode,
// muse and hermes, the Omp TUI command, plain text for Antigravity
// (settings) and Grok (first use inside the CLI). The command, when
// present, is the copyable piece and must appear inside the refusal sentence.
func TestAuthHintsAreDriverSpecific(t *testing.T) {
	cases := []struct {
		id      string
		wantCmd string
		wantIn  string
		notIn   string
	}{
		{id: "claude-code", wantCmd: "claude mcp login docs", wantIn: "claude mcp login docs"},
		{id: "codex", wantCmd: "codex mcp login docs", wantIn: "codex mcp login docs"},
		{id: "omp", wantCmd: "/mcp reauth docs", wantIn: "/mcp reauth docs", notIn: "omp /mcp"},
		{id: "agy", wantCmd: "", wantIn: "Antigravity"},
		{id: "opencode", wantCmd: "opencode mcp auth docs", wantIn: "opencode mcp auth docs"},
		{id: "grok", wantCmd: "", wantIn: "inside Grok", notIn: "grok "},
		{id: "muse", wantCmd: "muse mcp login docs", wantIn: "muse mcp login docs"},
		{id: "hermes", wantCmd: "hermes mcp login docs", wantIn: "hermes mcp login docs"},
	}
	for _, tc := range cases {
		d := For(tc.id)
		if d == nil {
			t.Fatalf("no driver for %s", tc.id)
		}
		hint := d.AuthHint("docs")
		if hint.Command != tc.wantCmd {
			t.Fatalf("%s command = %q, want %q", tc.id, hint.Command, tc.wantCmd)
		}
		if !strings.Contains(hint.Text, tc.wantIn) {
			t.Fatalf("%s text = %q, want it to contain %q", tc.id, hint.Text, tc.wantIn)
		}
		if tc.notIn != "" && strings.Contains(hint.Text, tc.notIn) {
			t.Fatalf("%s text = %q, want it to omit %q", tc.id, hint.Text, tc.notIn)
		}
	}
}
