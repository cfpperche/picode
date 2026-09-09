package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"

	"github.com/cfpperche/picode/internal/tmux"
)

func TestNativeSessionReportBinding(t *testing.T) {
	data := t.TempDir()
	s, err := store.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	w, _ := s.AddWorkspace("fixture", data)
	term, _ := s.CreateTerminalIn(w.ID, "Grok", data)
	s.SetTerminalLaunch(term.ID, "grok", clilaunch.Overrides{})
	deps := Deps{Store: s, TermRuntimes: NewTermRuntimes()}
	deps.TermRuntimes.Start(term.ID, TermRuntime{CLI: "grok", RunID: "run-1", PID: os.Getpid()})
	seq := time.Now().UnixNano()
	report := func(cli, run, id string, n int64) error {
		return recordNativeTerminalSession(deps, term.ID, cli, run, id, "", n)
	}
	if err := report("grok", "run-1", "native-A", seq); err != nil {
		t.Fatal(err)
	}
	p, token, err := s.EnablePeer("terminal", term.ID, "native-A")
	if err != nil {
		t.Fatal(err)
	}
	_ = p
	for _, tc := range []struct {
		cli, run, id string
		seq          int64
	}{{"grok", "old", "native-B", seq + 1}, {"hermes", "run-1", "native-B", seq + 1}, {"grok", "run-1", "native-B", seq - 1}, {"grok", "", "native-B", seq + 1}, {"grok", "run-1", "bad\nvalue", seq + 1}} {
		if err := report(tc.cli, tc.run, tc.id, tc.seq); err == nil {
			t.Fatal("accepted stale/invalid", tc)
		}
		if _, err := s.AuthorizePeer(token); err != nil {
			t.Fatal("invalid report changed binding")
		}
	}
	if err := report("grok", "run-1", "native-B", seq+2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AuthorizePeer(token); err == nil {
		t.Fatal("old conversation stayed authorized after /new")
	}
	deps.TermRuntimes.Start(term.ID, TermRuntime{CLI: "grok", RunID: "run-2", PID: os.Getpid()})
	if err := report("grok", "run-1", "native-A", seq+3); err == nil {
		t.Fatal("old process repinned after restart")
	}
	live, _ := deps.TermRuntimes.Get(term.ID)
	if live.SessionID != "" {
		t.Fatal("new process inherited identity")
	}
}

func TestNativePinNeverFallsBackToLatestConversation(t *testing.T) {
	data := t.TempDir()
	s, err := store.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	w, _ := s.AddWorkspace("fixture", data)
	for _, cli := range []string{"grok", "hermes"} {
		term, _ := s.CreateTerminalIn(w.ID, cli, data)
		s.SetTerminalLaunch(term.ID, cli, clilaunch.Overrides{})
		s.SetTerminalLastSession(term.ID, store.TerminalLastSession{CLI: cli, SessionID: "recorded-session"})
		// The runtime identity has deliberately been lost in a server restart.
		pinTerminalLastSession(Deps{Store: s}, term.ID, TermRuntime{CLI: cli, Source: "tmux-fallback", StartedAt: time.Time{}})
		launch, _ := s.TerminalLaunch(term.ID)
		if launch.LastSession.SessionID != "recorded-session" {
			t.Fatal("heuristic replaced native identity")
		}
	}
}

func TestNativeIdentityPlainTerminalRefusesWithoutPanic(t *testing.T) {
	data := t.TempDir()
	s, err := store.Open(filepath.Join(data, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	w, _ := s.AddWorkspace("fixture", data)
	term, _ := s.CreateTerminalIn(w.ID, "shell", data)
	deps := Deps{Store: s, TermRuntimes: NewTermRuntimes()}
	deps.TermRuntimes.Start(term.ID, TermRuntime{CLI: "grok", RunID: "run", PID: os.Getpid()})
	if err := recordNativeTerminalSession(deps, term.ID, "grok", "run", "native", "", time.Now().UnixNano()); err == nil {
		t.Fatal("plain terminal acquired launch binding")
	}
}

func TestNativeIdentityAndActivityStayOrdered(t *testing.T) {
	data := t.TempDir()
	s, err := store.Open(filepath.Join(data, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	w, _ := s.AddWorkspace("fixture", data)
	term, _ := s.CreateTerminalIn(w.ID, "grok", data)
	s.SetTerminalLaunch(term.ID, "grok", clilaunch.Overrides{})
	deps := Deps{Store: s, TermRuntimes: NewTermRuntimes(), TermStates: NewTermStates()}
	deps.TermRuntimes.Start(term.ID, TermRuntime{CLI: "grok", RunID: "run", PID: os.Getpid()})
	seq := time.Now().UnixNano()
	if err := recordNativeTerminalSession(deps, term.ID, "grok", "run", "new", "", seq, TermWorking); err != nil {
		t.Fatal(err)
	}
	if err := recordNativeTerminalSession(deps, term.ID, "grok", "run", "old", "", seq-1, TermIdle); err == nil {
		t.Fatal("stale report accepted")
	}
	state, _ := deps.TermStates.Get(term.ID)
	if state.State != TermWorking || state.SessionID != "new" || state.SessionSeq != seq {
		t.Fatalf("state and identity diverged: %+v", state)
	}
}

func TestNativeReportPreservesCodexNotifyIdentity(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 unavailable")
	}
	cmd := exec.Command("python3", "-c", hookMapPy)
	cmd.Env = append(os.Environ(), "PICODE_HOOK_REPORT=1", "PICODE_HOOK_CLI=codex", "PICODE_TUI_RUN_ID=fixture-run", "PICODE_TUI_PID=123")
	cmd.Stdin = strings.NewReader(`{"type":"agent-turn-complete","thread-id":"native-codex-id"}`)
	raw, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		SessionID string `json:"sessionId"`
		PID       int    `json:"pid"`
	}
	if err := json.Unmarshal(raw, &got); err != nil || got.SessionID != "native-codex-id" || got.PID != 123 {
		t.Fatalf("native notify lost identity: %s %v", raw, err)
	}
}

func TestNativeRuntimeRecoveryRequiresCurrentPane(t *testing.T) {
	m := tmux.New()
	if processStartToken(os.Getpid()) == "" {
		t.Skip("native recovery requires process start metadata")
	}
	if !m.Available() {
		t.Skip("tmux unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	term := "native-recovery-" + rand.Text()
	name := tmux.ShellSessionName(term)
	if err := m.NewSession(ctx, name, t.TempDir(), "sleep", "30"); err != nil {
		t.Fatal(err)
	}
	defer m.KillSession(context.Background(), name)
	pane, err := m.InputSnapshot(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	deps := Deps{Tmux: m, TermRuntimes: NewTermRuntimes()}
	deps.TermRuntimes.Start(term, TermRuntime{CLI: "grok", Source: "tmux-fallback", RunID: "fallback", PID: pane.PanePID})
	recoverNativeRuntime(ctx, deps, term, "grok", "foreign", os.Getpid())
	if rt, _ := deps.TermRuntimes.Get(term); rt.RunID != "fallback" {
		t.Fatal("foreign process acquired pane")
	}
	recoverNativeRuntime(ctx, deps, term, "grok", "original-wrapper", pane.PanePID)
	if rt, _ := deps.TermRuntimes.Get(term); rt.RunID != "original-wrapper" || rt.Source != "wrapper" {
		t.Fatal("original live wrapper not recovered", rt)
	}
	recoverNativeRuntime(ctx, deps, term, "grok", "other-wrapper", pane.PanePID)
	if rt, _ := deps.TermRuntimes.Get(term); rt.RunID != "original-wrapper" {
		t.Fatal("recovery displaced a wrapper")
	}
}

func TestNativeHookIgnoresChildAndDelayedCompletion(t *testing.T) {
	for _, tc := range []struct{ cli, payload, want string }{
		{"claude-code", `{"hook_event_name":"UserPromptSubmit"}`, "working"},
		{"claude-code", `{"hook_event_name":"TaskCompleted"}`, ""},
		{"claude-code", `{"hook_event_name":"SubagentStop"}`, ""},
		{"grok", `{"hook_event_name":"UserPromptSubmit","promptId":"B"}`, "working"},
		{"grok", `{"hook_event_name":"Stop","promptId":"A","timestamp":"2099-01-01T00:00:00Z"}`, ""},
		{"grok", `{"hook_event_name":"Stop","promptId":"B","stopHookActive":true}`, ""},
		{"grok", `{"hook_event_name":"StopCancelled","promptId":"A"}`, ""},
		{"grok", `{"hook_event_name":"SessionEnd","session_id":"child","subagentType":"explore"}`, ""},
		{"grok", `{"hook_event_name":"SessionEnd","parent_session_id":"root"}`, ""},
		{"grok", `{"hook_event_name":"SessionEnd","session_id":"old","timestamp":"2099-01-01T00:00:00Z"}`, ""},
		{"grok", `{"hook_event_name":"Notification","notification_type":"idle_prompt"}`, "idle"},
		{"grok", `{"hookEventName":"notification","notificationType":"idle_prompt"}`, "idle"},
	} {
		cmd := exec.Command("python3", "-c", hookMapPy)
		cmd.Env = append(os.Environ(), "PICODE_HOOK_CLI="+tc.cli, "PICODE_HOOK_REPORT=0")
		cmd.Stdin = strings.NewReader(tc.payload)
		raw, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(string(raw)) != tc.want {
			t.Fatalf("%s: %s => %q", tc.cli, tc.payload, raw)
		}
	}
}

func TestOpenCodeNativeActivityIgnoresOtherSessions(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node unavailable")
	}
	// Exercise the actual plugin with native SDK-shaped metadata and hook calls.
	source := strings.Replace(opencodeActivityJS, `import { spawn } from "node:child_process"`, `const reports=[]; function spawn(_hook,args,options){reports.push({state:args[0],id:options.env.PICODE_NATIVE_SESSION_ID,seq:options.env.PICODE_NATIVE_SESSION_SEQ});return {unref(){}}}`, 1)
	source = strings.Replace(source, "export default async function", "async function", 1)
	source += `
const client={session:{get:async ({path:{id}})=>{
 if(id==="missing") throw Error("unavailable")
 return {data:{id,parentID:id==="child"?"root":undefined}}
}}}
const hooks=await picodeActivity({client})
const event=(id,type="session.status",state="idle")=>hooks.event({event:{type,properties:{sessionID:id,status:{type:state}}}})
await event("other")
await hooks["chat.message"]({sessionID:"root"})
await event("child")
await event("child","permission.asked")
await hooks["chat.message"]({sessionID:"child"})
await event("other")
await hooks["chat.message"]({sessionID:"missing"})
await event("root")
await hooks["chat.message"]({sessionID:"new"})
await event("root")
await event("new")
if(JSON.stringify(reports.map(({state,id})=>[state,id]))!==JSON.stringify([["working","root"],["idle","root"],["working","new"],["idle","new"]])) throw Error(JSON.stringify(reports))
for(let i=1;i<reports.length;i++) if(BigInt(reports[i].seq)<=BigInt(reports[i-1].seq))throw Error("unordered reports")
process.env.PICODE_OPENCODE_SESSION_ID="resumed"
const resumed=await picodeActivity({client})
await resumed.event({event:{type:"session.idle",properties:{sessionID:"other"}}})
if(reports.length!==4) throw Error("background root stole resumed conversation")
await resumed.event({event:{type:"session.idle",properties:{sessionID:"resumed"}}})
if(reports.length!==5 || reports[4].id!=="resumed")throw Error("explicit resume failed to bootstrap")
process.env.PICODE_OPENCODE_SESSION_ID="child"
const child=await picodeActivity({client})
await child.event({event:{type:"session.idle",properties:{sessionID:"child"}}})
if(reports.length!==5)throw Error("child resume accepted")
`
	cmd := exec.Command("node", "--input-type=module", "-e", source)
	cmd.Env = append(os.Environ(), "PICODE_OPENCODE_HOOK=fixture", "PICODE_TERM_ID=fixture")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
}
