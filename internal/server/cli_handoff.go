package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/transcript"
)

// Cross-CLI session handoff (ADR-0088): continue a conversation recorded by
// one Agent CLI in another one.
//
//	POST /api/clis/{cli}/sessions/handoff/preview   read-only: what would travel, what would be left behind
//	POST /api/clis/{cli}/sessions/handoff           translate, create the target's native session (or a brief),
//	                                                open a CLI terminal (or adopt a pi agent), record lineage
//
// One plan function serves both, so the preview can never disagree with
// the execution (ADR-0070). Decision table (covered by
// TestHandoffDecisionTable):
//
//	source CLI has no Reader                        → 409 "Sessions of <src> cannot be read yet."
//	target has Writer, mode native or ""            → native
//	target has no Writer, has Prompter, mode native → 409 (preview lists modes:["brief"])
//	target has no Writer, has Prompter, mode brief/"" → brief
//	target has Writer and Prompter, mode brief      → brief
//	target has neither                              → 409 "<dst> cannot receive a handoff yet."
//	native, target format unknown                   → 409 ErrUnknownFormat (brief still offered)
//	source is live (agent running / terminal open), force=false → 409 with `live`
//	source is live, force=true                      → proceeds, manifest warning
//	target not installed (non-pi)                   → 400
//	source over the read cap                        → 413
//	to == cli                                       → 400 "Choose a different CLI."
//	window=recent without a compaction              → whole timeline
//	target pi, native                               → stopped managed agent, no terminal
//	unknown {cli} / unknown `to`                    → 404
type handoffRequest struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	Cwd         string `json:"cwd"`
	To          string `json:"to"`
	Mode        string `json:"mode"`   // native | brief | "" (best available)
	Window      string `json:"window"` // recent (default) | all
	Tools       string `json:"tools"`  // native (default) | text
	Force       bool   `json:"force"`
	WorkspaceID string `json:"workspaceId"`
}

// liveHolder names who is still writing to the source session.
type liveHolder struct {
	Kind string `json:"kind"` // terminal | agent
	ID   string `json:"id"`
	Name string `json:"name"`
}

type handoffPlan struct {
	req      handoffRequest
	ref      clisession.Ref
	src, dst clilaunch.CLI
	writer   clisession.Writer
	prompter clisession.Prompter
	full     transcript.Timeline // as read
	windowed transcript.Timeline // after Window
	summary  string              // compaction summary the window dropped behind
	mode     string
	modes    []string
	version  string // installed target version, "" = writer probes
	live     *liveHolder
	now      time.Time
}

const handoffModeNative, handoffModeBrief = "native", "brief"

func registerCLIHandoffRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("POST /api/clis/{cli}/sessions/handoff/preview", handleCLIHandoff(deps, false))
	mux.HandleFunc("POST /api/clis/{cli}/sessions/handoff", handleCLIHandoff(deps, true))
}

func handleCLIHandoff(deps Deps, execute bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		src, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		var req handoffRequest
		if !readCLIJSON(w, r, &req) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		plan, status, err := planHandoff(ctx, deps, src, req)
		if err != nil {
			body := map[string]any{"error": err.Error()}
			if plan != nil && plan.live != nil {
				body["live"] = plan.live
			}
			if plan != nil && len(plan.modes) > 0 {
				body["modes"] = plan.modes
			}
			writeJSON(w, status, body)
			return
		}
		if !execute {
			writeJSON(w, http.StatusOK, plan.preview(deps.DataDir))
			return
		}
		if plan.live != nil && !req.Force {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": plan.live.Name + " is still writing to this session.",
				"live":  plan.live,
				"modes": plan.modes,
			})
			return
		}
		if deps.Tmux == nil || !deps.Tmux.Available() {
			if !(plan.mode == handoffModeNative && plan.dst.ID == "pi") {
				writeErr(w, http.StatusServiceUnavailable, "Install tmux to open a terminal.")
				return
			}
		}
		unlock := terminalLock(deps, "handoff:"+src.ID+":"+plan.ref.ID)
		defer unlock()
		body, status, err := plan.commit(ctx, deps, r)
		if err != nil {
			writeErr(w, status, err.Error())
			return
		}
		writeJSON(w, status, body)
	}
}

// planHandoff resolves the request against the capabilities of both CLIs
// and reads the source. It never writes. The returned plan is non-nil on
// the 409 branches that carry extra payload (live holder, offered modes).
func planHandoff(ctx context.Context, deps Deps, src clilaunch.CLI, req handoffRequest) (*handoffPlan, int, error) {
	dst, ok := clilaunch.Find(strings.TrimSpace(req.To))
	if !ok {
		return nil, http.StatusNotFound, errors.New("Unknown target CLI.")
	}
	if dst.ID == src.ID {
		return nil, http.StatusBadRequest, errors.New("Choose a different CLI.")
	}
	reader, ok := clisession.ReaderFor(src.ID)
	if !ok {
		return nil, http.StatusConflict, errors.New("Sessions of " + src.Name + " cannot be read yet.")
	}
	p := &handoffPlan{req: req, src: src, dst: dst, now: time.Now().UTC()}
	p.writer, _ = clisession.WriterFor(dst.ID)
	p.prompter, _ = clisession.PrompterFor(dst.ID)
	if p.writer != nil {
		p.modes = append(p.modes, handoffModeNative)
	}
	if p.prompter != nil {
		p.modes = append(p.modes, handoffModeBrief)
	}
	if len(p.modes) == 0 {
		return nil, http.StatusConflict, errors.New(dst.Name + " cannot receive a handoff yet.")
	}
	switch req.Mode {
	case "":
		p.mode = p.modes[0]
	case handoffModeNative:
		if p.writer == nil {
			return p, http.StatusConflict, errors.New(dst.Name + " has no native import; use a brief.")
		}
		p.mode = handoffModeNative
	case handoffModeBrief:
		if p.prompter == nil {
			return p, http.StatusConflict, errors.New(dst.Name + " cannot start from a brief.")
		}
		p.mode = handoffModeBrief
	default:
		return nil, http.StatusBadRequest, errors.New("mode must be native or brief")
	}
	switch req.Window {
	case "":
		p.req.Window = "recent"
	case "recent", "all":
	default:
		return nil, http.StatusBadRequest, errors.New("window must be recent or all")
	}
	switch req.Tools {
	case "":
		p.req.Tools = clisession.ToolsNative
	case clisession.ToolsNative, clisession.ToolsText:
	default:
		return nil, http.StatusBadRequest, errors.New("tools must be native or text")
	}
	p.ref = clisession.Ref{ID: strings.TrimSpace(req.ID), Path: strings.TrimSpace(req.Path), Cwd: strings.TrimSpace(req.Cwd)}
	if p.ref.ID == "" && p.ref.Path == "" {
		return nil, http.StatusBadRequest, errors.New("session id or path required")
	}
	full, err := reader.Read(ctx, p.ref)
	switch {
	case errors.Is(err, clisession.ErrNotUnderRoot):
		return nil, http.StatusBadRequest, err
	case errors.Is(err, clisession.ErrTooLarge):
		return nil, http.StatusRequestEntityTooLarge, err
	case errors.Is(err, clisession.ErrUnknownFormat):
		return nil, http.StatusConflict, err
	case err != nil:
		if os.IsNotExist(err) {
			return nil, http.StatusNotFound, errors.New("That session is gone.")
		}
		return nil, http.StatusInternalServerError, err
	}
	full.Header.SourceName = src.Name
	if full.Header.Cwd == "" {
		full.Header.Cwd = p.ref.Cwd
	}
	if strings.TrimSpace(full.Header.Cwd) == "" {
		return nil, http.StatusBadRequest, errors.New("This session has no folder.")
	}
	if p.ref.ID == "" {
		p.ref.ID = full.Header.SourceID
	}
	p.full = full
	p.windowed, p.summary = full.Window(p.req.Window == "recent")
	if p.windowed.Counts().Messages == 0 && len(p.windowed.Events) == 0 {
		return nil, http.StatusConflict, errors.New("Nothing to hand off: the conversation has no turns.")
	}
	p.version = resolveFormatVersion(ctx, deps, dst)
	p.live = liveHolderFor(deps, src.ID, p.ref, p.full)
	if p.live != nil && req.Force {
		p.windowed.Manifest.Warn("%s was still writing to the source when it was translated; the newest turns may be missing.", p.live.Name)
	}
	// The target must be launchable before anything is created, except a
	// pi handoff, which becomes a stopped agent and needs no terminal.
	if !(p.mode == handoffModeNative && dst.ID == "pi") {
		if _, status, err := checkCLILaunch(deps, dst, clilaunch.Overrides{}); err != nil {
			return nil, status, err
		}
	}
	return p, 0, nil
}

// nativeTimeline is what a Writer receives: the handoff note first, then
// the windowed conversation.
func (p *handoffPlan) nativeTimeline() transcript.Timeline {
	t := p.windowed
	note := transcript.Event{Kind: transcript.KindMessage, Role: "user", Text: transcript.HandoffNote(p.full.Header, p.summary, p.now), Timestamp: p.now}
	t.Events = append([]transcript.Event{note}, t.Events...)
	return t
}

func (p *handoffPlan) brief() string {
	return transcript.Brief(p.windowed, p.summary, p.now)
}

func (p *handoffPlan) promptLine(briefPath string) string {
	return "Continue a session handed off from " + p.src.Name + ": read " + briefPath + " first, then carry on with the last request."
}

func (p *handoffPlan) preview(dataDir string) map[string]any {
	prepared := p.nativeTimeline().Prepare()
	out := map[string]any{
		"sourceCli":     p.src.ID,
		"sourceName":    p.src.Name,
		"targetCli":     p.dst.ID,
		"targetName":    p.dst.Name,
		"mode":          p.mode,
		"modes":         p.modes,
		"window":        p.req.Window,
		"tools":         p.req.Tools,
		"formatVersion": p.version,
		"hasCompaction": p.full.HasCompaction(),
		"source": map[string]any{
			"id":       p.full.Header.SourceID,
			"title":    p.full.Header.Title,
			"cwd":      p.full.Header.Cwd,
			"model":    p.full.Header.Model,
			"counts":   p.full.Counts(),
			"messages": p.full.Counts().Messages,
		},
		"counts":   prepared.Counts(),
		"manifest": prepared.Manifest,
		"live":     p.live,
	}
	if p.mode == handoffModeBrief {
		b := p.brief()
		out["briefPreview"] = b
		out["briefBytes"] = len(b)
		out["prompt"] = p.promptLine(filepath.Join(dataDir, "handoffs", "<id>", "brief.md"))
	}
	return out
}

// commit performs the handoff. Order: create the artifact (native session
// or brief), then the terminal or agent, then the lineage row — so the
// event carries the terminal id, and a failed launch still leaves a usable
// artifact plus a row pointing at it (the response carries launchError,
// as the plain create-terminal endpoint does).
func (p *handoffPlan) commit(ctx context.Context, deps Deps, r *http.Request) (map[string]any, int, error) {
	cwd := filepath.Clean(p.full.Header.Cwd)
	hid := transcript.NewID()
	row := store.SessionHandoff{
		ID:         hid,
		SourceCLI:  p.src.ID,
		SourceID:   p.ref.ID,
		SourcePath: p.full.Header.SourcePath,
		TargetCLI:  p.dst.ID,
		Mode:       p.mode,
		Window:     p.req.Window,
		Tools:      p.req.Tools,
	}
	workspaceID := strings.TrimSpace(p.req.WorkspaceID)
	if workspaceID == "" {
		workspaceID = workspaceOwning(deps, cwd)
	}
	out := map[string]any{"mode": p.mode, "targetCli": p.dst.ID}
	var manifest transcript.Manifest
	var args []string
	name := p.dst.Name + " · from " + p.src.Name
	if t := strings.TrimSpace(p.full.Header.Title); t != "" {
		name += " · " + clipRunes(t, 32)
	}

	switch p.mode {
	case handoffModeNative:
		tl := p.nativeTimeline()
		manifest = tl.Prepare().Manifest
		wreq := clisession.WriteRequest{Cwd: cwd, FormatVersion: p.version, Tools: p.req.Tools, Now: p.now, Run: cliRunner(deps, p.dst, cwd)}
		if p.dst.ID == "pi" {
			wsID, work := adoptHome(deps, cwd)
			agentName := strings.TrimSpace(p.full.Header.Title)
			if agentName == "" {
				agentName = clipRunes(firstUserText(p.windowed), 60)
			}
			if agentName == "" {
				agentName = "From " + p.src.Name
			}
			agent, err := deps.Store.AddAgent(wsID, clipRunes(agentName, 60), work)
			if err != nil {
				return nil, http.StatusBadRequest, err
			}
			wreq.Dir = session.AgentDir(agent.ID)
			sum, err := p.writer.Write(ctx, tl, wreq)
			if err != nil {
				_ = deps.Store.DeleteAgent(agent.ID)
				return nil, writeStatus(err), err
			}
			agent, err = deps.Store.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &sum.Path})
			if err != nil {
				return nil, http.StatusInternalServerError, err
			}
			row.TargetID, row.TargetPath, row.AgentID = sum.ID, sum.Path, agent.ID
			out["target"] = sum
			out["agent"] = agentView{Agent: agent, Mode: string(modeStopped)}
			break
		}
		sum, err := p.writer.Write(ctx, tl, wreq)
		if err != nil {
			return nil, writeStatus(err), err
		}
		row.TargetID, row.TargetPath = sum.ID, sum.Path
		out["target"] = sum
		args = sum.ResumeArgs
	case handoffModeBrief:
		manifest = p.windowed.Manifest
		dir := filepath.Join(deps.DataDir, "handoffs", hid)
		path := filepath.Join(dir, "brief.md")
		if err := writeInterceptFile(path, []byte(p.brief()), 0o644); err != nil {
			return nil, http.StatusInternalServerError, err
		}
		sessionID := transcript.NewID()
		args = p.prompter.PromptArgs(p.promptLine(path), sessionID)
		if len(args) > 1 { // the CLI took the pre-assigned id
			row.TargetID = sessionID
		}
		row.TargetPath = path
		out["brief"] = map[string]any{"path": path, "bytes": len(p.brief())}
	}
	if raw, err := json.Marshal(manifest); err == nil {
		row.Manifest = raw
	}
	out["manifest"] = manifest

	if args != nil {
		term, view, status, err := createCLITerminal(deps, r, p.dst, cliTerminalRequest{Name: name, WorkspaceID: workspaceID, Cwd: cwd, Overrides: clilaunch.Overrides{Args: &args}})
		if err != nil {
			// The artifact exists; record where it is so the Sessions tab
			// can still open it, then report the launch problem.
			if saved, e := deps.Store.AddSessionHandoff(row); e == nil {
				out["handoff"] = saved
			}
			return nil, status, err
		}
		row.TerminalID = term.ID
		out["terminal"] = view
	}
	saved, err := deps.Store.AddSessionHandoff(row)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	out["handoff"] = saved
	return out, http.StatusCreated, nil
}

func writeStatus(err error) int {
	switch {
	case errors.Is(err, clisession.ErrUnknownFormat):
		return http.StatusConflict
	case errors.Is(err, clisession.ErrNoDir):
		return http.StatusInternalServerError
	}
	return http.StatusInternalServerError
}

// cliRunner lets a writer publish through the target CLI's own import
// command (ADR-0094): the configured executable and environment, run in
// the session's folder, bounded in time and output. It never runs a
// conversation — only the subcommand the writer names.
func cliRunner(deps Deps, cli clilaunch.CLI, cwd string) func(context.Context, ...string) ([]byte, error) {
	return func(ctx context.Context, args ...string) ([]byte, error) {
		c, err := cliConfig(deps, cli.ID)
		if err != nil {
			return nil, err
		}
		binary, err := resolveCLIExecutable(cli, c)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Dir = cwd
		cmd.Env = cliEnvironment(c)
		var out boundedCLIOutput
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.WaitDelay = time.Second
		err = cmd.Run()
		return out.Bytes(), err
	}
}

var semverToken = regexp.MustCompile(`\d+\.\d+\.\d+`)

// resolveFormatVersion is the installed target's version as the setup
// check records it (ADR-0070 `--version` probe), reduced to its semver
// token. A missing or stale check runs now — one `--version`, the same
// call the CLIs page makes — so a target that never ran on this machine
// still gets a native session. "" lets the writer probe the newest local
// artifact instead.
func resolveFormatVersion(ctx context.Context, deps Deps, cli clilaunch.CLI) string {
	if deps.Store == nil {
		return ""
	}
	d, err := deps.Store.CLICheck(cli.ID)
	stale := true
	if err == nil && d != nil {
		if c, err := cliConfig(deps, cli.ID); err == nil {
			exe, _ := resolveCLIExecutable(cli, c)
			stale = d.Fingerprint != clilaunch.Fingerprint(c) || d.Identity != executableIdentity(exe)
		}
	}
	if stale {
		unlock := terminalLock(deps, "cli-config")
		c, err := cliConfig(deps, cli.ID)
		if err != nil {
			unlock()
			return ""
		}
		seed := CLIDiagnostic{CheckedAt: time.Now().UTC().Format(time.RFC3339), Fingerprint: clilaunch.Fingerprint(c)}
		out, err := computeCLICheck(deps, cli, c, seed, ctx)
		unlock()
		if err != nil || out.StoreErr != nil {
			return ""
		}
		fresh := out.Diagnostic
		d = &fresh
	}
	if d == nil || d.Error != "" {
		return ""
	}
	return semverToken.FindString(d.Version)
}

// liveHolderFor finds who is still writing to the source: for pi, the
// agent whose current session it is (when running); for terminal CLIs,
// a running CLI terminal whose pinned session (ADR-0084) is this one.
func liveHolderFor(deps Deps, cli string, ref clisession.Ref, full transcript.Timeline) *liveHolder {
	if deps.Store == nil {
		return nil
	}
	if cli == "pi" {
		path := ref.Path
		if path == "" {
			path = full.Header.SourcePath
		}
		if u, ok := sessionUseBy(deps)[path]; ok {
			if deps.Runtime != nil && deps.Runtime.Get(u.AgentID) != nil {
				return &liveHolder{Kind: "agent", ID: u.AgentID, Name: u.AgentName}
			}
		}
		return nil
	}
	if deps.TermRuntimes == nil {
		return nil
	}
	for termID, rt := range deps.TermRuntimes.Snapshot() {
		if rt.CLI != cli {
			continue
		}
		launch, err := deps.Store.TerminalLaunch(termID)
		if err != nil || launch == nil || launch.LastSession == nil || launch.LastSession.SessionID != ref.ID {
			continue
		}
		name := termID
		if t, err := deps.Store.GetTerminal(termID); err == nil {
			name = t.Name
		}
		return &liveHolder{Kind: "terminal", ID: termID, Name: name}
	}
	return nil
}

// workspaceOwning returns the id of the workspace whose path is cwd, or "".
func workspaceOwning(deps Deps, cwd string) string {
	list, err := deps.Store.ListWorkspaces()
	if err != nil {
		return ""
	}
	for _, wk := range list {
		if wk.ID != store.FreeWorkspaceID && wk.Path != "" && filepath.Clean(wk.Path) == filepath.Clean(cwd) {
			return wk.ID
		}
	}
	return ""
}

// firstUserText is the opening human turn of a timeline — the line a
// listing would show as the session's preview.
func firstUserText(t transcript.Timeline) string {
	for _, e := range t.Events {
		if e.Kind == transcript.KindMessage && e.Role == "user" && strings.TrimSpace(e.Text) != "" {
			return strings.TrimSpace(e.Text)
		}
	}
	return ""
}

func clipRunes(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n])
}

// handoffLinks is the lineage a Sessions row carries: where it came from
// and where it continued.
type handoffLinks struct {
	From *handoffRef  `json:"from,omitempty"`
	To   []handoffRef `json:"to,omitempty"`
}

type handoffRef struct {
	HandoffID  string `json:"handoffId"`
	CLI        string `json:"cli"`
	ID         string `json:"id,omitempty"`
	Mode       string `json:"mode"`
	CreatedAt  string `json:"createdAt"`
	TerminalID string `json:"terminalId,omitempty"`
	AgentID    string `json:"agentId,omitempty"`
}

// lineageIndex maps (cli, session id) to its handoff links. A brief
// handoff into a CLI that cannot pre-assign an id is resolved through the
// terminal's pinned session when one exists.
func lineageIndex(deps Deps) map[[2]string]*handoffLinks {
	out := map[[2]string]*handoffLinks{}
	if deps.Store == nil {
		return out
	}
	rows, err := deps.Store.SessionHandoffs(1000)
	if err != nil {
		return out
	}
	get := func(cli, id string) *handoffLinks {
		k := [2]string{cli, id}
		if l, ok := out[k]; ok {
			return l
		}
		l := &handoffLinks{}
		out[k] = l
		return l
	}
	for _, h := range rows {
		targetID := h.TargetID
		if targetID == "" && h.TerminalID != "" {
			if launch, err := deps.Store.TerminalLaunch(h.TerminalID); err == nil && launch != nil && launch.LastSession != nil && launch.LastSession.CLI == h.TargetCLI {
				targetID = launch.LastSession.SessionID
			}
		}
		get(h.SourceCLI, h.SourceID).To = append(get(h.SourceCLI, h.SourceID).To, handoffRef{HandoffID: h.ID, CLI: h.TargetCLI, ID: targetID, Mode: h.Mode, CreatedAt: h.CreatedAt, TerminalID: h.TerminalID, AgentID: h.AgentID})
		if targetID != "" {
			l := get(h.TargetCLI, targetID)
			if l.From == nil || l.From.CreatedAt < h.CreatedAt {
				l.From = &handoffRef{HandoffID: h.ID, CLI: h.SourceCLI, ID: h.SourceID, Mode: h.Mode, CreatedAt: h.CreatedAt, TerminalID: h.TerminalID, AgentID: h.AgentID}
			}
		}
	}
	return out
}
