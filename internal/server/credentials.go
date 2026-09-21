package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/clicreds"
	"github.com/cfpperche/picode/internal/clilaunch"
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
	mux.HandleFunc("POST /api/credentials/signin", handleCredentialSignin(deps))
	mux.HandleFunc("PATCH /api/credentials/{provider}/{id}", handleCredentialRename(deps))
	mux.HandleFunc("POST /api/credentials/{provider}/{id}/pause", handleCredentialPause(deps))
	mux.HandleFunc("POST /api/credentials/{provider}/{id}/verify", handleCredentialVerify(deps))
	mux.HandleFunc("POST /api/credentials/{provider}/{id}/activate", handleCredentialActivate(deps))
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
	Accounts []accountView     `json:"accounts"`
	// SingleOAuth says this CLI's login carries no account name, so the vault
	// keeps one subscription row per provider here (ADR-0013's rule). The pane
	// says so instead of letting a second import look like it vanished.
	SingleOAuth bool `json:"singleOAuth,omitempty"`
}

// accountView is one vault row as this CLI sees it. Active means "the file
// this CLI reads holds this account right now": the file is the truth, so a
// login made in the CLI's own TUI shows up here without PiCode remembering a
// click. Activatable says whether Use could write it (ADR-0166) — a row whose
// shape this CLI's file cannot hold offers no control instead of one that
// always fails.
type accountView struct {
	catalog.Account
	Activatable bool `json:"activatable"`
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
		isPi := spec.CLI == "pi"
		for _, p := range spec.Providers {
			// The CLI's own file, read once: it decides which row is in use and
			// whether Use could write each row.
			var existing []byte
			if path := clicreds.CredentialPath(spec.CLI, p.Provider); path != "" {
				existing, _ = os.ReadFile(path)
			}
			rows := accountsFor(p.Provider)
			// The CLI's own store, read once: what it holds decides which row
			// is in use and whether Use could write each row. "Detected" is the
			// store having a login at all; "live" is that login already being
			// one of the rows below.
			login, detected := clicreds.DetectProvider(spec.CLI, p.Provider)
			liveID := ""
			if detected {
				liveID = liveRowID(p.Provider, rows, login)
			}
			views := make([]accountView, 0, len(rows))
			for _, a := range rows {
				view := accountView{Account: a}
				if !isPi {
					// pi's roster is its own endpoint, where Active is pi's slot.
					view.Active = liveID != "" && a.ID == liveID
				}
				if p.Native != nil {
					if row, ok, err := credentials.Default().Row(p.Provider, a.ID); err == nil && ok {
						_, view.Activatable = clicreds.RenderLogin(p.Native.Format, p.Provider, row.Cred, existing)
					}
				}
				views = append(views, view)
			}
			view := providerView{
				ID: p.Provider, Kinds: p.Kinds, Env: p.Env, Note: p.Note,
				Verifier:    probeFor(p.Provider) != nil,
				Accounts:    views,
				SingleOAuth: p.Native != nil && !clicreds.IdentityBearing(p.Native.Format),
			}
			if p.Native != nil && detected {
				view.Native = &nativeView{
					Detected: true, Kind: login.Kind, Label: login.Label,
					Imported: liveID != "",
				}
			}
			providers = append(providers, view)
		}
		out["providers"] = providers
		if spec.Login != nil {
			out["signin"] = map[string]any{"available": true, "hint": spec.Login.Hint}
		} else {
			out["signin"] = map[string]any{"available": false}
		}
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
		row, err := credentials.Default().Import(provider, cred, strings.TrimSpace(req.Label), "vault", "")
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
			// As is the person's own name for this login, sent when a second
			// sign-in of a store that carries no account name has to be kept
			// beside the first instead of replacing it (ADR-0166's rule: the
			// vault never silently eats a login).
			As string `json:"as"`
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
		// What this login is called, in the order that keeps rows honest: the
		// name the person gave, the account the store itself names (Grok's
		// principal, Codex's account id), else the account the vendor's own
		// profile endpoint answers with on this explicit click. An unnamed
		// login is one row per provider, and the pane says so.
		who := login.Label
		if who == "" {
			who = deps.usageClient().Identity(r.Context(), login.Provider, accessTokenOf(login.Cred))
		}
		identity := strings.TrimSpace(req.As)
		if identity == "" {
			identity = login.Identity
		}
		if identity == "" {
			identity = who
		}
		before, _ := credentials.Default().Accounts(login.Provider)
		var row credentials.Row
		var err error
		if strings.TrimSpace(req.As) != "" {
			// Naming a login re-keys the row that holds it; it never leaves a
			// copy of the same credential behind.
			row, err = credentials.Default().Adopt(login.Provider, identity, login.Cred)
		} else {
			row, err = credentials.Default().Import(login.Provider, login.Cred, "", "imported:"+cli, identity)
		}
		if err != nil {
			writeVaultErr(w, err)
			return
		}
		if who != "" {
			_ = credentials.Default().SetIdentity(login.Provider, row.ID, who, "")
		}
		// Created means the roster grew. Naming a login re-keys the row that
		// holds it, so an id comparison would call that a new row.
		after, _ := credentials.Default().Accounts(login.Provider)
		created := len(after) > len(before)
		out := rowView(row)
		out["who"] = who
		out["identity"] = row.Identity
		out["created"] = created
		writeJSON(w, http.StatusCreated, out)
	}
}

// liveRowID names the vault row a CLI's own store currently holds — the id the
// pane marks in use, or the empty string when none of the rows is in that file
// (the pane then offers Import). The key is usually enough: the store names its
// account and the roster recomputes the same key with no call. Rows keyed by a
// name the file does not carry (the vendor's profile, or the person's) are
// matched by the token they were saved with — local, exact, and honest when it
// stops matching, which is why nothing here guesses.
func liveRowID(provider string, rows []catalog.Account, login clicreds.Login) string {
	key := credentials.Key(login.Cred, login.Identity)[:12]
	for _, a := range rows {
		if a.ID == key {
			return a.ID
		}
	}
	var live struct {
		Access string `json:"access"`
	}
	if json.Unmarshal(login.Cred, &live) != nil || live.Access == "" {
		return ""
	}
	for _, a := range rows {
		row, ok, err := credentials.Default().Row(provider, a.ID)
		if err != nil || !ok {
			continue
		}
		var saved struct {
			Access string `json:"access"`
		}
		if json.Unmarshal(row.Cred, &saved) != nil || saved.Access == "" {
			continue
		}
		if saved.Access == live.Access {
			return a.ID
		}
	}
	return ""
}

// accessTokenOf reads the OAuth access token out of a vault credential, for
// the callers that need to ask a vendor who the account is.
func accessTokenOf(cred json.RawMessage) string {
	var m struct {
		Access string `json:"access"`
	}
	if json.Unmarshal(cred, &m) != nil {
		return ""
	}
	return strings.TrimSpace(m.Access)
}

// handleCredentialSignin opens a terminal running the CLI's own sign-in (its
// binary, its client id) and answers with the hint to show while it runs.
// PiCode does not perform another tool's OAuth: it opens the door and imports
// the result (ADR-0166's shape, ADR-0167's flow).
func handleCredentialSignin(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CLI string `json:"cli"`
		}
		if !readCLIJSON(w, r, &req) {
			return
		}
		spec, ok := clicreds.For(strings.TrimSpace(req.CLI))
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		if spec.Login == nil {
			writeErr(w, http.StatusBadRequest, spec.Name+" publishes no sign-in PiCode can start — use Import once its own login is done.")
			return
		}
		cli, ok := clilaunch.Find(spec.CLI)
		if !ok || !cli.Launchable() {
			writeErr(w, http.StatusBadRequest, "This CLI cannot be launched here.")
			return
		}
		args := append([]string{}, spec.Login.Args...)
		v := cliTerminalRequest{
			Name:      spec.Name + " sign-in",
			Overrides: clilaunch.Overrides{Args: &args},
		}
		t, view, status, err := createCLITerminal(deps, r, cli, v)
		if err != nil {
			writeErr(w, status, err.Error())
			return
		}
		writeJSON(w, status, map[string]any{
			"terminalId": t.ID, "hint": spec.Login.Hint, "terminal": view,
		})
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

// handleCredentialActivate writes a saved account into the CLI's own
// credential file — the pi model generalized (ADR-0166). It refuses while a
// terminal of that CLI is live, keeps the file it replaces, and never invents
// a shape the CLI's file cannot hold.
func handleCredentialActivate(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CLI string `json:"cli"`
		}
		if !readCLIJSON(w, r, &req) {
			return
		}
		cli := strings.TrimSpace(req.CLI)
		spec, ok := clicreds.For(cli)
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		provider, id := r.PathValue("provider"), r.PathValue("id")
		var decl *clicreds.Provider
		for i := range spec.Providers {
			if spec.Providers[i].Provider == provider {
				decl = &spec.Providers[i]
				break
			}
		}
		if decl == nil {
			writeErr(w, http.StatusBadRequest, spec.Name+" does not use that provider.")
			return
		}
		if decl.Native == nil {
			writeErr(w, http.StatusBadRequest, spec.Name+" keeps its logins where PiCode does not write.")
			return
		}
		row, found, err := credentials.Default().Row(provider, id)
		if err != nil {
			writeVaultErr(w, err)
			return
		}
		if !found {
			writeErr(w, http.StatusNotFound, "Unknown account.")
			return
		}
		if n := liveTerminalsFor(deps, cli); n > 0 {
			writeErr(w, http.StatusConflict, fmt.Sprintf(
				"Close the %d running %s terminal(s) first — writing a login under a running agent can corrupt its session.", n, spec.Name))
			return
		}
		path := clicreds.CredentialPath(cli, provider)
		if path == "" {
			writeErr(w, http.StatusBadRequest, "PiCode does not know where "+spec.Name+" keeps that login.")
			return
		}
		existing, readErr := os.ReadFile(path)
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			writeErr(w, http.StatusInternalServerError, readErr.Error())
			return
		}
		out, ok := clicreds.RenderLogin(decl.Native.Format, provider, row.Cred, existing)
		if !ok {
			writeErr(w, http.StatusBadRequest, "PiCode will not write this login into "+spec.Name+"'s file — it cannot do that faithfully.")
			return
		}
		backup, err := keepCredentialBackup(deps.DataDir, cli, path, existing)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := writeInterceptFile(path, out, 0o600); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "path": path, "backup": backup})
	}
}

// keepCredentialBackup keeps the file PiCode is about to replace, once per
// CLI: the first activation copies it to <DataDir>/credfiles/<cli>-<unix>.bak
// and later activations leave that copy alone, because it is the *original*
// login the person may want back.
func keepCredentialBackup(dataDir, cli, path string, existing []byte) (string, error) {
	if len(existing) == 0 || dataDir == "" {
		return "", nil
	}
	dir := filepath.Join(dataDir, "credfiles")
	matches, _ := filepath.Glob(filepath.Join(dir, cli+"-*.bak"))
	if len(matches) > 0 {
		return matches[0], nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name := filepath.Join(dir, fmt.Sprintf("%s-%d.bak", cli, time.Now().Unix()))
	if err := writeInterceptFile(name, existing, 0o600); err != nil {
		return "", err
	}
	return name, nil
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
