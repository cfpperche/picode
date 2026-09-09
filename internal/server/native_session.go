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
	if deps.TermRuntimes == nil || run == "" || id == "" || len(id) > 256 || len(path) > 4096 || strings.ContainsAny(id, "\r\n\x00") || seq <= 0 || seq > time.Now().Add(5*time.Second).UnixNano() {
		return errors.New("invalid native identity")
	}
	r := deps.TermRuntimes
	r.mu.Lock()
	defer r.mu.Unlock()
	live, ok := r.m[term]
	if !ok || live.RunID != run || live.CLI != cli || !processAlive(live) || seq <= live.SessionSeq {
		return errors.New("stale native identity")
	}
	// Session IDs are protocol values, not paths. Pi alone resumes by its file.
	if cli == "pi" && path == "" {
		return errors.New("missing Pi session path")
	}
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
	}
	live.SessionID, live.SessionPath, live.SessionSeq = id, path, seq
	r.m[term] = live
	if len(state) > 0 && deps.TermStates != nil {
		deps.TermStates.mu.Lock()
		deps.TermStates.m[term] = TermState{State: state[0], CLI: cli, RunID: run, SessionID: id, SessionSeq: seq, At: time.Now()}
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
