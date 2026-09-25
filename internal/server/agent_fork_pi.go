package server

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/transcript"
)

// forkPiAgent is Fork agent… for a Pi agent. A Pi agent owns its
// conversation file (`--session` is reserved on its launch, SessionPath)
// and keeps its sessions in a private folder (ADR-0040), so the copy is
// made by pi's own `--fork` in the new agent's folder after the agent
// exists and before it starts, and becomes that agent's session: a restart
// reopens the copy, never forks again. The task, line breaks and all,
// travels through the prompt door once the TUI is ready (deliverForkTask).
func forkPiAgent(deps Deps, r *http.Request, agent store.Agent, req forkRequest) (map[string]any, int, error) {
	forker, ok := clisession.AgentForkerFor(store.CLIPi)
	cli, found := clilaunch.Find(store.CLIPi)
	if !ok || !found {
		return nil, http.StatusBadRequest, errors.New("Pi can't fork a conversation from PiCode yet.")
	}
	src := piAgentSessionFile(deps, agent)
	if src == "" {
		return nil, http.StatusConflict, errors.New("This agent has no conversation to fork yet.")
	}
	if len(req.Files)+len(req.Paths) > maxForkFiles {
		return nil, http.StatusBadRequest, errors.New("Up to 4 files.")
	}
	cwd := strings.TrimSpace(req.WorkPath)
	if cwd == "" {
		var err error
		if cwd, err = agentCwd(deps, agent.ID); err != nil {
			return nil, storeStatus(err), err
		}
	}
	if err := launchFolderExists(cwd); err != nil {
		return nil, http.StatusBadRequest, err
	}
	cwd = filepath.Clean(cwd)
	paths, err := forkAttachments(cwd, req)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	task := buildPromptPaste(req.Prompt, paths)

	// The source's other launch settings carry over, as for every fork.
	var overrides clilaunch.Overrides
	if agent.TerminalID != nil && *agent.TerminalID != "" {
		if launch, err := deps.Store.TerminalLaunch(*agent.TerminalID); err == nil && launch != nil {
			overrides = launch.Overrides
		}
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = agent.Name + " fork"
	}
	newID := transcript.NewID()
	ref := clisession.Ref{ID: piSessionID(src), Path: src, Cwd: cwd}
	forked, view, status, err := createCLIAgent(deps, r, cli, cliTerminalRequest{
		Name: name, WorkspaceID: agent.WorkspaceID, Cwd: cwd, Overrides: overrides,
		session: func(a store.Agent) (string, error) {
			return forker.ForkIntoDir(r.Context(), ref, session.AgentDir(a.ID), newID, cliCommand(deps, cli, cwd))
		},
	})
	if err != nil {
		return nil, status, err
	}
	out := map[string]any{"agent": agentView{Agent: forked, Mode: string(modeStopped)}, "terminal": view}
	if strings.TrimSpace(task) != "" && forked.TerminalID != nil {
		out["task"] = "pending"
		go deliverForkTask(deps, *forked.TerminalID, forked.WorkspaceID, name, task)
	}
	row := store.SessionHandoff{
		SourceCLI: cli.ID, SourceID: ref.ID, SourcePath: src,
		TargetCLI: cli.ID, TargetID: newID,
		Mode: "fork", Window: "all", Manifest: forkManifest(agent),
		AgentID: forked.ID,
	}
	if forked.TerminalID != nil {
		row.TerminalID = *forked.TerminalID
	}
	if saved, err := deps.Store.AddSessionHandoff(row); err == nil {
		out["handoff"] = saved
	}
	return out, http.StatusCreated, nil
}

// piAgentSessionFile is the Pi agent's current conversation file: its
// pinned SessionPath, else a pending id that has a file, else the newest
// file in its private folder — the same order the spawn and the exit use.
func piAgentSessionFile(deps Deps, a store.Agent) string {
	if a.SessionPath != nil && *a.SessionPath != "" {
		if _, err := os.Stat(*a.SessionPath); err == nil {
			return *a.SessionPath
		}
	}
	if p := deps.Store.ResolvePendingAgentSession(a.ID); p != "" {
		return p
	}
	a.SessionPath = nil
	return piSessionFallback(a)
}

// piSessionID is the id pi puts in a session file's name: <time>_<id>.jsonl.
func piSessionID(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	if _, id, ok := strings.Cut(base, "_"); ok {
		return id
	}
	return base
}
