package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"crypto/rand"
	"encoding/hex"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/termopts"
	"github.com/cfpperche/picode/internal/tmux"
)

func registerTerminalSettingsRoutes(mux *http.ServeMux, deps Deps) {
	// The literal segment wins over {id} in Go's mux, so "settings" is not a
	// terminal named "settings" — but a terminal *is* free to be called that,
	// which is why the per-terminal panel lives one level deeper.
	mux.HandleFunc("GET /api/terminals/settings", handleGetGlobalTermSettings(deps))
	mux.HandleFunc("PATCH /api/terminals/settings", handlePatchGlobalTermSettings(deps))
	mux.HandleFunc("GET /api/terminals/settings/catalog", handleGetTermCatalog(deps))
	mux.HandleFunc("GET /api/terminals/{id}/settings", handleGetTermSettings(deps))
	mux.HandleFunc("PATCH /api/terminals/{id}/settings", handlePatchTermSettings(deps))
}

// scopeBySession maps each terminal's live session name to the store scope
// holding its overrides. Sessions absent from the map — an agent's TUI — take
// the global defaults with nothing layered on top.
func scopeBySession(deps Deps) map[string]string {
	out := map[string]string{}
	list, err := deps.Store.ListTerminals()
	if err != nil {
		return out
	}
	for _, t := range list {
		out[tmux.ShellSessionName(t.ID)] = t.ID
	}
	return out
}

// ownSessions is every session name this instance is responsible for, derived
// from the store: one per terminal, one per agent.
//
// NOT from `tmux list-sessions`. That answers for the whole machine — every
// session carrying the prefix, whoever created it. Applying a global change to
// that list writes into sessions this instance has never heard of, which is
// not a hypothetical: a single `go test` against a throwaway database in /tmp
// flipped `mouse` on the developer's own running terminals, because they wore
// the same prefix. The store is the only honest answer to "mine".
func ownSessions(deps Deps) []string {
	var out []string
	if list, err := deps.Store.ListTerminals(); err == nil {
		for _, t := range list {
			out = append(out, tmux.ShellSessionName(t.ID))
		}
	}
	// Agents have no per-terminal overrides, but their TUI runs in a session
	// that takes the same global defaults, so a change has to reach it too.
	if agents, err := deps.Store.ListAllAgents(); err == nil {
		for _, a := range agents {
			out = append(out, tmux.SessionName(a.ID))
		}
	}
	return out
}

// scopeLookup answers "which tmux scope does this key live at". Curated flags
// know; anything else is looked up in the live catalog, fetched once and only
// when actually needed. Unknown keys land at session scope, where tmux will
// refuse them with its own message.
func scopeLookup(ctx context.Context, deps Deps) func(key string) string {
	var catalog map[string]string // name -> scope, lazily fetched
	return func(key string) string {
		if f, ok := termopts.Find(key); ok {
			return f.Scope
		}
		if catalog == nil {
			catalog = map[string]string{}
			if deps.Tmux != nil && deps.Tmux.Available() {
				if entries, err := deps.Tmux.OptionCatalog(ctx); err == nil {
					for _, e := range entries {
						// First scope wins; a handful of names exist at two
						// scopes and the narrower (earlier-sorted) is server —
						// harmless either way for set-option.
						if _, seen := catalog[e.Name]; !seen {
							catalog[e.Name] = e.Scope
						}
					}
				}
			}
		}
		if s, ok := catalog[key]; ok {
			return s
		}
		return tmux.ScopeSession
	}
}

// resolveScoped turns a resolved key/value map into scoped writes.
func resolveScoped(ctx context.Context, deps Deps, opts map[string]string) []tmux.ScopedValue {
	scopeOf := scopeLookup(ctx, deps)
	out := make([]tmux.ScopedValue, 0, len(opts))
	for k, v := range opts {
		out = append(out, tmux.ScopedValue{Scope: scopeOf(k), Key: k, Value: v})
	}
	return out
}

// termOptionsFor resolves the option map one session should be running with.
func termOptionsFor(deps Deps, global map[string]string, scopeOf map[string]string, session string) map[string]string {
	scope, ok := scopeOf[session]
	if !ok {
		return termopts.Resolve(global)
	}
	over, err := deps.Store.TerminalSettings(scope)
	if err != nil {
		return termopts.Resolve(global)
	}
	return termopts.Resolve(global, over)
}

// termOptionResolver is what the /ws/term bridge applies on every attach. It
// runs per attach, not per keystroke, so reading the store each time is the
// cheap option and always reflects a setting changed since the last one.
func termOptionResolver(deps Deps) func(session string) []tmux.ScopedValue {
	return func(session string) []tmux.ScopedValue {
		ctx := context.Background()
		global, err := deps.Store.TerminalSettings(termopts.GlobalScope)
		if err != nil {
			global = nil
		}
		return resolveScoped(ctx, deps, termOptionsFor(deps, global, scopeBySession(deps), session))
	}
}

// applyScoped pushes resolved options into tmux. Best-effort by design: a
// session that died between the store read and here is not an error worth
// failing a settings save over.
func applyScoped(ctx context.Context, deps Deps, session string, values []tmux.ScopedValue) {
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return
	}
	for _, sv := range values {
		_ = deps.Tmux.SetScopedOption(ctx, sv.Scope, session, sv.Key, sv.Value)
	}
}

// applyTermOptionsEverywhere re-resolves every session this instance owns.
// Used after a global change: each session gets its OWN resolution, so a
// terminal that pins a field keeps its value while the rest follow.
func applyTermOptionsEverywhere(ctx context.Context, deps Deps) {
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return
	}
	global, err := deps.Store.TerminalSettings(termopts.GlobalScope)
	if err != nil {
		return
	}
	scopeOf := scopeBySession(deps)
	for _, name := range ownSessions(deps) {
		// A record whose session is not running is skipped rather than
		// created: settings are not a reason to start a terminal.
		if live, err := deps.Tmux.HasSession(ctx, name); err != nil || !live {
			continue
		}
		applyScoped(ctx, deps, name, resolveScoped(ctx, deps, termOptionsFor(deps, global, scopeOf, name)))
	}
}

// validatePatch checks a patch the two ways it can be checked without lying:
// curated flags by their enum, everything else by (a) existing in the live
// catalog and (b) tmux itself accepting the value on a scratch session. tmux
// is the only honest validator of its own values — a whitelist here would
// drift from whatever tmux is actually installed.
//
// Server-scope values are NOT test-applied: setting one is machine-wide and
// is the action itself, so they are validated by name and applied for real by
// the caller, whose error still reaches the user.
func validatePatch(ctx context.Context, deps Deps, patch map[string]*string) error {
	if err := termopts.ValidateCurated(patch); err != nil {
		return err
	}
	scopeOf := scopeLookup(ctx, deps)
	var scratch string // created on first need, killed by the caller's defer via cleanup func below
	defer func() {
		if scratch != "" {
			_ = deps.Tmux.KillSession(ctx, scratch)
		}
	}()
	for key, val := range patch {
		if _, curated := termopts.Find(key); curated {
			continue
		}
		if deps.Tmux == nil || !deps.Tmux.Available() {
			return fmt.Errorf("need tmux to change terminal settings")
		}
		scope := scopeOf(key)
		if !knownToCatalog(ctx, deps, key) {
			return fmt.Errorf("%q is not an option this tmux knows", key)
		}
		if val == nil || scope == tmux.ScopeServer {
			continue
		}
		if scratch == "" {
			scratch = tmux.SessionName("optcheck-" + randomSuffix())
			if err := deps.Tmux.NewSession(ctx, scratch, "/", "cat"); err != nil {
				return fmt.Errorf("cannot validate the value: %v", err)
			}
		}
		if err := deps.Tmux.SetScopedOption(ctx, scope, scratch, key, *val); err != nil {
			return fmt.Errorf("tmux refused %s: %s", key, tmuxWords(err))
		}
	}
	return nil
}

func knownToCatalog(ctx context.Context, deps Deps, key string) bool {
	entries, err := deps.Tmux.OptionCatalog(ctx)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.Name == key {
			return true
		}
	}
	return false
}

// applyPatchLive pushes a validated, stored patch at one session, including
// the part storing cannot express: a cleared key with no default under it has
// to be UNSET on the live session, or it keeps the old value forever.
func applyPatchLive(ctx context.Context, deps Deps, session string, patch map[string]*string, resolved map[string]string) {
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return
	}
	scopeOf := scopeLookup(ctx, deps)
	for key, val := range patch {
		if val == nil {
			if _, still := resolved[key]; !still {
				_ = deps.Tmux.UnsetScopedOption(ctx, scopeOf(key), session, key)
			}
			continue
		}
	}
	applyScoped(ctx, deps, session, resolveScoped(ctx, deps, resolved))
}

// termSettingsView is the same shape for both scopes so the panel has one
// code path. `inherited` is what this scope falls back to field by field.
func termSettingsView(scope string, values, inherited map[string]string, term *store.Terminal) map[string]any {
	out := map[string]any{
		"scope":     scope,
		"flags":     termopts.Flags(),
		"defaults":  termopts.Defaults(),
		"values":    values,
		"inherited": inherited,
		"effective": termopts.Resolve(inherited, values),
	}
	if term != nil {
		out["terminal"] = map[string]any{"id": term.ID, "name": term.Name}
	}
	return out
}

// handleGetTermCatalog serves the full option space of the running tmux, for
// the settings page: every option, its scope, its current global value, a
// rendering kind, its warning if it has one, and whether a curated flag
// covers it (the page renders those with the rich controls instead).
func handleGetTermCatalog(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, http.StatusServiceUnavailable, "Need tmux to read terminal settings.")
			return
		}
		entries, err := deps.Tmux.OptionCatalog(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		type row struct {
			tmux.CatalogEntry
			Danger  string `json:"danger,omitempty"`
			Curated bool   `json:"curated,omitempty"`
		}
		out := make([]row, 0, len(entries))
		for _, e := range entries {
			_, curated := termopts.Find(e.Name)
			out = append(out, row{CatalogEntry: e, Danger: termopts.DangerFor(e.Name), Curated: curated})
		}
		writeJSON(w, http.StatusOK, map[string]any{"catalog": out})
	}
}

func decodeTermPatch(w http.ResponseWriter, r *http.Request) (map[string]*string, bool) {
	var patch map[string]*string
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return nil, false
	}
	return patch, true
}

func handleGetGlobalTermSettings(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		global, err := deps.Store.TerminalSettings(termopts.GlobalScope)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, termSettingsView(termopts.GlobalScope, global, termopts.Defaults(), nil))
	}
}

func handlePatchGlobalTermSettings(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patch, ok := decodeTermPatch(w, r)
		if !ok {
			return
		}
		if err := validatePatch(r.Context(), deps, patch); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		current, err := deps.Store.TerminalSettings(termopts.GlobalScope)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		next := termopts.Apply(current, patch)
		if err := deps.Store.SetTerminalSettings(termopts.GlobalScope, next); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Server-scope values apply once, machine-wide — that is what the
		// panel's labelling promises. tmux errors here surface to the user.
		scopeOf := scopeLookup(r.Context(), deps)
		for key, val := range patch {
			if scopeOf(key) != tmux.ScopeServer || deps.Tmux == nil || !deps.Tmux.Available() {
				continue
			}
			if val == nil {
				_ = deps.Tmux.UnsetScopedOption(r.Context(), tmux.ScopeServer, "", key)
			} else if err := deps.Tmux.SetScopedOption(r.Context(), tmux.ScopeServer, "", key, *val); err != nil {
				writeErr(w, http.StatusBadRequest, fmt.Sprintf("tmux refused %s: %s", key, tmuxWords(err)))
				return
			}
		}
		applyTermOptionsEverywhere(r.Context(), deps)
		writeJSON(w, http.StatusOK, termSettingsView(termopts.GlobalScope, next, termopts.Defaults(), nil))
	}
}

func handleGetTermSettings(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		global, err := deps.Store.TerminalSettings(termopts.GlobalScope)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		over, err := deps.Store.TerminalSettings(term.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, termSettingsView(term.ID, over, termopts.Resolve(global), &term))
	}
}

func handlePatchTermSettings(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		patch, ok := decodeTermPatch(w, r)
		if !ok {
			return
		}
		// A server-wide option cannot be promised per terminal; the panel
		// offers it only globally and the API holds the same line.
		scopeOf := scopeLookup(r.Context(), deps)
		for key := range patch {
			if scopeOf(key) == tmux.ScopeServer {
				writeErr(w, http.StatusBadRequest, fmt.Sprintf("%s is server-wide — set it in the global panel", key))
				return
			}
		}
		if err := validatePatch(r.Context(), deps, patch); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		current, err := deps.Store.TerminalSettings(term.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		next := termopts.Apply(current, patch)
		if err := deps.Store.SetTerminalSettings(term.ID, next); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		global, err := deps.Store.TerminalSettings(termopts.GlobalScope)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		inherited := termopts.Resolve(global)
		applyPatchLive(r.Context(), deps, tmux.ShellSessionName(term.ID), patch, termopts.Resolve(inherited, next))
		writeJSON(w, http.StatusOK, termSettingsView(term.ID, next, inherited, &term))
	}
}

// randomSuffix keeps scratch validation sessions from colliding when two
// PATCHes race.
func randomSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "0"
	}
	return hex.EncodeToString(b)
}

// tmuxWords strips the run-wrapper noise (the full command line, the scratch
// session's name, "exit status 1") down to what tmux actually said — "unknown
// value: sideways" — which is the part the user can act on.
func tmuxWords(err error) string {
	msg := err.Error()
	if i := strings.LastIndex(msg, "exit status 1: "); i >= 0 {
		msg = msg[i+len("exit status 1: "):]
	}
	return strings.TrimSpace(msg)
}
