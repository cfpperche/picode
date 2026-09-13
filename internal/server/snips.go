package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/snips"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func registerSnips(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/snips/picker", handleSnipPicker(deps))
	mux.HandleFunc("GET /api/snips", handleListSnips(deps))
	mux.HandleFunc("POST /api/snips", handleCreateSnip(deps))
	mux.HandleFunc("GET /api/snips/{id}", handleGetSnip(deps))
	mux.HandleFunc("PATCH /api/snips/{id}", handleUpdateSnip(deps))
	mux.HandleFunc("DELETE /api/snips/{id}", handleDeleteSnip(deps))
	mux.HandleFunc("POST /api/snips/{id}/starred", handleSnipStarred(deps))
	mux.HandleFunc("POST /api/snips/{id}/archived", handleSnipArchived(deps))
	mux.HandleFunc("POST /api/snips/{id}/expand", handleSnipExpand(deps))
	mux.HandleFunc("POST /api/snips/{id}/run", handleSnipRun(deps))
}

type snipReq struct {
	Title        string              `json:"title"`
	Slug         string              `json:"slug"`
	Description  string              `json:"description"`
	Kind         string              `json:"kind"`
	Body         string              `json:"body"`
	Tags         []string            `json:"tags"`
	Placeholders []snips.Placeholder `json:"placeholders"`
	IfUpdatedAt  string              `json:"ifUpdatedAt"`
}

func snipParams(req snipReq) store.SnipParams {
	return store.SnipParams{
		Title: req.Title, Slug: req.Slug, Description: req.Description,
		Kind: req.Kind, Body: req.Body, Tags: req.Tags, Placeholders: req.Placeholders,
	}
}

func handleListSnips(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := store.SnipListFilter{Q: r.URL.Query().Get("q"), Archived: queryFlag(r, "archived")}
		list, err := deps.Store.ListSnips(f)
		if err != nil {
			writePinErr(w, err)
			return
		}
		archived, _ := deps.Store.CountArchivedSnips()
		writeJSON(w, http.StatusOK, map[string]any{"snips": list, "archived": archived})
	}
}

func handleSnipPicker(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := deps.Store.ListSnipPicker(queryFlag(r, "shell"))
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"snips": list})
	}
}

func handleCreateSnip(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req snipReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, err := deps.Store.CreateSnip(snipParams(req))
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleGetSnip(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := deps.Store.GetSnip(r.PathValue("id"))
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleUpdateSnip(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req snipReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		precond := req.IfUpdatedAt
		if h := r.Header.Get("If-Match"); h != "" {
			precond = h
		}
		p, err := deps.Store.UpdateSnip(r.PathValue("id"), snipParams(req), precond)
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleDeleteSnip(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.DeleteSnip(r.PathValue("id")); err != nil {
			writePinErr(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleSnipStarred(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Starred bool `json:"starred"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, err := deps.Store.SetSnipStarred(r.PathValue("id"), req.Starred)
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleSnipArchived(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Archived bool `json:"archived"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		p, err := deps.Store.SetSnipArchived(r.PathValue("id"), req.Archived)
		if err != nil {
			writePinErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

type snipExpandReq struct {
	Values  map[string]string `json:"values"`
	Context map[string]string `json:"context"`
}

type snipRunReq struct {
	Target  snipRunTarget     `json:"target"`
	Values  map[string]string `json:"values"`
	Confirm bool              `json:"confirm"`
	// Preview expands with live context and runs the gates, but delivers
	// nothing — the confirm step of a command shows exactly what would run.
	Preview bool `json:"preview"`
}

type snipRunTarget struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func handleSnipExpand(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := deps.Store.GetSnip(r.PathValue("id"))
		if err != nil {
			writePinErr(w, err)
			return
		}
		var req snipExpandReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		text, missing, err := expandStoredSnip(p, req.Values, req.Context)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if missing == nil {
			missing = []string{}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"text":         text,
			"missing":      missing,
			"placeholders": p.Placeholders,
		})
	}
}

func handleSnipRun(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := deps.Store.GetSnip(r.PathValue("id"))
		if err != nil {
			writePinErr(w, err)
			return
		}
		var req snipRunReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		reason, status := "", 0
		defer func() {
			// A preview runs the gates and expands but delivers nothing —
			// it is not a run and must not be logged as one.
			if !req.Preview && deps.Feed != nil {
				data := map[string]any{
					"snipId": p.ID, "kind": p.Kind, "targetType": req.Target.Type, "targetId": req.Target.ID,
					"ok": status == http.StatusOK,
				}
				if reason != "" {
					data["reason"] = reason
				}
				deps.Feed.Ephemeral("snip.ran", data)
			}
		}()
		if p.Kind == "shell" {
			if req.Preview && !req.Confirm {
				status = http.StatusBadRequest
				writeErr(w, status, "confirm is required to run a command")
				return
			}
			if req.Target.Type != "terminal" {
				reason, status = "kind", http.StatusConflict
				writeJSON(w, status, map[string]any{"error": "A command runs in a terminal, not in an agent.", "reason": reason})
				return
			}
			if !req.Confirm {
				status = http.StatusBadRequest
				writeErr(w, status, "confirm is required to run a command")
				return
			}
			code, body := runShellIntoTerminal(deps, r, p, req)
			if rsn, _ := body["reason"].(string); rsn != "" {
				reason = rsn
			}
			status = code
			writeJSON(w, code, body)
			return
		}
		switch req.Target.Type {
		case "agent":
		case "terminal":
		default:
			status = http.StatusBadRequest
			writeErr(w, status, "target must be an agent or a terminal")
			return
		}
		if req.Target.ID == "" {
			status = http.StatusBadRequest
			writeErr(w, status, "target id is required")
			return
		}
		if req.Preview && p.Kind != "shell" {
			status = http.StatusBadRequest
			writeErr(w, status, "preview is only valid for command snippets")
			return
		}
		if req.Target.Type == "terminal" {
			code, body := runSnipIntoTerminal(deps, r, p, req)
			if rsn, _ := body["reason"].(string); rsn != "" {
				reason = rsn
			}
			status = code
			writeJSON(w, code, body)
			return
		}
		agent, err := deps.Store.GetAgent(req.Target.ID)
		if err != nil {
			status = storeStatus(err)
			reason = "stopped"
			writePinErr(w, err)
			return
		}
		wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
		if err != nil {
			status = storeStatus(err)
			writePinErr(w, err)
			return
		}
		cwd := store.AgentCwd(wk, agent)
		if cwd == "" || !filepath.IsAbs(cwd) || strings.ContainsRune(cwd, 0) {
			status = http.StatusBadRequest
			writeErr(w, status, "working folder is not an absolute path")
			return
		}
		ctx := map[string]string{
			"cwd":       cwd,
			"workspace": wk.Path,
			"agent":     agent.Name,
			"cli":       "",
			"branch":    currentBranch(cwd),
		}
		text, missing, err := expandStoredSnip(p, req.Values, ctx)
		if err != nil {
			status = http.StatusBadRequest
			writeErr(w, status, err.Error())
			return
		}
		if len(missing) > 0 {
			status = http.StatusBadRequest
			writeJSON(w, status, map[string]any{"error": "Fill in the missing fields.", "missing": missing})
			return
		}
		if deps.Runtime == nil {
			reason, status = "stopped", http.StatusConflict
			writeJSON(w, status, map[string]any{"error": "agent is not running", "reason": reason})
			return
		}
		ma := deps.Runtime.Get(agent.ID)
		if ma == nil {
			reason, status = "stopped", http.StatusConflict
			writeJSON(w, status, map[string]any{"error": "agent is not running", "reason": reason})
			return
		}
		if err := ma.SendTurn(store.TaskPrompt, text, nil); err != nil {
			status = http.StatusBadRequest
			writeErr(w, status, err.Error())
			return
		}
		status = http.StatusOK
		writeJSON(w, status, map[string]any{"ok": true, "typed": true, "text": text})
	}
}

// runSnipIntoTerminal delivers a prompt snippet through the ADR-0089 door.
// The pane must hold a CLI whose foreground is not a shell — re-checked
// here, at handler time (B6b): a launch row stays true after the TUI
// exits, and a prompt body pasted into bash is exactly the defect the
// owner refused (Q2b).
func runSnipIntoTerminal(deps Deps, r *http.Request, p store.Snip, req snipRunReq) (int, map[string]any) {
	t, err := deps.Store.GetTerminal(req.Target.ID)
	if err != nil {
		return storeStatus(err), map[string]any{"error": err.Error()}
	}
	if !termHoldsCLI(deps, t.ID) {
		return http.StatusConflict, map[string]any{"error": "Attach is for Agent CLI terminals.", "reason": "cli"}
	}
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return http.StatusServiceUnavailable, map[string]any{"error": "Need tmux to send to a terminal."}
	}
	// B6b, scoped to where it can be true. A launched CLI's pane leader is
	// the wrapper `sh` itself (launch.sh runs the TUI as a child), so a bare
	// isShell read would refuse every live launch. The refusal is for a
	// launch row whose lease is gone — the wrapper returns the pane to an
	// interactive shell when the CLI exits, and that shell must not be fed
	// a prompt body.
	if deps.TermRuntimes != nil {
		if _, live := deps.TermRuntimes.Get(t.ID); !live {
			if cmd := paneCommandFn(r.Context(), deps, tmux.ShellSessionName(t.ID)); cmd != "" && isShell(cmd) {
				return http.StatusConflict, map[string]any{
					"error":  "This pane is at a shell prompt — a prompt snippet would run as a command.",
					"reason": "kind",
				}
			}
		}
	}
	wsPath := ""
	if t.WorkspaceID != "" && t.WorkspaceID != store.FreeWorkspaceID {
		if wk, err := deps.Store.GetWorkspace(t.WorkspaceID); err == nil {
			wsPath = wk.Path
		}
	}
	cliID := ""
	if v, err := deps.Store.TerminalLaunch(t.ID); err == nil && v != nil {
		cliID = v.CLI
	}
	ctx := map[string]string{
		"cwd":       liveTermCwd(deps, r, t),
		"workspace": wsPath,
		"agent":     "",
		"cli":       cliID,
	}
	text, missing, err := expandStoredSnip(p, req.Values, ctx)
	if err != nil {
		return http.StatusBadRequest, map[string]any{"error": err.Error()}
	}
	if len(missing) > 0 {
		return http.StatusBadRequest, map[string]any{"error": "Fill in the missing fields.", "missing": missing}
	}
	status, body := pasteToTerminal(deps, r.Context(), t, text)
	if status == http.StatusOK {
		return http.StatusOK, map[string]any{"ok": true, "typed": true, "text": text}
	}
	return status, body
}

// runShellIntoTerminal delivers a command snippet through the snippet-run
// door (ADR-0128 in ADR-0130): a live shell pane only, re-checked at
// handler time, one in-flight run per terminal, ClearLine then a
// bracketed paste whose Enter submits. Confirm was already required by
// the caller — a UI invariant on official clients, not authorization
// (K14): a paired API client may set it, same as Pins.
func runShellIntoTerminal(deps Deps, r *http.Request, p store.Snip, req snipRunReq) (int, map[string]any) {
	t, err := deps.Store.GetTerminal(req.Target.ID)
	if err != nil {
		return storeStatus(err), map[string]any{"error": err.Error()}
	}
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return http.StatusServiceUnavailable, map[string]any{"error": "Need tmux to run a command in a terminal."}
	}
	// A live CLI lease means the TUI owns the pane — the wrapper `sh` is
	// the pane leader, so isShell alone would lie. Only a lease-less pane
	// whose foreground is a shell may run a command (a launch whose TUI
	// exited returned the pane to an interactive shell; Q2a allows it).
	if _, live := deps.TermRuntimes.Get(t.ID); live {
		return http.StatusConflict, map[string]any{"error": "This terminal is running its CLI.", "reason": "cli"}
	}
	ctx0, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	session := tmux.ShellSessionName(t.ID)
	cmd := paneCommandFn(ctx0, deps, session)
	if cmd == "" {
		return http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again.", "reason": "closed"}
	}
	if !isShell(cmd) {
		return http.StatusConflict, map[string]any{"error": "This terminal is running " + cmd + ".", "reason": "foreground"}
	}
	wsPath := ""
	if t.WorkspaceID != "" && t.WorkspaceID != store.FreeWorkspaceID {
		if wk, err := deps.Store.GetWorkspace(t.WorkspaceID); err == nil {
			wsPath = wk.Path
		}
	}
	cwd := liveTermCwd(deps, r, t)
	ctx := map[string]string{
		"cwd":       cwd,
		"workspace": wsPath,
		"agent":     "",
		"cli":       "",
		"branch":    currentBranch(cwd),
	}
	text, missing, err := expandStoredSnip(p, req.Values, ctx)
	if err != nil {
		return http.StatusBadRequest, map[string]any{"error": err.Error()}
	}
	if len(missing) > 0 {
		return http.StatusBadRequest, map[string]any{"error": "Fill in the missing fields.", "missing": missing}
	}
	if req.Preview {
		return http.StatusOK, map[string]any{"preview": true, "text": text}
	}
	if !tryLockPrompt(t.ID) {
		return http.StatusConflict, map[string]any{"error": "Already sending to this terminal.", "reason": "busy"}
	}
	defer unlockPrompt(t.ID)
	// The pane holds a shell prompt (checked above); clear whatever was
	// typed there so the command is not glued to it (ADR-0096), then paste.
	_ = deps.Tmux.ClearLine(ctx0, session)
	if err := deps.Tmux.PasteText(ctx0, session, text); err != nil {
		return http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again.", "reason": "closed"}
	}
	return http.StatusOK, map[string]any{"ok": true, "typed": true, "text": text}
}

func expandStoredSnip(p store.Snip, values, ctx map[string]string) (string, []string, error) {
	if values == nil {
		values = map[string]string{}
	}
	if err := checkSnipEnums(p.Placeholders, values); err != nil {
		return "", nil, err
	}
	return snips.Expand(p.Body, values, ctx)
}

func checkSnipEnums(placeholders []snips.Placeholder, values map[string]string) error {
	for _, ph := range placeholders {
		if len(ph.Enum) == 0 {
			continue
		}
		v, ok := values[ph.Name]
		if !ok || v == "" {
			continue
		}
		found := false
		for _, e := range ph.Enum {
			if v == e {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("value for %s is not one of the allowed choices", ph.Name)
		}
	}
	return nil
}
