package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cfpperche/picode/internal/gitinfo"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func registerTerminalRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/terminals", handleListTerminals(deps))
	mux.HandleFunc("POST /api/terminals", handleCreateTerminal(deps))
	mux.HandleFunc("DELETE /api/terminals/{id}", handleDeleteTerminal(deps))
	mux.HandleFunc("PATCH /api/terminals/{id}", handleRenameTerminal(deps))
	mux.HandleFunc("POST /api/terminals/{id}/open", handleOpenTerminal(deps))
	mux.HandleFunc("POST /api/terminals/{id}/state", handleSetTerminalState(deps))
	mux.HandleFunc("POST /api/terminals/{id}/open-url", handleTerminalOpenURL(deps))
	mux.HandleFunc("POST /api/terminals/{id}/runtime", handleSetTerminalRuntime(deps))
	mux.HandleFunc("GET /api/terminals/{id}/text", handleGetTerminalText(deps))
	mux.HandleFunc("PUT /api/terminals/{id}/text", handlePutTerminalText(deps))
	mux.HandleFunc("GET /api/terminals/{id}/blob", handleGetTerminalBlob(deps))
	mux.HandleFunc("GET /api/terminals/{id}/cwd", handleGetTerminalCwd(deps))
	mux.HandleFunc("GET /api/terminals/{id}/browse", handleTerminalBrowse(deps))
	registerTermPromptRoutes(mux, deps)
}

// defaultShell is the shell a new terminal runs. $SHELL wins when it names a
// file that exists — a daemon started by systemd, a container or a test
// binary has no SHELL at all, and this used to fall back to /bin/sh, which on
// Debian-family systems is dash: the pane then died on the first argument
// (measured 2026-09-20, see shellTakesRcfile) and took the tmux server with
// it under exit-empty, while the API reported a live terminal. bash is the
// fallback because the intercept rcfile and the rc the CLI wrappers expect
// are bash's.
func defaultShell() string {
	if s := strings.TrimSpace(os.Getenv("SHELL")); s != "" {
		if st, err := os.Stat(s); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
			return s
		}
	}
	if st, err := os.Stat(bashFallback); err == nil && !st.IsDir() && st.Mode()&0o111 != 0 {
		return bashFallback
	}
	return "/bin/sh"
}

const bashFallback = "/bin/bash"

// shellTakesRcfile reports whether `shell --rcfile <file>` is a command the
// shell will accept. bash is the only shell here that reads a replacement rc
// file by that flag: dash (what /bin/sh is on Debian and Ubuntu) exits with
// "Illegal option --" and status 2, so a pane given one dies before the user
// sees it. Symlinks are resolved because /bin/sh is exactly the ambiguous
// case — dash on Debian, bash on macOS, busybox elsewhere.
func shellTakesRcfile(shell string) bool {
	if filepath.Base(shell) == "bash" {
		return true
	}
	if resolved, err := filepath.EvalSymlinks(shell); err == nil {
		return filepath.Base(resolved) == "bash"
	}
	return false
}

func termView(t store.Terminal, session string, live bool) map[string]any {
	return map[string]any{
		"id":          t.ID,
		"name":        t.Name,
		"cwd":         t.Cwd,
		"workspaceId": t.WorkspaceID,
		"createdAt":   t.CreatedAt,
		"session":     session,
		"running":     live,
	}
}

// liveTermView is termView with the truth layered on: the pane's live cwd
// and the git facts read from it. EVERY handler that answers with a terminal
// uses this one — the app merges any such response into its list, so a
// response carrying the record cwd would overwrite the live one while the
// stale git survived the merge, pairing one directory's path with another's
// branch on the selected terminal.
// termViewForCreation picks the view for a request that may have just created
// the session: fresh (the folder it was created in) when it did, live
// otherwise.
func termViewForCreation(deps Deps, r *http.Request, t store.Terminal, session string, created bool) map[string]any {
	if created {
		return freshTermView(deps, r, t, session)
	}
	return liveTermView(deps, r, t, session, true)
}

func liveTermView(deps Deps, r *http.Request, t store.Terminal, session string, live bool) map[string]any {
	return termViewWith(deps, r, t, session, live, liveTermCwd(deps, r, t))
}

// freshTermView is the view for a session this request just created: its cwd
// is the one it was created in, not a live read. The live read races the
// pane's own process — tmux answers #{pane_current_path} with the *server's*
// directory while the pane's command has not spawned yet, and the server's
// directory is the daemon's own cwd. Measured 2026-09-15: 12 of 30 creation
// responses carried /home/goat/picode/internal/server (the daemon's cwd)
// instead of the workspace folder; that is the flake in
// TestCreateTerminalInWorkspaceUsesItsFolder, and a moment of the wrong folder
// in the UI and in the Inspector's root assertion. The next poll reads live
// again, which is where a user's own `cd` shows up.
func freshTermView(deps Deps, r *http.Request, t store.Terminal, session string) map[string]any {
	return termViewWith(deps, r, t, session, true, t.Cwd)
}

// termViewWith renders the terminal view for a known cwd — live read or the
// folder we just created the session in.
func termViewWith(deps Deps, r *http.Request, t store.Terminal, session string, live bool, cwd string) map[string]any {
	view := termView(t, session, live)
	view["cwd"] = cwd
	view["git"] = gitinfo.Inspect(cwd)
	// Terminal CLI presence is independent from lifecycle activity (ADR-0062).
	// The runtime is applied first, then a state report may add activity.
	applyTermRuntime(deps, view, t.ID)
	applyTermState(deps, view, t.ID)
	applyTerminalLaunch(deps, view, t.ID)
	applyTerminalChecklist(deps, view, t.ID)
	// Session forensics (ADR-0085): this terminal was alive at the previous
	// graceful shutdown and its session did not survive to this boot.
	if !live && deps.LostSessions[session] {
		view["lostAtRestart"] = true
	}
	return view
}

func handleListTerminals(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The CLIs page refetches on every terminal.* feed event; one round
		// costs up to a dozen subprocesses per terminal. Concurrent requests
		// share one computation and a snapshot younger than the TTL answers
		// without spawning anything (nil cache = compute directly — tests,
		// minimal embeddings).
		compute := func() ([]map[string]any, error) { return computeTerminals(r.Context(), deps) }
		var (
			out []map[string]any
			err error
		)
		if deps.TermCache == nil {
			out, err = compute()
		} else {
			out, err = deps.TermCache.View(DefaultTTL, compute)
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"terminals": out})
	}
}

// computeTerminals builds one full snapshot of every managed terminal. The
// per-terminal facts are independent, so the fleet is walked with a small
// worker pool instead of in sequence; the response order matches the store.
func computeTerminals(ctx context.Context, deps Deps) ([]map[string]any, error) {
	list, err := deps.Store.ListTerminals()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(list))
	const workers = 6
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, t := range list {
		wg.Add(1)
		go func(i int, t store.Terminal) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			name := tmux.ShellSessionName(t.ID)
			live := false
			if deps.Tmux != nil && deps.Tmux.Available() {
				live, _ = deps.Tmux.HasSession(ctx, name)
			}
			// The list speaks about where the terminal IS, not where it was
			// born (ADR-0022).
			out[i] = liveTermView(deps, requestWith(ctx), t, name, live)
		}(i, t)
	}
	wg.Wait()
	for _, v := range out {
		if v == nil {
			return nil, ctx.Err()
		}
	}
	return out, nil
}

// requestWith adapts a bare context to the *http.Request that liveTermView
// and liveTermCwd expect, without changing the signature used by the ~25
// single-terminal call sites.
func requestWith(ctx context.Context) *http.Request { return (&http.Request{}).WithContext(ctx) }

func invalidateTerminals(deps Deps) {
	if deps.TermCache != nil {
		deps.TermCache.Invalidate()
	}
}

func handleCreateTerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, http.StatusServiceUnavailable, "Need tmux to open a terminal.")
			return
		}
		var req struct {
			Name        string `json:"name"`
			Cwd         string `json:"cwd"`
			WorkspaceID string `json:"workspaceId"`
		}
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
		}
		t, err := deps.Store.CreateTerminalIn(req.WorkspaceID, req.Name, req.Cwd)
		if err != nil {
			// Covers both store messages on purpose: "that folder doesn't
			// exist" and "that workspace doesn't exist" are user errors.
			if strings.Contains(err.Error(), "doesn't exist") {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		name := tmux.ShellSessionName(t.ID)
		created, err := ensureShell(deps, r, name, t.ID, t.Cwd)
		if err != nil {
			_ = deps.Store.DeleteTerminal(t.ID)
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		invalidateTerminals(deps)
		writeJSON(w, http.StatusCreated, termViewForCreation(deps, r, t, name, created))
	}
}

func handleOpenTerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		unlockAgent := boundAgentLock(deps, r.PathValue("id"))
		defer unlockAgent()
		unlock := terminalLock(deps, r.PathValue("id"))
		defer unlock()
		t, err := deps.Store.GetTerminal(r.PathValue("id"))
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "That terminal is gone.")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, http.StatusServiceUnavailable, "Need tmux to open a terminal.")
			return
		}
		name := tmux.ShellSessionName(t.ID)
		launch, err := deps.Store.TerminalLaunch(t.ID)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if launch != nil {
			live, err := deps.Tmux.HasSession(r.Context(), name)
			if err != nil {
				writeErr(w, 500, err.Error())
				return
			}
			if !live {
				// Browser restoration uses /open too. Only an explicit Start
				// may relaunch a configured coding CLI after it was stopped.
				writeJSON(w, 200, liveTermView(deps, r, t, name, false))
				return
			}
		}
		created, err := ensureShell(deps, r, name, t.ID, t.Cwd)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, termViewForCreation(deps, r, t, name, created))
	}
}

func handleRenameTerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		t, err := deps.Store.RenameTerminal(r.PathValue("id"), req.Name)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "That terminal is gone.")
			return
		}
		if err != nil {
			if strings.Contains(err.Error(), "name is required") {
				writeErr(w, http.StatusBadRequest, "Give it a name.")
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		invalidateTerminals(deps)
		name := tmux.ShellSessionName(t.ID)
		live := false
		if deps.Tmux != nil && deps.Tmux.Available() {
			live, _ = deps.Tmux.HasSession(r.Context(), name)
		}
		writeJSON(w, http.StatusOK, liveTermView(deps, r, t, name, live))
	}
}

func handleDeleteTerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		unlockAgent := boundAgentLock(deps, r.PathValue("id"))
		defer unlockAgent()
		id := r.PathValue("id")
		if a, e := deps.Store.AgentByTerminal(id); e == nil && a.IsPi() {
			release := deps.Replies.Controls.BeginMutation(a.ID)
			defer release()
			if err := deps.stopAgentInteractive(r.Context(), a.ID); err != nil {
				writeErr(w, 500, err.Error())
				return
			}
			if deps.Runtime != nil {
				deps.Runtime.Stop(a.ID)
			}
		}
		unlock := terminalLock(deps, id)
		defer unlock()
		if _, err := deps.Store.GetTerminal(id); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "That terminal is gone.")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if deps.Tmux != nil && deps.Tmux.Available() {
			if err := deps.Tmux.KillSession(r.Context(), tmux.ShellSessionName(id)); err != nil {
				writeErr(w, 500, err.Error())
				return
			}
		}
		if deps.TermStates != nil {
			deps.TermStates.Drop(id)
		}
		if deps.TermRuntimes != nil {
			// The terminal.deleted event removes the row; no separate runtime
			// event is needed, but the in-memory lease must not linger until the
			// next reconciliation tick.
			deps.TermRuntimes.Drop(id)
		}
		if err := cleanCLILaunches(deps.DataDir, id, ""); deps.DataDir != "" && err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		if err := deps.Store.DeleteTerminal(id); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		invalidateTerminals(deps)
		w.WriteHeader(http.StatusNoContent)
	}
}

func liveTermCwd(deps Deps, r *http.Request, term store.Terminal) string {
	if deps.Tmux != nil && deps.Tmux.Available() {
		p, err := deps.Tmux.PaneCwd(r.Context(), tmux.ShellSessionName(term.ID))
		if err == nil && strings.TrimSpace(p) != "" {
			return p
		}
	}
	return term.Cwd
}

func handleGetTerminalCwd(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"cwd": liveTermCwd(deps, r, term)})
	}
}

func handleTerminalBrowse(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		cwd := liveTermCwd(deps, r, term)
		if !checkFileRoot(w, r, cwd) {
			return
		}
		out, err := browseAgentDir(cwd, r.URL.Query().Get("dir"))
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleGetTerminalBlob(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		cwd, ok := resolveGitWorktree(w, r, liveTermCwd(deps, r, term))
		if !ok {
			return
		}
		if !checkFileRoot(w, r, cwd) {
			return
		}
		mime, data, code, err := readAgentBlob(cwd, r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, code, err.Error())
			return
		}
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "private, max-age=0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}

func handleGetTerminalText(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		cwd, ok := resolveGitWorktree(w, r, liveTermCwd(deps, r, term))
		if !ok {
			return
		}
		if !checkFileRoot(w, r, cwd) {
			return
		}
		out, code, err := readAgentText(cwd, r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, code, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handlePutTerminalText(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		var req agentTextPut
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Path == "" {
			req.Path = r.URL.Query().Get("path")
		}
		cwd, ok := resolveGitWorktree(w, r, liveTermCwd(deps, r, term))
		if !ok {
			return
		}
		if !checkFileRoot(w, r, cwd) {
			return
		}
		out, code, err := writeAgentText(cwd, req.Path, req.Text, req.Mtime)
		if err != nil {
			writeErr(w, code, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ensureShell makes sure the terminal's session exists, and reports whether
// this call is the one that created it (the caller needs that: a cwd read
// taken in the same instant as the creation races the pane's own process).
func ensureShell(deps Deps, r *http.Request, name, termID, cwd string) (bool, error) {
	if !deps.Tmux.Available() {
		return false, errors.New("Need tmux to open a terminal.")
	}
	has, err := deps.Tmux.HasSession(r.Context(), name)
	if err != nil {
		return false, err
	}
	hadSession := has
	if !has && deps.Store != nil {
		launch, err := deps.Store.TerminalLaunch(termID)
		if err != nil {
			return false, err
		}
		if launch != nil {
			if err := launchCLITerminal(deps, r, name, cwd, launch); err != nil {
				return false, err
			}
			has = true
		}
	}
	if !has {
		// CLI lifecycle sensors correlate to this terminal through the session
		// environment (ADR-0056 tier 1): hooks inherit PICODE_TERM_ID from
		// the shell, and PICODE_TERM_URL spares them configuration. The env
		// must exist from the first pane, so it rides new-session (-e) — a
		// set-environment afterwards would miss the shell already running.
		env := []string{tmux.MarkerTermEnv + "=" + termID}
		if u := loopbackURL(deps); u != "" {
			env = append(env, tmux.MarkerURLEnv+"="+u)
		}
		// Wrappers live in <data>/bin and are only visible inside this
		// session (ADR-0056 intercept). Empty when nothing is enabled.
		ensureTmuxGuard(deps.DataDir) // ADR-0138: an upgrade or clean-up must not leave new sessions unguarded.
		if p := interceptSessionPath(deps.DataDir); p != "" {
			env = append(env, p)
		}
		if b := interceptBinEnv(deps.DataDir); b != "" {
			env = append(env, b)
		}
		env = append(env, openURLEnv(deps.DataDir)...) // ADR-0180: a CLI's "open in browser" leaves WSL
		cmd := defaultShell()
		var args []string
		if shellTakesRcfile(cmd) {
			if rc, err := ensureInterceptBashrc(deps.DataDir); err == nil {
				args = append(args, "--rcfile", rc)
			}
		}
		if err := deps.Tmux.NewSessionEnv(r.Context(), name, cwd, env, cmd, args...); err != nil {
			return false, err
		}
		// A pane whose command exits at once takes its session with it, and a
		// server with nothing else on it exits too (exit-empty) — while
		// new-session answers 0. Verify, or this reports a terminal that
		// never lived and every later tmux call answers "no server running"
		// (measured 2026-09-20: dash given --rcfile).
		if live, err := deps.Tmux.HasSession(r.Context(), name); err != nil || !live {
			return false, fmt.Errorf("The shell exited immediately (%s). Set SHELL to a shell that exists, or check the daemon's environment.", cmd)
		}
	}
	// Everything PiCode manages — status bar, passthrough, mouse, extended
	// keys — comes from the resolver (ADR-0024): the old forces are its
	// defaults, so a user override wins with no special case. A brand-new
	// terminal has no overrides yet, so this is the global default — which is
	// the point: a default, not a snapshot taken at creation.
	applyScoped(r.Context(), deps, name, termOptionResolver(deps)(name))
	_ = deps.Tmux.SetEnv(r.Context(), name, "TERM", "xterm-256color")
	_ = deps.Tmux.SetEnv(r.Context(), name, "COLORTERM", "truecolor")
	return !hadSession, nil
}
