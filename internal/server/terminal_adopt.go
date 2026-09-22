package server

import (
	"errors"
	"net/http"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// handleAdoptTerminal is "Make agent" (ADR-0184): a catalog CLI someone
// typed into a PiCode shell becomes an agent bound to that terminal — in
// the shell's workspace, or free — so it gets a name, grants and a place in
// the fleet. Nothing is adopted without this request; the CLI keeps running
// as it is, and the agent's launch (the CLI, PiCode's tools) applies from
// its next start.
func handleAdoptTerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		unlock := terminalLock(deps, id)
		defer unlock()
		t, err := deps.Store.GetTerminal(id)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		if t.Kind != "" {
			writeErr(w, http.StatusConflict, "This terminal belongs to a sign-in.")
			return
		}
		if _, err := deps.Store.AgentByTerminal(id); err == nil {
			writeErr(w, http.StatusConflict, "This terminal is already an agent.")
			return
		} else if !errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		cli, ok := adoptableCLI(deps, id)
		if !ok {
			writeErr(w, http.StatusConflict, "No CLI PiCode knows is running in this terminal.")
			return
		}
		if launch, err := deps.Store.TerminalLaunch(id); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		} else if launch != nil && launch.CLI != cli.ID {
			writeErr(w, http.StatusConflict, "This terminal is launched as a different CLI.")
			return
		}
		// The agent works where the shell is now, not where it was born.
		cwd := liveTermCwd(deps, r, t)
		work := cwd
		if t.WorkspaceID != store.FreeWorkspaceID {
			if wk, err := deps.Store.GetWorkspace(t.WorkspaceID); err == nil && wk.Path == cwd {
				work = ""
			}
		}
		agent, err := deps.Store.AddAgentWithCLI(t.WorkspaceID, cli.ID, t.Name, work)
		if err != nil {
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		if err := deps.Store.SetTerminalLaunch(id, cli.ID, fillManagedCLITools(clilaunch.Overrides{}, cli.ID)); err != nil {
			_ = deps.Store.DeleteAgent(agent.ID)
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		tid := id
		bound, err := deps.Store.UpdateAgent(agent.ID, store.AgentPatch{TerminalID: &tid})
		if err != nil {
			_ = deps.Store.DeleteAgent(agent.ID)
			writeErr(w, storeStatus(err), err.Error())
			return
		}
		live := false
		if deps.Tmux != nil && deps.Tmux.Available() {
			live, _ = deps.Tmux.HasSession(r.Context(), tmux.ShellSessionName(id))
		}
		publishTerminalState(deps, r, t, live)
		invalidateTerminals(deps)
		writeJSON(w, http.StatusCreated, agentView{Agent: bound, Mode: string(modeStopped)})
	}
}

// adoptableCLI is the launchable catalog CLI PiCode has seen running in
// the terminal (ADR-0062 runtime presence), if any.
func adoptableCLI(deps Deps, termID string) (clilaunch.CLI, bool) {
	if deps.TermRuntimes == nil {
		return clilaunch.CLI{}, false
	}
	rt, ok := deps.TermRuntimes.Get(termID)
	if !ok || rt.CLI == "" {
		return clilaunch.CLI{}, false
	}
	cli, ok := clilaunch.Find(rt.CLI)
	if !ok || !cli.Launchable() {
		return clilaunch.CLI{}, false
	}
	return cli, true
}
