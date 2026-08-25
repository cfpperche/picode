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
