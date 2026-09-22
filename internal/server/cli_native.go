package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/climemory"
	"github.com/cfpperche/picode/internal/climodels"
	"github.com/cfpperche/picode/internal/clisettings"
	"github.com/cfpperche/picode/internal/store"
)

// Native settings and native memory for the guest agent CLIs (ADR-0163).
// Pi is not routed here: it keeps `/api/pi-settings`, its own layers and its
// own trust rules (ADR-0101). Memory does answer for Pi, because the honest
// answer there is "Pi has none" and the pane needs to say it.

func registerCLINativeRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/cli-settings", handleCLISettingsGet(deps))
	mux.HandleFunc("PATCH /api/cli-settings", handleCLISettingsPatch(deps))
	mux.HandleFunc("GET /api/cli-models", handleCLIModelsGet(deps))
	mux.HandleFunc("GET /api/cli-memory", handleCLIMemoryGet(deps))
	mux.HandleFunc("GET /api/cli-memory/item", handleCLIMemoryRead(deps))
	mux.HandleFunc("PUT /api/cli-memory/item", handleCLIMemoryWrite(deps))
	mux.HandleFunc("DELETE /api/cli-memory/item", handleCLIMemoryDelete(deps))
}

// nativePaths resolves the workspace folder these drivers read and write. A
// stale workspace id drops the workspace layer rather than failing the pane,
// matching how connectors behave (ADR-0150).
func nativePaths(deps Deps, workspaceID string) (string, error) {
	if workspaceID == "" || deps.Store == nil {
		return "", nil
	}
	ws, err := deps.Store.GetWorkspace(workspaceID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	return ws.Path, nil
}

func handleCLISettingsGet(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli := r.URL.Query().Get("cli")
		if clisettings.For(cli) == nil {
			writeErr(w, http.StatusBadRequest, "PiCode has no settings editor for "+cliLabel(cli))
			return
		}
		cwd, err := nativePaths(deps, r.URL.Query().Get("workspace"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		rep, err := clisettings.Read(cli, clisettings.Paths{Cwd: cwd})
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

// handleCLIModelsGet asks a CLI which models it can reach, for the role
// matrix's picker (ADR-0181). It runs the vendor's own read-only command in the
// workspace being edited, because the answer depends on the directory: a
// project that disables a provider gets a shorter list than the folder next to
// it (measured 2026-09-22, omp 18.2.8). The pane calls this when a picker
// opens, never on mount — the probe costs a subprocess.
func handleCLIModelsGet(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli := r.URL.Query().Get("cli")
		if !climodels.Supports(cli) {
			writeErr(w, http.StatusBadRequest, "PiCode cannot ask "+cliLabel(cli)+" which models it has")
			return
		}
		cwd, err := nativePaths(deps, r.URL.Query().Get("workspace"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		rep, err := climodels.Read(r.Context(), cli, cwd)
		if err != nil {
			// The vendor's own words, not a status code: a CLI that is not
			// installed, not signed in, or slow says so differently each time.
			writeErr(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

type cliSettingsPatchReq struct {
	CLI         string         `json:"cli"`
	WorkspaceID string         `json:"workspaceId"`
	Scope       string         `json:"scope"`
	Set         map[string]any `json:"set"`
	Reset       []string       `json:"reset"`
	Revision    string         `json:"revision"`
	Force       bool           `json:"force"`
}

func handleCLISettingsPatch(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req cliSettingsPatchReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if clisettings.For(req.CLI) == nil {
			writeErr(w, http.StatusBadRequest, "PiCode has no settings editor for "+cliLabel(req.CLI))
			return
		}
		cwd, err := nativePaths(deps, req.WorkspaceID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		p := clisettings.Paths{Cwd: cwd}
		err = clisettings.Apply(req.CLI, p, clisettings.Patch{
			Scope:    req.Scope,
			Set:      req.Set,
			Reset:    req.Reset,
			Revision: req.Revision,
			Force:    req.Force,
		})
		switch {
		case errors.Is(err, clisettings.ErrStale):
			// The CLI writes this file too. 409 is the pane's cue to re-read
			// and show what changed before offering Replace.
			writeErr(w, http.StatusConflict, err.Error())
			return
		case err != nil && clisettings.IsParseError(err):
			writeErr(w, http.StatusConflict, err.Error())
			return
		case err != nil:
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		// The file stays authoritative; this only invalidates open views.
		if deps.Feed != nil {
			deps.Feed.Ephemeral("cli.settings", map[string]any{"cli": req.CLI, "scope": req.Scope})
		}
		rep, err := clisettings.Read(req.CLI, p)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

// memoryPaths wires the memory driver to the settings driver, so a store the
// user moved in their CLI's own settings is found where they moved it.
func memoryPaths(cwd string) climemory.Paths {
	return climemory.Paths{
		Cwd: cwd,
		Setting: func(cli, key string) (string, bool) {
			// The same workspace the memory request names, so a key set in the
			// workspace layer is the one that wins here too.
			rep, err := clisettings.Read(cli, clisettings.Paths{Cwd: cwd})
			if err != nil {
				return "", false
			}
			for i := len(rep.Layers) - 1; i >= 0; i-- {
				if v, ok := rep.Layers[i].Values[key]; ok {
					if s, isString := v.(string); isString {
						return s, true
					}
				}
			}
			return "", false
		},
	}
}

func handleCLIMemoryGet(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli := r.URL.Query().Get("cli")
		if climemory.For(cli) == nil {
			writeErr(w, http.StatusBadRequest, "PiCode has no memory view for "+cliLabel(cli))
			return
		}
		cwd, err := nativePaths(deps, r.URL.Query().Get("workspace"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		p := memoryPaths(cwd)
		rep, err := climemory.Describe(cli, p)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		body := map[string]any{"report": rep}
		// One store's items come back with the report so the pane paints in a
		// single round trip; `scope` picks which.
		scope := r.URL.Query().Get("scope")
		if scope == "" {
			for _, st := range rep.Stores {
				if st.Resolved && st.Exists {
					scope = st.Scope
					break
				}
			}
		}
		if scope != "" {
			body["scope"] = scope
			items, audit, err := climemory.Survey(cli, p, scope)
			if err == nil {
				body["items"] = items
				// The index the CLI loads every session, measured against the
				// limits that CLI applies — a reader cannot see them anywhere
				// else (2026-09-20).
				if audit.Index.Exists {
					body["index"] = audit.Index
				}
				// A store the CLI keeps under version control answers when its
				// content actually changed; Codex is the one that does.
				if audit.History != nil {
					body["history"] = audit.History
				}
			} else {
				body["itemsError"] = err.Error()
			}
		}
		// The pane warns before writing into a store whose CLI is mid-turn.
		body["running"] = runningTerminalsFor(deps, cli)
		writeJSON(w, http.StatusOK, body)
	}
}

func handleCLIMemoryRead(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cli := q.Get("cli")
		if climemory.For(cli) == nil {
			writeErr(w, http.StatusBadRequest, "PiCode has no memory view for "+cliLabel(cli))
			return
		}
		cwd, err := nativePaths(deps, q.Get("workspace"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		body, err := climemory.Read(cli, memoryPaths(cwd), q.Get("scope"), q.Get("id"))
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": q.Get("id"), "body": body})
	}
}

type cliMemoryWriteReq struct {
	CLI         string `json:"cli"`
	WorkspaceID string `json:"workspaceId"`
	Scope       string `json:"scope"`
	ID          string `json:"id"`
	Body        string `json:"body"`
}

func handleCLIMemoryWrite(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req cliMemoryWriteReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if climemory.For(req.CLI) == nil {
			writeErr(w, http.StatusBadRequest, "PiCode has no memory view for "+cliLabel(req.CLI))
			return
		}
		cwd, err := nativePaths(deps, req.WorkspaceID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		if err := climemory.Write(req.CLI, memoryPaths(cwd), req.Scope, req.ID, req.Body); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		announceMemory(deps, req.CLI)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func handleCLIMemoryDelete(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cli := q.Get("cli")
		if climemory.For(cli) == nil {
			writeErr(w, http.StatusBadRequest, "PiCode has no memory view for "+cliLabel(cli))
			return
		}
		cwd, err := nativePaths(deps, q.Get("workspace"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		if err := climemory.Delete(cli, memoryPaths(cwd), q.Get("scope"), q.Get("id")); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		announceMemory(deps, cli)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

// announceMemory invalidates open views. The event carries the CLI and nothing
// else: memory text never travels on the feed (ADR-0163).
func announceMemory(deps Deps, cli string) {
	if deps.Feed != nil {
		deps.Feed.Ephemeral("cli.memory", map[string]any{"cli": cli})
	}
}

// runningTerminalsFor names the terminals this CLI holds right now, from the
// authoritative presence registry (ADR-0062), so the pane can say who else is
// writing before the reader edits a memory file. Presence, never a guess from
// the terminal's name.
func runningTerminalsFor(deps Deps, cli string) []map[string]string {
	if deps.TermRuntimes == nil {
		return []map[string]string{}
	}
	names := map[string]string{}
	if deps.Store != nil {
		if rows, err := deps.Store.ListTerminals(); err == nil {
			for _, t := range rows {
				names[t.ID] = t.Name
			}
		}
	}
	out := []map[string]string{}
	for termID, rt := range deps.TermRuntimes.Snapshot() {
		if !strings.EqualFold(rt.CLI, cli) {
			continue
		}
		name := names[termID]
		if name == "" {
			name = "Terminal"
		}
		out = append(out, map[string]string{"id": termID, "name": name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["name"] < out[j]["name"] })
	return out
}

func cliLabel(cli string) string {
	if cli == "" {
		return "this CLI"
	}
	return cli
}
