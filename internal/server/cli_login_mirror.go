package server

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/cfpperche/picode/internal/clicreds"
	"github.com/cfpperche/picode/internal/credentials"
	"github.com/cfpperche/picode/internal/usage"
)

// A CLI that keeps a nameless OAuth login in its own file (Claude Code's
// subscription) and the vault row that stands for it are one login: the
// file is what the CLI uses, so the row mirrors it (credentials.Mirror), and
// the row's refresh token is the CLI's to spend, never PiCode's — renewing
// it here would rotate the token out from under the CLI (measured 2026-09-24:
// the in-use Anthropic row had held a dead token since 09-14 while Claude
// Code's own login worked).

// mirrorCLILogins copies each nameless CLI login for provider into its row.
// pi is not a mirror source: its auth.json is written from the vault's own
// active slot (ADR-0013). The first CLI holding such a login wins, so two
// CLIs never take turns rewriting one row.
func mirrorCLILogins(provider string) {
	for _, spec := range clicreds.Declarations() {
		if spec.CLI == "pi" {
			continue
		}
		login, ok := clicreds.DetectProvider(spec.CLI, provider)
		if !ok || login.Kind != "oauth" || credentials.Exact(login.Cred, login.Identity) {
			continue
		}
		credentials.Default().Mirror(provider, login.Cred)
		return
	}
}

// cliHoldingRefresh names the CLI whose own live login holds this refresh
// token, or "".
func cliHoldingRefresh(provider, refresh string) string {
	if refresh == "" {
		return ""
	}
	for _, spec := range clicreds.Declarations() {
		if spec.CLI == "pi" {
			continue
		}
		login, ok := clicreds.DetectProvider(spec.CLI, provider)
		if !ok || login.Kind != "oauth" {
			continue
		}
		var live struct {
			Refresh string `json:"refresh"`
		}
		if json.Unmarshal(login.Cred, &live) == nil && live.Refresh == refresh {
			return spec.Name
		}
	}
	return ""
}

var usageHooksOnce sync.Once

// installUsageHooks points the production usage client at the mirror and the
// refresh owner once per process.
func installUsageHooks() {
	usageHooksOnce.Do(func() {
		usage.Default.Sync = mirrorCLILogins
		usage.Default.HeldBy = cliHoldingRefresh
	})
}

// refreshHolders lists every CLI — pi included — whose own live login holds
// this refresh token. Two holders of one rotating token sign each other out
// on the first renewal (ADR-0166 amendment 2026-09-25).
func refreshHolders(provider, refresh string) []clicreds.Spec {
	if refresh == "" {
		return nil
	}
	var out []clicreds.Spec
	for _, spec := range clicreds.Declarations() {
		login, ok := clicreds.DetectProvider(spec.CLI, provider)
		if !ok || login.Kind != "oauth" {
			continue
		}
		var live struct {
			Refresh string `json:"refresh"`
		}
		if json.Unmarshal(login.Cred, &live) == nil && live.Refresh == refresh {
			out = append(out, spec)
		}
	}
	return out
}

// refreshOf is an OAuth credential's refresh token ("" for anything else).
func refreshOf(cred json.RawMessage) string {
	var c struct {
		Type    string `json:"type"`
		Refresh string `json:"refresh"`
	}
	if json.Unmarshal(cred, &c) != nil || c.Type != "oauth" {
		return ""
	}
	return c.Refresh
}

// sharedLoginHolder names another CLI that already holds this row's refresh
// token live, which Use for target would duplicate; "" when Use is safe.
func sharedLoginHolder(provider string, cred json.RawMessage, target string) (cli, name string) {
	for _, spec := range refreshHolders(provider, refreshOf(cred)) {
		if spec.CLI != target {
			return spec.CLI, spec.Name
		}
	}
	return "", ""
}

// refuseSharedLogin writes the 409 Use answers when the login is another
// CLI's: the pane offers a separate sign-in instead.
func refuseSharedLogin(w http.ResponseWriter, targetName, holderCLI, holderName string) {
	writeJSON(w, http.StatusConflict, map[string]any{
		"error": "This login is " + holderName + "'s. Using it in " + targetName +
			" too would share one renewal, and the first CLI to renew signs the other out. Sign in to " +
			targetName + " separately — the same account can hold two logins.",
		"heldBy":     holderName,
		"heldByCli":  holderCLI,
		"signInHere": true,
	})
}
