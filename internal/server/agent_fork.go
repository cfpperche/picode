package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/transcript"
)

// Fork agent… (POST /api/agents/{id}/fork-agent; "Fork" in
// docs/architecture/cli-session-handoff.md): a new agent of the same CLI
// opens a copy of the source agent's pinned conversation (ADR-0084)
// through the vendor's own fork, and starts on the task the user gave it.
// The source keeps running untouched — a fork only reads its session file
// — so there is no live-holder check here, unlike a handoff. Where the copy
// works is the caller's choice: the source's folder, or a worktree the
// browser created first through ADR-0096's visible door (this handler never
// runs git).

// maxForkFiles and maxForkBody bound the inline attachments: the attach
// bar's own limits (four files, 4 MB each), base64 included.
const (
	maxForkFiles = 4
	maxForkBody  = maxForkFiles*maxDropBytes*4/3 + 64<<10
)

type forkRequest struct {
	Name string `json:"name"`
	// Prompt is the fork's first task; "" opens the copy waiting.
	Prompt string `json:"prompt"`
	// WorkPath is the folder the fork works in; "" is the source's own.
	WorkPath string `json:"workPath"`
	// Files are staged into the fork's folder (.picode/drop/) and named in
	// the prompt, as the attach bar does for a running terminal.
	Files []dropBody `json:"files"`
	// Paths are files already in the folder, relative to it.
	Paths []string `json:"paths"`
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
	if agent.TerminalID == nil || *agent.TerminalID == "" {
		return nil, http.StatusBadRequest, errors.New("Only an agent running a CLI in its terminal can be forked.")
	}
	launch, err := deps.Store.TerminalLaunch(*agent.TerminalID)
	if err != nil || launch == nil {
		return nil, http.StatusBadRequest, errors.New("Only an agent running a CLI in its terminal can be forked.")
	}
	ls := launch.LastSession
	if ls == nil || (ls.SessionID == "" && ls.Path == "") {
		return nil, http.StatusConflict, errors.New("This agent has no conversation to fork yet.")
	}
	cli, ok := clilaunch.Find(launch.CLI)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("Unknown CLI.")
	}
	forker, ok := clisession.ForkerFor(cli.ID)
	if !ok || (ls.CLI != "" && ls.CLI != cli.ID) {
		return nil, http.StatusBadRequest, errors.New(cli.Name + " can't fork a conversation from PiCode yet.")
	}
	if len(req.Files)+len(req.Paths) > maxForkFiles {
		return nil, http.StatusBadRequest, errors.New("Up to 4 files.")
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

	// Attachments first: a file that cannot be staged refuses the fork
	// before an agent exists, so the dialog keeps everything and says why.
	paths, err := checkPromptPaths(cwd, req.Paths)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	for _, f := range req.Files {
		raw, err := decodeDropData(f.Data)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		saved, err := writeDropFile(cwd, f.Name, raw)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		paths = append(paths, saved["path"].(string))
	}
	prompt, err := forkPrompt(req.Prompt, paths)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	fork := forker.ForkArgs(clisession.Ref{ID: ls.SessionID, Path: ls.Path, Cwd: ls.Cwd}, prompt, transcript.NewID())
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
	// the CLI reports its session would fork again and resend the task. A
	// CLI that took the pre-assigned id is pinned now (a restart resumes the
	// copy); the others pin on their first turn report (ADR-0084).
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

// forkPrompt is the task as one launch argument. A launch setting is one
// line of at most 8192 characters (clilaunch.Validate: it is stored and
// shown as the terminal's launch), so line breaks become spaces and each
// attachment follows as @path, which the CLI reads like a mention. A task
// that begins with a dash would reach the CLI's flag parser; a leading
// space is invisible to the model and keeps it an operand for every parser.
func forkPrompt(message string, paths []string) (string, error) {
	parts := strings.Fields(message)
	for _, p := range paths {
		if !strings.HasPrefix(p, "@") {
			p = "@" + p
		}
		parts = append(parts, p)
	}
	out := strings.Join(parts, " ")
	if strings.HasPrefix(out, "-") {
		out = " " + out
	}
	if len(out) > 8192 {
		return "", errors.New("The task is too long to start a fork with. Shorten it, or attach the details as a file.")
	}
	return out, nil
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
