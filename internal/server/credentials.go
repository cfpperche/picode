package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/clicreds"
	"github.com/cfpperche/picode/internal/credentials"
)

// The credential vault's own surface (ADR-0165): every provider account for
// every agent CLI, listed per CLI, plus the three verbs that create or take
// one out of play — import the CLI's own login, add a key, and verify.
//
// The pi provider endpoints (POST /api/providers/…) stay the only writer of
// pi's auth.json slot; nothing here activates a credential for the agent. That
// is deliberate: adding an Anthropic key in Claude Code's pane must not change
// what pi is using.
func registerCredentialRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/credentials", handleCredentials(deps))
	mux.HandleFunc("POST /api/credentials", handleCredentialAdd(deps))
	mux.HandleFunc("POST /api/credentials/import", handleCredentialImport(deps))
	mux.HandleFunc("PATCH /api/credentials/{provider}/{id}", handleCredentialRename(deps))
	mux.HandleFunc("POST /api/credentials/{provider}/{id}/pause", handleCredentialPause(deps))
	mux.HandleFunc("POST /api/credentials/{provider}/{id}/verify", handleCredentialVerify(deps))
	mux.HandleFunc("DELETE /api/credentials/{provider}/{id}", handleCredentialDelete(deps))
}

// providerView is one provider the CLI can use, with the vault rows it holds.
type providerView struct {
	ID       string            `json:"id"`
	Kinds    []string          `json:"kinds"`
	Env      map[string]string `json:"env,omitempty"`
	Note     string            `json:"note,omitempty"`
	Verifier bool              `json:"verifier"`
	Native   *nativeView       `json:"native,omitempty"`
	Accounts []catalog.Account `json:"accounts"`
}

type nativeView struct {
	Detected bool   `json:"detected"`
	Kind     string `json:"kind,omitempty"`
	Label    string `json:"label,omitempty"`
	Imported bool   `json:"imported,omitempty"`
}

// handleCredentials answers the roster for one CLI: which providers it can
// read, and which saved accounts exist for each. The vault's own state is
// reported honestly — a locked vault says so instead of looking empty.
func handleCredentials(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli := strings.TrimSpace(r.URL.Query().Get("cli"))
		spec, ok := clicreds.For(cli)
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		out := map[string]any{"cli": spec.CLI, "cliName": spec.Name}
		out["vault"] = vaultState()
		providers := []providerView{}
		for _, p := range spec.Providers {
			view := providerView{
				ID: p.Provider, Kinds: p.Kinds, Env: p.Env, Note: p.Note,
				Verifier: probeFor(p.Provider) != nil,
				Accounts: accountsFor(p.Provider),
			}
			if p.Native != nil {
				nv := &nativeView{}
				if login, found := clicreds.Detect(spec.CLI); found && login.Provider == p.Provider {
					nv.Detected = true
					nv.Kind = login.Kind
					nv.Label = login.Label
					nv.Imported = importedAlready(p.Provider, login.Cred)
				}
				if nv.Detected {
					view.Native = nv
				}
			}
			providers = append(providers, view)
		}
		out["providers"] = providers
		writeJSON(w, http.StatusOK, out)
	}
}

// vaultState reports whether the vault can be read at all.
func vaultState() map[string]any {
	_, err := credentials.Default().Load()
	switch {
	case errors.Is(err, credentials.ErrLocked):
		return map[string]any{"readable": false, "problem": err.Error()}
	case errors.Is(err, credentials.ErrCorrupt):
		return map[string]any{"readable": false, "problem": err.Error()}
	case err != nil:
		return map[string]any{"readable": false, "problem": err.Error()}
	}
	return map[string]any{"readable": true}
}

func accountsFor(provider string) []catalog.Account {
	accounts := catalog.AccountsFor(provider)
	if accounts == nil {
		accounts = []catalog.Account{}
	}
	return accounts
}

// importedAlready reports whether this exact credential is already a row, so
// the pane offers Import once instead of on every load.
func importedAlready(provider string, cred json.RawMessage) bool {
	fp := credentials.Fingerprint(cred)
	for _, a := range accountsFor(provider) {
		if a.ID == fp[:12] {
			return true
		}
	}
	return false
}

func handleCredentialAdd(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Provider string `json:"provider"`
			Label    string `json:"label"`
			Key      string `json:"key"`
		}
		if !readCLIJSON(w, r, &req) {
			return
		}
		provider := strings.TrimSpace(req.Provider)
		key := strings.TrimSpace(req.Key)
		if provider == "" || key == "" {
			writeErr(w, http.StatusBadRequest, "A provider and a key are required.")
			return
		}
		if !knownProvider(provider) {
			writeErr(w, http.StatusBadRequest, "Unknown provider.")
			return
		}
		cred, err := json.Marshal(map[string]string{"type": catalog.LoginAPIKey, "key": key})
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		row, err := credentials.Default().Import(provider, cred, strings.TrimSpace(req.Label), "vault")
		if err != nil {
			writeVaultErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, rowView(row))
	}
}

func handleCredentialImport(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CLI string `json:"cli"`
		}
		if !readCLIJSON(w, r, &req) {
			return
		}
		cli := strings.TrimSpace(req.CLI)
		login, ok := clicreds.Detect(cli)
		if !ok {
			writeErr(w, http.StatusNotFound, "No saved login was found for this CLI on this machine.")
			return
		}
		row, err := credentials.Default().Import(login.Provider, login.Cred, "", "imported:"+cli)
		if err != nil {
			writeVaultErr(w, err)
			return
		}
		if login.Label != "" {
			// The vendor volunteered an identity (an email): store it as the
			// row's identity, never as the user's label.
			_ = credentials.Default().SetIdentity(login.Provider, row.ID, login.Label, "")
		}
		writeJSON(w, http.StatusCreated, rowView(row))
	}
}

func handleCredentialRename(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Label string `json:"label"`
		}
		if !readCLIJSON(w, r, &req) {
			return
		}
		provider, id := r.PathValue("provider"), r.PathValue("id")
		if err := catalog.RenameAccount(provider, id, req.Label); err != nil {
			writeVaultErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id})
	}
}

func handleCredentialPause(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Paused bool `json:"paused"`
		}
		if !readCLIJSON(w, r, &req) {
			return
		}
		if err := catalog.PauseAccount(r.PathValue("provider"), r.PathValue("id"), req.Paused); err != nil {
			writeVaultErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": r.PathValue("id"), "paused": req.Paused})
	}
}

func handleCredentialDelete(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider, id := r.PathValue("provider"), r.PathValue("id")
		if err := catalog.RemoveAccount(provider, id); err != nil {
			writeVaultErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleCredentialVerify spends one listing call against the provider with the
// stored credential — the cheapest honest answer to "does this key still
// work". Nothing else here talks to a vendor, and the result is cached on the
// row with its age.
func handleCredentialVerify(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provider, id := r.PathValue("provider"), r.PathValue("id")
		probe := probeFor(provider)
		if probe == nil {
			writeErr(w, http.StatusBadRequest, "Nothing to verify this provider with yet.")
			return
		}
		row, ok, err := credentials.Default().Row(provider, id)
		if err != nil {
			writeVaultErr(w, err)
			return
		}
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown account.")
			return
		}
		state, message := probe.verify(r.Context(), row.Cred)
		if err := credentials.Default().SetHealth(provider, id, state, message); err != nil {
			writeVaultErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"state": state, "message": message})
	}
}

func writeVaultErr(w http.ResponseWriter, err error) {
	code := http.StatusBadRequest
	if errors.Is(err, credentials.ErrLocked) || errors.Is(err, credentials.ErrCorrupt) {
		code = http.StatusConflict
	}
	writeErr(w, code, err.Error())
}

// rowView is one vault row as the browser sees it: never the credential.
func rowView(row credentials.Row) map[string]any {
	out := map[string]any{
		"id": row.ID, "label": row.Label, "type": row.Type,
		"paused": row.Paused, "origin": row.Origin,
	}
	if row.Hint != "" {
		out["hint"] = row.Hint
	}
	if row.Email != "" {
		out["email"] = row.Email
	}
	if row.Plan != "" {
		out["plan"] = row.Plan
	}
	return out
}

// probe is one provider's "does this credential still work" call. Nothing here
// sends a prompt: every one of these is a listing or a key-info endpoint, the
// same class of call ADR-0129 allowed and ADR-0031 already makes for quota.
type probe struct {
	url    string
	header func(key string) (string, string)
	bearer bool
	query  bool
}

func (p *probe) verify(ctx context.Context, cred json.RawMessage) (string, string) {
	var m struct {
		Type string `json:"type"`
		Key  string `json:"key"`
	}
	if err := json.Unmarshal(cred, &m); err != nil || strings.TrimSpace(m.Key) == "" {
		return "unknown", "Only API keys can be verified here."
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	reqURL := p.url
	if p.query {
		reqURL = p.url + "?key=" + url.QueryEscape(m.Key)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "unknown", err.Error()
	}
	req.Header.Set("Accept", "application/json")
	if !p.query {
		k, v := p.header(m.Key)
		req.Header.Set(k, v)
	}
	resp, err := probeClient.Do(req)
	if err != nil {
		return "unknown", "Could not reach the provider: " + err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	switch {
	case resp.StatusCode == http.StatusOK:
		return "ok", ""
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return "invalid", "The provider refused this key (" + fmt.Sprint(resp.StatusCode) + ")."
	case resp.StatusCode == http.StatusPaymentRequired:
		return "no_credit", "The provider says this account has no credit left."
	case resp.StatusCode == http.StatusTooManyRequests:
		return "rate_limited", "The provider is rate limiting this key right now."
	default:
		return "unknown", fmt.Sprintf("%s answered %d: %s", probeHost(reqURL), resp.StatusCode, clipMessage(string(body)))
	}
}

func clipMessage(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

func probeHost(raw string) string {
	if u, err := url.Parse(raw); err == nil {
		return u.Host
	}
	return "the provider"
}

var probeClient = &http.Client{Timeout: 12 * time.Second}

func bearer(key string) (string, string)  { return "Authorization", "Bearer " + key }
func xAPIKey(key string) (string, string) { return "x-api-key", key }

// probeTable maps a provider to its listing call. A provider that is not here
// has no honest answer yet, and the pane hides Verify instead of inventing
// one. A package variable so a test can point a provider at a local server.
var probeTable = map[string]*probe{
	"anthropic":      {url: "https://api.anthropic.com/v1/models", header: xAPIKey},
	"openai":         {url: "https://api.openai.com/v1/models", header: bearer},
	"openai-codex":   {url: "https://api.openai.com/v1/models", header: bearer},
	"openrouter":     {url: "https://openrouter.ai/api/v1/key", header: bearer},
	"xai":            {url: "https://api.x.ai/v1/models", header: bearer},
	"google":         {url: "https://generativelanguage.googleapis.com/v1beta/models", query: true},
	"zai":            {url: "https://api.z.ai/api/paas/v4/models", header: bearer},
	"deepseek":       {url: "https://api.deepseek.com/models", header: bearer},
	"mistral":        {url: "https://api.mistral.ai/v1/models", header: bearer},
	"groq":           {url: "https://api.groq.com/openai/v1/models", header: bearer},
	"cerebras":       {url: "https://api.cerebras.ai/v1/models", header: bearer},
	"fireworks":      {url: "https://api.fireworks.ai/inference/v1/models", header: bearer},
	"together":       {url: "https://api.together.xyz/v1/models", header: bearer},
	"nvidia":         {url: "https://integrate.api.nvidia.com/v1/models", header: bearer},
	"huggingface":    {url: "https://huggingface.co/api/whoami-v2", header: bearer},
	"kimi-coding":    {url: "https://api.moonshot.ai/v1/models", header: bearer},
	"moonshot":       {url: "https://api.moonshot.ai/v1/models", header: bearer},
	"minimax":        {url: "https://api.minimax.io/v1/models", header: bearer},
	"github-copilot": {url: "https://api.github.com/user", header: bearer},
}

func probeFor(provider string) *probe {
	return probeTable[strings.ToLower(strings.TrimSpace(provider))]
}

// knownProvider reports whether the vault can hold a row for this provider id:
// pi's /login set plus whatever the CLI declarations add.
func knownProvider(provider string) bool {
	if _, ok := catalog.LoginMethods[strings.ToLower(provider)]; ok {
		return true
	}
	for _, spec := range clicreds.Declarations() {
		for _, p := range spec.Providers {
			if strings.EqualFold(p.Provider, provider) {
				return true
			}
		}
	}
	return false
}

// providerIDs lists every provider the vault can hold a row for, sorted: pi's
// /login set plus the ids the CLI declarations add.
func providerIDs() []string {
	seen := map[string]bool{}
	for id := range catalog.LoginMethods {
		seen[id] = true
	}
	for _, spec := range clicreds.Declarations() {
		for _, p := range spec.Providers {
			seen[p.Provider] = true
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
