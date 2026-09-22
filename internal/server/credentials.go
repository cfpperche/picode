package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/clicreds"
	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/credentials"
	"github.com/cfpperche/picode/internal/oauth"
	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/usage"
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
	ID    string            `json:"id"`
	Kinds []string          `json:"kinds"`
	Env   map[string]string `json:"env,omitempty"`
	Note  string            `json:"note,omitempty"`
	// Custom marks a provider defined in pi's models.json (ADR-0129): the pane
	// offers Edit provider for it, not only Add.
	Custom bool `json:"custom,omitempty"`
	// Definition carries a custom provider's editable shape for the CLIs
	// whose definitions ride this roster (omp's models.yml; pi's Edit reads
	// /api/catalog). The key never travels — Keyed inside the row is the only
	// statement about it.
	Definition *catalog.CustomRow `json:"definition,omitempty"`
	// Verify is how this CLI answers "does this credential still work":
	// verifyByProvider for pi, whose own check answers for a whole provider, or
	// verifyByRow for a guest CLI, whose check is a listing call with one row's
	// key. Empty where the CLI has no honest check for that provider, so the
	// pane offers no Verify rather than one that always fails.
	Verify   string        `json:"verify,omitempty"`
	Native   *nativeView   `json:"native,omitempty"`
	Accounts []accountView `json:"accounts"`
	// SingleOAuth says this CLI's login carries no account name, so the vault
	// keeps one subscription row per provider here (ADR-0013's rule). The pane
	// says so instead of letting a second import look like it vanished.
	SingleOAuth bool `json:"singleOAuth,omitempty"`
}

// The two ways a CLI can check a credential (providerView.Verify).
const (
	verifyByProvider = "provider"
	verifyByRow      = "row"
)

// accountView is one vault row as this CLI sees it. Active means "the file
// this CLI reads holds this account right now": the file is the truth, so a
// login made in the CLI's own TUI shows up here without PiCode remembering a
// click. Activatable says whether Use could write it (ADR-0166) — a row whose
// shape this CLI's file cannot hold offers no control instead of one that
// always fails.
type accountView struct {
	catalog.Account
	Activatable bool `json:"activatable"`
	// Usage is the cached report for this row. The key is absent when the
	// cache holds none: a pane load never calls a vendor, and the pane renders
	// "unknown" with Check instead of a guessed bar (ADR-0031).
	Usage *usage.Entry `json:"usage,omitempty"`
}

type nativeView struct {
	Detected bool   `json:"detected"`
	Kind     string `json:"kind,omitempty"`
	Label    string `json:"label,omitempty"`
	Imported bool   `json:"imported,omitempty"`
}

// rosterSource is one provider a CLI can use, from wherever that CLI keeps its
// list: pi's catalog (the /login set, the models it lists and the models.json
// definitions) or a guest CLI's clicreds declaration.
type rosterSource struct {
	ID          string
	Kinds       []string
	Env         map[string]string
	Note        string
	Custom      bool
	Native      *clicreds.Native
	Verify      string
	SingleOAuth bool
}

// handleCredentials answers the roster for one CLI: which providers it can
// read, and which saved accounts exist for each. The vault's own state is
// reported honestly — a locked vault says so instead of looking empty.
//
// One roster for every CLI (ADR-0169): pi's providers come from the catalog,
// a guest's from its declaration, and both carry the same rows with the cached
// usage report each one has.
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
		var sources []rosterSource
		if spec.CLI == "pi" {
			fromCatalog, err := catalogSources(deps)
			if err != nil {
				writeErr(w, http.StatusServiceUnavailable, err.Error())
				return
			}
			sources = fromCatalog
			// The bar's primary opens pi's own dialog — OAuth or a key — and
			// the custom-endpoint page hangs off this pane (ADR-0169).
			out["add"] = map[string]any{"kind": "provider", "label": "Add provider"}
			out["custom"] = map[string]any{"available": true, "href": "#/clis/pi/providers/custom"}
		} else {
			sources = declarationSources(spec)
			out["add"] = map[string]any{"kind": "key", "label": "Add provider"}
			// omp keeps provider definitions of its own in models.yml (the
			// owner's amendment to ADR-0169): the same custom door pi has,
			// with the definitions riding this roster as rows below.
			if spec.CLI == "omp" {
				out["custom"] = map[string]any{"available": true, "href": "#/clis/omp/providers/custom"}
			}
		}
		providers := providerViews(spec, sources)
		if spec.CLI == "omp" {
			providers = appendOMPDefinitions(providers)
		}
		attachRowUsage(providers, time.Now())
		out["providers"] = providers
		if spec.Login != nil {
			out["signin"] = map[string]any{"available": true, "hint": spec.Login.Hint}
		} else {
			out["signin"] = map[string]any{"available": false}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// catalogSources is pi's provider list, read from the catalog (ADR-0169). The
// catalog is the authority on what pi can talk to — its /login set, the models
// it lists, and the custom definitions in models.json; pi's declaration still
// says which credential shapes it takes and which variable each kind is passed
// in for the providers it names, and a provider only the model list knows
// carries the catalog's own answer.
func catalogSources(deps Deps) ([]rosterSource, error) {
	rep, err := loadCatalog(deps)
	if err != nil {
		return nil, err
	}
	spec, _ := clicreds.For("pi")
	out := make([]rosterSource, 0, len(rep.Providers))
	for _, p := range rep.Providers {
		s := rosterSource{ID: p.ID, Custom: p.Custom, Verify: verifyByProvider}
		if d := declaredProvider(spec, p.ID); d != nil {
			s.Kinds, s.Env, s.Note, s.Native = d.Kinds, d.Env, d.Note, d.Native
			s.SingleOAuth = d.Native != nil && !clicreds.IdentityBearing(d.Native.Format)
		} else {
			s.Kinds = kindsOfLogin(p.Login)
			s.Env = envOfProvider(p.ID)
		}
		out = append(out, s)
	}
	return out, nil
}

// declarationSources is a guest CLI's provider list: its own declaration, in
// declaration order, with the check that CLI can actually run.
func declarationSources(spec clicreds.Spec) []rosterSource {
	out := make([]rosterSource, 0, len(spec.Providers))
	for _, p := range spec.Providers {
		s := rosterSource{
			ID: p.Provider, Kinds: p.Kinds, Env: p.Env, Note: p.Note, Native: p.Native,
			SingleOAuth: p.Native != nil && !clicreds.IdentityBearing(p.Native.Format),
		}
		if probeFor(p.Provider) != nil {
			s.Verify = verifyByRow
		}
		out = append(out, s)
	}
	return out
}

// appendOMPDefinitions appends omp's models.yml definitions as roster rows,
// after the declared ones, in id order (the file's order is the map's secret;
// a pane that reshuffles between loads reads as a bug). The row carries the
// editable shape and the keyed flag — never the key.
func appendOMPDefinitions(providers []providerView) []providerView {
	defs := catalog.OMPLoadCustomDefinitions()
	ids := make([]string, 0, len(defs))
	for id := range defs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		row := defs[id]
		providers = append(providers, providerView{
			ID: id, Kinds: []string{clicreds.KindAPIKey}, Custom: true,
			Accounts: []accountView{}, Definition: &row,
		})
	}
	return providers
}

// providerViews turns a provider list into the roster's rows, reading the
// CLI's own credential file once per provider: it decides whether Use could
// write each row, and whether the CLI is already signed in by itself.
func providerViews(spec clicreds.Spec, sources []rosterSource) []providerView {
	isPi := spec.CLI == "pi"
	out := make([]providerView, 0, len(sources))
	for _, s := range sources {
		var existing []byte
		if path := clicreds.CredentialPath(spec.CLI, s.ID); path != "" {
			existing, _ = os.ReadFile(path)
		}
		// The CLI's own store, read once: what it holds decides which row is
		// in use and whether Use could write each row. "Detected" is the
		// store having a login at all; "live" is that login already being one
		// of the rows below.
		login, detected := clicreds.DetectProvider(spec.CLI, s.ID)
		if detected && login.Kind == "oauth" {
			// The vendor may have renewed this login since the vault saw it:
			// pull the renewal in before the rows are read, so the row and
			// the file agree again (no write to the CLI's own file happens
			// here — that is Use, ADR-0166).
			credentials.Default().Harvest(s.ID, login.Cred)
		}
		rows := accountsFor(s.ID)
		liveID := ""
		if detected {
			liveID = liveRowID(s.ID, rows, login)
		}
		views := make([]accountView, 0, len(rows))
		for _, a := range rows {
			view := accountView{Account: a}
			if !isPi {
				// pi's rows are marked in use by the vault's own active slot,
				// which mirrors auth.json (ADR-0013).
				view.Active = liveID != "" && a.ID == liveID
			}
			if s.Native != nil {
				if row, ok, err := credentials.Default().Row(s.ID, a.ID); err == nil && ok {
					_, view.Activatable = clicreds.RenderLogin(s.Native.Format, s.ID, row.Cred, existing)
				}
			}
			views = append(views, view)
		}
		view := providerView{
			ID: s.ID, Kinds: s.Kinds, Env: s.Env, Note: s.Note, Custom: s.Custom,
			Verify: s.Verify, Accounts: views, SingleOAuth: s.SingleOAuth,
		}
		if s.Native != nil && detected {
			view.Native = &nativeView{
				Detected: true, Kind: login.Kind, Label: login.Label,
				Imported: liveID != "",
			}
		}
		out = append(out, view)
	}
	return out
}

// attachRowUsage fills each row's usage from the cache. The rows themselves
// are handed to usage.Summary, so the entry is the JSON /api/providers/usage
// serves, and Lookup decides presence: a row the cache never saw keeps no
// usage key at all. Nothing here reaches a vendor (ADR-0031).
func attachRowUsage(providers []providerView, now time.Time) {
	rows := make([]catalog.Provider, 0, len(providers))
	for _, p := range providers {
		if len(p.Accounts) == 0 {
			continue
		}
		signed := catalog.Provider{ID: p.ID, SignedIn: true}
		for _, a := range p.Accounts {
			signed.Accounts = append(signed.Accounts, a.Account)
		}
		rows = append(rows, signed)
	}
	index := map[[2]string]usage.Entry{}
	for _, e := range usage.Summary(rows, now) {
		if _, _, cached := usage.Lookup(e.Provider, e.AccountID); !cached {
			continue
		}
		index[[2]string{e.Provider, e.AccountID}] = e
	}
	for i := range providers {
		for j := range providers[i].Accounts {
			a := &providers[i].Accounts[j]
			e, ok := index[[2]string{providers[i].ID, a.ID}]
			if !ok {
				continue
			}
			entry := e
			a.Usage = &entry
		}
	}
}

// declaredProvider is the CLI's own row for a provider, where it declares one.
func declaredProvider(spec clicreds.Spec, provider string) *clicreds.Provider {
	for i := range spec.Providers {
		if spec.Providers[i].Provider == provider {
			return &spec.Providers[i]
		}
	}
	return nil
}

// kindsOfLogin is the catalog's /login method as the roster's credential
// shapes.
func kindsOfLogin(login string) []string {
	switch login {
	case catalog.LoginOAuth:
		return []string{clicreds.KindOAuth}
	case catalog.LoginBoth:
		return []string{clicreds.KindAPIKey, clicreds.KindOAuth}
	default:
		return []string{clicreds.KindAPIKey}
	}
}

// envOfProvider is the variable pi reads for a provider it never declared.
// The catalog lists pi's names in the order pi reads them; anthropic is the
// only provider with more than one, and pi declares it.
func envOfProvider(provider string) map[string]string {
	names := catalog.APIKeyEnvVars[provider]
	if len(names) == 0 {
		return nil
	}
	return map[string]string{clicreds.KindAPIKey: names[0]}
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
			// Keep asks the handler to keep the login that is already saved
			// (named from its own known identity) and file the new login as
			// its own row — "keep both", the pane's second door.
			Keep    bool `json:"keep"`
			Preview bool `json:"preview"`
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
		// What this import would do, before doing it: the pane asks the
		// person to name a login BEFORE the write when the write would
		// replace an unnamed one — after the write, the replaced tokens are
		// already gone (2026-09-21).
		if req.Preview {
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
			id := credentials.Key(login.Cred, identity)[:12]
			existing, found, err := credentials.Default().Row(login.Provider, id)
			if err != nil {
				writeVaultErr(w, err)
				return
			}
			out := map[string]any{
				"preview": true,
				"created": !found,
				"who":     who,
				"stamp":   tokenStamp(login.Cred),
			}
			if identity != "" {
				out["identity"] = identity
			}
			if found {
				out["existing"] = rowView(existing)
			}
			writeJSON(w, http.StatusOK, out)
			return
		}
		// What this login is called, in the order that keeps rows honest: the
		// name the person gave, the account the store itself names, else the
		// account the vendor's own profile endpoint answers with on this
		// explicit click. An unnamed login is one row per provider, and the
		// pane says so.
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
		// Keep both: the login already saved is named from what the vault
		// knows about it (its email, else its label), and the new login
		// lands as its own unnamed row — the roster then shows both, with
		// the live file matching the new one.
		if req.Keep {
			id := credentials.Key(login.Cred, "")[:12]
			old, found, err := credentials.Default().Row(login.Provider, id)
			if err != nil {
				writeVaultErr(w, err)
				return
			}
			if found {
				oldName := old.Email
				if oldName == "" {
					oldName = old.Label
				}
				if isDefaultLabel(oldName) || oldName == "" {
					oldName = "Previous login"
				}
				if _, err := credentials.Default().Adopt(login.Provider, oldName, old.Cred); err != nil {
					writeVaultErr(w, err)
					return
				}
				if err := catalog.RenameAccount(login.Provider, credentials.Key(old.Cred, oldName)[:12], oldName); err != nil {
					writeVaultErr(w, err)
					return
				}
			}
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
		// The name the person typed is what the row shows — not the generic
		// label the counter handed out.
		if strings.TrimSpace(req.As) != "" && isDefaultLabel(row.Label) {
			if err := catalog.RenameAccount(login.Provider, row.ID, strings.TrimSpace(req.As)); err != nil {
				writeVaultErr(w, err)
				return
			}
			row.Label = strings.TrimSpace(req.As)
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
		// A stamp of the access token, not the token: Check now must tell a
		// finished sign-in from the account that was already in the file.
		// Nameless stores (Claude Code) keep one row id either way.
		out["stamp"] = tokenStamp(login.Cred)
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

// isDefaultLabel reports the labels the counter hands out ("Default",
// "Account 2", …): a named login replaces them, a person's own label stays.
func isDefaultLabel(label string) bool {
	label = strings.TrimSpace(label)
	if label == "" || label == "Default" {
		return true
	}
	return defaultLabelRe.MatchString(label)
}

var defaultLabelRe = regexp.MustCompile(`^Account \d+$`)

// accessTokenOf reads the OAuth access token out of a vault credential, for
// the callers that need to ask a vendor who the account is.
// tokenStamp is a short hash of the access token. It is safe to send to
// the pane: it cannot be turned back into the token, and it changes exactly
// when the CLI's file starts holding a different login.
func tokenStamp(cred json.RawMessage) string {
	access := accessTokenOf(cred)
	if access == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(access))
	return hex.EncodeToString(sum[:8])
}

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
			CLI      string `json:"cli"`
			Provider string `json:"provider"`
		}
		if !readCLIJSON(w, r, &req) {
			return
		}
		spec, ok := clicreds.For(strings.TrimSpace(req.CLI))
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		// An omp provider the OAuth engine can sign in (ADR-0176) runs the
		// browser flow from here — the authorize page opens in a tab, the
		// loopback/device callback returns to PiCode, and the minted
		// credential lands in the vault, from which omp reads it through its
		// declared env names. The terminal strip stays for everything else.
		if req.CLI == "omp" && strings.TrimSpace(req.Provider) != "" && oauth.Supports(strings.TrimSpace(req.Provider)) {
			provider := strings.TrimSpace(req.Provider)
			url, userCode, err := oauth.StartSink(provider, "", ompVaultSink)
			if err != nil {
				writeErr(w, http.StatusConflict, err.Error())
				return
			}
			out := map[string]any{"cli": req.CLI, "oauth": true, "url": url}
			if userCode != "" {
				out["userCode"] = userCode
			}
			writeJSON(w, http.StatusOK, out)
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
		// One sign-in terminal per CLI. A second click must lead back to the
		// terminal that is already waiting — thirteen "Claude Code sign-in"
		// sessions piled up in one machine's tmux before this check existed
		// (2026-09-21). A record whose session is gone is a husk: remove it so
		// the terminals list does not fill with sign-in ghosts.
		want := spec.Name + " sign-in"
		if rows, err := deps.Store.ListTerminals(); err == nil {
			for _, t := range rows {
				if t.Name != want {
					continue
				}
				alive, e := deps.Tmux.HasSession(r.Context(), tmux.ShellSessionName(t.ID))
				if e == nil && alive {
					writeJSON(w, http.StatusOK, map[string]any{
						"terminalId": t.ID, "hint": spec.Login.Hint, "reused": true,
					})
					return
				}
				_ = deps.Store.DeleteTerminal(t.ID)
			}
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
		prior := ""
		if login, found := clicreds.Detect(spec.CLI); found {
			prior = tokenStamp(login.Cred)
		}
		writeJSON(w, status, map[string]any{
			"terminalId": t.ID, "hint": spec.Login.Hint, "terminal": view, "stamp": prior,
		})
	}
}

// ompVaultSink lands a minted OAuth credential in the vault as the
// provider's own row — the same shape the pi flow writes to auth.json
// (ADR-0176). Origin "vault"; the roster shows it like any other account.
func ompVaultSink(provider string, cred map[string]any) error {
	raw, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	_, err = credentials.Default().Import(provider, raw, "", "vault", "")
	return err
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
		decl := declaredProvider(spec, provider)
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
