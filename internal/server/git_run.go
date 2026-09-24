package server

import (
	"context"
	"encoding/json"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/gitcmd"
	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// Stage 2 of the Inspector's Git actions (ADR-0078). With the viewer's "run
// when no agent is working here" preference on, PiCode presses Enter on the
// command it just typed — still in the user's own shell, with the user's
// credentials, hooks and signing — but only after an interlock finds nobody
// else writing that repository: no managed agent mid-turn, no interactive
// TUI working, no automation run, no terminal that its CLI reports working
// or that holds a foreground process, and the target pane itself waiting at
// a shell prompt. Any of those is a 409 naming who is busy, and the rail
// falls back to preparing the command. The interlock is advisory: it sees
// only what PiCode knows, and only at the moment it looks. Nothing here
// runs git in the service process; the request also asserts the rail's
// root, so a terminal that moved never runs a command meant for elsewhere.
func registerGitRunRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/terminals/{id}/run", handleTerminalRun(deps))
}

type busyOwner struct {
	Kind string `json:"kind"` // agent | terminal
	ID   string `json:"id"`
	Name string `json:"name"`
	Why  string `json:"why"`
}

// Probes are package variables so tests can stand in for pi, tmux and the
// CLI hooks without spawning any of them.
var (
	agentBusyFn    = agentBusy
	terminalBusyFn = terminalBusy
	paneCommandFn  = paneCommand
)

// paneCommand names the pane's foreground process, giving a shell that is
// momentarily busy with a prompt hook or an rc-file child a second or two
// to come back: the last answer is what the caller judges.
func paneCommand(ctx context.Context, deps Deps, session string) string {
	if deps.Tmux == nil || !deps.Tmux.Available() {
		return ""
	}
	last := ""
	for i := 0; i < 12; i++ {
		cmd, err := deps.Tmux.PaneCommand(ctx, session)
		if err != nil {
			return ""
		}
		last = cmd
		if isShell(cmd) {
			return cmd
		}
		select {
		case <-ctx.Done():
			return last
		case <-time.After(150 * time.Millisecond):
		}
	}
	return last
}

// isShell says whether a pane's foreground process is a shell waiting for
// input — the only place a prepared command may be typed or run.
func isShell(cmd string) bool {
	base := strings.TrimPrefix(path.Base(strings.TrimSpace(cmd)), "-")
	switch base {
	case "bash", "zsh", "fish", "sh", "dash", "ksh", "mksh", "ash", "tcsh", "csh", "nu", "pwsh":
		return true
	}
	return false
}

func agentBusy(ctx context.Context, deps Deps, a store.Agent) (bool, string) {
	if deps.Runtime != nil {
		if ma := deps.Runtime.Get(a.ID); ma != nil && ma.Snapshot().Streaming {
			return true, "mid-turn"
		}
	}
	if deps.automationRunOn(a.ID) {
		return true, "running an automation"
	}
	if a.TerminalID != nil && deps.agentSession(a.ID) == tmux.ShellSessionName(*a.TerminalID) {
		if deps.Runtime != nil && deps.Runtime.Active(a.ID) {
			return false, ""
		}
		if t, e := deps.Store.GetTerminal(*a.TerminalID); e == nil {
			return terminalBusy(ctx, deps, t)
		}
	}
	if deps.Tmux != nil && deps.Tmux.Available() {
		name := deps.agentSession(a.ID)
		if has, err := deps.Tmux.HasSession(ctx, name); err == nil && has {
			if tail, err := deps.Tmux.CaptureTail(ctx, name, 8); err == nil && tmux.LooksWorking(tail) {
				return true, "working in its terminal"
			}
		}
	}
	return false, ""
}

// terminalBusy: a terminal with a CLI runtime is busy when the CLI reports
// working; a plain terminal is busy when something other than its shell
// holds the foreground (a paused rebase, an editor, a long command).
func terminalBusy(ctx context.Context, deps Deps, t store.Terminal) (bool, string) {
	if deps.TermStates != nil {
		if st, ok := deps.TermStates.Get(t.ID); ok && (st.State == TermWorking || st.State == TermCompacting) {
			return true, "working"
		}
	}
	if deps.TermRuntimes != nil {
		if _, present := deps.TermRuntimes.Get(t.ID); present {
			return false, ""
		}
	}
	if cmd := paneCommandFn(ctx, deps, tmux.ShellSessionName(t.ID)); cmd != "" && !isShell(cmd) {
		return true, "running " + cmd
	}
	return false, ""
}

// repoBusy lists every PiCode-known writer of the repository that contains
// cwd — agents whose cwd resolves to the same git common dir, and the other
// terminals sitting in it.
func repoBusy(ctx context.Context, deps Deps, r *http.Request, cwd, targetTermID string) []busyOwner {
	key := gitgraph.Key(cwd)
	if key == "" {
		return nil
	}
	out := []busyOwner{}
	if agents, err := deps.Store.ListAllAgents(); err == nil {
		for _, a := range agents {
			wk, err := deps.Store.GetWorkspace(a.WorkspaceID)
			if err != nil {
				continue
			}
			if gitgraph.Key(store.AgentCwd(wk, a)) != key {
				continue
			}
			if busy, why := agentBusyFn(ctx, deps, a); busy {
				out = append(out, busyOwner{Kind: "agent", ID: a.ID, Name: a.Name, Why: why})
			}
		}
	}
	if terms, err := deps.Store.ListTerminals(); err == nil {
		for _, t := range terms {
			if t.ID == targetTermID {
				continue
			}
			if gitgraph.Key(liveTermCwd(deps, r, t)) != key {
				continue
			}
			if busy, why := terminalBusyFn(ctx, deps, t); busy {
				out = append(out, busyOwner{Kind: "terminal", ID: t.ID, Name: t.Name, Why: why})
			}
		}
	}
	return out
}

func busyMessage(busy []busyOwner) string {
	if len(busy) == 0 {
		return ""
	}
	b := busy[0]
	who := b.Name
	if who == "" {
		who = b.Kind
	}
	if b.Kind == "terminal" {
		who = "Terminal " + who
	}
	if len(busy) > 1 {
		return who + " and " + strconv.Itoa(len(busy)-1) + " more are busy in this repository."
	}
	return who + " is " + b.Why + " in this repository."
}

func handleTerminalRun(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if refuseInspectorCLI(w, deps, t.ID) {
			return
		}
		var req struct {
			Text string `json:"text"`
			Root string `json:"root"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if msg := typeTextProblem(req.Text); msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
		// The request is judged before the machine: a bad body, an unknown
		// terminal or a stale root is the caller's problem whether or not
		// tmux is installed. The capability check sits ahead of the pane
		// probes, which are what needs tmux.
		cwd := liveTermCwd(deps, r, t)
		if req.Root == "" || req.Root != canonDir(cwd) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "This terminal moved to " + cwd + ".", "reason": "moved", "cwd": cwd})
			return
		}
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, http.StatusServiceUnavailable, "Need tmux to run a command in a terminal.")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		session := tmux.ShellSessionName(t.ID)
		cmd := paneCommandFn(ctx, deps, session)
		if cmd == "" {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again.", "reason": "closed"})
			return
		}
		if !isShell(cmd) {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "This terminal is running " + cmd + ".", "reason": "foreground"})
			return
		}
		// Creating a worktree makes a new folder and a new branch and touches
		// nothing another agent is working on (ADR-0202): that one command,
		// aimed at this terminal's own repository, skips the busy check.
		if !worktreeCreateHere(req.Text, cwd) {
			if busy := repoBusy(ctx, deps, r, cwd, t.ID); len(busy) > 0 {
				writeJSON(w, http.StatusConflict, map[string]any{"error": busyMessage(busy), "reason": "busy", "busy": busy})
				return
			}
		}
		// The pane is at a shell prompt (checked above), but that prompt may
		// already hold a command an earlier Prepare typed and nobody
		// submitted. Typing onto its end would glue two commands into one
		// word; clear the line first (ADR-0096).
		_ = deps.Tmux.ClearLine(ctx, session)
		if err := deps.Tmux.TypeText(ctx, session, req.Text); err != nil {
			writeJSON(w, http.StatusConflict, map[string]any{"error": "Open the terminal first, then try again.", "reason": "closed"})
			return
		}
		if err := deps.Tmux.SendKeys(ctx, session, "Enter"); err != nil {
			writeErr(w, http.StatusBadGateway, "The command was typed but could not be submitted: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ran": true, "command": req.Text})
	}
}

// worktreeCreateHere reports whether text is the worktree-creating command
// (gitcmd.WorktreeCreate) for the repository cwd belongs to: an absolute
// root must be that repository's, a relative one resolves from cwd.
func worktreeCreateHere(text, cwd string) bool {
	root, ok := gitcmd.WorktreeCreate(text)
	if !ok {
		return false
	}
	key := gitgraph.Key(cwd)
	if key == "" {
		return false
	}
	if root == "" {
		return canonDir(cwd) == canonDir(filepath.Dir(key))
	}
	return canonDir(root) == canonDir(filepath.Dir(key))
}
