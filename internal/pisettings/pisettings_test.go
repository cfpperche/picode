package pisettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyKeepsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":"dark","packages":[],"compaction":{"reserveTokens":8}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	off := false
	if err := Apply(path, Patch{CompactionEnabled: &off}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["theme"] != "dark" {
		t.Fatalf("theme lost: %v", doc["theme"])
	}
	c, _ := doc["compaction"].(map[string]any)
	if c["enabled"] != false {
		t.Fatalf("enabled = %v", c["enabled"])
	}
	if c["reserveTokens"] != float64(8) {
		t.Fatalf("reserveTokens lost: %v", c["reserveTokens"])
	}
}

func TestLoadHasOnlyPresentKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"defaultProvider":"xai"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	layer, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !layer.Has["defaultProvider"] || layer.Has["steeringMode"] {
		t.Fatalf("has = %+v", layer.Has)
	}
}

func TestLoadMissing(t *testing.T) {
	layer, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if layer.Exists || !layer.CompactionEnabled || layer.SteeringMode != "one-at-a-time" {
		t.Fatalf("%+v", layer)
	}
}

func TestApplyEnabledModels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":"dark"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	list := []string{"claude-*", "gpt-4o"}
	if err := Apply(path, Patch{EnabledModels: &list}); err != nil {
		t.Fatal(err)
	}
	layer, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !layer.Has["enabledModels"] || len(layer.EnabledModels) != 2 {
		t.Fatalf("%+v", layer)
	}
	empty := []string{}
	if err := Apply(path, Patch{EnabledModels: &empty}); err != nil {
		t.Fatal(err)
	}
	layer, _ = Load(path)
	if layer.Has["enabledModels"] {
		t.Fatal("empty list should drop the key")
	}
}

func TestApplyRejectsBadMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	bad := "sometimes"
	if err := Apply(path, Patch{SteeringMode: &bad}); err == nil {
		t.Fatal("expected error")
	}
}

// A reset hands one value back to the parent layer: only the named keys leave
// the file, and a nested object goes with its last key.
func TestResetRemovesOnlyTheNamedKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	body := `{"theme":"dark","steeringMode":"all","followUpMode":"all","defaultProvider":"xai","enabledModels":["a"],"compaction":{"enabled":false,"reserveTokens":8}}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(path, Patch{Reset: []string{"steeringMode", "compactionEnabled"}}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["steeringMode"]; ok {
		t.Fatalf("steeringMode kept: %v", doc)
	}
	c, _ := doc["compaction"].(map[string]any)
	if _, ok := c["enabled"]; ok {
		t.Fatalf("compaction.enabled kept: %v", c)
	}
	if c["reserveTokens"] != float64(8) || doc["theme"] != "dark" || doc["followUpMode"] != "all" || doc["defaultProvider"] != "xai" {
		t.Fatalf("reset touched another key: %v", doc)
	}
	if _, ok := doc["enabledModels"].([]any); !ok {
		t.Fatalf("enabledModels lost: %v", doc)
	}
}

// The nested object leaves with its last key; a sibling keeps it alive.
func TestResetDropsAnEmptyCompactionObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"compaction":{"enabled":true}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(path, Patch{Reset: []string{"compactionEnabled"}}); err != nil {
		t.Fatal(err)
	}
	layer, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if layer.Has["compactionEnabled"] {
		t.Fatalf("still overridden: %+v", layer.Has)
	}
	raw, _ := os.ReadFile(path)
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["compaction"]; ok {
		t.Fatalf("empty compaction object left behind: %q", raw)
	}
}

// An unknown name is refused, not ignored: the UI must never report
// "inherited" while the override stayed in the file.
func TestResetRefusesUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"steeringMode":"all"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(path, Patch{Reset: []string{"model"}}); err == nil {
		t.Fatal("unknown reset key accepted")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != `{"steeringMode":"all"}` {
		t.Fatalf("refused patch still wrote: %q", raw)
	}
}

// A patch that sets and resets the same field sets it: the write is explicit,
// the reset is the fallback it replaces.
func TestResetWithSetWins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"steeringMode":"all"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mode := "one-at-a-time"
	if err := Apply(path, Patch{Reset: []string{"steeringMode"}, SteeringMode: &mode}); err != nil {
		t.Fatal(err)
	}
	layer, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !layer.Has["steeringMode"] || layer.SteeringMode != "one-at-a-time" {
		t.Fatalf("layer = %+v", layer)
	}
}

// TestMachineKeysRoundTrip covers the five keys the Pi pane gained on
// 2026-09-20. Each name was read out of pi 0.86.1's own setter before it was
// declared; `defaultProjectTrust` carries the value domain its getter
// enforces, because pi resolves anything else to "ask" and the row would
// never look set.
func TestMachineKeysRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"defaultModel":"opus","packages":["npm:pi-roles"]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	theme, shell, trust := "dark", "/bin/zsh", "always"
	hide, quiet := true, true
	if err := Apply(path, Patch{
		Theme: &theme, ShellPath: &shell, DefaultProjectTrust: &trust,
		HideThinkingBlock: &hide, QuietStartup: &quiet,
	}); err != nil {
		t.Fatal(err)
	}
	rep, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Theme != "dark" || rep.ShellPath != "/bin/zsh" || rep.DefaultProjectTrust != "always" {
		t.Fatalf("strings: %#v", rep)
	}
	if !rep.HideThinkingBlock || !rep.QuietStartup {
		t.Fatalf("bools: %#v", rep)
	}
	for _, key := range []string{"theme", "shellPath", "defaultProjectTrust", "hideThinkingBlock", "quietStartup"} {
		if !rep.Has[key] {
			t.Errorf("%s is set but not reported as set here", key)
		}
	}
	// A key PiCode does not manage survives untouched.
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), "pi-roles") {
		t.Fatalf("an unknown key was dropped:\n%s", raw)
	}
	// A value outside the domain pi resolves is refused, not written.
	bad := "maybe"
	if err := Apply(path, Patch{DefaultProjectTrust: &bad}); err == nil {
		t.Fatal("want a refusal for a trust value pi would resolve to ask")
	}
	// Reset hands every one of them back.
	if err := Apply(path, Patch{Reset: []string{"theme", "shellPath", "defaultProjectTrust", "hideThinkingBlock", "quietStartup"}}); err != nil {
		t.Fatal(err)
	}
	rep, _ = Load(path)
	for _, key := range []string{"theme", "shellPath", "defaultProjectTrust", "hideThinkingBlock", "quietStartup"} {
		if rep.Has[key] {
			t.Errorf("%s survived a reset", key)
		}
	}
	if rep.DefaultModel != "opus" {
		t.Fatalf("a sibling key was lost: %#v", rep)
	}
}
