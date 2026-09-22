package clikeys

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The pane captures chords in its own syntax (`ctrl+alt+k`); codex's file takes
// its own (`ctrl-alt-k`). The engine renders one into the other at the write, and
// refuses what codex cannot express — because codex validates its keymap at
// startup and a chord it cannot parse is a CLI that does not start.
func TestCodexChordSpeaksTheFileVocabulary(t *testing.T) {
	for in, want := range map[string]string{
		"ctrl+alt+k":   "ctrl-alt-k",
		"ctrl+o":       "ctrl-o",
		"f5":           "f5",
		"ctrl+f24":     "ctrl-f24", // the last function key codex's parser takes
		"shift+enter":  "shift-enter",
		"ctrl+pageup":  "ctrl-page-up",
		"escape":       "esc",
		"return":       "enter",
		"backspace":    "backspace",
		"alt+space":    "alt-space",
		"ctrl+-":       "ctrl-minus",
		"ctrl+alt+up":  "ctrl-alt-up",
		"shift+ctrl+x": "shift-ctrl-x", // order is the pane's; both are keys codex knows
	} {
		got, err := codexChord(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got != want {
			t.Fatalf("%s rendered %q, want %q", in, got, want)
		}
	}
	for _, bad := range []string{"super+k", "meta+j", "ctrl+f25", "ctr+up", ""} {
		if got, err := codexChord(bad); err == nil {
			t.Fatalf("%q must be refused, got %q", bad, got)
		}
	}
}

// The whole path: a captured chord written through the engine lands in codex's
// spelling, and a chord codex cannot express leaves the file exactly as it was.
func TestCodexWriteRendersTheChord(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.toml")
	before := "model = \"gpt-5\"\n"
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ReadFlat(CodexMap)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteFlat(CodexMap, "composer.submit", []string{"ctrl+alt+m"}, false, report.Revision); err != nil {
		t.Fatal(err)
	}
	if got := read(t, path); !strings.Contains(got, `submit = ["ctrl-alt-m"]`) {
		t.Fatalf("the chord was not rendered in codex's spelling:\n%s", got)
	}
	// A chord codex has no word for is refused by name, and the file is untouched.
	report, err = ReadFlat(CodexMap)
	if err != nil {
		t.Fatal(err)
	}
	err = WriteFlat(CodexMap, "composer.submit", []string{"super+m"}, false, report.Revision)
	if err == nil || !strings.Contains(err.Error(), "super") {
		t.Fatalf("a chord codex cannot express must be refused by name, got %v", err)
	}
	if got := read(t, path); !strings.Contains(got, `ctrl-alt-m`) {
		t.Fatalf("a refused write touched the file:\n%s", got)
	}
}
