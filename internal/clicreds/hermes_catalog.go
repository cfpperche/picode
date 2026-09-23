package clicreds

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Hermes's own provider list (ADR-0193). Hermes keeps its registry in its
// Python package (hermes_cli.auth.PROVIDER_REGISTRY): each provider's id,
// name, auth type and the variables it reads a key from. PiCode asks Hermes's
// own interpreter for it and appends every provider the declaration lacks, so
// the pane offers what `hermes auth add` offers. Aliases (one provider under
// several ids) are listed once, by the first id Hermes names.
//
// The registry is Hermes's code, not an API: when it moves, the reader finds
// nothing and the roster is the declaration alone.

// hermesCatalogEnv points the reader at a JSON file of the reader's own shape
// (tests), skipping the interpreter.
const hermesCatalogEnv = "PICODE_HERMES_CATALOG"

// hermesDeclared are Hermes's ids for providers the declaration carries under
// the vault's vocabulary — listed again they would be the same vendor twice.
var hermesDeclared = map[string]bool{
	"openai-codex": true, "anthropic": true, "xai": true, "xai-oauth": true,
	"gemini": true, "openrouter": true, "copilot": true, "opencode-zen": true,
	"zai": true, "kimi-coding": true, "kimi-coding-cn": true, "kimi-for-coding": true,
}

// hermesSkipped are auth types PiCode cannot drive from a dialog yet: cloud
// SDK chains, an external process, and Qwen's login that needs its own CLI.
var hermesSkipped = map[string]bool{"aws_sdk": true, "vertex": true, "external_process": true}

// HermesAddID is the id `hermes auth add` takes for a vault provider and a
// credential kind: the vault's vocabulary differs from Hermes's for a few.
func HermesAddID(provider, kind string) string {
	switch provider {
	case "google":
		return "gemini"
	case "github-copilot":
		return "copilot"
	case "opencode":
		return "opencode-zen"
	case "xai":
		if kind == KindOAuth {
			return "xai-oauth"
		}
	}
	return provider
}

// HermesDeviceSignin says whether `hermes auth add <id> --type oauth` runs a
// device-code sign-in PiCode can show (measured on 0.21.4: Codex, xAI, Nous,
// MiniMax). Anthropic and GitHub Copilot are declared with an oauth kind for
// the vault's sake, but Hermes signs neither in that way — a key for the
// first, an external process for the second.
func HermesDeviceSignin(provider string) bool {
	switch provider {
	case "anthropic", "github-copilot":
		return false
	}
	return true
}

type hermesRow struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	AuthType string   `json:"auth_type"`
	Env      []string `json:"env"`
}

const hermesTTL = 60 * time.Second

var hermesCatalog struct {
	sync.Mutex
	at   time.Time
	src  string
	rows []Provider
}

func resetHermesCatalog() {
	hermesCatalog.Lock()
	defer hermesCatalog.Unlock()
	hermesCatalog.at, hermesCatalog.src, hermesCatalog.rows = time.Time{}, "", nil
}

func withHermesCatalog(s Spec) Spec {
	if s.CLI != "hermes" {
		return s
	}
	extra := hermesCatalogProviders(s.Providers)
	if len(extra) == 0 {
		return s
	}
	out := s
	out.Providers = append(append(make([]Provider, 0, len(s.Providers)+len(extra)), s.Providers...), extra...)
	return out
}

// hermesHome is Hermes's own home: $HERMES_HOME, else ~/.hermes.
func hermesHome() string {
	if h := strings.TrimSpace(os.Getenv("HERMES_HOME")); h != "" {
		return h
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".hermes")
}

const hermesDumpScript = `import json
from hermes_cli.auth import PROVIDER_REGISTRY as R
print(json.dumps([{"id": k, "name": p.name, "auth_type": p.auth_type, "env": list(p.api_key_env_vars or [])} for k, p in R.items()]))`

func hermesCatalogProviders(declared []Provider) []Provider {
	hermesCatalog.Lock()
	defer hermesCatalog.Unlock()
	// The cache is keyed by where the catalog comes from, so a changed
	// source (a test's file, another HERMES_HOME) is read again at once.
	src := os.Getenv(hermesCatalogEnv) + "|" + hermesHome()
	if !hermesCatalog.at.IsZero() && hermesCatalog.src == src && time.Since(hermesCatalog.at) < hermesTTL {
		return hermesCatalog.rows
	}
	hermesCatalog.at, hermesCatalog.src = time.Now(), src
	hermesCatalog.rows = parseHermesCatalog(readHermesCatalog(), declared)
	return hermesCatalog.rows
}

func readHermesCatalog() []byte {
	if p := strings.TrimSpace(os.Getenv(hermesCatalogEnv)); p != "" {
		raw, _ := os.ReadFile(p)
		return raw
	}
	python, dir := hermesInterpreter()
	if python == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, python, "-c", hermesDumpScript)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PYTHONPATH="+dir, "PYTHONHOME=")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	return out
}

// hermesWrapperExec is the line Hermes's installer writes into the `hermes`
// on PATH: a shell script that execs the venv's own entry point.
var hermesWrapperExec = regexp.MustCompile(`exec\s+"?([^"\s]+/venv/bin/hermes)"?`)

// hermesInterpreter finds the Python Hermes runs on, and its source dir,
// from the `hermes` on PATH — the install the person actually uses, wherever
// HERMES_HOME points — falling back to ~/.hermes/hermes-agent.
func hermesInterpreter() (python, dir string) {
	fromVenvBin := func(bin string) (string, string) {
		venvBin := filepath.Dir(bin)
		py := filepath.Join(venvBin, "python")
		if _, err := os.Stat(py); err != nil {
			return "", ""
		}
		return py, filepath.Dir(filepath.Dir(venvBin))
	}
	if bin, err := exec.LookPath("hermes"); err == nil {
		if real, err := filepath.EvalSymlinks(bin); err == nil {
			bin = real
		}
		if filepath.Base(filepath.Dir(bin)) == "bin" && filepath.Base(filepath.Dir(filepath.Dir(bin))) == "venv" {
			if py, d := fromVenvBin(bin); py != "" {
				return py, d
			}
		}
		if raw, err := os.ReadFile(bin); err == nil && len(raw) < 64<<10 {
			if m := hermesWrapperExec.FindSubmatch(raw); m != nil {
				if py, d := fromVenvBin(string(m[1])); py != "" {
					return py, d
				}
			}
		}
	}
	if home := hermesHome(); home != "" {
		if py, d := fromVenvBin(filepath.Join(home, "hermes-agent", "venv", "bin", "hermes")); py != "" {
			return py, d
		}
	}
	return "", ""
}

// parseHermesCatalog turns Hermes's registry into declaration rows: a key
// provider takes an API key (its first variable is what the pane names), an
// OAuth provider takes a sign-in Hermes runs as a device code. Declared ids,
// aliases and the doors PiCode cannot drive are left out.
func parseHermesCatalog(raw []byte, declared []Provider) []Provider {
	var rows []hermesRow
	if json.Unmarshal(raw, &rows) != nil {
		return nil
	}
	seenID := map[string]bool{}
	for _, p := range declared {
		seenID[p.Provider] = true
	}
	seenName := map[string]bool{}
	var out []Provider
	for _, r := range rows {
		id, name := strings.TrimSpace(r.ID), strings.TrimSpace(r.Name)
		if id == "" || seenID[id] || hermesDeclared[id] || hermesSkipped[r.AuthType] || id == "qwen-oauth" {
			continue
		}
		if name != "" && seenName[name] {
			continue
		}
		seenID[id], seenName[name] = true, true
		p := Provider{Provider: id, Name: name, Native: nativeHermes, Note: noteHermesPool}
		switch {
		case r.AuthType == "api_key":
			p.Kinds = []string{KindAPIKey}
			if len(r.Env) > 0 {
				p.Env = map[string]string{KindAPIKey: r.Env[0]}
			}
			p.Note = ""
		case strings.HasPrefix(r.AuthType, "oauth"):
			p.Kinds = []string{KindOAuth}
		default:
			continue
		}
		out = append(out, p)
	}
	return out
}
