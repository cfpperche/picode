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
	for _, id := range []string{"claude-code", "codex", "omp", "agy"} {
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

// Guest sign-in hints: terminal commands for claude-code and codex, the
// Omp TUI command, plain text for Antigravity. The command, when present,
// is the copyable piece and must appear inside the refusal sentence.
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
