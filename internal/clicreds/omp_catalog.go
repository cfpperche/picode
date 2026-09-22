package clicreds

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Omp's own provider list. Omp ships its whole /login roster — about ninety
// providers, each with its name, its sign-in kind and the variables it reads
// a key from — compiled into one file of its npm package
// (@oh-my-pi/pi-catalog/src/compat/rules.json). The declaration in specs.go
// names the providers PiCode shares with pi and the other CLIs; this file
// appends every other provider Omp can sign in to, read from the installed
// package, so the pane's Add offers what Omp's own /login offers.
//
// The file is Omp's internals, not an API: when it moves or its shape
// changes, the reader finds nothing and the roster is the declaration alone
// (the owner's call, 2026-09-22: fix it then).

// ompRulesEnv points the reader at a rules.json of the caller's choosing
// (tests, or an Omp installed where LookPath cannot see it).
const ompRulesEnv = "PICODE_OMP_RULES"

// ompAliases are Omp's ids for providers the declaration already carries
// under the vault's vocabulary: listing them again would offer the same
// vendor twice under two names.
var ompAliases = map[string]bool{
	"meta":                true, // meta-ai
	"moonshot":            true, // kimi-coding
	"kimi-code":           true, // kimi-coding
	"opencode-zen":        true, // opencode
	"opencode-go":         true, // opencode
	"xai-oauth":           true, // xai
	"openai-codex-device": true, // openai-codex
}

// noteOmpLogin is the line a catalog provider with no key variable shows:
// its only door is Omp's own /login, whose result PiCode does not read.
const noteOmpLogin = "Omp signs in to this provider in its own /login and keeps the login in a SQLite database PiCode does not read."

// ompLocateTTL bounds how often omp is looked up on PATH again: For runs
// several times per provider in one roster request, and LookPath walks every
// PATH entry. The file itself is re-read only when its mtime moves.
const ompLocateTTL = 30 * time.Second

var ompCatalog struct {
	sync.Mutex
	locatedAt time.Time
	located   string
	path      string
	mod       time.Time
	providers []Provider
}

// resetOMPCatalog drops the cached read (tests).
func resetOMPCatalog() {
	ompCatalog.Lock()
	defer ompCatalog.Unlock()
	ompCatalog.locatedAt, ompCatalog.located = time.Time{}, ""
	ompCatalog.path, ompCatalog.mod, ompCatalog.providers = "", time.Time{}, nil
}

// withOMPCatalog returns the omp declaration with the installed package's
// other providers appended, in Omp's /login order. Any other spec is returned
// as is.
func withOMPCatalog(s Spec) Spec {
	if s.CLI != "omp" {
		return s
	}
	extra := ompCatalogProviders(s.Providers)
	if len(extra) == 0 {
		return s
	}
	out := s
	out.Providers = append(append(make([]Provider, 0, len(s.Providers)+len(extra)), s.Providers...), extra...)
	return out
}

func ompCatalogProviders(declared []Provider) []Provider {
	ompCatalog.Lock()
	defer ompCatalog.Unlock()
	path := strings.TrimSpace(os.Getenv(ompRulesEnv))
	if path == "" {
		if ompCatalog.locatedAt.IsZero() || time.Since(ompCatalog.locatedAt) >= ompLocateTTL {
			ompCatalog.located, ompCatalog.locatedAt = locateOMPRules(), time.Now()
		}
		path = ompCatalog.located
	}
	info, err := os.Stat(path)
	if path == "" || err != nil {
		ompCatalog.path, ompCatalog.providers = "", nil
		return nil
	}
	if path == ompCatalog.path && info.ModTime().Equal(ompCatalog.mod) {
		return ompCatalog.providers
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		ompCatalog.path, ompCatalog.providers = "", nil
		return nil
	}
	ompCatalog.path, ompCatalog.mod = path, info.ModTime()
	ompCatalog.providers = parseOMPRules(raw, declared)
	return ompCatalog.providers
}

// locateOMPRules finds rules.json next to the omp on PATH: the package's own
// node_modules first, then the hoisted layout one level up.
func locateOMPRules() string {
	bin, err := exec.LookPath("omp")
	if err != nil {
		return ""
	}
	if real, err := filepath.EvalSymlinks(bin); err == nil {
		bin = real
	}
	// bin is <pkg>/dist/cli.js (npm) or a wrapper inside <pkg>; walk up to
	// the package root, the directory whose package.json names Omp.
	dir := filepath.Dir(bin)
	for i := 0; i < 4 && dir != filepath.Dir(dir); i++ {
		if isOMPPackage(filepath.Join(dir, "package.json")) {
			for _, c := range []string{
				filepath.Join(dir, "node_modules", "@oh-my-pi", "pi-catalog", "src", "compat", "rules.json"),
				filepath.Join(filepath.Dir(dir), "pi-catalog", "src", "compat", "rules.json"),
			} {
				if _, err := os.Stat(c); err == nil {
					return c
				}
			}
			return ""
		}
		dir = filepath.Dir(dir)
	}
	return ""
}

func isOMPPackage(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var pkg struct {
		Name string `json:"name"`
	}
	return json.Unmarshal(raw, &pkg) == nil && pkg.Name == "@oh-my-pi/pi-coding-agent"
}

// ompRules is the part of rules.json this reader needs.
type ompRules struct {
	Auth struct {
		Providers []struct {
			ID    string `json:"id"`
			Name  string `json:"name"`
			Login *struct {
				Kind string `json:"kind"`
			} `json:"login"`
			Env *struct {
				Vars []string `json:"vars"`
			} `json:"env"`
		} `json:"providers"`
	} `json:"auth"`
	Providers map[string]struct {
		EnvVars []string `json:"envVars"`
	} `json:"providers"`
}

// parseOMPRules turns Omp's roster into declaration rows: a provider with a
// key variable takes an API key by env (the channel ompCredentialEnv
// injects), a provider with a sign-in flow also offers a subscription — done
// in Omp's /login — and one with neither (local models, cloud SDK chains) is
// left out, since nothing here could reach it. Declared ids and their Omp
// aliases are skipped.
func parseOMPRules(raw []byte, declared []Provider) []Provider {
	var rules ompRules
	if json.Unmarshal(raw, &rules) != nil {
		return nil
	}
	seen := map[string]bool{}
	for _, p := range declared {
		seen[p.Provider] = true
	}
	var out []Provider
	for _, a := range rules.Auth.Providers {
		id := strings.TrimSpace(a.ID)
		if id == "" || seen[id] || ompAliases[id] {
			continue
		}
		seen[id] = true
		env := ""
		if vars := rules.Providers[id].EnvVars; len(vars) > 0 {
			env = vars[0]
		} else if a.Env != nil && len(a.Env.Vars) > 0 {
			env = a.Env.Vars[0]
		}
		signIn := a.Login != nil && a.Login.Kind != "" && a.Login.Kind != "api-key"
		// Omp tells two doors to one vendor apart in its /login list with a
		// "· Sign in" tag ("Z.AI (GLM Coding Plan · Sign in)"); the pane
		// already says how each provider signs in, so the tag would only
		// repeat itself in the dialog's sentence.
		name := strings.TrimSpace(strings.ReplaceAll(a.Name, " · Sign in", ""))
		p := Provider{Provider: id, Name: name}
		if env != "" {
			p.Kinds = append(p.Kinds, KindAPIKey)
			p.Env = map[string]string{KindAPIKey: env}
		}
		if signIn {
			p.Kinds = append(p.Kinds, KindOAuth)
			p.Note = noteOmpStore
			if env == "" {
				p.Note = noteOmpLogin
			}
		}
		if len(p.Kinds) == 0 {
			continue
		}
		out = append(out, p)
	}
	return out
}
