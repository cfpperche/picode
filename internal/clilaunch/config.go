// Package clilaunch describes installed coding CLIs and terminal launch settings.
// It has no agent protocol, session history or orchestration responsibilities.
package clilaunch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type CLI struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
	Docs    string `json:"docs"`
}

func Catalog() []CLI {
	return []CLI{
		{"pi", "Pi", "pi", "https://pi.dev"},
		{"claude-code", "Claude Code", "claude", "https://code.claude.com/docs/en/setup"},
		{"codex", "Codex", "codex", "https://developers.openai.com/codex/cli"},
		{"grok", "Grok", "grok", "https://grok.com/build"},
	}
}

func Find(id string) (CLI, bool) {
	for _, c := range Catalog() {
		if c.ID == id {
			return c, true
		}
	}
	return CLI{}, false
}

type Config struct {
	Executable  string            `json:"executable"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env"`
	Path        []string          `json:"path"`
	Integration bool              `json:"integration"`
}

type Overrides struct {
	Executable  *string            `json:"executable,omitempty"`
	Args        *[]string          `json:"args,omitempty"`
	Env         map[string]*string `json:"env,omitempty"`
	Path        *[]string          `json:"path,omitempty"`
	Integration *bool              `json:"integration,omitempty"`
}

func Resolve(base Config, v Overrides) Config {
	base.Args = append([]string{}, base.Args...)
	base.Path = append([]string{}, base.Path...)
	env := map[string]string{}
	for k, x := range base.Env {
		env[k] = x
	}
	for k, x := range v.Env {
		if x == nil {
			delete(env, k)
		} else {
			env[k] = *x
		}
	}
	base.Env = env
	if v.Executable != nil {
		base.Executable = *v.Executable
	}
	if v.Args != nil {
		base.Args = append([]string{}, (*v.Args)...)
	}
	if v.Path != nil {
		base.Path = append([]string{}, (*v.Path)...)
	}
	if v.Integration != nil {
		base.Integration = *v.Integration
	}
	return base
}

var envKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func Validate(c Config) error {
	if len(c.Args) > 128 || len(c.Path) > 32 || len(c.Env) > 64 {
		return fmt.Errorf("Too many launch options.")
	}
	check := func(v string) bool { return len(v) <= 8192 && !strings.ContainsAny(v, "\x00\r\n") }
	if !check(c.Executable) {
		return fmt.Errorf("Executable must be one path or command name.")
	}
	for _, a := range c.Args {
		if !check(a) {
			return fmt.Errorf("Each argument must be a single line.")
		}
	}
	for _, p := range c.Path {
		if !check(p) || !filepath.IsAbs(p) || strings.ContainsRune(p, ':') {
			return fmt.Errorf("PATH entries must be absolute directories without colons.")
		}
	}
	for k, v := range c.Env {
		if !envKey.MatchString(k) || !check(v) {
			return fmt.Errorf("Environment variables need a valid name and a single-line value.")
		}
		if strings.HasPrefix(k, "PICODE_") || k == "PATH" || k == "HOME" || k == "SHELL" || k == "GROK_HOME" {
			return fmt.Errorf("%s is managed by the launcher.", k)
		}
	}
	return nil
}

func Fingerprint(c Config) string {
	raw, _ := json.Marshal(Resolve(c, Overrides{}))
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// Snapshot contains diagnostics only. Configurations are read separately by
// the editor; secrets never ride terminal views or change-feed events.
type Snapshot struct {
	CLI         string           `json:"cli"`
	Executable  string           `json:"executable"`
	Identity    string           `json:"identity,omitempty"`
	Args        []string         `json:"args"`
	EnvKeys     []string         `json:"envKeys"`
	Path        []string         `json:"path"`
	Integration bool             `json:"integration"`
	Fingerprint string           `json:"fingerprint"`
	StartedAt   string           `json:"startedAt"`
	Injection   *IntegrationPlan `json:"injection,omitempty"`
}

func Describe(c Config, executable, at string) Snapshot {
	keys := []string{}
	for k := range c.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	args := append([]string{}, c.Args...)
	secret := false
	for i, a := range args {
		lower := strings.ToLower(a)
		if secret {
			args[i] = "••••"
			secret = false
			continue
		}
		if strings.HasPrefix(a, "-") && (strings.Contains(lower, "token") || strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "api-key") || strings.Contains(lower, "api_key")) {
			if before, _, ok := strings.Cut(a, "="); ok {
				args[i] = before + "=••••"
			} else {
				secret = true
			}
		}
	}
	return Snapshot{Executable: executable, Args: args, EnvKeys: keys, Path: c.Path, Integration: c.Integration, Fingerprint: Fingerprint(c), StartedAt: at}
}
