package climodels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Codex's catalog is `codex debug models` — "Render the raw model catalog as
// JSON". Measured 2026-09-23 against codex-cli 0.156.1:
//
//   - One JSON object, {"models":[…]}, ~600 KB because every row carries its
//     base instructions; the reader keeps slug, display name, context window,
//     reasoning levels and input modalities. Rows marked visibility "hide"
//     are internal (codex-auto-review) and never offered.
//   - 0.1–0.2 s; it answers from ~/.codex/models_cache.json (fetched by the
//     CLI itself, keyed to the signed-in account) and wrote nothing when run.
//     Without auth it lists its bundled catalog (11 models against 9).
//   - A project .codex/config.toml was ignored in an untrusted folder; a
//     trusted project's effect was not measured, so the folder is an input.
//   - The selector is the slug: Codex's `model` setting takes it as is.
func codexInputs(command, dir string) []string {
	root := os.Getenv("CODEX_HOME")
	if root == "" {
		root = filepath.Join(home(), ".codex")
	}
	files := []string{
		filepath.Join(root, "auth.json"),
		filepath.Join(root, "models_cache.json"),
		filepath.Join(root, "config.toml"),
	}
	if bin, err := exec.LookPath(command); err == nil {
		files = append(files, bin)
	}
	if dir != "" {
		files = append(files, filepath.Join(dir, ".codex", "config.toml"))
	}
	return files
}

func probeCodex(ctx context.Context, command, dir string) (Report, error) {
	out, err := runQuiet(ctx, dir, nil, command, "debug", "models")
	if err != nil {
		return Report{}, fmt.Errorf("codex debug models: %s", err)
	}
	models, err := parseCodex(out)
	if err != nil {
		return Report{}, err
	}
	return Report{CLI: "codex", Dir: dir, Models: models, Kinds: []string{"chat"}}, nil
}

func parseCodex(raw []byte) ([]Model, error) {
	var body struct {
		Models []struct {
			Slug          string   `json:"slug"`
			DisplayName   string   `json:"display_name"`
			ContextWindow int      `json:"context_window"`
			Visibility    string   `json:"visibility"`
			Input         []string `json:"input_modalities"`
			Levels        []struct {
				Effort string `json:"effort"`
			} `json:"supported_reasoning_levels"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("codex debug models answered something PiCode could not read: %w", err)
	}
	rows := []Model{}
	for _, m := range body.Models {
		if m.Slug == "" || m.Visibility == "hide" {
			continue
		}
		row := Model{Provider: "openai", ID: m.Slug, Name: m.DisplayName, Kind: "chat", Selector: m.Slug, Context: m.ContextWindow}
		for _, l := range m.Levels {
			if l.Effort != "" {
				row.Thinking = append(row.Thinking, l.Effort)
			}
		}
		row.Reasoning = len(row.Thinking) > 0
		for _, in := range m.Input {
			if in != "text" {
				row.Input = append(row.Input, in)
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// runQuiet runs a CLI's read-only listing and returns stdout, or the CLI's own
// first words on failure (bounded; never a stack of Go wrapping).
func runQuiet(ctx context.Context, dir string, env []string, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	cmd.Stdin = nil
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
		return nil, fmt.Errorf("%s", bounded(msg, 400))
	}
	return stdout.Bytes(), nil
}
