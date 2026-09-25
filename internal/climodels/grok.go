package climodels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Grok's catalog is the file Grok keeps itself: `$GROK_HOME/models_cache.json`
// (~/.grok by default). Measured 2026-09-25 against grok 1.0.41:
//
//   - Its own command, `grok models`, prints text only and re-extracts 27 files
//     under ~/.grok/docs every run (2026-09-23), so the reader never runs Grok:
//     it reads the list Grok fetched for the signed-in account
//     (`origin` …/v1/models), and writes nothing.
//   - The file is per account, not per folder; a Grok that never signed in
//     has none, and the report says so instead of guessing a list.
//   - The selector is the model id, the form `[models] default` takes;
//     thinking levels are the model's `reasoning_efforts`. Rows marked
//     `hidden` are never offered.
func grokRoot() string {
	if v := os.Getenv("GROK_HOME"); v != "" {
		return v
	}
	return filepath.Join(home(), ".grok")
}

func grokInputs(_, _ string) []string {
	return []string{filepath.Join(grokRoot(), "models_cache.json")}
}

func probeGrok(_ context.Context, _, dir string) (Report, error) {
	raw, err := os.ReadFile(filepath.Join(grokRoot(), "models_cache.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return Report{}, fmt.Errorf("Grok has not saved its model list yet — sign in and open Grok once")
	}
	if err != nil {
		return Report{}, fmt.Errorf("grok models: %w", err)
	}
	models, err := parseGrok(raw)
	if err != nil {
		return Report{}, err
	}
	return Report{CLI: "grok", Dir: dir, Models: models, Kinds: []string{"chat"}}, nil
}

func parseGrok(raw []byte) ([]Model, error) {
	var body struct {
		Models map[string]struct {
			Info struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Family   string `json:"model_family"`
				Context  int    `json:"context_window"`
				MaxOut   int    `json:"max_completion_tokens"`
				Hidden   bool   `json:"hidden"`
				Supports bool   `json:"supports_reasoning_effort"`
				Efforts  []struct {
					Value string `json:"value"`
				} `json:"reasoning_efforts"`
			} `json:"info"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("Grok's model list holds something PiCode could not read: %w", err)
	}
	rows := []Model{}
	for key, m := range body.Models {
		id := m.Info.ID
		if id == "" {
			id = key
		}
		if m.Info.Hidden {
			continue
		}
		provider := m.Info.Family
		if provider == "" {
			provider = "xai"
		}
		row := Model{Provider: provider, ID: id, Name: m.Info.Name, Kind: "chat", Selector: id,
			Context: m.Info.Context, MaxOut: m.Info.MaxOut}
		for _, e := range m.Info.Efforts {
			if e.Value != "" {
				row.Thinking = append(row.Thinking, e.Value)
			}
		}
		row.Thinking = orderLevels(row.Thinking)
		row.Reasoning = m.Info.Supports || len(row.Thinking) > 0
		rows = append(rows, row)
	}
	// A map has no order; a stable answer keeps the tests honest (the pane
	// sorts by name on its own).
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows, nil
}
