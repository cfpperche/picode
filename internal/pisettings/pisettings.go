// Package pisettings reads and patches pi's settings.json (ADR-0012).
// Unknown keys are preserved. There is no pi settings CLI.
package pisettings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cfpperche/picode/internal/pipkg"
)

// Layer is the GUI view of one settings file.
type Layer struct {
	Path                 string          `json:"path"`
	Exists               bool            `json:"exists"`
	CompactionEnabled    bool            `json:"compactionEnabled"`
	SteeringMode         string          `json:"steeringMode"`
	FollowUpMode         string          `json:"followUpMode"`
	DefaultProvider      string          `json:"defaultProvider,omitempty"`
	DefaultModel         string          `json:"defaultModel,omitempty"`
	DefaultThinkingLevel string          `json:"defaultThinkingLevel,omitempty"`
	EnabledModels        []string        `json:"enabledModels,omitempty"`
	DefaultTools         []string        `json:"defaultTools,omitempty"`
	Has                  map[string]bool `json:"has,omitempty"`
}

// Patch is an allowlisted update. Nil pointer = leave alone.
type Patch struct {
	CompactionEnabled    *bool     `json:"compactionEnabled"`
	SteeringMode         *string   `json:"steeringMode"`
	FollowUpMode         *string   `json:"followUpMode"`
	DefaultProvider      *string   `json:"defaultProvider"`
	DefaultModel         *string   `json:"defaultModel"`
	DefaultThinkingLevel *string   `json:"defaultThinkingLevel"`
	EnabledModels        *[]string `json:"enabledModels"`
	DefaultTools         *[]string `json:"defaultTools"`
}

// UserFile is ~/.pi/agent/settings.json.
func UserFile() string {
	return filepath.Join(pipkg.UserDir(), "settings.json")
}

// ProjectFile is <cwd>/.pi/settings.json.
func ProjectFile(cwd string) string {
	if cwd == "" {
		return ""
	}
	return filepath.Join(cwd, ".pi", "settings.json")
}

// Load extracts the GUI layer. Missing file is an empty layer with defaults.
func Load(path string) (Layer, error) {
	out := Layer{
		Path:              path,
		CompactionEnabled: true,
		SteeringMode:      "one-at-a-time",
		FollowUpMode:      "one-at-a-time",
	}
	if path == "" {
		return out, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return out, err
	}
	out.Exists = true
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return out, fmt.Errorf("pi settings: %s: %w", path, err)
	}
	fill(&out, doc)
	return out, nil
}

// Apply writes patch into path (create + mkdir). Preserves other keys.
func Apply(path string, p Patch) error {
	if path == "" {
		return fmt.Errorf("pi settings: empty path")
	}
	doc := map[string]any{}
	raw, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("pi settings: %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := merge(doc, p); err != nil {
		return err
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

func fill(out *Layer, doc map[string]any) {
	out.Has = map[string]bool{}
	if c, ok := doc["compaction"].(map[string]any); ok {
		if _, ok := c["enabled"]; ok {
			out.Has["compactionEnabled"] = true
			if v, ok := c["enabled"].(bool); ok {
				out.CompactionEnabled = v
			}
		}
	}
	if s, ok := doc["steeringMode"].(string); ok && s != "" {
		out.Has["steeringMode"] = true
		out.SteeringMode = s
	}
	if s, ok := doc["followUpMode"].(string); ok && s != "" {
		out.Has["followUpMode"] = true
		out.FollowUpMode = s
	}
	markStr(out, doc, "defaultProvider", &out.DefaultProvider)
	markStr(out, doc, "defaultModel", &out.DefaultModel)
	markStr(out, doc, "defaultThinkingLevel", &out.DefaultThinkingLevel)
	markStrs(out, doc, "enabledModels", &out.EnabledModels)
	markStrs(out, doc, "defaultTools", &out.DefaultTools)
}

func markStr(out *Layer, doc map[string]any, key string, dest *string) {
	if _, ok := doc[key]; !ok {
		return
	}
	out.Has[key] = true
	*dest = str(doc, key)
}

func markStrs(out *Layer, doc map[string]any, key string, dest *[]string) {
	if _, ok := doc[key]; !ok {
		return
	}
	out.Has[key] = true
	*dest = asStrings(doc[key])
}

func merge(doc map[string]any, p Patch) error {
	if p.CompactionEnabled != nil {
		c, _ := doc["compaction"].(map[string]any)
		if c == nil {
			c = map[string]any{}
			doc["compaction"] = c
		}
		c["enabled"] = *p.CompactionEnabled
	}
	if err := setMode(doc, "steeringMode", p.SteeringMode); err != nil {
		return err
	}
	if err := setMode(doc, "followUpMode", p.FollowUpMode); err != nil {
		return err
	}
	setOrDelete(doc, "defaultProvider", p.DefaultProvider)
	setOrDelete(doc, "defaultModel", p.DefaultModel)
	if p.DefaultThinkingLevel != nil {
		s := *p.DefaultThinkingLevel
		if s != "" && !validThinking(s) {
			return fmt.Errorf("pi settings: bad thinking %q", s)
		}
		setOrDelete(doc, "defaultThinkingLevel", p.DefaultThinkingLevel)
	}
	if p.EnabledModels != nil {
		if len(*p.EnabledModels) == 0 {
			delete(doc, "enabledModels")
		} else {
			doc["enabledModels"] = *p.EnabledModels
		}
	}
	if p.DefaultTools != nil {
		doc["defaultTools"] = *p.DefaultTools
	}
	return nil
}

func setMode(doc map[string]any, key string, v *string) error {
	if v == nil {
		return nil
	}
	s := *v
	if s != "all" && s != "one-at-a-time" {
		return fmt.Errorf("pi settings: bad %s %q", key, s)
	}
	doc[key] = s
	return nil
}

func setOrDelete(doc map[string]any, key string, v *string) {
	if v == nil {
		return
	}
	if *v == "" {
		delete(doc, key)
		return
	}
	doc[key] = *v
}

func str(doc map[string]any, key string) string {
	s, _ := doc[key].(string)
	return s
}

func asStrings(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func validThinking(s string) bool {
	switch s {
	case "off", "minimal", "low", "medium", "high", "xhigh", "max":
		return true
	}
	return false
}
