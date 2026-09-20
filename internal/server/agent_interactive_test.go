package server

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func TestPiRuntimeMatrix(t *testing.T) {
	data := t.TempDir()
	tm := tmux.NewWithSocket(filepath.Join(data, "tmux.sock"))
	if !tm.Available() {
		t.Skip("tmux unavailable")
	}
	st, e := store.Open(filepath.Join(data, "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = st.Close() })
	cmd := fakeBlockingAgentCmd(t)
	deps := Deps{Store: st, DataDir: data, Tmux: tm, Runtime: rpc.NewRuntime(cmd, st, nil), AgentCmd: cmd, CLIs: NewCLITerminals(), Replies: NewTuiReplies(), TermStates: NewTermStates(), TermRuntimes: NewTermRuntimes()}
	// The fixture owns its socket and every process launched here.
	a, e := st.AddAgent(store.FreeWorkspaceID, "fixture", data)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = deps.stopAgent(context.Background(), a.ID) })
	if e = st.SetCLIConfig("pi", clilaunch.Config{Executable: cmd, Integration: true}); e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := deps.openAgentTUI(ctx, a.ID, false); errCh <- e }()
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		if e != nil {
			t.Fatal(e)
		}
	}
	a, _ = st.GetAgent(a.ID)
	terms, _ := st.ListTerminals()
	if len(terms) != 1 || a.TerminalID == nil {
		t.Fatalf("duplicate bindings: %+v", terms)
	}
	name := deps.agentSession(a.ID)
	pid, e := tm.PanePID(ctx, name)
	if e != nil {
		t.Fatal(e)
	}
	old := TermRuntime{PID: pid, ProcStart: processStartToken(pid)}
	v := map[string]any{}
	applyTerminalLaunch(deps, v, *a.TerminalID)
	if v["launchPending"] != false {
		t.Fatalf("fresh launch pending: %v", v)
	}
	// Invalid restart must leave the old process alive.
	bad := "/does/not/exist/pi"
	if e = st.SetTerminalLaunch(*a.TerminalID, "pi", clilaunch.Overrides{Executable: &bad}); e != nil {
		t.Fatal(e)
	}
	if _, e = deps.openAgentTUI(ctx, a.ID, true); e == nil {
		t.Fatal("invalid restart accepted")
	}
	if got, _ := tm.PanePID(ctx, name); got != pid {
		t.Fatal("failed prepare replaced old pane")
	}
	if e = st.SetTerminalLaunch(*a.TerminalID, "pi", clilaunch.Overrides{}); e != nil {
		t.Fatal(e)
	}
	if _, e = deps.openAgentTUI(ctx, a.ID, true); e != nil {
		t.Fatal(e)
	}
	if processAlive(old) {
		t.Fatal("old writer survived restart")
	}
	// Work-path changes are used through the terminal launch door too.
	if got := agentTerminalCwd(deps, *a.TerminalID, t.TempDir()); got != data {
		t.Fatalf("cwd=%s, want agent work path", got)
	}
	// A shutdown receipt blocks both replacement and a managed start.
	if e = deps.stopAgentInteractive(ctx, a.ID); e != nil {
		t.Fatal(e)
	}
	if has, _ := tm.HasSession(ctx, name); has {
		t.Fatal("pane survived stop")
	}
	if e = savePeerStop(deps, *a.TerminalID, map[int]string{os.Getpid(): processStartToken(os.Getpid())}); e != nil {
		t.Fatal(e)
	}
	defer os.Remove(peerStopPath(deps, *a.TerminalID))
	req := httptest.NewRequest("POST", "/", nil)
	req.SetPathValue("id", a.ID)
	rec := httptest.NewRecorder()
	handleManagedStart(deps)(rec, req)
	if rec.Code < 400 || deps.Runtime.Active(a.ID) {
		t.Fatal("managed mode bypassed shutdown receipt")
	}
}

func TestPiAgentReservedLaunchArgs(t *testing.T) {
	for _, arg := range []string{"--mode=rpc", "--session", "--session-dir=x", "--session-id=x", "--continue", "--resume", "-r", "-c", "--print", "-p", "--no-session", "--model=x", "--provider=x", "--tools=all", "--"} {
		if validatePiAgentArgs([]string{arg}) == nil {
			t.Errorf("accepted %s", arg)
		}
	}
	if err := validatePiAgentArgs([]string{"-e", "./extension.ts"}); err != nil {
		t.Fatal(err)
	}
}

func TestPiLegacyReceiptBlocksEveryRPCDoor(t *testing.T) {
	t.Setenv("PICODE_FAKE_RPC", "1")
	data := t.TempDir()
	st, err := store.Open(filepath.Join(data, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	a, err := st.AddAgent(store.FreeWorkspaceID, "receipt", data)
	if err != nil {
		t.Fatal(err)
	}
	deps := Deps{Store: st, DataDir: data, Runtime: rpc.NewRuntime(os.Args[0], st, nil), AgentCmd: os.Args[0], CLIs: NewCLITerminals(), Replies: NewTuiReplies()}
	defer deps.Runtime.Stop(a.ID)
	if err = savePeerStop(deps, a.ID, map[int]string{os.Getpid(): processStartToken(os.Getpid())}); err != nil {
		t.Fatal(err)
	}
	// The legacy receipt remains authoritative even after lazy binding.
	for _, bind := range []bool{false, true} {
		if bind {
			a, err = st.EnsureAgentTerminal(a.ID, data)
			if err != nil {
				t.Fatal(err)
			}
		}
		r := httptest.NewRequest("POST", "/", nil)
		r.SetPathValue("id", a.ID)
		for name, handler := range map[string]func(){
			"managed": func() {
				w := httptest.NewRecorder()
				handleManagedStart(deps)(w, r)
				if w.Code < 400 {
					t.Fatal("managed bypass")
				}
			},
			"compact": func() {
				w := httptest.NewRecorder()
				handleAgentCompact(deps)(w, r)
				if w.Code < 400 {
					t.Fatal("compact bypass")
				}
			},
			"clone/fork": func() {
				if _, e := ensureChat(r, deps, a.ID); e == nil {
					t.Fatal("session operation bypass")
				}
			},
			"extension/automation": func() {
				if e := deps.startManaged(a); e == nil {
					t.Fatal("implicit start bypass")
				}
			},
		} {
			t.Run(name+map[bool]string{false: "/legacy", true: "/bound"}[bind], func(t *testing.T) {
				handler()
				if deps.Runtime.Active(a.ID) {
					t.Fatal("writer started")
				}
			})
		}
	}
}

func TestPiNativeIdentityWaitsForSessionMutation(t *testing.T) {
	data := t.TempDir()
	st, err := store.Open(filepath.Join(data, "db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	a, _ := st.AddAgent(store.FreeWorkspaceID, "native", data)
	a, err = st.EnsureAgentTerminal(a.ID, data)
	if err != nil {
		t.Fatal(err)
	}
	deps := Deps{Store: st, CLIs: NewCLITerminals(), TermRuntimes: NewTermRuntimes()}
	deps.TermRuntimes.Start(*a.TerminalID, TermRuntime{CLI: "pi", RunID: "old", PID: os.Getpid()})
	unlock := terminalLock(deps, "agent:"+a.ID)
	done := make(chan error, 1)
	go func() {
		done <- recordNativeTerminalSession(deps, *a.TerminalID, "pi", "old", "old-session", "/old.jsonl", time.Now().UnixNano())
	}()
	select {
	case <-done:
		unlock()
		t.Fatal("native event bypassed lifecycle lock")
	case <-time.After(30 * time.Millisecond):
	}
	selected := "/selected.jsonl"
	if _, err = st.UpdateAgent(a.ID, store.AgentPatch{SessionPath: &selected}); err != nil {
		unlock()
		t.Fatal(err)
	}
	deps.TermRuntimes.Drop(*a.TerminalID)
	unlock()
	if err = <-done; err == nil {
		t.Fatal("old generation accepted after session change")
	}
	got, _ := st.GetAgent(a.ID)
	if got.SessionPath == nil || *got.SessionPath != selected {
		t.Fatal("explicit session selection overwritten")
	}
}

func TestPiLegacyAndRPC(t *testing.T) {
	t.Setenv("PICODE_FAKE_RPC", "1")
	data := t.TempDir()
	tm := tmux.NewWithSocket(filepath.Join(data, "tmux.sock"))
	if !tm.Available() {
		t.Skip("tmux unavailable")
	}
	st, e := store.Open(filepath.Join(data, "db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = st.Close() })
	cmd := fakeBlockingAgentCmd(t)
	deps := Deps{Store: st, DataDir: data, Tmux: tm, Runtime: rpc.NewRuntime(os.Args[0], st, nil), AgentCmd: cmd, CLIs: NewCLITerminals(), Replies: NewTuiReplies(), TermStates: NewTermStates(), TermRuntimes: NewTermRuntimes()}
	a, e := st.AddAgent(store.FreeWorkspaceID, "legacy", data)
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	legacy := tmux.SessionName(a.ID)
	t.Cleanup(func() {
		deps.Runtime.Stop(a.ID)
		_ = tm.KillSession(context.Background(), legacy)
		if a, e := st.GetAgent(a.ID); e == nil && a.TerminalID != nil {
			_ = stopInteractivePane(context.Background(), deps, tmux.ShellSessionName(*a.TerminalID), *a.TerminalID)
		}
	})
	if e = tm.NewSessionEnv(ctx, legacy, data, nil, cmd); e != nil {
		t.Fatal(e)
	}
	if already, e := deps.openAgentTUI(ctx, a.ID, false); e != nil || !already {
		t.Fatalf("legacy open: %v %v", already, e)
	}
	a, _ = st.GetAgent(a.ID)
	if a.TerminalID != nil {
		t.Fatal("ordinary reconnect migrated legacy")
	}
	if e = st.SetCLIConfig("pi", clilaunch.Config{Executable: "/missing/pi"}); e != nil {
		t.Fatal(e)
	}
	if _, e = deps.openAgentTUI(ctx, a.ID, true); e == nil {
		t.Fatal("invalid restart accepted")
	}
	a, _ = st.GetAgent(a.ID)
	req := httptest.NewRequest("GET", "/", nil)
	if deps.agentSession(a.ID) != legacy || !deps.legacyAgentInteractive(a) || deps.agentTerminalView(req, a) != nil {
		t.Fatal("failed prepare hid live legacy")
	}
	if e = st.SetCLIConfig("pi", clilaunch.Config{Executable: cmd, Integration: true}); e != nil {
		t.Fatal(e)
	}
	v, _ := st.TerminalLaunch(*a.TerminalID)
	p, e := prepareCLITerminal(deps, data, v)
	if e != nil {
		t.Fatal(e)
	}
	defer p.discard()
	if e = p.start(deps, req, tmux.ShellSessionName(*a.TerminalID), data); e == nil {
		t.Fatal("terminal route duplicated legacy")
	}
	// A receipt saved before a failed legacy kill must not prevent retrying
	// shutdown of that same live pane (even after the lazy binding exists).
	legacyPID, e := tm.PanePID(ctx, legacy)
	if e != nil {
		t.Fatal(e)
	}
	if e = savePeerStop(deps, a.ID, map[int]string{legacyPID: processStartToken(legacyPID)}); e != nil {
		t.Fatal(e)
	}
	if _, e = deps.openAgentTUI(ctx, a.ID, true); e != nil {
		t.Fatal(e)
	}
	if has, _ := tm.HasSession(ctx, legacy); has {
		t.Fatal("legacy process not migrated")
	}
	req = httptest.NewRequest("POST", "/", nil)
	req.SetPathValue("id", a.ID)
	rec := httptest.NewRecorder()
	handleManagedStart(deps)(rec, req)
	if rec.Code != 201 || !deps.Runtime.Active(a.ID) {
		t.Fatalf("RPC transition: %d %s", rec.Code, rec.Body.String())
	}
	if has, _ := tm.HasSession(ctx, tmux.ShellSessionName(*a.TerminalID)); has {
		t.Fatal("terminal survived RPC switch")
	}
	p2, e := prepareCLITerminal(deps, data, v)
	if e != nil {
		t.Fatal(e)
	}
	defer p2.discard()
	if e = p2.start(deps, req, tmux.ShellSessionName(*a.TerminalID), data); e == nil {
		t.Fatal("terminal duplicated RPC")
	}
	req = httptest.NewRequest("DELETE", "/", nil)
	req.SetPathValue("id", *a.TerminalID)
	rec = httptest.NewRecorder()
	handleDeleteTerminal(deps)(rec, req)
	if rec.Code != 204 || deps.Runtime.Active(a.ID) {
		t.Fatalf("delete RPC owner: %d %s", rec.Code, rec.Body.String())
	}
	if _, e = st.GetAgent(a.ID); e == nil {
		t.Fatal("owner not deleted")
	}
}

func TestPiAgentLaunchReusesIntegrationAndIdentity(t *testing.T) {
	for _, integration := range []bool{true, false} {
		t.Run(map[bool]string{true: "activity", false: "no-activity"}[integration], func(t *testing.T) {
			data := t.TempDir()
			st, err := store.Open(filepath.Join(data, "picode.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			a, err := st.AddAgent(store.FreeWorkspaceID, "pi", data)
			if err != nil {
				t.Fatal(err)
			}
			model := "fixture-model"
			a, err = st.UpdateAgent(a.ID, store.AgentPatch{Model: &model})
			if err != nil {
				t.Fatal(err)
			}
			a, err = st.EnsureAgentTerminal(a.ID, data)
			if err != nil {
				t.Fatal(err)
			}
			if err = st.SetCLIConfig("pi", clilaunch.Config{Executable: "/bin/cat", Integration: integration}); err != nil {
				t.Fatal(err)
			}
			deps := Deps{Store: st, DataDir: data, AgentCmd: "/bin/cat", TermStates: NewTermStates()}
			v, err := st.TerminalLaunch(*a.TerminalID)
			if err != nil {
				t.Fatal(err)
			}
			p, err := prepareCLITerminal(deps, data, v)
			if err != nil {
				t.Fatal(err)
			}
			defer p.discard()
			body, err := os.ReadFile(p.script)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"--model", "fixture-model", "--session-dir", "--session-id", "PICODE_AGENT_ID", a.ID, "PICODE_DATA", data} {
				if !strings.Contains(string(body), want) {
					t.Errorf("launch missing %s", want)
				}
			}
			if integration {
				wrapper, err := os.ReadFile(wrapperPath(p.dir, "pi"))
				if err != nil {
					t.Fatal(err)
				}
				for _, want := range []string{"pi-terminal-state.ts", "pi-inbox-reply.ts"} {
					if !strings.Contains(string(wrapper), want) {
						t.Errorf("missing shared %s", want)
					}
				}
			}
			identity := strings.Join(launchIdentityEnv(deps, *a.TerminalID), "\n")
			if !strings.Contains(identity, "PICODE_AGENT_ID="+a.ID) || !strings.Contains(identity, "PICODE_TERM_ID="+*a.TerminalID) {
				t.Fatal(identity)
			}
			reportTermState(deps, *a.TerminalID, TermNeedsYou, "pi", time.Now())
			reportTermState(deps, *a.TerminalID, TermNeedsYou, "pi", time.Now())
			items, err := st.ActiveInboxBySourceReason(store.InboxFromAgent, a.ID, store.InboxNeedsYouReason)
			if err != nil || len(items) != 1 {
				t.Fatalf("attention: %v %v", items, err)
			}
			reportTermState(deps, *a.TerminalID, TermIdle, "pi", time.Now())
			items, _ = st.ActiveInboxBySourceReason(store.InboxFromAgent, a.ID, store.InboxNeedsYouReason)
			if len(items) != 0 {
				t.Fatal("attention remained open")
			}
		})
	}
}
