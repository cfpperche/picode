package pisettings

import (
	"encoding/json"
	"os"
	"path/filepath"
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
