package server

import (
	"context"
	"encoding/json"
	"net/http"

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
	mux.HandleFunc("GET /api/terminals/{id}/settings", handleGetTermSettings(deps))
	mux.HandleFunc("PATCH /api/terminals/{id}/settings", handlePatchTermSettings(deps))
}

// scopeBySession maps each terminal's live session name to the store scope
// holding its overrides. Sessions absent from the map — an agent's TUI — take
// the global defaults with nothing layered on top.
//
// The mapping only runs forwards. A session name is the terminal id sanitised
// and clipped to 60 characters, so two ids can produce one name and no id can
// be recovered from a name; deriving it backwards would be a guess.
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

// termOptionsFor resolves the tmux options one session should be running with.
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
func termOptionResolver(deps Deps) func(string) map[string]string {
	return func(session string) map[string]string {
		global, err := deps.Store.TerminalSettings(termopts.GlobalScope)
		if err != nil {
			global = nil
		}
		return termOptionsFor(deps, global, scopeBySession(deps), session)
	}
}

// applyTermOptions pushes a session's resolved options into tmux. Best-effort
// by design: a session that died between the store read and here is not an
// error worth failing a settings save over.
func applyTermOptions(ctx context.Context, deps Deps, session string, opts map[string]string) {
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return
	}
	for key, value := range opts {
		_ = deps.Tmux.SetOption(ctx, session, key, value)
	}
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

// applyTermOptionsEverywhere re-resolves every session this instance owns.
// Used after a global change: each session gets its OWN resolution, so a
// terminal that pins a field keeps its value while the rest follow.
// Blanket-applying the new global instead would silently overwrite exactly the
// overrides the panel just promised to respect.
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
		applyTermOptions(ctx, deps, name, termOptionsFor(deps, global, scopeOf, name))
	}
}

// termSettingsView is the same shape for both scopes so the panel has one code
// path. `inherited` is what this scope falls back to field by field, which is
// what lets the panel show "Inherit (on)" rather than a bare "Inherit".
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

func decodeTermPatch(w http.ResponseWriter, r *http.Request) (map[string]*string, bool) {
	var patch map[string]*string
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return nil, false
	}
	if err := termopts.Validate(patch); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
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
		applyTermOptions(r.Context(), deps, tmux.ShellSessionName(term.ID), termopts.Resolve(inherited, next))
		writeJSON(w, http.StatusOK, termSettingsView(term.ID, next, inherited, &term))
	}
}
