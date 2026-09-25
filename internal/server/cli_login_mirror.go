package server

import (
	"encoding/json"
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
