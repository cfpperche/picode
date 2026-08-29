package pikeys

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/pipkg"
)

func TestSetRoundTripAndReset(t *testing.T) {
	home := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return filepath.Join(home, ".pi", "agent") }
	t.Cleanup(func() { pipkg.UserDir = old })

	got, err := LoadUser()
	if err != nil || len(got) != 0 {
		t.Fatalf("empty: %v %v", got, err)
	}
	if err := Set("tui.editor.deleteWordBackward", []string{"ctrl+w", "ctrl+backspace"}); err != nil {
		t.Fatal(err)
	}
	got, err = LoadUser()
	if err != nil {
		t.Fatal(err)
	}
	if len(got["tui.editor.deleteWordBackward"]) != 2 {
		t.Fatalf("%v", got)
	}
	raw, err := os.ReadFile(File())
	if err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if err := Set("nope.action", []string{"a"}); err == nil {
		t.Fatal("unknown action")
	}
	if err := Set("tui.editor.deleteWordBackward", nil); err != nil {
		t.Fatal(err)
	}
	got, err = LoadUser()
	if err != nil || len(got) != 0 {
		t.Fatalf("reset: %v %v", got, err)
	}
}

func TestSetOffAndKeepsUnknown(t *testing.T) {
	home := t.TempDir()
	old := pipkg.UserDir
	dir := filepath.Join(home, ".pi", "agent")
	pipkg.UserDir = func() string { return dir }
	t.Cleanup(func() { pipkg.UserDir = old })
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(File(), []byte(`{"future.thing":"x","tui.input.submit":"enter"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Set("tui.altScreen.pageUp", []string{}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(File())
	if err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["future.thing"] != "x" {
		t.Fatalf("lost unknown: %s", raw)
	}
	arr, _ := doc["tui.altScreen.pageUp"].([]any)
	if len(arr) != 0 {
		t.Fatalf("off: %v", arr)
	}
}

func TestValidKey(t *testing.T) {
	ok := []string{"enter", "ctrl+w", "ctrl+backspace", "shift+enter", "ctrl+shift+f", "pageUp", "["}
	for _, k := range ok {
		if !ValidKey(k) {
			t.Fatalf("want valid %q", k)
		}
	}
	bad := []string{"", "ctrl+", "foo+enter", "ctrl w"}
	for _, k := range bad {
		if ValidKey(k) {
			t.Fatalf("want bad %q", k)
		}
	}
}
