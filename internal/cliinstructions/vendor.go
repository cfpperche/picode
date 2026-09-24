package cliinstructions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The few vendor settings the rules depend on, read from each CLI's own
// file with the standard library. A file that is missing or does not parse
// means the vendor's default, which is what the CLI itself would use.

// claudeMode is Claude Code's Project instructions setting: the built-in
// agents-md plugin's instructionFiles option, read from managed settings
// first and then the user's (project and local settings are ignored for it).
// The legacy projectInstructions spelling is still honoured by 2.1.280.
func claudeMode(home string) string {
	for _, p := range []string{"/etc/claude-code/managed-settings.json", filepath.Join(home, ".claude", "settings.json")} {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var doc struct {
			PluginConfigs map[string]struct {
				Options map[string]any `json:"options"`
			} `json:"pluginConfigs"`
		}
		if json.Unmarshal(b, &doc) != nil {
			continue
		}
		opts := doc.PluginConfigs["agents-md@builtin"].Options
		for _, k := range []string{"instructionFiles", "projectInstructions"} {
			if v, ok := opts[k].(string); ok {
				switch v {
				case "claude-md-or-agents-md", "claude-md-and-agents-md", "claude-md", "managed-only":
					return v
				}
			}
		}
	}
	return "claude-md-or-agents-md"
}

type codexConfig struct {
	maxBytes  int
	fallbacks []string
}

// readCodexConfig reads the two top-level keys of ~/.codex/config.toml that
// shape AGENTS.md discovery. Only keys before the first [table] are top-level.
func readCodexConfig(path string) codexConfig {
	cfg := codexConfig{maxBytes: 32 * 1024}
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	quoted := regexp.MustCompile(`"([^"]*)"|'([^']*)'`)
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			break
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch strings.TrimSpace(key) {
		case "project_doc_max_bytes":
			if n, err := strconv.Atoi(strings.ReplaceAll(val, "_", "")); err == nil && n > 0 {
				cfg.maxBytes = n
			}
		case "project_doc_fallback_filenames":
			for _, m := range quoted.FindAllStringSubmatch(val, -1) {
				name := m[1] + m[2]
				if name != "" && !strings.ContainsAny(name, `/\`) {
					cfg.fallbacks = append(cfg.fallbacks, name)
				}
			}
		}
	}
	return cfg
}

// hermesCap is context_file_max_chars from ~/.hermes/config.yaml, or 0 when
// Hermes sizes the cap from the model's window.
func hermesCap(home string) int {
	b, err := os.ReadFile(filepath.Join(home, ".hermes", "config.yaml"))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "context_file_max_chars:") {
			continue
		}
		v := strings.TrimSpace(strings.TrimPrefix(line, "context_file_max_chars:"))
		if i := strings.Index(v, "#"); i >= 0 {
			v = strings.TrimSpace(v[:i])
		}
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 0
}

// grokTrusted reports whether ~/.grok/trusted_folders.toml trusts exactly
// this folder. Measured: trusting a parent folder does not trust the
// repository inside it.
func grokTrusted(home, folder string) bool {
	b, err := os.ReadFile(filepath.Join(home, ".grok", "trusted_folders.toml"))
	if err != nil {
		return false
	}
	// Both sides are resolved: the file names folders the way the user typed
	// them, the scan reports what the kernel resolved, and on macOS those
	// differ under /var (the aliasing `canon` exists for — measured
	// 2026-09-24, the first run of the macOS leg on a push).
	want := filepath.Clean(folder)
	if resolved, err := canon(want); err == nil {
		want = resolved
	}
	header := regexp.MustCompile(`^\[folders\."(.*)"\]\s*$`)
	current := ""
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if m := header.FindStringSubmatch(line); m != nil {
			current = filepath.Clean(strings.ReplaceAll(m[1], `\\`, `\`))
			if resolved, err := canon(current); err == nil {
				current = resolved
			}
			continue
		}
		if strings.HasPrefix(line, "[") {
			current = ""
			continue
		}
		if current == want {
			if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) == "trusted" {
				return strings.TrimSpace(v) == "true"
			}
		}
	}
	return false
}

// imports resolves a file's import lines to absolute paths: Claude Code and
// Omp's `@path` (relative to the importing file, `~/` for home) and
// Antigravity's `@[label](path)`.
func imports(f *File) []string {
	var out []string
	base := filepath.Dir(f.abs)
	for _, m := range importLine.FindAllStringSubmatch(f.text, -1) {
		p := m[1]
		if strings.HasPrefix(p, "[") {
			if i := strings.Index(p, "]("); i >= 0 {
				p = strings.TrimSuffix(p[i+2:], ")")
			}
		}
		switch {
		case strings.HasPrefix(p, "~/"):
			if h, err := os.UserHomeDir(); err == nil {
				p = filepath.Join(h, p[2:])
			}
		case !filepath.IsAbs(p):
			p = filepath.Join(base, p)
		}
		out = append(out, filepath.Clean(p))
	}
	return out
}
