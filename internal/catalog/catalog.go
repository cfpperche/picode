// Package catalog reads pi's model list and auth.json keys (ADR-0009).
package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Model is one row from `pi --list-models`, plus thinking levels from models-store.json.
type Model struct {
	ID             string   `json:"id"`
	Context        string   `json:"context,omitempty"`
	MaxOut         string   `json:"maxOut,omitempty"`
	Thinking       bool     `json:"thinking"`
	Images         bool     `json:"images"`
	ThinkingLevels []string `json:"thinkingLevels,omitempty"`
}

// Provider is a catalog group plus auth.json presence (never key values).
type Provider struct {
	ID       string  `json:"id"`
	SignedIn bool    `json:"signedIn"`
	AuthType string  `json:"authType,omitempty"` // api_key | oauth
	Login    string  `json:"login"`              // api_key | oauth | both
	Models   []Model `json:"models"`
}

// Report is the payload for GET /api/catalog.
type Report struct {
	Providers []Provider `json:"providers"`
	Thinking  []string   `json:"thinking"`
}

// ThinkingLevels is the full pi scale. Per-model subsets come from thinkingLevelMap.
var ThinkingLevels = []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"}

// Load runs pi --list-models --offline and merges auth.json key names.
func Load(piCmd string) (Report, error) {
	rep := Report{Providers: []Provider{}, Thinking: ThinkingLevels}
	if piCmd == "" {
		piCmd = "pi"
	}
	out, err := exec.Command(piCmd, "--list-models", "--offline").Output()
	if err != nil {
		return rep, fmt.Errorf("catalog: list-models: %w", err)
	}
	info := authInfo()
	store := loadThinkingMaps()
	byID := map[string]*Provider{}
	var order []string
	for _, m := range ParseListModels(string(out)) {
		p := byID[m.provider]
		if p == nil {
			order = append(order, m.provider)
			p = &Provider{ID: m.provider, Models: []Model{}}
			byID[m.provider] = p
		}
		levels := SupportedThinking(m.thinking, store[m.provider+"/"+m.model])
		p.Models = append(p.Models, Model{
			ID: m.model, Context: m.context, MaxOut: m.maxOut,
			Thinking: m.thinking, Images: m.images, ThinkingLevels: levels,
		})
	}
	for id := range LoginMethods {
		if byID[id] == nil {
			order = append(order, id)
			byID[id] = &Provider{ID: id, Models: []Model{}}
		}
	}
	for _, id := range order {
		p := byID[id]
		p.Login = loginMethod(id)
		if a, ok := info[id]; ok {
			p.SignedIn = true
			p.AuthType = a
		}
		rep.Providers = append(rep.Providers, *p)
	}
	return rep, nil
}

type parsedRow struct {
	provider, model, context, maxOut string
	thinking, images                 bool
}

// ParseListModels turns the table from `pi --list-models` into rows.
func ParseListModels(text string) []parsedRow {
	var rows []parsedRow
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		if fields[0] == "provider" {
			continue
		}
		// provider | model… | context | max-out | thinking | images
		images := fields[len(fields)-1]
		thinking := fields[len(fields)-2]
		maxOut := fields[len(fields)-3]
		context := fields[len(fields)-4]
		model := strings.Join(fields[1:len(fields)-4], " ")
		rows = append(rows, parsedRow{
			provider: fields[0],
			model:    model,
			context:  context,
			maxOut:   maxOut,
			thinking: thinking == "yes",
			images:   images == "yes",
		})
	}
	return rows
}

// SupportedThinking mirrors pi-ai getSupportedThinkingLevels:
// null in thinkingLevelMap hides a level; xhigh/max need an explicit map entry.
func SupportedThinking(reasoning bool, levelMap map[string]any) []string {
	if !reasoning {
		return []string{"off"}
	}
	var out []string
	for _, level := range ThinkingLevels {
		mapped, has := levelMap[level]
		if has && mapped == nil {
			continue
		}
		if (level == "xhigh" || level == "max") && !has {
			continue
		}
		out = append(out, level)
	}
	if len(out) == 0 {
		return []string{"off"}
	}
	return out
}

func loadThinkingMaps() map[string]map[string]any {
	out := map[string]map[string]any{}
	home, err := os.UserHomeDir()
	if err != nil {
		return out
	}
	b, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "models-store.json"))
	if err != nil {
		return out
	}
	var store map[string]struct {
		Models []struct {
			ID               string         `json:"id"`
			Reasoning        bool           `json:"reasoning"`
			ThinkingLevelMap map[string]any `json:"thinkingLevelMap"`
		} `json:"models"`
	}
	if json.Unmarshal(b, &store) != nil {
		return out
	}
	for provider, entry := range store {
		for _, m := range entry.Models {
			out[provider+"/"+m.ID] = m.ThinkingLevelMap
		}
	}
	return out
}

func authInfo() map[string]string {
	out := map[string]string{}
	home, err := os.UserHomeDir()
	if err != nil {
		return out
	}
	b, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "auth.json"))
	if err != nil {
		return out
	}
	var obj map[string]struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(b, &obj) != nil {
		return out
	}
	for k, v := range obj {
		t := v.Type
		if t == "" {
			t = LoginAPIKey
		}
		out[k] = t
	}
	return out
}

// AuthPath is where pi stores credentials (keys only are ever read).
func AuthPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "auth.json")
}

// PutAPIKey writes {type:api_key,key} for provider. Other keys stay. The key is never logged.
func PutAPIKey(provider, key string) error {
	provider = strings.TrimSpace(provider)
	key = strings.TrimSpace(key)
	if provider == "" {
		return fmt.Errorf("provider required")
	}
	if key == "" {
		return fmt.Errorf("key required")
	}
	cred, err := json.Marshal(map[string]string{"type": "api_key", "key": key})
	if err != nil {
		return err
	}
	return mutateAuth(func(obj map[string]json.RawMessage) error {
		obj[provider] = cred
		return nil
	})
}

// RemoveAuth deletes one provider entry from auth.json. Other keys stay.
func RemoveAuth(provider string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return fmt.Errorf("provider required")
	}
	return mutateAuth(func(obj map[string]json.RawMessage) error {
		delete(obj, provider)
		return nil
	})
}

func mutateAuth(fn func(map[string]json.RawMessage) error) error {
	path := AuthPath()
	if path == "" {
		return fmt.Errorf("no home directory")
	}
	obj := map[string]json.RawMessage{}
	raw, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(raw, &obj); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := fn(obj); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	out, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o600)
}
