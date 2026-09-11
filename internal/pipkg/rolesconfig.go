// pi-roles configuration files (ADR-0028/0033) read and written by the
// Packages view. The GUI edits the same files the extension reads —
// <workspace>/.pi/roles.json and the per-agent overlay
// <agent cwd>/.pi/roles/<agentID>.json — never a copy in SQLite (ADR-0005).
// Validation mirrors packages/pi-roles/src/logic.ts so the GUI and the
// extension accept the same documents; unknown keys survive every write.
package pipkg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// BuiltinRoles are pi-roles' three builtin slots, in picker order.
var BuiltinRoles = []string{"default", "vision", "plan"}

var rolesThinking = map[string]bool{
	"off": true, "minimal": true, "low": true, "medium": true,
	"high": true, "xhigh": true, "max": true,
}

var rolesNameRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// Reserved covers pi command names and builtin slots (logic.ts RESERVED).
var rolesReserved = map[string]bool{
	"auto": true, "default": true, "vision": true, "plan": true,
	"role": true, "roles": true,
}

// RolesAssignment is one role → model + optional thinking level.
type RolesAssignment struct {
	Model    string `json:"model"`
	Thinking string `json:"thinking,omitempty"`
}

// RolesCustom is one named preset.
type RolesCustom struct {
	Name     string `json:"name"`
	Model    string `json:"model"`
	Thinking string `json:"thinking,omitempty"`
}

// RolesConfig is the typed shape of a roles file. Unknown keys live only
// in the raw map and are preserved on write.
type RolesConfig struct {
	Builtin map[string]*RolesAssignment `json:"builtin"`
	Custom  []RolesCustom               `json:"custom"`
}

// NewRolesConfig returns an empty config with the builtin map allocated.
func NewRolesConfig() RolesConfig {
	return RolesConfig{Builtin: map[string]*RolesAssignment{}, Custom: []RolesCustom{}}
}

// ParseRolesModelID splits a `provider/id` model string. nil when invalid —
// same rule as logic.ts parseModelId (a leading slash is a path, not a model).
func ParseRolesModelID(model string) (provider, id string, ok bool) {
	i := strings.Index(model, "/")
	if i <= 0 || i == len(model)-1 {
		return "", "", false
	}
	return model[:i], model[i+1:], true
}

func rolesAssignmentFromMap(path string, raw map[string]any) (RolesAssignment, error) {
	a := RolesAssignment{}
	model, _ := raw["model"].(string)
	if strings.TrimSpace(model) == "" {
		return a, fmt.Errorf("%s.model is required", path)
	}
	if _, _, ok := ParseRolesModelID(model); !ok {
		return a, fmt.Errorf("%s.model %q must be provider/id", path, model)
	}
	a.Model = model
	if raw["thinking"] != nil {
		t, _ := raw["thinking"].(string)
		if !rolesThinking[t] {
			return a, fmt.Errorf("%s.thinking %q is not a valid thinking level", path, raw["thinking"])
		}
		a.Thinking = t
	}
	return a, nil
}

// RolesConfigFromMap validates a decoded roles JSON document. Unknown keys
// anywhere are ignored (the extension ignores them too). Builtin slots set
// to JSON null mean "no assignment" — the GUI's way to clear a slot.
func RolesConfigFromMap(raw map[string]any) (RolesConfig, error) {
	cfg := NewRolesConfig()
	if raw == nil {
		return cfg, nil
	}
	if b, ok := raw["builtin"].(map[string]any); ok {
		for _, name := range BuiltinRoles {
			v, present := b[name]
			if !present || v == nil {
				continue
			}
			obj, ok := v.(map[string]any)
			if !ok {
				return cfg, fmt.Errorf("builtin.%s must be an object", name)
			}
			a, err := rolesAssignmentFromMap("builtin."+name, obj)
			if err != nil {
				return cfg, err
			}
			cfg.Builtin[name] = &a
		}
	}
	list, present := raw["custom"]
	if !present {
		return cfg, nil
	}
	items, ok := list.([]any)
	if !ok {
		return cfg, fmt.Errorf("custom must be an array")
	}
	seen := map[string]bool{}
	for i, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			return cfg, fmt.Errorf("custom[%d] must be an object", i)
		}
		name, _ := obj["name"].(string)
		if name == "" {
			return cfg, fmt.Errorf("custom[%d].name is required", i)
		}
		if !rolesNameRE.MatchString(name) {
			return cfg, fmt.Errorf("custom[%d].name %q is not a valid role name", i, name)
		}
		if rolesReserved[name] {
			return cfg, fmt.Errorf("custom[%d].name %q is reserved", i, name)
		}
		if seen[name] {
			return cfg, fmt.Errorf("custom role %q is duplicated", name)
		}
		seen[name] = true
		a, err := rolesAssignmentFromMap(fmt.Sprintf("custom[%d]", i), obj)
		if err != nil {
			return cfg, err
		}
		cfg.Custom = append(cfg.Custom, RolesCustom{Name: name, Model: a.Model, Thinking: a.Thinking})
	}
	return cfg, nil
}

// MergeRolesConfigs overlays the agent layer on the workspace layer: builtin
// slots and custom names present in the overlay win; everything else is
// inherited (logic.ts mergeConfigs).
func MergeRolesConfigs(base, overlay RolesConfig) RolesConfig {
	out := NewRolesConfig()
	for _, name := range BuiltinRoles {
		if a := overlay.Builtin[name]; a != nil {
			cp := *a
			out.Builtin[name] = &cp
		} else if a := base.Builtin[name]; a != nil {
			cp := *a
			out.Builtin[name] = &cp
		}
	}
	for _, c := range base.Custom {
		if o, ok := overlayCustom(overlay, c.Name); ok {
			out.Custom = append(out.Custom, o)
		} else {
			out.Custom = append(out.Custom, c)
		}
	}
	for _, o := range overlay.Custom {
		if _, ok := overlayCustom(base, o.Name); !ok {
			out.Custom = append(out.Custom, o)
		}
	}
	return out
}

func overlayCustom(cfg RolesConfig, name string) (RolesCustom, bool) {
	for _, c := range cfg.Custom {
		if c.Name == name {
			return c, true
		}
	}
	return RolesCustom{}, false
}

func rolesAssignmentJSON(a RolesAssignment) map[string]any {
	row := map[string]any{"model": a.Model}
	if a.Thinking != "" {
		row["thinking"] = a.Thinking
	}
	return row
}

// SerializeRolesConfig merges the typed config back onto the original
// document so unknown keys survive (logic.ts serializeConfig). Builtin slots
// absent from config are removed; custom replaces the array wholesale.
func SerializeRolesConfig(config RolesConfig, raw map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range raw {
		out[k] = v
	}
	builtin := map[string]any{}
	if prev, ok := raw["builtin"].(map[string]any); ok {
		for k, v := range prev {
			builtin[k] = v
		}
	}
	for _, name := range BuiltinRoles {
		if a := config.Builtin[name]; a != nil {
			builtin[name] = rolesAssignmentJSON(*a)
		} else {
			delete(builtin, name)
		}
	}
	out["builtin"] = builtin
	custom := make([]map[string]any, 0, len(config.Custom))
	for _, c := range config.Custom {
		row := map[string]any{"name": c.Name, "model": c.Model}
		if c.Thinking != "" {
			row["thinking"] = c.Thinking
		}
		custom = append(custom, row)
	}
	out["custom"] = custom
	return out
}

// RolesLayer is one roles file as the GUI sees it.
type RolesLayer struct {
	Path    string      `json:"path"`              // absolute file path
	Rel     string      `json:"rel"`               // workspace-relative display path
	Exists  bool        `json:"exists"`            // file exists on disk
	Invalid string      `json:"invalid,omitempty"` // parse/validation error; config is zero when set
	Config  RolesConfig `json:"config"`
}

// ReadRolesLayer reads one roles file. A missing file is Exists:false, not
// an error — dormant routing is a normal state (ADR-0028). Invalid JSON or
// an invalid document is reported in Invalid, never as a transport error,
// so the GUI can show the file problem without pretending the layer is
// empty (an empty layer would invite a silent overwrite).
func ReadRolesLayer(abs, rel string) RolesLayer {
	layer := RolesLayer{Path: abs, Rel: rel, Config: NewRolesConfig()}
	b, err := os.ReadFile(abs)
	if err != nil {
		return layer
	}
	layer.Exists = true
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		layer.Invalid = fmt.Sprintf("%s is not valid JSON: %v", rel, err)
		return layer
	}
	if raw == nil {
		layer.Invalid = fmt.Sprintf("%s must be a JSON object", rel)
		return layer
	}
	cfg, err := RolesConfigFromMap(raw)
	if err != nil {
		layer.Invalid = fmt.Sprintf("%s: %v", rel, err)
		return layer
	}
	layer.Config = cfg
	return layer
}

// WriteRolesFile atomically writes config merged onto raw. Unknown keys in
// raw survive; the tmp+rename pair means a crash never truncates the file.
func WriteRolesFile(abs string, config RolesConfig, raw map[string]any) error {
	if raw == nil {
		raw = map[string]any{}
	}
	doc := SerializeRolesConfig(config, raw)
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".roles-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpName, abs)
}

// RolesWorkspaceRel is the workspace file relative to the workspace folder.
const RolesWorkspaceRel = ".pi/roles.json"

// RolesOverlayRel is the per-agent overlay relative to the agent's working
// directory. An id that could not be a PI_ROLES_AGENT value (logic.ts
// parseAgentKey) returns "" — the GUI then has no agent layer.
func RolesOverlayRel(agentID string) string {
	id := strings.TrimSpace(agentID)
	if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`).MatchString(id) {
		return ""
	}
	return filepath.Join(".pi", "roles", id+".json")
}

// ConfigKindOf names a known config-capable package adapter, "" when the
// package has no GUI editor yet. Matching follows the role-state gate
// (roles_state.go): a substring on the source, so path installs of this
// repository (`…/packages/pi-roles`) and future npm installs both match.
// Descriptor packages (docs/plans/package-config-manifest.md) resolve the
// same way — catalog by name/source, then a picode.config manifest beside
// the installed package — and answer with the descriptor's id.
func ConfigKindOf(sources ...string) string {
	for _, s := range sources {
		if strings.Contains(strings.ToLower(s), "pi-roles") {
			return "roles"
		}
	}
	for _, s := range sources {
		if s == "" {
			continue
		}
		if d := DescriptorFor(s, ""); d != nil {
			return d.ID
		}
		if filepath.IsAbs(s) {
			if d := DescriptorFor("", s); d != nil {
				return d.ID
			}
		}
	}
	return ""
}
