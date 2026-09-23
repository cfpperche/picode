package clisession

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := filepath.Join(home, "proj")
	claudeDir := filepath.Join(home, ".claude", "projects", encodeClaudeTestDir(proj))
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"user","message":{"role":"user","content":"hi"},"timestamp":"2026-09-01T10:00:00Z","cwd":"` + proj + `","sessionId":"cc-9"}` + "\n"
	if err := os.WriteFile(filepath.Join(claudeDir, "cc-9.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	piFile := filepath.Join(home, "s.jsonl")
	if err := os.WriteFile(piFile, []byte(`{"type":"session","id":"p1","cwd":"/w"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	l := NewLocator()
	for _, tc := range []struct {
		name, cli, id, path, cwd string
		found                    bool
	}{
		{"claude in its folder", "claude-code", "cc-9", "", proj, true},
		{"claude, folder unknown: machine-wide", "claude-code", "cc-9", "", "", true},
		{"claude, wrong folder: falls back to machine-wide", "claude-code", "cc-9", "", "/elsewhere", true},
		{"claude, unknown id", "claude-code", "nope", "", proj, false},
		{"no id is never a match", "claude-code", "", filepath.Join(claudeDir, "cc-9.jsonl"), proj, false},
		{"pi by file", "pi", "", piFile, "", true},
		{"pi file gone", "pi", "", filepath.Join(home, "gone.jsonl"), "", false},
		{"unknown CLI", "nope", "x", "", "", false},
	} {
		got, err := l.Locate(tc.cli, tc.id, tc.path, tc.cwd)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if (got != nil) != tc.found {
			t.Fatalf("%s: found = %v", tc.name, got)
		}
		if got != nil && tc.cli == "claude-code" && (len(got.ResumeArgs) != 2 || got.ResumeArgs[1] != "cc-9") {
			t.Fatalf("%s: resume args = %v", tc.name, got.ResumeArgs)
		}
	}
}

func encodeClaudeTestDir(cwd string) string {
	return strings.NewReplacer("/", "-", ".", "-").Replace(cwd)
}
