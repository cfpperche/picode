package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// A child copy of this test binary is a deterministic Pi stand-in. The parent
// process has another PICODE_AGENT_ID, so only the tmux/RPC subprocesses enter
// this path. TestMain intercepts them before m.Run parses Pi's CLI flags.
func TestMain(m *testing.M) {
	if strings.HasPrefix(os.Getenv(store.AgentIDEnv), "burst-e2e-") {
		runFakeBurstPi()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runFakeBurstPi() {
	modeRPC := false
	sessionPath := ""
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--mode":
			if i+1 < len(os.Args) && os.Args[i+1] == "rpc" {
				modeRPC = true
			}
			i++
		case "--session":
			if i+1 < len(os.Args) {
				sessionPath = os.Args[i+1]
			}
			i++
		}
	}
	if !modeRPC {
		fmt.Println("fake pi ready")
		for {
			time.Sleep(time.Hour)
		}
	}
	enc := json.NewEncoder(os.Stdout)
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		var cmd struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Message string `json:"message"`
		}
		if json.Unmarshal(scan.Bytes(), &cmd) != nil {
			continue
		}
		switch cmd.Type {
		case "get_state":
			_ = enc.Encode(map[string]any{"id": cmd.ID, "type": "response", "command": cmd.Type, "success": true, "data": map[string]any{"sessionFile": sessionPath}})
		case "prompt":
			_ = enc.Encode(map[string]any{"id": cmd.ID, "type": "response", "command": cmd.Type, "success": true})
			behavior := os.Getenv("PICODE_FAKE_BEHAVIOR")
			if behavior == "exit-before-start" {
				os.Exit(7)
			}
			if behavior == "passive-widget" {
				// Real sessions decorate themselves at startup: the checklist
				// widget and status lines ride extension_ui_request without
				// ever waiting for an answer.
				_ = enc.Encode(map[string]any{"type": "extension_ui_request", "id": "widget-1", "method": "setWidget", "widgetKey": "checklist", "widgetLines": []string{"checklist 1/2"}})
				_ = enc.Encode(map[string]any{"type": "extension_ui_request", "id": "status-1", "method": "setStatus", "statusKey": "branch", "statusText": "main"})
			}
			if behavior == "blocking-dialog" {
				_ = enc.Encode(map[string]any{"type": "extension_ui_request", "id": "ask-1", "method": "confirm", "title": "Proceed?"})
				time.Sleep(30 * time.Second)
				return
			}
			_ = enc.Encode(map[string]any{"type": "agent_start"})
			appendFakeSession(sessionPath, "user", cmd.Message)
			_ = enc.Encode(map[string]any{"type": "message_update", "assistantMessageEvent": map[string]any{"type": "text_delta", "delta": "Burst complete"}})
			if behavior == "wait-after-start" {
				time.Sleep(30 * time.Second)
			}
			time.Sleep(250 * time.Millisecond)
			appendFakeSession(sessionPath, "assistant", "Burst complete")
			message := map[string]any{"role": "assistant", "content": []map[string]string{{"type": "text", "text": "Burst complete"}}}
			_ = enc.Encode(map[string]any{"type": "message_end", "message": message})
			_ = enc.Encode(map[string]any{"type": "agent_end", "messages": []any{message}})
			time.Sleep(50 * time.Millisecond)
			_ = enc.Encode(map[string]any{"type": "agent_settled"})
		default:
			_ = enc.Encode(map[string]any{"id": cmd.ID, "type": "response", "command": cmd.Type, "success": true})
		}
	}
}

func appendFakeSession(path, role, text string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_ = json.NewEncoder(f).Encode(map[string]any{
		"type":    "message",
		"message": map[string]any{"role": role, "content": []map[string]string{{"type": "text", "text": text}}},
	})
}

func TestBurstDecisionTable(t *testing.T) {
	ready := burstPreflight{TmuxAvailable: true, SessionExists: true, PiAvailable: true, SessionSafe: true, LeaseClear: true}
	cases := []struct {
		name string
		in   burstPreflight
		want string
	}{
		{"ready", ready, ""},
		{"another burst", withBurstPreflight(ready, func(p *burstPreflight) { p.Active = true }), "already processing"},
		{"managed writer", withBurstPreflight(ready, func(p *burstPreflight) { p.Managed = true }), "no longer in its terminal"},
		{"stale lease", withBurstPreflight(ready, func(p *burstPreflight) { p.LeaseClear = false }), "still recovering"},
		{"tmux missing", withBurstPreflight(ready, func(p *burstPreflight) { p.TmuxAvailable = false }), "integration is unavailable"},
		{"session missing", withBurstPreflight(ready, func(p *burstPreflight) { p.SessionExists = false }), "terminal is no longer running"},
		{"tui working", withBurstPreflight(ready, func(p *burstPreflight) { p.TUIWorking = true }), "still working"},
		{"pi missing", withBurstPreflight(ready, func(p *burstPreflight) { p.PiAvailable = false }), "pi is not installed"},
		{"unsafe session", withBurstPreflight(ready, func(p *burstPreflight) { p.SessionSafe = false }), "identified safely"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := burstRefusal(tc.in)
			if tc.want == "" && got != "" {
				t.Fatalf("refused ready burst: %q", got)
			}
			if tc.want != "" && !strings.Contains(got, tc.want) {
				t.Fatalf("refusal = %q, want substring %q", got, tc.want)
			}
		})
	}
}

func withBurstPreflight(in burstPreflight, mutate func(*burstPreflight)) burstPreflight {
	mutate(&in)
	return in
}

func TestBurstCoordinatorIgnoresOldGeneration(t *testing.T) {
	b := NewBurstCoordinator()
	first, ctx, err := b.Reserve("a")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.Reserve("a"); err == nil {
		t.Fatal("second in-flight generation was accepted")
	}
	requested, cleared := b.Cancel("a", first.Generation)
	if !requested || cleared {
		t.Fatalf("cancel = requested %v cleared %v", requested, cleared)
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("burst context was not cancelled")
	}
	if !b.Clear("a", first.Generation) {
		t.Fatal("first generation not cleared")
	}
	second, _, err := b.Reserve("a")
	if err != nil {
		t.Fatal(err)
	}
	if second.Generation == first.Generation {
		t.Fatal("generation token reused")
	}
	if _, ok := b.Update("a", first.Generation, func(s *BurstState) { s.Phase = burstFailed }); ok {
		t.Fatal("late first-generation update changed the new burst")
	}
	if got := b.Snapshot("a"); got == nil || got.Generation != second.Generation || got.Phase != burstReceiving {
		t.Fatalf("new generation overwritten: %+v", got)
	}
}

func TestBurstCoordinatorBlocksReplyDuringControlMutation(t *testing.T) {
	b := NewBurstCoordinator()
	release := b.BeginControl("a")
	if _, _, err := b.Reserve("a"); err == nil || !strings.Contains(err.Error(), "terminal is changing") {
		t.Fatalf("reserve during control mutation = %v", err)
	}
	if _, err := b.TryBeginControl("a"); err == nil {
		t.Fatal("exclusive control overlapped an existing mutation")
	}
	release()
	release() // idempotent: a duplicated defer cannot underflow the guard
	exclusive, err := b.TryBeginControl("a")
	if err != nil {
		t.Fatalf("exclusive control after release: %v", err)
	}
	exclusive()
	st, _, err := b.Reserve("a")
	if err != nil || st.Generation == "" {
		t.Fatalf("reserve after control mutation = %+v, %v", st, err)
	}
}

func TestStartReplyBurstKeepsInboxOpenDuringControlMutation(t *testing.T) {
	st := newTestStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "workspace", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	question, err := st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: agent.ID,
		Reason: "test", Title: "Control race", Body: "Please answer", SessionPath: filepath.Join(t.TempDir(), "session.jsonl"),
	})
	if err != nil {
		t.Fatal(err)
	}
	bursts := NewBurstCoordinator()
	release := bursts.BeginControl(agent.ID)
	defer release()
	deps := Deps{Store: st, Bursts: bursts}
	if _, _, err := deps.startReplyBurst(context.Background(), question.ID, store.VerbRespond, "reply"); err == nil || !strings.Contains(err.Error(), "terminal is changing") {
		t.Fatalf("reply during control mutation = %v", err)
	}
	item, err := st.GetInboxItem(question.ID)
	if err != nil {
		t.Fatal(err)
	}
	if item.State != store.InboxUnread || item.Response != nil {
		t.Fatalf("refused reply mutated Inbox item: %+v", item)
	}
	if tasks, err := st.ListTasks(agent.ID, 10); err != nil || len(tasks) != 0 {
		t.Fatalf("refused reply tasks = %+v, %v", tasks, err)
	}
}

func TestBurstCoordinatorAllowsOnlyOneConcurrentReply(t *testing.T) {
	b := NewBurstCoordinator()
	const contenders = 24
	start := make(chan struct{})
	results := make(chan error, contenders)
	for i := 0; i < contenders; i++ {
		go func() {
			<-start
			_, _, err := b.Reserve("same-agent")
			results <- err
		}()
	}
	close(start)
	succeeded := 0
	for i := 0; i < contenders; i++ {
		if <-results == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("concurrent reservations succeeded = %d, want 1", succeeded)
	}
}

func TestBurstCancelHandlerRequiresCurrentGeneration(t *testing.T) {
	b := NewBurstCoordinator()
	st, burstCtx, err := b.Reserve("a")
	if err != nil {
		t.Fatal(err)
	}
	handler := handleBurstCancel(Deps{Bursts: b})
	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/agents/a/burst/cancel", strings.NewReader(body))
		req.SetPathValue("id", "a")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if got := call(`{}`).Code; got != http.StatusBadRequest {
		t.Fatalf("missing generation status = %d", got)
	}
	if got := call(`{"generation":"stale"}`).Code; got != http.StatusOK {
		t.Fatalf("stale generation status = %d", got)
	}
	select {
	case <-burstCtx.Done():
		t.Fatal("stale generation cancelled the current burst")
	default:
	}
	if got := call(`{"generation":"` + st.Generation + `"}`).Code; got != http.StatusOK {
		t.Fatalf("current generation status = %d", got)
	}
	select {
	case <-burstCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("current generation was not cancelled")
	}
}

func TestBurstCancelKeepsTerminalUnavailableCardForRestart(t *testing.T) {
	b := NewBurstCoordinator()
	burst, _, err := b.Reserve("a")
	if err != nil {
		t.Fatal(err)
	}
	b.Update("a", burst.Generation, func(s *BurstState) {
		s.Phase = burstFailed
		s.TerminalUnavailable = true
	})
	handler := handleBurstCancel(Deps{Bursts: b})
	req := httptest.NewRequest(http.MethodPost, "/api/agents/a/burst/cancel", strings.NewReader(`{"generation":"`+burst.Generation+`"}`))
	req.SetPathValue("id", "a")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("terminal-unavailable cancel status = %d", rec.Code)
	}
	if got := b.Snapshot("a"); got == nil || got.Generation != burst.Generation || !got.TerminalUnavailable {
		t.Fatalf("terminal-unavailable cancel cleared recovery card: %+v", got)
	}
}

func TestBurstCoordinatorCancelAllWaitsForTerminalPhase(t *testing.T) {
	b := NewBurstCoordinator()
	st, burstCtx, err := b.Reserve("a")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		<-burstCtx.Done()
		time.Sleep(75 * time.Millisecond)
		b.Update("a", st.Generation, func(s *BurstState) { s.Phase = burstFailed })
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	started := time.Now()
	if err := b.CancelAllAndWait(ctx); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) < 50*time.Millisecond {
		t.Fatal("shutdown returned before the burst reached a restored terminal phase")
	}
}

func TestProjectBurstEventShowsOutputButNotThoughts(t *testing.T) {
	deps := Deps{Bursts: NewBurstCoordinator()}
	st, _, err := deps.Bursts.Reserve("a")
	if err != nil {
		t.Fatal(err)
	}
	blocked := make(chan struct{}, 1)
	deps.projectBurstEvent("a", st.Generation, rpc.Event(`{"type":"message_update","assistantMessageEvent":{"type":"thinking_delta","delta":"private thought"}}`), blocked)
	deps.projectBurstEvent("a", st.Generation, rpc.Event(`{"type":"message_update","assistantMessageEvent":{"type":"text_delta","delta":"Visible answer"}}`), blocked)
	got := deps.Bursts.Snapshot("a")
	if got == nil || got.Output != "Visible answer" || strings.Contains(got.Output, "private") {
		t.Fatalf("projected state = %+v", got)
	}
	// Passive extension UI is decoration: it must never stop a reply.
	for _, method := range []string{"", "notify", "setStatus", "setWidget", "setTitle", "set_editor_text"} {
		deps.projectBurstEvent("a", st.Generation, rpc.Event(`{"type":"extension_ui_request","id":"ui-1","method":"`+method+`"}`), blocked)
		select {
		case <-blocked:
			t.Fatalf("passive extension UI method %q stopped the reply", method)
		default:
		}
	}
	for _, method := range []string{"select", "confirm", "input", "editor"} {
		deps.projectBurstEvent("a", st.Generation, rpc.Event(`{"type":"extension_ui_request","id":"ui-2","method":"`+method+`"}`), blocked)
		select {
		case <-blocked:
		default:
			t.Fatalf("blocking extension UI method %q was not signalled", method)
		}
		select {
		case <-blocked:
			t.Fatalf("blocked channel did not drain for method %q", method)
		default:
		}
	}
}

func TestAppendBurstOutputKeepsUTF8AtCap(t *testing.T) {
	got := appendBurstOutput(strings.Repeat("🙂", 25_001), "x")
	if len(got) > 100_000 || !utf8.ValidString(got) || !strings.HasSuffix(got, "x") {
		t.Fatalf("invalid capped output: bytes=%d valid=%v", len(got), utf8.ValidString(got))
	}
}

func TestResolveBurstSessionRequiresItemSessionPath(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cwd := t.TempDir()
	path := filepath.Join(session.Dir(cwd), "exact.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{\"type\":\"session\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent := store.Agent{ID: "exact-agent", SessionPath: &path}
	deps := Deps{}
	if got, ok := deps.resolveBurstSession(agent, store.InboxItem{}, cwd); ok || got != "" {
		t.Fatalf("missing item session fell back to %q", got)
	}
	if got, ok := deps.resolveBurstSession(agent, store.InboxItem{SessionPath: path}, cwd); !ok || got != path {
		t.Fatalf("captured item session = %q, %v", got, ok)
	}
}

func TestInstallBurstSessionRollsBackPointerWhenPaneInstallFails(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ws, _ := st.AddWorkspace("burst rollback", t.TempDir())
	agent, _ := st.AddAgent(ws.ID, "agent", "")
	prior := filepath.Join(t.TempDir(), "prior.jsonl")
	exact := filepath.Join(t.TempDir(), "exact.jsonl")
	agent, err = st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &prior})
	if err != nil {
		t.Fatal(err)
	}
	installErr := errors.New("pane disappeared")
	seenExact := false
	_, err = (Deps{Store: st}).installBurstSession(agent, exact, func(selected store.Agent) error {
		seenExact = selected.SessionPath != nil && *selected.SessionPath == exact
		persisted, _ := st.GetAgent(agent.ID)
		if persisted.SessionPath == nil || *persisted.SessionPath != exact {
			t.Fatalf("exact session was not persisted before pane install: %+v", persisted.SessionPath)
		}
		return installErr
	})
	if !errors.Is(err, installErr) || !seenExact {
		t.Fatalf("install error = %v, saw exact=%v", err, seenExact)
	}
	rolledBack, _ := st.GetAgent(agent.ID)
	if rolledBack.SessionPath == nil || *rolledBack.SessionPath != prior {
		t.Fatalf("session pointer after failed install = %+v, want %q", rolledBack.SessionPath, prior)
	}
}

func TestRequireBurstSessionIsExact(t *testing.T) {
	want := filepath.Join(t.TempDir(), "session.jsonl")
	body, _ := json.Marshal(map[string]string{"sessionFile": want})
	if err := requireBurstSession(rpc.Response{Data: body}, want); err != nil {
		t.Fatalf("exact session refused: %v", err)
	}
	body, _ = json.Marshal(map[string]string{"sessionFile": want + ".other"})
	if err := requireBurstSession(rpc.Response{Data: body}, want); err == nil {
		t.Fatal("different session accepted")
	}
}

func TestWaitForRestoredPaneUsesLeaseInsteadOfExecutableName(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux is not installed")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "pi")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nsleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	name := tmux.SessionName("burst-interpreter-" + strconv.FormatInt(time.Now().UnixNano(), 36))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := manager.NewSession(ctx, name, dir, script); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })

	marker := filepath.Join(dir, "reply.hold")
	if err := os.WriteFile(marker, []byte("released\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	blockedCtx, blockedCancel := context.WithTimeout(context.Background(), 220*time.Millisecond)
	if err := waitForRestoredPane(blockedCtx, manager, name, marker); err == nil {
		blockedCancel()
		t.Fatal("live interpreter was accepted before its holder lease cleared")
	}
	blockedCancel()
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	readyCtx, readyCancel := context.WithTimeout(context.Background(), time.Second)
	defer readyCancel()
	if err := waitForRestoredPane(readyCtx, manager, name, marker); err != nil {
		t.Fatalf("interpreter-backed restored pane = %v", err)
	}
	if command, err := manager.PaneCommand(context.Background(), name); err != nil || command != "sh" {
		t.Fatalf("pane command = %q, %v; test did not exercise an interpreter-backed command", command, err)
	}
}

func TestRestoreFallbackGetsFreshDeadline(t *testing.T) {
	calls := 0
	fallbackCalled := false
	err := restoreWithFallback(50*time.Millisecond,
		func(ctx context.Context) error {
			calls++
			if calls == 1 {
				<-ctx.Done()
				return ctx.Err()
			}
			return ctx.Err()
		},
		func(ctx context.Context) error {
			fallbackCalled = true
			return ctx.Err()
		},
	)
	if err != nil || !fallbackCalled || calls != 2 {
		t.Fatalf("fallback did not receive a fresh deadline: calls=%d fallback=%v err=%v", calls, fallbackCalled, err)
	}
}

func TestOpenAgentTUIRestartReplacesStalePane(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	st, err := store.Open(filepath.Join(home, ".picode", "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cwd := t.TempDir()
	ws, _ := st.AddWorkspace("restart workspace", cwd)
	agent, _ := st.AddAgent(ws.ID, "burst-e2e-restart", cwd)
	name := tmux.SessionName(agent.ID)
	if err := manager.NewSession(context.Background(), name, cwd, "sleep", "30"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	before, err := manager.PaneSessionID(context.Background(), name)
	if err != nil {
		t.Fatal(err)
	}
	runtime := rpc.NewRuntime(os.Args[0], st, nil)
	changes := &feed.Feed{Store: st}
	var burstNotices []BurstState
	changes.Listen(func(ev store.Event) {
		if ev.Type != "agent.burst" {
			return
		}
		var state BurstState
		if json.Unmarshal(ev.Data, &state) == nil {
			burstNotices = append(burstNotices, state)
		}
	})
	deps := Deps{Store: st, Runtime: runtime, Tmux: manager, AgentCmd: os.Args[0], DataDir: filepath.Join(home, ".picode"), Bursts: NewBurstCoordinator(), Feed: changes}
	if running, err := deps.openAgentTUI(context.Background(), agent.ID, false); err != nil || !running {
		t.Fatalf("ordinary open = already %v, %v", running, err)
	}
	preserved, _ := manager.PaneSessionID(context.Background(), name)
	if preserved != before {
		t.Fatalf("ordinary open replaced session: %q -> %q", before, preserved)
	}
	burst, _, err := deps.Bursts.Reserve(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	deps.Bursts.Update(agent.ID, burst.Generation, func(s *BurstState) {
		s.Phase = burstFailed
		s.TerminalUnavailable = true
		s.Error = "The terminal could not be restored automatically."
	})
	marker, err := createBurstMarker(deps.DataDir, agent.ID, burst.Generation)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/agents/"+agent.ID+"/open?restart=1", nil)
	req.SetPathValue("id", agent.ID)
	rec := httptest.NewRecorder()
	handleAgentOpen(deps).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("restart open status = %d: %s", rec.Code, rec.Body.String())
	}
	after, err := manager.PaneSessionID(context.Background(), name)
	if err != nil || after == before {
		t.Fatalf("restart did not replace session: %q -> %q (%v)", before, after, err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("restart left stale burst marker: %v", err)
	}
	if state := deps.Bursts.Snapshot(agent.ID); state != nil {
		t.Fatalf("successful restart left recovery card: %+v", state)
	}
	if len(burstNotices) == 0 || burstNotices[len(burstNotices)-1].Phase != burstIdle || burstNotices[len(burstNotices)-1].TerminalUnavailable {
		t.Fatalf("successful restart final notice = %+v", burstNotices)
	}
	selected, err := st.GetAgent(agent.ID)
	if err != nil || selected.LastStatus != store.StatusRunning {
		t.Fatalf("successful restart runtime state = %+v, %v", selected, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if command, err := manager.PaneCommand(context.Background(), name); err == nil && command == filepath.Base(os.Args[0]) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("restarted pane did not launch the TUI command")
}

func TestOpenAgentTUIRestartFailureKeepsRecoveryCard(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	st, err := store.Open(filepath.Join(home, ".picode", "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cwd := t.TempDir()
	ws, _ := st.AddWorkspace("failed restart workspace", cwd)
	agent, _ := st.AddAgent(ws.ID, "failed-restart", cwd)
	bursts := NewBurstCoordinator()
	burst, _, err := bursts.Reserve(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	bursts.Update(agent.ID, burst.Generation, func(s *BurstState) {
		s.Phase = burstFailed
		s.TerminalUnavailable = true
	})
	missing := filepath.Join(t.TempDir(), "missing-pi")
	deps := Deps{
		Store: st, Runtime: rpc.NewRuntime(missing, st, nil), Tmux: manager,
		AgentCmd: missing, DataDir: filepath.Join(home, ".picode"), Bursts: bursts,
	}
	if _, err := deps.openAgentTUI(context.Background(), agent.ID, true); !errors.Is(err, errAgentCmdMissing) {
		t.Fatalf("failed restart = %v, want missing pi", err)
	}
	state := bursts.Snapshot(agent.ID)
	if state == nil || state.Generation != burst.Generation || state.Phase != burstFailed || !state.TerminalUnavailable {
		t.Fatalf("failed restart recovery card = %+v", state)
	}
	if !strings.Contains(state.Error, "try again") {
		t.Fatalf("failed restart has no actionable retry copy: %+v", state)
	}
	if err := bursts.CheckControl(agent.ID); err != nil {
		t.Fatalf("failed restart leaked control guard: %v", err)
	}
}

func TestReplyBurstEndToEndKeepsTmuxAndSession(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	dataDir := filepath.Join(home, ".picode")
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cwd := t.TempDir()
	ws, _ := st.AddWorkspace("burst workspace", cwd)
	agent, err := st.AddAgent(ws.ID, "burst-e2e-helper", cwd)
	if err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(session.Dir(cwd), "exact.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, []byte("{\"type\":\"session\",\"id\":\"exact\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent, err = st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &sessionPath})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "interactive"); err != nil {
		t.Fatal(err)
	}
	question, err := st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: agent.ID,
		Reason: "test", Title: "Continue?", Body: "Please answer", SessionPath: sessionPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	unrelated, err := st.EnqueueTask(agent.ID, store.TaskPrompt, "unrelated queued work", "user")
	if err != nil {
		t.Fatal(err)
	}

	name := tmux.SessionName(agent.ID)
	env := append(agent.SpawnEnv(), "GO_WANT_FAKE_PI=1")
	if err := manager.NewSessionEnv(context.Background(), name, cwd, env, os.Args[0], agent.CLIFlagsForSpawn("")...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	before, err := manager.PaneSessionID(context.Background(), name)
	if err != nil {
		t.Fatal(err)
	}

	runtime := rpc.NewRuntime(os.Args[0], st, nil)
	runtime.DataDir = dataDir
	deps := Deps{Store: st, Runtime: runtime, Tmux: manager, AgentCmd: os.Args[0], DataDir: dataDir, Bursts: NewBurstCoordinator()}
	agentID, generation, err := deps.startReplyBurst(context.Background(), question.ID, store.VerbRespond, "continue in place")
	if err != nil {
		t.Fatal(err)
	}
	if agentID != agent.ID || generation == "" {
		t.Fatalf("start = %q %q", agentID, generation)
	}
	seen := map[string]bool{burstReceiving: true}
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		state := deps.Bursts.Snapshot(agent.ID)
		if state == nil {
			break
		}
		seen[state.Phase] = true
		if state.Phase == burstFailed {
			t.Fatalf("burst failed: %+v", state)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if state := deps.Bursts.Snapshot(agent.ID); state != nil {
		t.Fatalf("burst did not restore: %+v", state)
	}
	if !seen[burstProcessing] || !seen[burstRestoring] {
		t.Fatalf("lifecycle phases seen = %v", seen)
	}
	after, err := manager.PaneSessionID(context.Background(), name)
	if err != nil || before != after {
		t.Fatalf("tmux session changed: %q -> %q (%v)", before, after, err)
	}
	if command, err := manager.PaneCommand(context.Background(), name); err != nil || command != filepath.Base(os.Args[0]) {
		t.Fatalf("restored pane command = %q, %v", command, err)
	}
	body, err := os.ReadFile(sessionPath)
	if err != nil || !strings.Contains(string(body), "continue in place") || !strings.Contains(string(body), "Burst complete") {
		t.Fatalf("session continuity missing: %v\n%s", err, body)
	}
	tasks, _ := st.ListTasks(agent.ID, 10)
	statuses := map[string]string{}
	for _, task := range tasks {
		statuses[task.ID] = task.Status
	}
	if statuses[unrelated.ID] != store.TaskQueued {
		t.Fatalf("unrelated task was drained: %+v", tasks)
	}
	var burstTask store.Task
	for _, task := range tasks {
		if strings.HasPrefix(task.Source, "inbox-burst:") {
			burstTask = task
		}
	}
	if burstTask.Status != store.TaskDelivered || burstTask.Attempts != 1 {
		t.Fatalf("burst task = %+v", burstTask)
	}
	item, _ := st.GetInboxItem(question.ID)
	if item.State != store.InboxDone {
		t.Fatalf("question state = %s", item.State)
	}
	if runtime.Get(agent.ID) != nil {
		t.Fatal("transient runtime survived restoration")
	}
	if mode := deps.runMode(httptest.NewRequest("GET", "/", nil), agent.ID); mode != modeInteractive {
		t.Fatalf("logical mode = %s", mode)
	}
}

func TestReplyBurstIgnoresPassiveExtensionUI(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PICODE_FAKE_BEHAVIOR", "passive-widget")
	dataDir := filepath.Join(home, ".picode")
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cwd := t.TempDir()
	ws, _ := st.AddWorkspace("burst widget workspace", cwd)
	agent, _ := st.AddAgent(ws.ID, "burst-e2e-widget", cwd)
	sessionPath := filepath.Join(session.Dir(cwd), "exact.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, []byte("{\"type\":\"session\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent, err = st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &sessionPath})
	if err != nil {
		t.Fatal(err)
	}
	_ = st.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "interactive")
	question, _ := st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: agent.ID,
		Reason: "test", Title: "Continue?", Body: "Please answer", SessionPath: sessionPath,
	})
	name := tmux.SessionName(agent.ID)
	if err := manager.NewSessionEnv(context.Background(), name, cwd, append(agent.SpawnEnv(), "GO_WANT_FAKE_PI=1"), os.Args[0], agent.CLIFlagsForSpawn("")...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	before, err := manager.PaneSessionID(context.Background(), name)
	if err != nil {
		t.Fatal(err)
	}
	runtime := rpc.NewRuntime(os.Args[0], st, nil)
	runtime.DataDir = dataDir
	deps := Deps{Store: st, Runtime: runtime, Tmux: manager, AgentCmd: os.Args[0], DataDir: dataDir, Bursts: NewBurstCoordinator()}
	if _, _, err := deps.startReplyBurst(context.Background(), question.ID, store.VerbRespond, "widget-safe answer"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		if deps.Bursts.Snapshot(agent.ID) == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if state := deps.Bursts.Snapshot(agent.ID); state != nil {
		t.Fatalf("burst did not restore: %+v", state)
	}
	tasks, _ := st.ListTasks(agent.ID, 10)
	var burstTask store.Task
	for _, task := range tasks {
		if strings.HasPrefix(task.Source, "inbox-burst:") {
			burstTask = task
		}
	}
	if burstTask.Status != store.TaskDelivered || burstTask.Attempts != 1 {
		t.Fatalf("widget-decorated burst task = %+v", burstTask)
	}
	item, _ := st.GetInboxItem(question.ID)
	if item.State != store.InboxDone {
		t.Fatalf("question state = %s", item.State)
	}
	body, err := os.ReadFile(sessionPath)
	if err != nil || !strings.Contains(string(body), "widget-safe answer") {
		t.Fatalf("session continuity missing: %v", err)
	}
	after, err := manager.PaneSessionID(context.Background(), name)
	if err != nil || before != after {
		t.Fatalf("tmux session changed: %q -> %q (%v)", before, after, err)
	}
}

func TestReplyBurstFailsOnBlockingExtensionDialog(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PICODE_FAKE_BEHAVIOR", "blocking-dialog")
	dataDir := filepath.Join(home, ".picode")
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cwd := t.TempDir()
	ws, _ := st.AddWorkspace("burst dialog workspace", cwd)
	agent, _ := st.AddAgent(ws.ID, "burst-e2e-dialog", cwd)
	sessionPath := filepath.Join(session.Dir(cwd), "exact.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, []byte("{\"type\":\"session\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent, err = st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &sessionPath})
	if err != nil {
		t.Fatal(err)
	}
	_ = st.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "interactive")
	question, _ := st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: agent.ID,
		Reason: "test", Title: "Continue?", Body: "Please answer", SessionPath: sessionPath,
	})
	name := tmux.SessionName(agent.ID)
	if err := manager.NewSessionEnv(context.Background(), name, cwd, append(agent.SpawnEnv(), "GO_WANT_FAKE_PI=1"), os.Args[0], agent.CLIFlagsForSpawn("")...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	before, _ := manager.PaneSessionID(context.Background(), name)
	runtime := rpc.NewRuntime(os.Args[0], st, nil)
	runtime.DataDir = dataDir
	deps := Deps{Store: st, Runtime: runtime, Tmux: manager, AgentCmd: os.Args[0], DataDir: dataDir, Bursts: NewBurstCoordinator()}
	if _, _, err := deps.startReplyBurst(context.Background(), question.ID, store.VerbRespond, "answer behind a dialog"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(15 * time.Second)
	var failed *BurstState
	for time.Now().Before(deadline) {
		state := deps.Bursts.Snapshot(agent.ID)
		if state != nil && state.Phase == burstFailed {
			failed = state
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if failed == nil || failed.Error == "" {
		t.Fatalf("failure state = %+v", failed)
	}
	tasks, _ := st.ListTasks(agent.ID, 10)
	var burstTask store.Task
	for _, task := range tasks {
		if strings.HasPrefix(task.Source, "inbox-burst:") {
			burstTask = task
		}
	}
	if burstTask.Status != store.TaskFailed || burstTask.Attempts != 3 {
		t.Fatalf("dialog-blocked task = %+v", burstTask)
	}
	item, _ := st.GetInboxItem(question.ID)
	if item.State != store.InboxUnread || item.Response == nil || !strings.Contains(*item.Response, "answer behind a dialog") {
		t.Fatalf("reopened inbox item = %+v", item)
	}
	body, _ := os.ReadFile(sessionPath)
	if strings.Contains(string(body), "answer behind a dialog") {
		t.Fatalf("dialog-blocked fake materialized the message: %s", body)
	}
	after, err := manager.PaneSessionID(context.Background(), name)
	if err != nil || before != after {
		t.Fatalf("tmux session changed on dialog failure: %q -> %q (%v)", before, after, err)
	}
	if runtime.Get(agent.ID) != nil {
		t.Fatal("dialog-blocked transient runtime survived")
	}
}

func TestReplyBurstCancelRestoresPaneForManualTakeover(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PICODE_FAKE_BEHAVIOR", "wait-after-start")
	dataDir := filepath.Join(home, ".picode")
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cwd := t.TempDir()
	ws, _ := st.AddWorkspace("burst cancel workspace", cwd)
	agent, _ := st.AddAgent(ws.ID, "burst-e2e-cancel", cwd)
	sessionPath := filepath.Join(session.Dir(cwd), "exact.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, []byte("{\"type\":\"session\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent, err = st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &sessionPath})
	if err != nil {
		t.Fatal(err)
	}
	_ = st.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "interactive")
	question, _ := st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: agent.ID,
		Reason: "test", Title: "Continue?", Body: "Please answer", SessionPath: sessionPath,
	})
	name := tmux.SessionName(agent.ID)
	if err := manager.NewSessionEnv(context.Background(), name, cwd, agent.SpawnEnv(), os.Args[0], agent.CLIFlagsForSpawn("")...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	before, _ := manager.PaneSessionID(context.Background(), name)
	runtime := rpc.NewRuntime(os.Args[0], st, nil)
	runtime.DataDir = dataDir
	deps := Deps{Store: st, Runtime: runtime, Tmux: manager, AgentCmd: os.Args[0], DataDir: dataDir, Bursts: NewBurstCoordinator()}
	_, _, err = deps.startReplyBurst(context.Background(), question.ID, store.VerbRespond, "cancel me after delivery")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(8 * time.Second)
	reachedProcessing := false
	for time.Now().Before(deadline) {
		state := deps.Bursts.Snapshot(agent.ID)
		body, _ := os.ReadFile(sessionPath)
		if state != nil && state.Phase == burstProcessing && strings.Contains(string(body), "cancel me after delivery") {
			reachedProcessing = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !reachedProcessing {
		t.Fatal("burst did not reach materialized processing before cancellation")
	}
	if runtime.Get(agent.ID) != nil || !runtime.Active(agent.ID) {
		t.Fatal("transient writer leaked into ordinary managed control")
	}
	if mode := deps.runMode(httptest.NewRequest("GET", "/", nil), agent.ID); mode != modeInteractive {
		t.Fatalf("burst leaked logical mode %q", mode)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	release, err := deps.cancelBurstAndWait(ctx, agent.ID)
	if err != nil {
		t.Fatalf("manual takeover cancellation: %v", err)
	}
	if state := deps.Bursts.Snapshot(agent.ID); state != nil {
		t.Fatalf("burst state survived takeover: %+v", state)
	}
	if runtime.Get(agent.ID) != nil {
		t.Fatal("transient writer survived takeover")
	}
	after, err := manager.PaneSessionID(context.Background(), name)
	if err != nil || before != after {
		t.Fatalf("tmux session changed on cancel: %q -> %q (%v)", before, after, err)
	}
	if command, err := manager.PaneCommand(context.Background(), name); err != nil || command != filepath.Base(os.Args[0]) {
		t.Fatalf("restored pane command = %q, %v", command, err)
	}
	release()

	// Simulate a stale selected-session pointer: the item-owned path must win
	// for both the temporary writer and the restored TUI.
	otherSession := filepath.Join(session.Dir(cwd), "other.jsonl")
	if err := os.WriteFile(otherSession, []byte("{\"type\":\"session\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &otherSession}); err != nil {
		t.Fatal(err)
	}

	// The agent-scoped Stop used by the mobile header must cancel and await
	// a second burst before it kills the holder's pane.
	question2, _ := st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: agent.ID,
		Reason: "test", Title: "Stop?", Body: "Please answer", SessionPath: sessionPath,
	})
	if _, _, err := deps.startReplyBurst(context.Background(), question2.ID, store.VerbRespond, "stop the whole agent"); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.Active(agent.ID) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !runtime.Active(agent.ID) {
		t.Fatal("second burst did not start")
	}
	selected, _ := st.GetAgent(agent.ID)
	if selected.SessionPath == nil || filepath.Clean(*selected.SessionPath) != filepath.Clean(sessionPath) {
		t.Fatalf("item-owned session did not replace stale pointer: %+v", selected.SessionPath)
	}
	deadline = time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		exactBody, _ := os.ReadFile(sessionPath)
		otherBody, _ := os.ReadFile(otherSession)
		if strings.Contains(string(otherBody), "stop the whole agent") {
			t.Fatal("stale selected session received the reply")
		}
		if strings.Contains(string(exactBody), "stop the whole agent") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	exactBody, _ := os.ReadFile(sessionPath)
	if !strings.Contains(string(exactBody), "stop the whole agent") {
		t.Fatal("item-owned session did not materialize the second reply")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/agents/"+agent.ID+"/close", nil)
	req.SetPathValue("id", agent.ID)
	rec := httptest.NewRecorder()
	handleAgentClose(deps).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("agent stop status = %d: %s", rec.Code, rec.Body.String())
	}
	if runtime.Active(agent.ID) || deps.Bursts.Snapshot(agent.ID) != nil {
		t.Fatal("agent stop left a burst writer or state behind")
	}
	if exists, err := manager.HasSession(context.Background(), name); err != nil || exists {
		t.Fatalf("agent stop left tmux session: exists=%v err=%v", exists, err)
	}
}

func TestReplyBurstRetriesExactTaskThenReopensInbox(t *testing.T) {
	manager := tmux.New()
	if !manager.Available() {
		t.Skip("tmux not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PICODE_FAKE_BEHAVIOR", "exit-before-start")
	dataDir := filepath.Join(home, ".picode")
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cwd := t.TempDir()
	ws, _ := st.AddWorkspace("burst failure workspace", cwd)
	agent, _ := st.AddAgent(ws.ID, "burst-e2e-failure", cwd)
	sessionPath := filepath.Join(session.Dir(cwd), "exact.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sessionPath, []byte("{\"type\":\"session\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	agent, err = st.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &sessionPath})
	if err != nil {
		t.Fatal(err)
	}
	_ = st.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "interactive")
	question, _ := st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: agent.ID,
		Reason: "test", Title: "Continue?", Body: "Please answer", SessionPath: sessionPath,
	})
	unrelated, _ := st.EnqueueTask(agent.ID, store.TaskPrompt, "older task", "user")
	name := tmux.SessionName(agent.ID)
	if err := manager.NewSessionEnv(context.Background(), name, cwd, append(agent.SpawnEnv(), "GO_WANT_FAKE_PI=1"), os.Args[0], agent.CLIFlagsForSpawn("")...); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	before, _ := manager.PaneSessionID(context.Background(), name)
	runtime := rpc.NewRuntime(os.Args[0], st, nil)
	runtime.DataDir = dataDir
	deps := Deps{Store: st, Runtime: runtime, Tmux: manager, AgentCmd: os.Args[0], DataDir: dataDir, Bursts: NewBurstCoordinator()}
	_, generation, err := deps.startReplyBurst(context.Background(), question.ID, store.VerbRespond, "answer that must retry")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(12 * time.Second)
	var failed *BurstState
	for time.Now().Before(deadline) {
		state := deps.Bursts.Snapshot(agent.ID)
		if state != nil && state.Phase == burstFailed {
			failed = state
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if failed == nil || failed.Generation != generation || failed.Error == "" {
		t.Fatalf("failure state = %+v", failed)
	}
	after, err := manager.PaneSessionID(context.Background(), name)
	if err != nil || before != after {
		t.Fatalf("tmux session changed on failure: %q -> %q (%v)", before, after, err)
	}
	if runtime.Get(agent.ID) != nil {
		t.Fatal("failed transient runtime survived")
	}
	tasks, _ := st.ListTasks(agent.ID, 10)
	var retried store.Task
	for _, task := range tasks {
		if strings.HasPrefix(task.Source, "inbox-burst:") {
			retried = task
		}
		if task.ID == unrelated.ID && task.Status != store.TaskQueued {
			t.Fatalf("unrelated task was claimed: %+v", task)
		}
	}
	if retried.Status != store.TaskFailed || retried.Attempts != 3 {
		t.Fatalf("retried task = %+v", retried)
	}
	item, _ := st.GetInboxItem(question.ID)
	if item.State != store.InboxUnread || item.Response == nil || !strings.Contains(*item.Response, "answer that must retry") || !strings.Contains(item.Body, "Send it again") {
		t.Fatalf("reopened inbox item = %+v", item)
	}
	body, _ := os.ReadFile(sessionPath)
	if strings.Contains(string(body), "answer that must retry") {
		t.Fatalf("failed fake materialized the message: %s", body)
	}
	requested, cleared := deps.Bursts.Cancel(agent.ID, generation)
	if requested || !cleared || deps.Bursts.Snapshot(agent.ID) != nil {
		t.Fatalf("failed card did not dismiss: requested=%v cleared=%v", requested, cleared)
	}
}

func TestBurstHolderReleaseKillsLeasedRPCAcrossReexec(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX crash-holder test")
	}
	dataDir := t.TempDir()
	dir := filepath.Join(dataDir, "bursts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "burst-test.hold")
	restored := filepath.Join(dir, "restored")
	restore := filepath.Join(dir, "restore.sh")
	if err := os.WriteFile(restore, []byte("#!/bin/sh\nprintf restored > \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	worker := exec.Command("sleep", "30")
	if err := worker.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = worker.Process.Kill() }()
	workerDone := make(chan error, 1)
	go func() { workerDone <- worker.Wait() }()
	if err := os.WriteFile(marker, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := recordBurstRPCPID(marker, worker.Process.Pid); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	holder := exec.CommandContext(ctx, "/bin/sh", burstHolderArgs(marker, restore, []string{restored})...)
	holderDone := make(chan error, 1)
	go func() { holderDone <- holder.Run() }()
	time.Sleep(350 * time.Millisecond)
	if err := ReleaseInterruptedBurstMarkers(dataDir); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-holderDone:
		if err != nil {
			t.Fatalf("holder: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("holder did not restore after startup released the old lease")
	}
	select {
	case <-workerDone:
	case <-time.After(time.Second):
		t.Fatal("leased RPC process survived startup release")
	}
	if body, err := os.ReadFile(restored); err != nil || string(body) != "restored" {
		t.Fatalf("restore output = %q, %v", body, err)
	}
}

func TestReleaseInterruptedBurstMarkersClearsLeaseWithoutHolder(t *testing.T) {
	dataDir := t.TempDir()
	dir := filepath.Join(dataDir, "bursts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "pre-holder.hold")
	if err := os.WriteFile(marker, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := releaseInterruptedBurstMarkers(dataDir, 100*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out waiting") {
		t.Fatalf("orphan release error = %v", err)
	}
	if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("orphan lease still blocks replies: %v", statErr)
	}
}

func TestReleaseInterruptedBurstMarkersRetainsPossibleLiveWriter(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("/proc PID identity check is Linux-specific")
	}
	dataDir := t.TempDir()
	dir := filepath.Join(dataDir, "bursts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "live-writer.hold")
	if err := os.WriteFile(marker, []byte(fmt.Sprintf("%d\n%d\n", os.Getpid(), os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := releaseInterruptedBurstMarkers(dataDir, 100*time.Millisecond); err == nil {
		t.Fatal("possible live writer did not fail closed")
	}
	if !BurstRecoveryPending(dataDir) {
		t.Fatal("possible live writer lease was removed")
	}
}

func TestReleaseInterruptedBurstMarkersContinuesPastDamagedLease(t *testing.T) {
	dataDir := t.TempDir()
	dir := filepath.Join(dataDir, "bursts")
	if err := os.MkdirAll(filepath.Join(dir, "bad.hold"), 0o700); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(dir, "good.hold")
	if err := os.WriteFile(good, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
			body, _ := os.ReadFile(good)
			if strings.HasPrefix(string(body), "released") {
				_ = os.Remove(good)
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	err := ReleaseInterruptedBurstMarkers(dataDir)
	<-done
	if err == nil || !strings.Contains(err.Error(), "bad.hold") {
		t.Fatalf("release error = %v, want damaged marker detail", err)
	}
	if _, statErr := os.Stat(good); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("healthy marker was not released: %v", statErr)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.hold"))
	if len(matches) != 0 {
		t.Fatalf("damaged lease still blocks future replies: %v", matches)
	}
}

func TestBurstHolderKillsLeasedRPCAndRestores(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX crash-holder test")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "burst.hold")
	restored := filepath.Join(dir, "restored")
	restore := filepath.Join(dir, "restore.sh")
	if err := os.WriteFile(restore, []byte("#!/bin/sh\nprintf restored > \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	daemon := exec.Command("sleep", "30")
	if err := daemon.Start(); err != nil {
		t.Fatal(err)
	}
	daemonDone := make(chan error, 1)
	go func() { daemonDone <- daemon.Wait() }()
	defer func() { _ = daemon.Process.Kill() }()
	worker := exec.Command("sleep", "30")
	if err := worker.Start(); err != nil {
		t.Fatal(err)
	}
	workerDone := make(chan error, 1)
	go func() { workerDone <- worker.Wait() }()
	if err := os.WriteFile(marker, []byte(fmt.Sprintf("%d\n%d\n", daemon.Process.Pid, worker.Process.Pid)), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	holderArgs := burstHolderArgs(marker, restore, []string{restored})
	holderArgs[3] = strconv.Itoa(daemon.Process.Pid)
	holder := exec.CommandContext(ctx, "/bin/sh", holderArgs...)
	holderDone := make(chan error, 1)
	go func() { holderDone <- holder.Run() }()
	time.Sleep(350 * time.Millisecond)
	if _, err := os.Stat(restored); !os.IsNotExist(err) {
		t.Fatalf("holder restored while daemon was alive: %v", err)
	}
	if err := daemon.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-daemonDone:
	case <-time.After(time.Second):
		t.Fatal("fake daemon did not exit")
	}
	select {
	case err := <-holderDone:
		if err != nil {
			t.Fatalf("holder: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("holder did not restore after daemon death")
	}
	select {
	case <-workerDone:
	case <-time.After(time.Second):
		_ = worker.Process.Kill()
		t.Fatal("leased RPC process survived holder recovery")
	}
	if err := worker.Process.Signal(syscall.Signal(0)); err == nil {
		t.Fatal("leased RPC process still exists")
	}
	if body, err := os.ReadFile(restored); err != nil || string(body) != "restored" {
		t.Fatalf("restore output = %q, %v", body, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("marker survived recovery: %v", err)
	}
}
