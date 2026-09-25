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
	"strings"
)

// Claude Code has no listing subcommand; its /model catalog answers one
// `list_models` control request over `-p` stream-json, without an API turn
// (adapted from orca's claude-model-list-probe). Measured 2026-09-25 against
// the owner's Claude Code:
//
//   - Every start rewrites the config it runs with (`.claude.json`) and runs
//     the folder's hooks, so the probe runs with CLAUDE_CONFIG_DIR and the
//     working directory set to a throwaway folder: the owner's config and
//     projects are never touched, and nothing is signed in.
//   - Signed out, it still answers the full catalog in ~0.7 s: aliases
//     (`opus[1m]`, `sonnet`, `haiku`) and ids, with each one's effort levels.
//     Rows marked `disabled` are placeholders ("update required") and the
//     `default` row mirrors another, so neither is offered.
//   - The account's extra models (`additionalModelOptionsCache` in the
//     owner's own `.claude.json`, written by Claude Code) are added from that
//     file — read, never written.
//   - The selector is the row's `value`, the form the `model` setting takes.
func claudeConfigDir() string {
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
		return v
	}
	return home()
}

func claudeInputs(command, _ string) []string {
	// The catalog is the binary's; the owner's .claude.json is rewritten by
	// every running Claude Code, so keying on it would never hit (as omp's
	// credential store did). The age bound covers the account's extras.
	if bin, err := exec.LookPath(command); err == nil {
		return []string{bin}
	}
	return nil
}

const claudeListRequest = `{"type":"control_request","request_id":"picode-models","request":{"subtype":"list_models"}}` + "\n"

func probeClaude(ctx context.Context, command, dir string) (Report, error) {
	scratch, err := os.MkdirTemp("", "picode-claude-models-")
	if err != nil {
		return Report{}, fmt.Errorf("claude models: %w", err)
	}
	defer os.RemoveAll(scratch)
	cmd := exec.CommandContext(ctx, command, "-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose")
	cmd.Dir = scratch
	cmd.Env = append(claudeEnv(), "CLAUDE_CONFIG_DIR="+filepath.Join(scratch, "config"))
	cmd.Stdin = strings.NewReader(claudeListRequest)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if ctx.Err() != nil {
			msg = "it did not answer in time"
		}
		return Report{}, fmt.Errorf("claude models: %s", bounded(msg, 400))
	}
	models, err := parseClaude(stdout.Bytes())
	if err != nil {
		return Report{}, err
	}
	models = addClaudeExtras(models, filepath.Join(claudeConfigDir(), ".claude.json"))
	return Report{CLI: "claude-code", Dir: dir, Models: models, Kinds: []string{"chat"}}, nil
}

// claudeEnv is the daemon's environment without the variables a Claude Code
// session sets for its children: a probe started from inside one (PiCode run
// by an agent) would otherwise act as that session's child.
func claudeEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if k == "CLAUDECODE" || k == "CLAUDE_CONFIG_DIR" || strings.HasPrefix(k, "CLAUDE_CODE_") {
			continue
		}
		env = append(env, kv)
	}
	return env
}

type claudeRow struct {
	Value       string   `json:"value"`
	DisplayName string   `json:"displayName"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Efforts     []string `json:"supportedEffortLevels"`
	Disabled    bool     `json:"disabled"`
}

func (r claudeRow) model() Model {
	name := r.DisplayName
	if name == "" {
		name = r.Label
	}
	m := Model{Provider: "anthropic", ID: r.Value, Name: name, Kind: "chat", Selector: r.Value}
	m.Thinking = orderLevels(append([]string{}, r.Efforts...))
	m.Reasoning = len(m.Thinking) > 0
	return m
}

func parseClaude(raw []byte) ([]Model, error) {
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if !bytes.Contains(line, []byte("control_response")) {
			continue
		}
		var msg struct {
			Type     string `json:"type"`
			Response struct {
				Subtype  string `json:"subtype"`
				Error    string `json:"error"`
				Response struct {
					Models []claudeRow `json:"models"`
				} `json:"response"`
			} `json:"response"`
		}
		if err := json.Unmarshal(line, &msg); err != nil || msg.Type != "control_response" {
			continue
		}
		if msg.Response.Subtype != "success" {
			// A Claude Code older than the request answers an error and exits 0.
			return nil, fmt.Errorf("this Claude Code cannot list its models (%s) — update it", bounded(msg.Response.Error, 200))
		}
		rows := []Model{}
		seen := map[string]bool{}
		for _, r := range msg.Response.Response.Models {
			if r.Value == "" || r.Value == "default" || r.Disabled || seen[r.Value] {
				continue
			}
			seen[r.Value] = true
			rows = append(rows, r.model())
		}
		return rows, nil
	}
	return nil, fmt.Errorf("Claude Code answered something PiCode could not read")
}

// addClaudeExtras appends the account's extra models Claude Code cached in
// the owner's config, skipping any the catalog already lists. A missing or
// unreadable file adds nothing: the catalog stands on its own.
func addClaudeExtras(rows []Model, config string) []Model {
	raw, err := os.ReadFile(config)
	if err != nil {
		return rows
	}
	var body struct {
		Extras []claudeRow `json:"additionalModelOptionsCache"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return rows
	}
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.ID] = true
	}
	for _, r := range body.Extras {
		if r.Value == "" || r.Disabled || seen[r.Value] {
			continue
		}
		seen[r.Value] = true
		rows = append(rows, r.model())
	}
	return rows
}
