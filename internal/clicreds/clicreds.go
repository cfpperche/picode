// Package clicreds declares, per agent CLI, which provider credentials it can
// use and where it keeps its own login (ADR-0165). It reads; it never writes a
// CLI's files. Step 1 of the credential work uses it to answer three
// questions in the Providers pane — which accounts this CLI could use, whether
// it is already logged in by itself, and how to verify a stored key — and step
// 2 uses the same declarations to inject an account into a launch.
//
// One declaration per CLI, the vendor's own names, verified against the
// installed CLIs (docs/benchmarks/2026-09-20-agent-cli-credentials.md) — a
// claim this file cannot support is not here.
package clicreds

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Kinds of credential a provider row can hold.
const (
	KindAPIKey = "api_key"
	KindOAuth  = "oauth"
)

// Spec is one CLI's declaration.
type Spec struct {
	CLI       string     `json:"cli"`
	Name      string     `json:"name"`
	Providers []Provider `json:"providers"`
}

// Provider is one provider this CLI can talk to.
type Provider struct {
	// Provider is the vault's provider id ("anthropic", "openai-codex", …) —
	// the same vocabulary pi uses, so usage adapters and identity keep working.
	Provider string `json:"provider"`
	// Kinds lists the credential shapes this CLI accepts for that provider.
	Kinds []string `json:"kinds"`
	// Env names the variable each kind is passed in, when the vendor ships
	// one (step 2 injection, and the honest answer to "can two terminals run
	// two accounts at once").
	Env map[string]string `json:"env,omitempty"`
	// Native points at the CLI's own login, when it has one PiCode can read.
	Native *Native `json:"native,omitempty"`
	// Note is the one line the pane shows where a row cannot be injected —
	// what the person should expect instead of a control that does nothing.
	Note string `json:"note,omitempty"`
}

// Native describes a CLI's own credential store.
type Native struct {
	// Path is the credential file inside the CLI's home, with $HOME and
	// $XDG_DATA_HOME expanded. It is the *default* home: a per-account
	// directory (step 2) holds the same file name.
	Path string `json:"path"`
	// Format names the parser for that file (see parse.go).
	Format string `json:"format"`
	// DirEnv is the environment variable that moves the CLI's whole home to
	// another directory — how step 2 runs two accounts of one CLI at once.
	DirEnv string `json:"dirEnv,omitempty"`
	// VendorDir is that home relative to $HOME (".claude"), used to build a
	// per-account directory.
	VendorDir string `json:"vendorDir,omitempty"`
	// CredFile is the credential's file name inside the home.
	CredFile string `json:"credFile,omitempty"`
	// Seed lists the entries a per-account directory links back to the real
	// home so settings, sessions and memory survive the isolation.
	Seed []string `json:"seed,omitempty"`
}

// Login is what reading a CLI's own store found: a credential in the vault's
// shape, ready to be imported.
type Login struct {
	Provider string
	Kind     string
	Cred     json.RawMessage
	// Label is a vendor-volunteered name (an email), when the file carries
	// one. Never typed by PiCode.
	Label string
}

// Declarations returns every CLI's declaration.
func Declarations() []Spec { return catalog }

// For returns one CLI's declaration.
func For(cli string) (Spec, bool) {
	for _, s := range catalog {
		if s.CLI == cli {
			return s, true
		}
	}
	return Spec{}, false
}

// ProvidersFor lists the vault providers a CLI can use, in declaration order.
func ProvidersFor(cli string) []string {
	spec, ok := For(cli)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(spec.Providers))
	for _, p := range spec.Providers {
		out = append(out, p.Provider)
	}
	return out
}

// CanUse reports whether the CLI declares that provider, and in which kinds.
func CanUse(cli, provider string) ([]string, bool) {
	spec, ok := For(cli)
	if !ok {
		return nil, false
	}
	for _, p := range spec.Providers {
		if p.Provider == provider {
			return p.Kinds, true
		}
	}
	return nil, false
}

// EnvVar is the variable a kind is passed in for that provider, when the
// vendor ships one.
func EnvVar(cli, provider, kind string) (string, bool) {
	spec, ok := For(cli)
	if !ok {
		return "", false
	}
	for _, p := range spec.Providers {
		if p.Provider != provider {
			continue
		}
		name, ok := p.Env[kind]
		return name, ok && name != ""
	}
	return "", false
}

// Detect reads the CLI's own login, if it has one and it parses. Read-only:
// a file PiCode cannot understand is reported as "no login", never guessed at.
func Detect(cli string) (Login, bool) {
	spec, ok := For(cli)
	if !ok {
		return Login{}, false
	}
	for _, p := range spec.Providers {
		if p.Native == nil {
			continue
		}
		path := expandPath(p.Native.Path)
		if path == "" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil || len(raw) == 0 {
			continue
		}
		if login, ok := parseLogin(p.Native.Format, p.Provider, raw); ok {
			return login, true
		}
	}
	return Login{}, false
}

// expandPath resolves a declaration's path: $HOME, $XDG_DATA_HOME, $XDG_CONFIG_HOME.
func expandPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(p, "$XDG_DATA_HOME"):
		base := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			base = filepath.Join(home, ".local", "share")
		}
		return filepath.Join(base, strings.TrimPrefix(p, "$XDG_DATA_HOME/"))
	case strings.HasPrefix(p, "$XDG_CONFIG_HOME"):
		base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			base = filepath.Join(home, ".config")
		}
		return filepath.Join(base, strings.TrimPrefix(p, "$XDG_CONFIG_HOME/"))
	case strings.HasPrefix(p, "$HOME"):
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, strings.TrimPrefix(p, "$HOME/"))
	}
	return p
}
