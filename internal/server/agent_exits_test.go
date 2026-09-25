package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func exitServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PICODE_DATA", t.TempDir())
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store:    st,
		Tmux:     tmux.New(),
		Runtime:  rpc.NewRuntime("cat", st, nil),
		AgentCmd: "cat",
		DataDir:  t.TempDir(),
	}).Handler)
	t.Cleanup(ts.Close)
	return ts, st
}

func sendJSON(t *testing.T, ts *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()
	var rd *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, ts.URL+path, rd)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return do(t, ts.Client(), req)
}

func freeAgent(t *testing.T, ts *httptest.Server, name string) agentView {
	t.Helper()
	res := postJSON(t, ts, "/api/agents", map[string]string{"name": name, "path": t.TempDir()})
	var ag agentView
	if err := json.NewDecoder(res.Body).Decode(&ag); err != nil || ag.ID == "" {
		t.Fatalf("create agent: %v", err)
	}
	return ag
}

func TestAgentCleanupPreviewCarriesTheQuestion(t *testing.T) {
	ts, st := exitServer(t)
	ag := freeAgent(t, ts, "fresh")
	var preview cleanupPreview
	_ = json.NewDecoder(do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+ag.ID+"/cleanup")).Body).Decode(&preview)
	if preview.Exit == nil || preview.Exit.Ask || preview.Exit.Skip != store.ExitSkipIdle || len(preview.Exit.Taxonomy.Outcomes) != 4 {
		t.Fatalf("a brand-new agent that never worked is not asked: %+v", preview.Exit)
	}
	if err := st.SetExitAskOn(false); err != nil {
		t.Fatal(err)
	}
	_ = json.NewDecoder(do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+ag.ID+"/cleanup")).Body).Decode(&preview)
	if preview.Exit.Skip != store.ExitSkipOff {
		t.Fatalf("the switch wins: %+v", preview.Exit)
	}
	// The workspace preview has no question.
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces"))
	res.Body.Close()
}

func TestDeleteAgentRecordsTheAnswer(t *testing.T) {
	ts, _ := exitServer(t)
	ag := freeAgent(t, ts, "fixer")
	res := sendJSON(t, ts, http.MethodDelete, "/api/agents/"+ag.ID, map[string]any{"exit": map[string]any{
		"origin": "desktop", "asked": true, "outcome": "unresolved", "reasons": []string{"stuck"}, "note": "loops",
	}})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	var out struct {
		Exit store.AgentExit `json:"exit"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)
	ex := out.Exit
	if ex.AgentID != ag.ID || ex.Origin != store.ExitFromDesktop || !ex.Asked || ex.Outcome != store.ExitUnresolved || strings.Join(ex.Reasons, ",") != "stuck" || ex.Note != "loops" {
		t.Fatalf("exit = %+v", ex)
	}
	gone := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+ag.ID+"/cleanup"))
	if gone.StatusCode != http.StatusNotFound {
		t.Fatalf("agent still there: %d", gone.StatusCode)
	}
}

func TestDeleteAgentRefusesABadAnswerBeforeStoppingIt(t *testing.T) {
	ts, st := exitServer(t)
	ag := freeAgent(t, ts, "keep")
	for _, body := range []any{
		map[string]any{"exit": map[string]any{"outcome": "meh"}},
		map[string]any{"exit": map[string]any{"outcome": "resolved", "reasons": []string{"stuck"}}},
	} {
		if res := sendJSON(t, ts, http.MethodDelete, "/api/agents/"+ag.ID, body); res.StatusCode != http.StatusBadRequest {
			t.Fatalf("bad answer = %d", res.StatusCode)
		}
	}
	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/agents/"+ag.ID, strings.NewReader("{not json"))
	if res := do(t, ts.Client(), req); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("broken body = %d", res.StatusCode)
	}
	if _, err := st.GetAgent(ag.ID); err != nil {
		t.Fatalf("a refused answer removed the agent: %v", err)
	}
}

// ADR-0194: when the client did not show the question, the skip says why —
// the server's own reason, or "client" when the server would have asked.
func TestExitInputSkipReason(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	deps := Deps{Store: st}
	a, _ := st.AddAgent(store.FreeWorkspaceID, "a", "")
	born, _ := time.Parse(time.RFC3339Nano, a.CreatedAt)
	later := born.Add(2 * time.Hour)

	if in := deps.exitInput(a, exitRequest{}, false, false, later); in.Asked || in.AskSkip != store.ExitSkipIdle || in.Origin != store.ExitFromAPI {
		t.Fatalf("idle agent, no body = %+v", in)
	}
	_ = st.NoteAgentTurn(a.ID)
	if in := deps.exitInput(a, exitRequest{}, false, false, later); in.AskSkip != store.ExitSkipClient {
		t.Fatalf("worked agent, no body = %+v", in)
	}
	if in := deps.exitInput(a, exitRequest{}, false, false, born.Add(10*time.Second)); in.AskSkip != store.ExitSkipBrief {
		t.Fatalf("worked but brief = %+v", in)
	}
	asked := exitRequest{Exit: &exitBody{Origin: "mobile", Asked: true, Outcome: "trial"}}
	if in := deps.exitInput(a, asked, true, false, later); !in.Asked || in.AskSkip != "" || in.Origin != "mobile" || !in.SessionsPurged || in.Label.Outcome != "trial" {
		t.Fatalf("asked = %+v", in)
	}
}

func TestAgentExitRoutes(t *testing.T) {
	ts, _ := exitServer(t)
	one := freeAgent(t, ts, "one")
	two := freeAgent(t, ts, "two")
	sendJSON(t, ts, http.MethodDelete, "/api/agents/"+one.ID, map[string]any{"exit": map[string]any{"origin": "desktop", "asked": true, "outcome": "resolved"}})
	res := sendJSON(t, ts, http.MethodDelete, "/api/agents/"+two.ID, nil)
	var removed struct {
		Exit store.AgentExit `json:"exit"`
	}
	_ = json.NewDecoder(res.Body).Decode(&removed)

	var list struct {
		Exits    []store.AgentExit  `json:"exits"`
		Ask      bool               `json:"ask"`
		Taxonomy store.ExitTaxonomy `json:"taxonomy"`
	}
	_ = json.NewDecoder(do(t, ts.Client(), mustGet(t, ts.URL+"/api/agent-exits")).Body).Decode(&list)
	if len(list.Exits) != 2 || !list.Ask || list.Taxonomy.Version != store.ExitTaxonomyVersion {
		t.Fatalf("list = %+v", list)
	}
	unanswered := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agent-exits?outcome=unanswered&range=7d"))
	_ = json.NewDecoder(unanswered.Body).Decode(&list)
	if len(list.Exits) != 1 || list.Exits[0].ID != removed.Exit.ID {
		t.Fatalf("unanswered = %+v", list.Exits)
	}

	labeled := sendJSON(t, ts, http.MethodPatch, "/api/agent-exits/"+removed.Exit.ID, map[string]any{"outcome": "partial", "reasons": []string{"slow_costly"}})
	var ex store.AgentExit
	_ = json.NewDecoder(labeled.Body).Decode(&ex)
	if labeled.StatusCode != http.StatusOK || ex.Outcome != store.ExitPartial || ex.LabeledAt == nil {
		t.Fatalf("label = %d %+v", labeled.StatusCode, ex)
	}
	if bad := sendJSON(t, ts, http.MethodPatch, "/api/agent-exits/"+removed.Exit.ID, map[string]any{"outcome": "nah"}); bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad label = %d", bad.StatusCode)
	}

	var sum struct {
		Range   string            `json:"range"`
		Summary store.ExitSummary `json:"summary"`
	}
	_ = json.NewDecoder(do(t, ts.Client(), mustGet(t, ts.URL+"/api/agent-exits/summary?range=30d")).Body).Decode(&sum)
	if sum.Range != "30d" || sum.Summary.Total != 2 || sum.Summary.Answered != 2 || sum.Summary.Outcomes[store.ExitPartial] != 1 {
		t.Fatalf("summary = %+v", sum)
	}

	if undo := sendJSON(t, ts, http.MethodPost, "/api/agent-exits/"+removed.Exit.ID+"/undo", map[string]string{"agentId": "two-back"}); undo.StatusCode != http.StatusOK {
		t.Fatalf("undo = %d", undo.StatusCode)
	}
	_ = json.NewDecoder(do(t, ts.Client(), mustGet(t, ts.URL+"/api/agent-exits")).Body).Decode(&list)
	if len(list.Exits) != 1 {
		t.Fatalf("an undone exit leaves the list: %d", len(list.Exits))
	}

	export := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agent-exits/export"))
	if ct := export.Header.Get("Content-Type"); ct != "application/x-ndjson" {
		t.Fatalf("content type = %q", ct)
	}
	lines := 0
	sc := bufio.NewScanner(export.Body)
	for sc.Scan() {
		var row store.AgentExit
		if err := json.Unmarshal(sc.Bytes(), &row); err != nil || row.ID == "" {
			t.Fatalf("export line %d: %v", lines, err)
		}
		lines++
	}
	if lines != 2 {
		t.Fatalf("export keeps undone exits too: %d lines", lines)
	}

	if del := sendJSON(t, ts, http.MethodDelete, "/api/agent-exits/"+removed.Exit.ID, nil); del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete exit = %d", del.StatusCode)
	}
	if missing := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agent-exits/"+removed.Exit.ID)); missing.StatusCode != http.StatusNotFound {
		t.Fatalf("deleted exit = %d", missing.StatusCode)
	}

	if bad := sendJSON(t, ts, http.MethodPut, "/api/agent-exits/prefs", map[string]any{}); bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("prefs without ask = %d", bad.StatusCode)
	}
	sendJSON(t, ts, http.MethodPut, "/api/agent-exits/prefs", map[string]bool{"ask": false})
	var prefs struct {
		Ask bool `json:"ask"`
	}
	_ = json.NewDecoder(do(t, ts.Client(), mustGet(t, ts.URL+"/api/agent-exits/prefs")).Body).Decode(&prefs)
	if prefs.Ask {
		t.Fatal("prefs still on")
	}
}

// A turn is work that follows idle or no state; work after needs-you is
// the same turn resuming once the person answered.
func TestTerminalReportsCountTurns(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	a, _ := st.AddAgentWithCLI(store.FreeWorkspaceID, "codex", "reviewer", "")
	term, _ := st.CreateTerminalIn(store.FreeWorkspaceID, "reviewer", t.TempDir())
	if _, err := st.UpdateAgent(a.ID, store.AgentPatch{TerminalID: &term.ID}); err != nil {
		t.Fatal(err)
	}
	shell, _ := st.CreateTerminalIn(store.FreeWorkspaceID, "shell", t.TempDir())
	deps := Deps{Store: st, TermStates: &TermStates{}}
	now := time.Now()
	steps := []struct {
		state string
		want  int64
	}{
		{TermWorking, 1},  // no state before: a turn
		{TermWorking, 1},  // the same report again: nothing changed
		{TermNeedsYou, 1}, // waiting on the person
		{TermWorking, 1},  // resumed after the answer: the same turn
		{TermIdle, 1},
		{TermWorking, 2}, // a new prompt
	}
	for i, step := range steps {
		reportTermState(deps, term.ID, step.state, "codex", now.Add(time.Duration(i)*time.Second))
		act, _ := st.AgentActivityOf(a.ID)
		if act.Turns == nil || *act.Turns != step.want {
			t.Fatalf("step %d (%s): turns = %v, want %d", i, step.state, act.Turns, step.want)
		}
	}
	// A plain shell has no agent: reporting on it counts nothing and fails nothing.
	reportTermState(deps, shell.ID, TermWorking, "", now)
}

// Integrated CLIs (Claude Code, Codex, Omp…) report with their native
// session, which writes the state on its own path; it counts turns by the
// same rule. Production recorded 0 turns for a 175-turn Omp agent before.
func TestNativeSessionReportsCountTurns(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	a, _ := st.AddAgentWithCLI(store.FreeWorkspaceID, "codex", "reviewer", "")
	term, _ := st.CreateTerminalIn(store.FreeWorkspaceID, "reviewer", t.TempDir())
	if _, err := st.UpdateAgent(a.ID, store.AgentPatch{TerminalID: &term.ID}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(term.ID, "codex", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	deps := Deps{Store: st, TermRuntimes: NewTermRuntimes(), TermStates: NewTermStates()}
	deps.TermRuntimes.Start(term.ID, TermRuntime{CLI: "codex", RunID: "run", PID: os.Getpid()})
	seq := time.Now().UnixNano()
	steps := []struct {
		state string
		want  int64
	}{
		{TermWorking, 1},  // no state before: a turn
		{TermWorking, 1},  // still working
		{TermNeedsYou, 1}, // waiting on the person
		{TermWorking, 1},  // resumed after the answer: the same turn
		{TermIdle, 1},
		{TermWorking, 2}, // a new prompt
	}
	for i, step := range steps {
		if err := recordNativeTerminalObservation(deps, term.ID, "codex", "run", "session", "", seq+int64(i), step.state, "codex-hook", ""); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		act, _ := st.AgentActivityOf(a.ID)
		if act.Turns == nil || *act.Turns != step.want {
			t.Fatalf("step %d (%s): turns = %v, want %d", i, step.state, act.Turns, step.want)
		}
	}
}

func TestWorkspaceRemovalEndsItsAgentsWithExits(t *testing.T) {
	ts, st := exitServer(t)
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "Gone", "path": t.TempDir()})
	var wk store.Workspace
	if err := json.NewDecoder(res.Body).Decode(&wk); err != nil || wk.ID == "" {
		t.Fatalf("workspace: %v", err)
	}
	if _, err := st.AddAgent(wk.ID, "inside", ""); err != nil {
		t.Fatal(err)
	}
	if del := sendJSON(t, ts, http.MethodDelete, "/api/workspaces/"+wk.ID, map[string]any{"exit": map[string]string{"origin": "desktop"}}); del.StatusCode != http.StatusNoContent {
		t.Fatalf("remove workspace = %d", del.StatusCode)
	}
	exits, _ := st.ListAgentExits(store.ExitFilter{WorkspaceID: wk.ID})
	if len(exits) != 1 || exits[0].AskSkip != store.ExitSkipWorkspace || exits[0].Origin != store.ExitFromDesktop || exits[0].WorkspaceName != "Gone" {
		t.Fatalf("exits = %+v", exits)
	}
}

// A CLI agent's TUI is its terminal (ADR-0160): removing the terminal ends
// the agent, and the removal writes its exit (ADR-0194).
func TestTerminalRemovalEndsItsAgentWithAnExit(t *testing.T) {
	ts, st := exitServer(t)
	a, _ := st.AddAgentWithCLI(store.FreeWorkspaceID, "codex", "reviewer", "")
	term, _ := st.CreateTerminalIn(store.FreeWorkspaceID, "reviewer", t.TempDir())
	if _, err := st.UpdateAgent(a.ID, store.AgentPatch{TerminalID: &term.ID}); err != nil {
		t.Fatal(err)
	}
	shell, _ := st.CreateTerminalIn(store.FreeWorkspaceID, "shell", t.TempDir())
	if bad := sendJSON(t, ts, http.MethodDelete, "/api/terminals/"+term.ID, map[string]any{"exit": map[string]any{"outcome": "nope"}}); bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad answer = %d", bad.StatusCode)
	}
	res := sendJSON(t, ts, http.MethodDelete, "/api/terminals/"+term.ID, map[string]any{"exit": map[string]any{"origin": "mobile", "asked": true, "outcome": "resolved"}})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("remove terminal = %d", res.StatusCode)
	}
	exits, _ := st.ListAgentExits(store.ExitFilter{})
	if len(exits) != 1 || exits[0].AgentID != a.ID || exits[0].Origin != store.ExitFromMobile || exits[0].Outcome != store.ExitResolved {
		t.Fatalf("exits = %+v", exits)
	}
	if _, err := st.GetAgent(a.ID); err == nil {
		t.Fatal("agent survived its terminal")
	}
	if res := sendJSON(t, ts, http.MethodDelete, "/api/terminals/"+shell.ID, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("remove shell = %d", res.StatusCode)
	}
	if exits, _ := st.ListAgentExits(store.ExitFilter{}); len(exits) != 1 {
		t.Fatalf("a plain shell has no agent and no exit: %d", len(exits))
	}
}

// An exit freezes the instruction files the agent read, with a hash of
// each (ADR-0194, amended 2026-09-24): a free Pi agent, which keeps no record
// of its own, gets the rules' answer for its folder.
func TestExitRecordsInstructionRevisions(t *testing.T) {
	ts, _ := exitServer(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Build\nmake\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := postJSON(t, ts, "/api/agents", map[string]string{"name": "reader", "path": dir})
	var ag agentView
	if err := json.NewDecoder(res.Body).Decode(&ag); err != nil || ag.ID == "" {
		t.Fatalf("create agent: %v", err)
	}
	del := sendJSON(t, ts, http.MethodDelete, "/api/agents/"+ag.ID, nil)
	var out struct {
		Exit store.AgentExit `json:"exit"`
	}
	_ = json.NewDecoder(del.Body).Decode(&out)
	in := out.Exit.Config.Instructions
	if in == nil || in.Source != "declared" || len(in.Files) != 1 || in.Files[0].Path != "AGENTS.md" || len(in.Files[0].SHA) != 12 || in.Files[0].Bytes != 13 {
		t.Fatalf("instructions = %+v", in)
	}
}

func TestPathWithin(t *testing.T) {
	for _, c := range []struct {
		root, p string
		want    bool
	}{{"/w", "/w", true}, {"/w", "/w/sub", true}, {"/w", "/w2", false}, {"/w", "/", false}, {"/w/a", "/w", false}} {
		if got := pathWithin(c.root, c.p); got != c.want {
			t.Errorf("pathWithin(%s, %s) = %v", c.root, c.p, got)
		}
	}
}
