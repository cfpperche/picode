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

// Model is one row from `pi --list-models`.
type Model struct {
	ID       string `json:"id"`
	Context  string `json:"context,omitempty"`
	MaxOut   string `json:"maxOut,omitempty"`
	Thinking bool   `json:"thinking"`
	Images   bool   `json:"images"`
}

// Provider is a catalog group plus whether auth.json has a key for it.
type Provider struct {
	ID       string  `json:"id"`
	SignedIn bool    `json:"signedIn"`
	Models   []Model `json:"models"`
}

// Report is the payload for GET /api/catalog.
type Report struct {
	Providers []Provider `json:"providers"`
	Thinking  []string   `json:"thinking"`
}

// ThinkingLevels matches `pi --thinking`.
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
	signed := authKeys()
	byID := map[string]*Provider{}
	var order []string
	for _, m := range ParseListModels(string(out)) {
		p := byID[m.provider]
		if p == nil {
			order = append(order, m.provider)
			p = &Provider{ID: m.provider, SignedIn: signed[m.provider], Models: []Model{}}
			byID[m.provider] = p
		}
		p.Models = append(p.Models, Model{
			ID: m.model, Context: m.context, MaxOut: m.maxOut,
			Thinking: m.thinking, Images: m.images,
		})
	}
	for _, id := range order {
		rep.Providers = append(rep.Providers, *byID[id])
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

func authKeys() map[string]bool {
	out := map[string]bool{}
	home, err := os.UserHomeDir()
	if err != nil {
		return out
	}
	b, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "auth.json"))
	if err != nil {
		return out
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(b, &obj) != nil {
		return out
	}
	for k := range obj {
		out[k] = true
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
