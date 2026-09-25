package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func piAgentLaunchFingerprint(c clilaunch.Config, a store.Agent) string {
	// Session pointers change as the running CLI creates/switches files;
	// they are not unapplied launch configuration.
	a.SessionPath = nil
	c.Args = append(append([]string{}, c.Args...), a.CLIFlags()...)
	return clilaunch.Fingerprint(c)
}

func validatePiAgentArgs(args []string) error {
	reserved := map[string]bool{"--mode": true, "--session": true, "--session-id": true, "--session-dir": true, "--continue": true, "--resume": true, "--no-session": true, "--print": true, "-p": true, "-c": true, "-r": true, "--provider": true, "--model": true, "--thinking": true, "--tools": true, "--append-system-prompt": true}
	for _, arg := range args {
		key, _, _ := strings.Cut(arg, "=")
		if reserved[key] || arg == "--" {
			return fmt.Errorf("Configure %s through this Pi agent's settings or session controls.", key)
		}
	}
	return nil
}

func (deps Deps) agentTerminalView(r *http.Request, a store.Agent) map[string]any {
	if a.TerminalID == nil {
		return nil
	}
	t, err := deps.Store.GetTerminal(*a.TerminalID)
	if err != nil || deps.Tmux == nil {
		return nil
	}
	name := tmux.ShellSessionName(t.ID)
	live, _ := deps.Tmux.HasSession(r.Context(), name)
	return liveTermView(deps, r, t, name, live)
}

func (deps Deps) stopAgentInteractive(ctx context.Context, id string) error {
	name := deps.agentSession(id)
	key := id
	a, err := deps.Store.GetAgent(id)
	if err != nil {
		return err
	}
	if deps.Tmux == nil || !deps.Tmux.Available() {
		keys := []string{id}
		if a.TerminalID != nil {
			keys = append(keys, *a.TerminalID)
		}
		for _, key := range keys {
			if blocked, e := peerStopPending(deps, key); e != nil {
				return e
			} else if blocked {
				return errAgentTUIInFlight
			}
		}
		return nil
	}
	if live, e := deps.Tmux.HasSession(ctx, name); e != nil {
		return e
	} else if !live {
		if blocked, e := peerStopPending(deps, id); e != nil {
			return e
		} else if blocked {
			return errAgentTUIInFlight
		}
	}
	if a.TerminalID != nil {
		key = *a.TerminalID
	}
	if err := stopInteractivePane(ctx, deps, name, key); err != nil {
		return err
	}
	if a.TerminalID != nil {
		if deps.TermStates != nil {
			deps.TermStates.Drop(key)
		}
		if deps.TermRuntimes != nil {
			deps.TermRuntimes.Drop(key)
		}
		if t, e := deps.Store.GetTerminal(key); e == nil {
			publishTerminalState(deps, requestWith(ctx), t, false)
			syncManagedCLIInbox(deps, key, TermIdle)
		}
	}
	return nil
}

// Caller holds the agent lifecycle lock. All RPC entry points use this guard,
// including implicit starts after a timed-out interactive shutdown.
func (deps Deps) startAgentRPC(ctx context.Context, id, cwd string) error {
	if deps.Tmux != nil && deps.Tmux.Available() {
		live, err := deps.Tmux.HasSession(ctx, deps.agentSession(id))
		if err != nil {
			return err
		}
		if live {
			return errors.New("Close this agent's terminal before starting chat.")
		}
	}
	a, err := deps.Store.GetAgent(id)
	if err != nil {
		return err
	}
	keys := []string{id}
	if a.TerminalID != nil {
		keys = append(keys, *a.TerminalID)
	}
	for _, key := range keys {
		if blocked, e := peerStopPending(deps, key); e != nil {
			return e
		} else if blocked {
			return errAgentTUIInFlight
		}
	}
	return deps.Runtime.Start(id, cwd)
}

func boundAgentLock(deps Deps, termID string) func() {
	if a, err := deps.Store.AgentByTerminal(termID); err == nil {
		return terminalLock(deps, "agent:"+a.ID)
	}
	return func() {}
}

func agentAwareResumeLaunch(deps Deps, v *store.TerminalLaunch) *store.TerminalLaunch {
	if v != nil && v.CLI == "pi" {
		if a, e := deps.Store.AgentByTerminal(v.TerminalID); e == nil && a.IsPi() {
			return v
		}
	}
	return launchWithPinnedSession(v)
}

// agentSession is the single address resolver for an agent's interactive
// process: its bound terminal's session. An agent with no terminal answers
// with its own name, which no live session carries outside tests (the
// pre-ADR-0162 sessions it once addressed were retired 2026-09-25).
func (deps Deps) agentSession(id string) string {
	if deps.Store != nil {
		if a, err := deps.Store.GetAgent(id); err == nil && a.TerminalID != nil {
			return tmux.ShellSessionName(*a.TerminalID)
		}
	}
	return tmux.SessionName(id)
}

func (deps Deps) preparePiInteractive(ctx context.Context, agent store.Agent, cwd string) (*preparedCLILaunch, store.Agent, error) {
	a, err := deps.Store.EnsureAgentTerminal(agent.ID, cwd)
	if err != nil {
		return nil, agent, err
	}
	v, err := deps.Store.TerminalLaunch(*a.TerminalID)
	if err != nil {
		return nil, a, err
	}
	p, err := prepareCLITerminal(deps, cwd, v)
	return p, a, err
}

func agentTerminalCwd(deps Deps, termID, fallback string) string {
	if a, e := deps.Store.AgentByTerminal(termID); e == nil && a.IsPi() {
		if w, e := deps.Store.GetWorkspace(a.WorkspaceID); e == nil {
			return store.AgentCwd(w, a)
		}
	}
	return fallback
}

func routeBoundPi(deps Deps, w http.ResponseWriter, r *http.Request, handler http.HandlerFunc) bool {
	a, e := deps.Store.AgentByTerminal(r.PathValue("id"))
	if e != nil || !a.IsPi() {
		return false
	}
	if deps.runMode(r, a.ID) != modeInteractive {
		writeErr(w, http.StatusConflict, "This agent is not running in a terminal.")
		return true
	}
	r = r.Clone(r.Context())
	r.SetPathValue("id", a.ID)
	handler(w, r)
	return true
}
