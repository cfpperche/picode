package climodels

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// OpenCode's catalog is `opencode models --verbose`: a `provider/model` line,
// then that model's JSON, for every model. Measured 2026-09-23 against
// opencode 1.18.32:
//
//   - ~1 s. It refreshes the models.dev cache from the network when stale;
//     OPENCODE_DISABLE_MODELS_FETCH=1 keeps it on the cache (same output).
//   - The answer depends on the folder: a project opencode.json with
//     disabled_providers ["openai"] gave 86 rows against 105, and
//     enabled_providers ["xai"] gave 12. It depends on credentials too —
//     without auth.json only the free opencode/* models are listed.
//   - Every run touches its own opencode.db (a project row's time) — its
//     state, not the owner's settings; accepted.
//   - The selector is the `provider/model` line, the form OpenCode's `model`
//     setting takes; thinking levels are the model's `variants`.
func opencodeInputs(command, dir string) []string {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home(), ".config")
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(home(), ".local", "share")
	}
	cache := os.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		cache = filepath.Join(home(), ".cache")
	}
	files := []string{
		filepath.Join(config, "opencode", "opencode.json"),
		filepath.Join(config, "opencode", "opencode.jsonc"),
		filepath.Join(data, "opencode", "auth.json"),
		filepath.Join(cache, "opencode", "models.json"),
	}
	if bin, err := exec.LookPath(command); err == nil {
		files = append(files, bin)
	}
	if dir != "" {
		files = append(files, filepath.Join(dir, "opencode.json"), filepath.Join(dir, "opencode.jsonc"))
	}
	return files
}

func probeOpencode(ctx context.Context, command, dir string) (Report, error) {
	out, err := runQuiet(ctx, dir, []string{"OPENCODE_DISABLE_MODELS_FETCH=1"}, command, "models", "--verbose")
	if err != nil {
		return Report{}, fmt.Errorf("opencode models: %s", err)
	}
	models, err := parseOpencode(out)
	if err != nil {
		return Report{}, err
	}
	return Report{CLI: "opencode", Dir: dir, Models: models, Kinds: []string{"chat"}}, nil
}

// thinkingOrder is the usual order of effort names; anything else follows,
// sorted, so a vendor's new level is shown rather than dropped.
var thinkingOrder = []string{"none", "off", "minimal", "low", "medium", "high", "xhigh", "max"}

func orderLevels(names []string) []string {
	rank := map[string]int{}
	for i, n := range thinkingOrder {
		rank[n] = i
	}
	sort.SliceStable(names, func(i, j int) bool {
		ri, iok := rank[names[i]]
		rj, jok := rank[names[j]]
		switch {
		case iok && jok:
			return ri < rj
		case iok != jok:
			return iok
		}
		return names[i] < names[j]
	})
	return names
}

func parseOpencode(raw []byte) ([]Model, error) {
	rows := []Model{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	var selector string
	var block strings.Builder
	inBlock := false
	flush := func() error {
		var m struct {
			ID         string `json:"id"`
			ProviderID string `json:"providerID"`
			Name       string `json:"name"`
			Limit      struct {
				Context int `json:"context"`
				Output  int `json:"output"`
			} `json:"limit"`
			Cost *struct {
				Input  float64 `json:"input"`
				Output float64 `json:"output"`
			} `json:"cost"`
			Capabilities struct {
				Reasoning bool            `json:"reasoning"`
				Input     map[string]bool `json:"input"`
			} `json:"capabilities"`
			Variants map[string]json.RawMessage `json:"variants"`
		}
		if err := json.Unmarshal([]byte(block.String()), &m); err != nil {
			return fmt.Errorf("opencode models answered something PiCode could not read: %w", err)
		}
		provider, id := m.ProviderID, m.ID
		if selector == "" {
			selector = provider + "/" + id
		}
		row := Model{Provider: provider, ID: id, Name: m.Name, Kind: "chat", Selector: selector,
			Context: m.Limit.Context, MaxOut: m.Limit.Output, Reasoning: m.Capabilities.Reasoning}
		if m.Cost != nil && (m.Cost.Input > 0 || m.Cost.Output > 0) {
			row.Cost = &Cost{Input: m.Cost.Input, Output: m.Cost.Output}
		}
		for kind, ok := range m.Capabilities.Input {
			if ok && kind != "text" {
				row.Input = append(row.Input, kind)
			}
		}
		sort.Strings(row.Input)
		for name := range m.Variants {
			row.Thinking = append(row.Thinking, name)
		}
		row.Thinking = orderLevels(row.Thinking)
		rows = append(rows, row)
		selector = ""
		block.Reset()
		return nil
	}
	for sc.Scan() {
		line := sc.Text()
		switch {
		case !inBlock && line == "{":
			inBlock = true
			block.WriteString(line)
		case inBlock:
			block.WriteString(line)
			if line == "}" {
				inBlock = false
				if err := flush(); err != nil {
					return nil, err
				}
			}
		case strings.TrimSpace(line) != "":
			selector = strings.TrimSpace(line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("opencode models: %w", err)
	}
	return rows, nil
}
