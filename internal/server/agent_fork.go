package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/transcript"
)

// Fork agent… (POST /api/agents/{id}/fork-agent; "Fork" in
// docs/architecture/cli-session-handoff.md): a new agent of the same CLI
// opens a copy of the source agent's pinned conversation (ADR-0084)
// through the vendor's own fork, and opens waiting: the person gives it its
// first task there, like any agent (owner, 2026-09-25: a task field in the
// dialog cost more than it saved).
// The source keeps running untouched — a fork only reads its session file
// — so there is no live-holder check here, unlike a handoff. Where the copy
// works is the caller's choice: the source's folder, or a worktree the
// browser created first through ADR-0096's visible door (this handler never
// runs git).

// maxForkBody bounds the request: a name and a folder.
const maxForkBody = 16 << 10

type forkRequest struct {
	Name string `json:"name"`
	// WorkPath is the folder the fork works in; "" is the source's own.
	WorkPath string `json:"workPath"`
}

func handleForkAgent(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req forkRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxForkBody))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		out, status, err := forkAgent(deps, r, r.PathValue("id"), req)
		if err != nil {
			writeErr(w, status, err.Error())
			return
		}
		writeJSON(w, status, out)
	}
}

func forkAgent(deps Deps, r *http.Request, id string, req forkRequest) (map[string]any, int, error) {
	agent, err := deps.Store.GetAgent(id)
	if err != nil {
		return nil, storeStatus(err), err
	}
	if agent.IsPi() {
		return forkPiAgent(deps, r, agent, req)
	}
	if agent.TerminalID == nil || *agent.TerminalID == "" {
		return nil, http.StatusBadRequest, errors.New("Only an agent running a CLI in its terminal can be forked.")
	}
	launch, err := deps.Store.TerminalLaunch(*agent.TerminalID)
	if err != nil || launch == nil {
		return nil, http.StatusBadRequest, errors.New("Only an agent running a CLI in its terminal can be forked.")
	}
	ls := launch.LastSession
	if ls == nil || (ls.SessionID == "" && ls.Path == "") {
		// A CLI with no runtime integration (Muse Code, Antigravity) pins
		// only at stop time; resolve its conversation now, the same way.
		if t, err := deps.Store.GetTerminal(*agent.TerminalID); err == nil {
			pinCLITerminalLastSession(deps, t.ID, t)
			pinLiveSession(deps, r.Context(), launch.CLI, t)
			if again, err := deps.Store.TerminalLaunch(t.ID); err == nil && again != nil {
				launch, ls = again, again.LastSession
			}
		}
	}
	if ls == nil || (ls.SessionID == "" && ls.Path == "") {
		return nil, http.StatusConflict, errors.New("This agent has no conversation to fork yet.")
	}
	cli, ok := clilaunch.Find(launch.CLI)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("Unknown CLI.")
	}
	forker, flagFork := clisession.ForkerFor(cli.ID)
	sessionForker, protocolFork := clisession.SessionForkerFor(cli.ID)
	if !(flagFork || protocolFork) || (ls.CLI != "" && ls.CLI != cli.ID) {
		return nil, http.StatusBadRequest, errors.New(cli.Name + " can't fork a conversation from PiCode yet.")
	}

	cwd := strings.TrimSpace(req.WorkPath)
	if cwd == "" {
		cwd = ls.Cwd
	}
	if cwd == "" {
		if cwd, err = agentCwd(deps, agent.ID); err != nil {
			return nil, storeStatus(err), err
		}
	}
	if err := launchFolderExists(cwd); err != nil {
		return nil, http.StatusBadRequest, err
	}
	cwd = filepath.Clean(cwd)

	src := clisession.Ref{ID: ls.SessionID, Path: ls.Path, Cwd: ls.Cwd}
	var fork clisession.Fork
	if flagFork {
		fork = forker.ForkArgs(src, transcript.NewID())
	} else {
		// A protocol fork makes the copy before any agent exists; its launch
		// only reopens it.
		srcCwd := ls.Cwd
		if srcCwd == "" {
			srcCwd = cwd
		}
		fork, err = sessionForker.ForkSession(r.Context(), src, cliCommand(deps, cli, srcCwd))
		if err != nil {
			return nil, http.StatusBadGateway, err
		}
	}
	// Resume's rule (launchWithPinnedSession): the arguments are the
	// recipe, every other launch setting of the source carries over.
	overrides := launch.Overrides
	overrides.Args = &fork.Args

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = agent.Name + " fork"
	}
	forked, view, status, err := createCLIAgent(deps, r, cli, cliTerminalRequest{Name: name, WorkspaceID: agent.WorkspaceID, Cwd: cwd, Overrides: overrides})
	if err != nil {
		return nil, status, err
	}
	// The copy's launch arguments are the fork recipe, so a restart before
	// the CLI reports its session would fork again. A CLI that took the
	// pre-assigned id is pinned now (a restart resumes the copy); the others
	// pin on their first turn report (ADR-0084).
	if fork.ID != "" {
		_ = deps.Store.SetTerminalLastSession(*forked.TerminalID, store.TerminalLastSession{CLI: cli.ID, SessionID: fork.ID, Cwd: cwd, Name: name, ResumeArgs: fork.ResumeArgs})
	}
	out := map[string]any{"agent": agentView{Agent: forked, Mode: string(modeStopped)}, "terminal": view}
	sourceID := ls.SessionID
	if sourceID == "" {
		sourceID = ls.Path
	}
	row := store.SessionHandoff{
		SourceCLI: cli.ID, SourceID: sourceID, SourcePath: ls.Path,
		TargetCLI: cli.ID, TargetID: fork.ID,
		Mode: "fork", Window: "all", Manifest: forkManifest(agent),
		TerminalID: *forked.TerminalID, AgentID: forked.ID,
	}
	if saved, err := deps.Store.AddSessionHandoff(row); err == nil {
		out["handoff"] = saved
	}
	return out, http.StatusCreated, nil
}

// forkManifest names the source agent on a fork's lineage row: the session
// ids say which conversation was copied, this says whose, so the sidebar
// can show "fork of <name>" even after the source agent is gone.
func forkManifest(source store.Agent) json.RawMessage {
	raw, err := json.Marshal(map[string]string{"sourceAgentId": source.ID, "sourceAgentName": source.Name})
	if err != nil {
		return json.RawMessage("{}")
	}
	return raw
}

// forkOrigin is what an agent row shows about the agent it was forked from:
// the current name while that agent exists, the name it had at fork time
// (and Gone) once it was removed.
type forkOrigin struct {
	AgentID string `json:"agentId"`
	Name    string `json:"name"`
	Gone    bool   `json:"gone,omitempty"`
}

// forkOrigins maps each forked agent to its source, read from the lineage
// rows (session_handoffs, mode "fork"). Newest row wins for an agent; the
// read is best effort — a store error leaves the sidebar without the line.
func forkOrigins(deps Deps) map[string]*forkOrigin {
	rows, err := deps.Store.SessionHandoffs(1000)
	if err != nil {
		return nil
	}
	out := map[string]*forkOrigin{}
	for _, h := range rows {
		if h.Mode != "fork" || h.AgentID == "" || out[h.AgentID] != nil {
			continue
		}
		var m struct {
			SourceAgentID   string `json:"sourceAgentId"`
			SourceAgentName string `json:"sourceAgentName"`
		}
		if json.Unmarshal(h.Manifest, &m) != nil || m.SourceAgentID == "" {
			continue
		}
		o := &forkOrigin{AgentID: m.SourceAgentID, Name: m.SourceAgentName}
		if a, err := deps.Store.GetAgent(m.SourceAgentID); err == nil {
			o.Name = a.Name
		} else {
			o.Gone = true
		}
		out[h.AgentID] = o
	}
	return out
}

// cliCommand builds the CLI's own command the way cliRunner runs it
// (ADR-0094): its configured executable and environment, in dir. A
// SessionForker drives the process's stdio itself.
func cliCommand(deps Deps, cli clilaunch.CLI, dir string) func(ctx context.Context, args ...string) *exec.Cmd {
	return func(ctx context.Context, args ...string) *exec.Cmd {
		binary := cli.Command
		env := os.Environ()
		if c, err := cliConfig(deps, cli.ID); err == nil {
			if b, err := resolveCLIExecutable(cli, c); err == nil {
				binary = b
			}
			env = cliEnvironment(c)
		}
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = dir
		cmd.Env = env
		cmd.WaitDelay = time.Second
		return cmd
	}
}

// pinLiveSession pins the conversation a running TUI is writing when its
// CLI records it only at exit and has not been pinned otherwise (Muse
// Code: clisession.LiveSessionFinder). Best effort: nothing found leaves
// the terminal unpinned and the fork answers "no conversation yet".
func pinLiveSession(deps Deps, ctx context.Context, cliID string, t store.Terminal) {
	launch, err := deps.Store.TerminalLaunch(t.ID)
	if err != nil || launch == nil || (launch.LastSession != nil && launch.LastSession.SessionID != "") {
		return
	}
	src, ok := clisession.Get(cliID)
	if !ok {
		return
	}
	finder, ok := src.(clisession.LiveSessionFinder)
	if !ok {
		return
	}
	cli, ok := clilaunch.Find(cliID)
	if !ok {
		return
	}
	// The current run's start, not the terminal's birth: a terminal
	// restarted in a shared folder must not claim an earlier run's session.
	since, _ := time.Parse(time.RFC3339Nano, t.CreatedAt)
	if launch.Applied != nil {
		if at, err := time.Parse(time.RFC3339Nano, launch.Applied.StartedAt); err == nil {
			since = at
		}
	}
	taken := takenSessions(deps, cliID, t.ID)
	id, resume, err := finder.LiveSession(ctx, t.Cwd, since, func(id string) bool { return taken[id] }, cliCommand(deps, cli, t.Cwd))
	if err != nil || id == "" {
		return
	}
	_ = deps.Store.SetTerminalLastSession(t.ID, store.TerminalLastSession{CLI: cliID, SessionID: id, Cwd: t.Cwd, ResumeArgs: resume})
}

// takenSessions are the cli's sessions PiCode knows belong somewhere else:
// pinned by another terminal, or the copy a fork made (its lineage row's
// target). In a shared folder they are newer than the conversation a
// terminal is running, so a live lookup must not mistake one for it.
func takenSessions(deps Deps, cliID, termID string) map[string]bool {
	out := map[string]bool{}
	if terms, err := deps.Store.ListTerminals(); err == nil {
		for _, t := range terms {
			if t.ID == termID {
				continue
			}
			if l, err := deps.Store.TerminalLaunch(t.ID); err == nil && l != nil && l.LastSession != nil && l.LastSession.CLI == cliID && l.LastSession.SessionID != "" {
				out[l.LastSession.SessionID] = true
			}
		}
	}
	if rows, err := deps.Store.SessionHandoffs(1000); err == nil {
		for _, h := range rows {
			if h.TargetCLI == cliID && h.TargetID != "" {
				out[h.TargetID] = true
			}
		}
	}
	return out
}
