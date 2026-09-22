package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// The other agent CLIs travel in a snapshot the way Pi does (ADR-0179):
// settings always, credentials only with secrets, file-based sessions only
// with sessions — from the paths each CLI's own declarations name — and a
// restore puts them back under $HOME.
func TestSnapshotRestoreOtherCLIs(t *testing.T) {
	e, _, _ := testEngine(t)
	home := os.Getenv("HOME")

	claudeSettings := filepath.Join(home, ".claude", "settings.json")
	claudeCreds := filepath.Join(home, ".claude", ".credentials.json")
	claudeSession := filepath.Join(home, ".claude", "projects", "-work", "s1.jsonl")
	codexSession := filepath.Join(home, ".codex", "sessions", "2026", "09", "22", "rollout.jsonl")
	writeFile(t, claudeSettings, `{"model":"opus"}`)
	writeFile(t, claudeCreds, `{"token":"secret"}`)
	writeFile(t, claudeSession, "{}\n")
	writeFile(t, codexSession, "{}\n")

	// Without secrets or sessions: settings only.
	bare, err := e.Snapshot(false, false, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(bare.Path, "clis", "claude-code", "home", ".claude", "settings.json")); err != nil {
		t.Fatalf("settings missing from a bare snapshot: %v", err)
	}
	for _, never := range []string{
		filepath.Join(bare.Path, "clis", "claude-code", "home", ".claude", ".credentials.json"),
		filepath.Join(bare.Path, "clis", "claude-code", "trees"),
		filepath.Join(bare.Path, "clis", "codex", "trees"),
	} {
		if _, err := os.Stat(never); err == nil {
			t.Fatalf("%s copied without its toggle", never)
		}
	}

	dest := t.TempDir()
	snap, err := e.Snapshot(true, true, dest)
	if err != nil {
		t.Fatal(err)
	}
	cred := filepath.Join(snap.Path, "clis", "claude-code", "home", ".claude", ".credentials.json")
	if st, err := os.Stat(cred); err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("credential copy = %v %v, want 0600", st, err)
	}

	// Change everything live, then restore.
	writeFile(t, claudeSettings, `{"model":"changed"}`)
	writeFile(t, claudeCreds, `{"token":"changed"}`)
	if err := os.RemoveAll(filepath.Join(home, ".claude", "projects")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, codexSession, "changed\n")
	schema, err := e.Store.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Restore(dest, snap.ID, schema); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, claudeSettings); got != `{"model":"opus"}` {
		t.Fatalf("settings = %q", got)
	}
	if got := readFile(t, claudeCreds); got != `{"token":"secret"}` {
		t.Fatalf("credentials = %q", got)
	}
	if got := readFile(t, claudeSession); got != "{}\n" {
		t.Fatalf("claude session = %q", got)
	}
	if got := readFile(t, codexSession); got != "{}\n" {
		t.Fatalf("codex session = %q", got)
	}
}

func TestUnderHomeRefusesOutsidePaths(t *testing.T) {
	for _, c := range []struct {
		p  string
		ok bool
	}{
		{"/h/.claude/settings.json", true},
		{"/h", false},
		{"/other/x", false},
		{"/h/../etc/passwd", false},
	} {
		if _, ok := underHome("/h", c.p); ok != c.ok {
			t.Errorf("underHome(%q) = %v", c.p, ok)
		}
	}
}
