package server

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// Serialize native session publication with runtime replacement. The recorded
// binding invalidates the previous conversation before a new live identity is shown.
func recordNativeTerminalSession(deps Deps, term, cli, run, id, path string, seq int64, state ...string) error {
	value := ""
	if len(state) > 0 {
		value = state[0]
	}
	return recordNativeTerminalObservation(deps, term, cli, run, id, path, seq, value, "", "")
}

func recordNativeTerminalObservation(deps Deps, term, cli, run, id, path string, seq int64, state, source, attention string, observed ...bool) error {
	if deps.TermRuntimes == nil || run == "" || id == "" || len(id) > 256 || len(path) > 4096 || strings.ContainsAny(id, "\r\n\x00") || seq <= 0 || seq > time.Now().Add(5*time.Second).UnixNano() {
		return errors.New("invalid native identity")
	}
	// A report that begins a turn counts it (ADR-0194) once every lock
	// below is released: this path writes the state itself, bypassing
	// reportTermStateForRun, and every integrated CLI reports through it.
	turn := false
	defer func() {
		if turn {
			noteTerminalTurn(deps, term)
		}
	}()
	unlock := boundAgentLock(deps, term)
	defer unlock()
	r := deps.TermRuntimes
	r.mu.Lock()
	defer r.mu.Unlock()
	live, ok := r.m[term]
	if !ok || live.RunID != run || live.CLI != cli || !processAlive(live) {
		return errors.New("stale native identity")
	}
	stale := seq <= live.SessionSeq
	// A held question is stamped when its card appeared, which can be older
	// than a sibling tool's completion in the same parallel batch. Accept the
	// hold without rewinding the identity fence; every other stale report is
	// rejected as before.
	if stale && attention == "" {
		return errors.New("stale native identity")
	}
	// Old launchers inject both paths. Once native hooks have spoken in this
	// incarnation, legacy notifications (including auxiliary threads) cannot
	// replace their identity or activity. A new incarnation starts fresh.
	if cli == "codex" && source == "codex-notify" && live.CodexHooks {
		return nil
	}
	// Session IDs are protocol values, not paths. Pi alone resumes by its file.
	if cli == "pi" && path == "" {
		return errors.New("missing Pi session path")
	}
	if !stale {
		if live.SessionID != id || live.SessionPath != path {
			launch, err := deps.Store.TerminalLaunch(term)
			if err != nil || launch == nil || launch.CLI != cli {
				return errors.New("terminal launch changed")
			}
			resume := []string{"--resume", id}
			switch cli {
			case "pi":
				resume = []string{"--session", path}
			case "codex":
				resume = []string{"resume", id}
			case "opencode":
				resume = []string{"--session", id}
			}
			// Retain catalog presentation when it describes this exact native session.
			last := store.TerminalLastSession{CLI: cli, SessionID: id, Path: path, ResumeArgs: resume}
			if launch.LastSession != nil && launch.LastSession.SessionID == id {
				last = *launch.LastSession
				last.ResumeArgs = resume
				if path != "" {
					last.Path = path
				}
			}
			if err := deps.Store.SetTerminalLastSession(term, last); err != nil {
				return err
			}
			if cli == "pi" {
				if a, e := deps.Store.AgentByTerminal(term); e == nil && a.IsPi() && (a.SessionPath == nil || *a.SessionPath != path) {
					if _, e = deps.Store.UpdateAgent(a.ID, store.AgentPatch{SessionPath: &path}); e != nil {
						return e
					}
				}
			}
		}
		live.SessionID, live.SessionPath, live.SessionSeq = id, path, seq
		if len(observed) > 0 && observed[0] {
			live.Observation = true
		}
		if cli == "codex" && source == "codex-hook" {
			live.CodexHooks = true
		}
		r.m[term] = live
	}
	if state != "" && deps.TermStates != nil {
		deps.TermStates.mu.Lock()
		prev, had := deps.TermStates.m[term]
		if !had || !heldQuestion(prev, state, attention) {
			sessionID, sessionSeq := id, seq
			if stale {
				// The hold is older than the binding it describes; keep the
				// runtime's newest identity rather than rewinding it.
				sessionID, sessionSeq = live.SessionID, live.SessionSeq
			}
			turn = startsTurn(prev, had, state)
			deps.TermStates.m[term] = TermState{State: state, CLI: cli, RunID: run, SessionID: sessionID, SessionSeq: sessionSeq, Attention: attentionFor(state, attention), At: time.Now()}
		}
		deps.TermStates.mu.Unlock()
	}
	return nil
}

// A native event can renew a wrapper after daemon restart, but cannot replace
// another wrapper incarnation. Match the current pane and process ancestry;
// no session-file discovery participates in this recovery.
func recoverNativeRuntime(ctx context.Context, deps Deps, term, cli, run string, pid int) {
	if deps.TermRuntimes == nil || deps.Tmux == nil || pid <= 0 || run == "" || len(run) > runtimeRunIDCap || normalizeTerminalCLI(cli) == "" {
		return
	}
	if rt, ok := deps.TermRuntimes.Get(term); ok && rt.Source == "wrapper" {
		return
	}
	pane, err := deps.Tmux.InputSnapshot(ctx, tmux.ShellSessionName(term))
	if err != nil {
		return
	}
	procs := readProcSnapshot()
	descendant := func(child, parent int) bool {
		for n := 0; n < 256 && child > 0; n++ {
			if child == parent {
				return true
			}
			child = procs.ppid[child]
		}
		return false
	}
	if !descendant(pid, pane.PanePID) {
		return
	}
	rt := TermRuntime{CLI: cli, Source: "wrapper", RunID: run, PID: pid, ProcStart: processStartToken(pid), StartedAt: time.Now()}
	if rt.ProcStart == "" || !processAlive(rt) {
		return
	}
	r := deps.TermRuntimes
	r.mu.Lock()
	previous, had := r.m[term]
	if had && (previous.Source == "wrapper" || previous.CLI != cli || !descendant(previous.PID, pid)) {
		r.mu.Unlock()
		return
	}
	r.m[term] = rt
	r.mu.Unlock()
	publishTermRuntime(deps, "started", rt, term)
}
