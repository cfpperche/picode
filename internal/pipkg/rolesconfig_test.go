package pipkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustJSON(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return m
}

func TestRolesConfigFromMap(t *testing.T) {
	tests := []struct {
		name    string
		doc     string
		wantErr string
		check   func(t *testing.T, cfg RolesConfig)
	}{
		{
			name: "full valid document",
			doc:  `{"builtin":{"default":{"model":"zai/glm-5.3","thinking":"medium"},"vision":{"model":"xai/grok-4.6"}},"custom":[{"name":"redteam","model":"kimi-coding/k3","thinking":"low"}]}`,
			check: func(t *testing.T, cfg RolesConfig) {
				if cfg.Builtin["default"] == nil || cfg.Builtin["default"].Model != "zai/glm-5.3" || cfg.Builtin["default"].Thinking != "medium" {
					t.Fatalf("default slot = %+v", cfg.Builtin["default"])
				}
				if cfg.Builtin["vision"] == nil || cfg.Builtin["vision"].Thinking != "" {
					t.Fatalf("vision slot = %+v (omitted thinking stays empty)", cfg.Builtin["vision"])
				}
				if len(cfg.Custom) != 1 || cfg.Custom[0].Name != "redteam" {
					t.Fatalf("custom = %+v", cfg.Custom)
				}
			},
		},
		{name: "null builtin slot clears", doc: `{"builtin":{"default":null}}`, check: func(t *testing.T, cfg RolesConfig) {
			if cfg.Builtin["default"] != nil {
				t.Fatalf("null slot should stay unset, got %+v", cfg.Builtin["default"])
			}
		}},
		{name: "unknown keys ignored", doc: `{"future":"kept","custom":[{"name":"x","model":"p/m","extra":1}]}`, check: func(t *testing.T, cfg RolesConfig) {
			if len(cfg.Custom) != 1 || cfg.Custom[0].Model != "p/m" {
				t.Fatalf("custom = %+v", cfg.Custom)
			}
		}},
		{name: "model without provider", doc: `{"builtin":{"default":{"model":"glm-5.3"}}}`, wantErr: `builtin.default.model "glm-5.3" must be provider/id`},
		{name: "model trailing slash", doc: `{"custom":[{"name":"x","model":"zai/"}]}`, wantErr: `custom[0].model "zai/" must be provider/id`},
		{name: "bad thinking", doc: `{"builtin":{"plan":{"model":"p/m","thinking":"turbo"}}}`, wantErr: `not a valid thinking level`},
		{name: "bad name", doc: `{"custom":[{"name":"1bad","model":"p/m"}]}`, wantErr: `not a valid role name`},
		{name: "reserved name", doc: `{"custom":[{"name":"auto","model":"p/m"}]}`, wantErr: `"auto" is reserved`},
		{name: "duplicate custom", doc: `{"custom":[{"name":"x","model":"p/m"},{"name":"x","model":"p/q"}]}`, wantErr: `duplicated`},
		{name: "custom missing name", doc: `{"custom":[{"model":"p/m"}]}`, wantErr: `custom[0].name is required`},
		{name: "custom not array", doc: `{"custom":{}}`, wantErr: `custom must be an array`},
		{name: "builtin slot not object", doc: `{"builtin":{"vision":"p/m"}}`, wantErr: `builtin.vision must be an object`},
		{name: "assignment missing model", doc: `{"custom":[{"name":"x"}]}`, wantErr: `custom[0].model is required`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := RolesConfigFromMap(mustJSON(t, tt.doc))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want contains %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestMergeRolesConfigs(t *testing.T) {
	base := RolesConfig{Builtin: map[string]*RolesAssignment{
		"default": {Model: "zai/glm-5.3"}, "vision": {Model: "xai/grok-4.6"},
	}, Custom: []RolesCustom{{Name: "shared", Model: "p/a"}, {Name: "ws-only", Model: "p/b"}}}
	overlay := RolesConfig{Builtin: map[string]*RolesAssignment{
		"default": {Model: "anthropic/claude-haiku-4-5"},
	}, Custom: []RolesCustom{{Name: "shared", Model: "p/c"}, {Name: "agent-only", Model: "p/d"}}}

	got := MergeRolesConfigs(base, overlay)
	if got.Builtin["default"].Model != "anthropic/claude-haiku-4-5" {
		t.Fatalf("overlay must win the slot, got %v", got.Builtin["default"].Model)
	}
	if got.Builtin["vision"] == nil || got.Builtin["vision"].Model != "xai/grok-4.6" {
		t.Fatalf("unset overlay slot inherits, got %+v", got.Builtin["vision"])
	}
	names := map[string]string{}
	for _, c := range got.Custom {
		names[c.Name] = c.Model
	}
	if names["shared"] != "p/c" || names["ws-only"] != "p/b" || names["agent-only"] != "p/d" {
		t.Fatalf("custom merge wrong: %v", names)
	}
	if len(got.Custom) != 3 {
		t.Fatalf("custom count = %d, want 3", len(got.Custom))
	}
}

func TestSerializeRolesConfigPreservesUnknownKeys(t *testing.T) {
	raw := mustJSON(t, `{"future":{"nested":true},"builtin":{"default":{"model":"old/p","note":"keepme"},"vision":{"model":"x/y"}},"custom":[{"name":"a","model":"p/a"}]}`)
	cfg := RolesConfig{Builtin: map[string]*RolesAssignment{"default": {Model: "new/q"}}, Custom: []RolesCustom{{Name: "a", Model: "p/a2", Thinking: "low"}}}
	out := SerializeRolesConfig(cfg, raw)
	if out["future"] == nil {
		t.Fatal("unknown top-level key lost")
	}
	b := out["builtin"].(map[string]any)
	if _, ok := b["vision"]; ok {
		t.Fatal("cleared slot must be removed")
	}
	if b["default"].(map[string]any)["model"] != "new/q" {
		t.Fatal("slot not updated")
	}
	if len(out["custom"].([]map[string]any)) != 1 {
		t.Fatal("custom must replace the array")
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, ".pi", "roles.json")
	raw := mustJSON(t, `{"keep":"me","custom":[]}`)
	cfg := RolesConfig{Builtin: map[string]*RolesAssignment{"plan": {Model: "p/m", Thinking: "high"}}, Custom: []RolesCustom{}}
	if err := WriteRolesFile(abs, cfg, raw); err != nil {
		t.Fatalf("write: %v", err)
	}
	layer := ReadRolesLayer(abs, RolesWorkspaceRel)
	if !layer.Exists || layer.Invalid != "" {
		t.Fatalf("layer = exists:%v invalid:%q", layer.Exists, layer.Invalid)
	}
	if layer.Config.Builtin["plan"].Model != "p/m" {
		t.Fatalf("plan = %+v", layer.Config.Builtin["plan"])
	}
	// Unknown key survives the write.
	b, _ := os.ReadFile(abs)
	if !strings.Contains(string(b), `"keep": "me"`) {
		t.Fatalf("unknown key lost: %s", b)
	}
	// Missing file is dormant, not an error.
	missing := ReadRolesLayer(filepath.Join(dir, ".pi", "missing.json"), ".pi/missing.json")
	if missing.Exists || missing.Invalid != "" {
		t.Fatalf("missing layer = %+v", missing)
	}
}

func TestReadRolesLayerInvalid(t *testing.T) {
	dir := t.TempDir()
	abs := filepath.Join(dir, "roles.json")
	if err := os.WriteFile(abs, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	layer := ReadRolesLayer(abs, "roles.json")
	if !layer.Exists || layer.Invalid == "" {
		t.Fatalf("invalid JSON must surface in Invalid, got %+v", layer)
	}
	if err := os.WriteFile(abs, []byte(`{"custom":"nope"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if layer := ReadRolesLayer(abs, "roles.json"); layer.Invalid == "" {
		t.Fatal("invalid document must surface in Invalid")
	}
}

func TestRolesOverlayRel(t *testing.T) {
	if got := RolesOverlayRel("mobile-1a2b3c"); got != filepath.Join(".pi", "roles", "mobile-1a2b3c.json") {
		t.Fatalf("overlay rel = %q", got)
	}
	for _, bad := range []string{"", "  ", "/etc/passwd", "..", "a/b", strings.Repeat("x", 65)} {
		if got := RolesOverlayRel(bad); got != "" {
			t.Fatalf("RolesOverlayRel(%q) = %q, want empty", bad, got)
		}
	}
}

func TestConfigKindOf(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"pi-roles"}, "roles"},
		{[]string{"/home/u/picode/packages/pi-roles"}, "roles"},
		{[]string{"npm:pi-roles@0.5.1"}, "roles"},
		{[]string{"npm:pi-web-search"}, ""},
		{[]string{"", ""}, ""},
	}
	for _, c := range cases {
		if got := ConfigKindOf(c.in...); got != c.want {
			t.Fatalf("ConfigKindOf(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
