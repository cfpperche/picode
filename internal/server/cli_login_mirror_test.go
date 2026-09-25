package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/credentials"
)

// The in-use Anthropic row follows Claude Code's own file, and its refresh
// token is reported as Claude Code's (2026-09-24: the row had held a dead
// token for ten days while Claude Code's login worked).
func TestMirrorCLILoginsFollowsClaudeCode(t *testing.T) {
	home, data := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PICODE_DATA", data)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	vault := credentials.Default()
	if _, err := vault.Import("anthropic", json.RawMessage(`{"type":"oauth","access":"a-dead","refresh":"r-dead","expires":1}`), "", "vault", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	file := `{"claudeAiOauth":{"accessToken":"a-live","refreshToken":"r-live","expiresAt":4102444800000,"scopes":["user:inference"],"subscriptionType":"max"}}`
	if err := os.WriteFile(filepath.Join(home, ".claude", ".credentials.json"), []byte(file), 0o600); err != nil {
		t.Fatal(err)
	}
	mirrorCLILogins("anthropic")
	row, ok, err := vault.Row("anthropic", "6e306c515177")
	if err != nil || !ok {
		t.Fatalf("row: %v %v", ok, err)
	}
	var got struct{ Access, Refresh string }
	_ = json.Unmarshal(row.Cred, &got)
	if got.Access != "a-live" || got.Refresh != "r-live" {
		t.Fatalf("row after mirror = %+v", got)
	}
	if who := cliHoldingRefresh("anthropic", "r-live"); who != "Claude Code" {
		t.Fatalf("holder of the live refresh = %q", who)
	}
	if who := cliHoldingRefresh("anthropic", "r-dead"); who != "" {
		t.Fatalf("a token no CLI holds is held by %q", who)
	}
}
