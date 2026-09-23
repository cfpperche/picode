package server

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/clicreds"
	"github.com/cfpperche/picode/internal/credentials"
)

// Claude Code's logins through PiCode's GUI (ADR-0187). Claude Code reads a
// subscription from ~/.claude/.credentials.json (written by Use, ADR-0166)
// and an Anthropic Console key from ANTHROPIC_API_KEY — and a key in the
// environment outranks the subscription (measured on 2.1.280: "API-key auth
// precedence active" with both present). So exactly one login is in use: the
// key PiCode injects only when the person chose that key row with Use, and
// choosing the subscription again clears that choice.

// claudeKeySetting holds the vault row of the Console key in use for Claude
// Code, as "<provider>/<row id>"; absent or empty means the subscription (the
// file) is what Claude Code uses.
const claudeKeySetting = "credentials.claude-code.key"

const claudeKeyEnv = "ANTHROPIC_API_KEY"

// claudeKeyInUse returns the key row Claude Code is set to use, when the
// setting names a row that still exists, is not paused, and is a key.
func claudeKeyInUse(deps Deps) (credentials.Row, bool) {
	if deps.Store == nil {
		return credentials.Row{}, false
	}
	v, ok, err := deps.Store.GetSetting(claudeKeySetting)
	if err != nil || !ok || v == "" {
		return credentials.Row{}, false
	}
	provider, id, found := strings.Cut(v, "/")
	if !found || provider == "" || id == "" {
		return credentials.Row{}, false
	}
	row, found, err := credentials.Default().Row(provider, id)
	if err != nil || !found || row.Paused || row.Type != catalog.LoginAPIKey {
		return credentials.Row{}, false
	}
	return row, true
}

// setClaudeKeyInUse records (or, with an empty id, clears) the key row.
func setClaudeKeyInUse(deps Deps, provider, id string) error {
	if deps.Store == nil {
		return errors.New("no store")
	}
	v := ""
	if id != "" {
		v = provider + "/" + id
	}
	return deps.Store.SetSetting(claudeKeySetting, v)
}

// rowKey is the API key a vault key row holds.
func rowKey(row credentials.Row) string {
	var cred struct {
		Key string `json:"key"`
	}
	if json.Unmarshal(row.Cred, &cred) != nil {
		return ""
	}
	return strings.TrimSpace(cred.Key)
}

// claudeCredentialEnv is the env a Claude Code launch gets from the vault:
// the chosen Console key, unless the person's own launch config already sets
// the variable.
func claudeCredentialEnv(deps Deps, have map[string]string) [][2]string {
	if have[claudeKeyEnv] != "" {
		return nil
	}
	// A platform in Claude Code's settings outranks a key (ADR-0189).
	if _, ok := claudePlatformInUse(); ok {
		return nil
	}
	row, ok := claudeKeyInUse(deps)
	if !ok {
		return nil
	}
	key := rowKey(row)
	if key == "" {
		return nil
	}
	return [][2]string{{claudeKeyEnv, key}}
}

// claudeConfigPath is Claude Code's own config file: $CLAUDE_CONFIG_DIR's
// .claude.json when that is set, else ~/.claude.json.
func claudeConfigPath() string {
	if dir := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); dir != "" {
		return filepath.Join(dir, ".claude.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude.json")
}

// approveClaudeKey records the key as approved the way Claude Code does when
// a person answers its "use this API key?" prompt — the key's last 20
// characters in customApiKeyResponses.approved (read from 2.1.280's own
// code: trim().slice(-20)) — so a terminal opened after Use starts on the
// key instead of asking. Every other field of the file is kept as it was.
func approveClaudeKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	path := claudeConfigPath()
	if path == "" {
		return errors.New("PiCode cannot find Claude Code's config file")
	}
	doc := map[string]any{}
	mode := os.FileMode(0o600)
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if len(strings.TrimSpace(string(raw))) > 0 {
			if err := json.Unmarshal(raw, &doc); err != nil {
				return errors.New("Claude Code's config file is not JSON PiCode can edit safely")
			}
		}
		if info, statErr := os.Stat(path); statErr == nil {
			mode = info.Mode().Perm()
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return err
	}
	tail := key
	if len(tail) > 20 {
		tail = tail[len(tail)-20:]
	}
	responses, _ := doc["customApiKeyResponses"].(map[string]any)
	if responses == nil {
		responses = map[string]any{}
	}
	approved := stringList(responses["approved"])
	rejected := stringList(responses["rejected"])
	if !containsString(approved, tail) {
		approved = append(approved, tail)
	}
	kept := rejected[:0]
	for _, r := range rejected {
		if r != tail {
			kept = append(kept, r)
		}
	}
	responses["approved"] = approved
	responses["rejected"] = kept
	doc["customApiKeyResponses"] = responses
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return writeInterceptFile(path, append(out, '\n'), mode)
}

func stringList(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, it := range items {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// markClaudeInUse corrects the roster's "in use" for Claude Code: a chosen
// key is what Claude Code runs on (it outranks the file), so that row is the
// active one and no subscription row is; key rows can always be chosen.
func markClaudeInUse(deps Deps, providers []providerView) {
	key, keyed := claudeKeyInUse(deps)
	// A platform outranks both (ADR-0189): no row is what Claude Code runs on.
	_, platform := claudePlatformInUse()
	for i := range providers {
		for j := range providers[i].Accounts {
			a := &providers[i].Accounts[j]
			if a.Type == catalog.LoginAPIKey {
				a.Activatable = !a.Paused
				a.Active = !platform && keyed && a.ID == key.ID && providers[i].ID == claudeKeyProvider(deps)
				continue
			}
			if keyed || platform {
				a.Active = false
			}
		}
	}
}

// claudeKeyProvider is the provider half of the key setting.
func claudeKeyProvider(deps Deps) string {
	if deps.Store == nil {
		return ""
	}
	v, _, _ := deps.Store.GetSetting(claudeKeySetting)
	provider, _, _ := strings.Cut(v, "/")
	return provider
}

// claudeVaultSink files a browser sign-in for Claude Code: the minted
// subscription lands in the vault, and when no Claude Code terminal is
// running it is written into Claude Code's own file and becomes the login in
// use — the GUI equivalent of finishing /login. With terminals running it
// waits in the vault for Use, which refuses to write under a live agent.
func claudeVaultSink(deps Deps) func(provider string, cred map[string]any) error {
	return func(provider string, cred map[string]any) error {
		raw, err := json.Marshal(cred)
		if err != nil {
			return err
		}
		row, err := credentials.Default().Import(provider, raw, "", "vault", "")
		if err != nil {
			return err
		}
		if liveTerminalsFor(deps, "claude-code") > 0 {
			return nil
		}
		return activateClaudeSubscription(deps, provider, row)
	}
}

// activateClaudeSubscription writes a vault subscription into Claude Code's
// file and makes it the login in use.
func activateClaudeSubscription(deps Deps, provider string, row credentials.Row) error {
	spec, ok := clicreds.For("claude-code")
	if !ok {
		return errors.New("Claude Code is not declared")
	}
	decl := declaredProvider(spec, provider)
	if decl == nil || decl.Native == nil {
		return errors.New("Claude Code does not keep that login")
	}
	_, _, _, err := writeCLILogin(deps, spec, decl, provider, row)
	return err
}
