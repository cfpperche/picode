package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// answerHarness is one agent of any CLI with a question in the Inbox. When
// running, its tmux pane is a fake TUI that appends every submitted line to
// typed (the bracketed-paste markers stripped, as a real editor consumes
// them) and, given a session file, also writes it there as a user row.
type answerHarness struct {
	deps  Deps
	store *store.Store
	agent store.Agent
	item  store.InboxItem
	typed string
}

func newAnswerHarness(t *testing.T, cli string, running bool, sessionPath func(dataDir, agentID string) string) *answerHarness {
	t.Helper()
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
	t.Cleanup(func() { _ = st.Close() })
	cwd := t.TempDir()
	ws, _ := st.AddWorkspace("answer workspace", cwd)
	agent, err := st.AddAgentWithCLI(ws.ID, cli, "answer-agent", cwd)
	if err != nil {
		t.Fatal(err)
	}
	session := ""
	if sessionPath != nil {
		session = sessionPath(dataDir, agent.ID)
		if err := os.MkdirAll(filepath.Dir(session), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(session, []byte("{\"type\":\"session\",\"id\":\"exact\"}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	item, err := st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: agent.ID,
		Reason: "agent needs your input", Title: "Adopt the table?", Body: "Please answer", SessionPath: session,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := &answerHarness{
		deps: Deps{
			Store: st, Tmux: manager, DataDir: dataDir,
			Runtime: rpc.NewRuntime("/nonexistent-pi", st, nil),
			Replies: NewTuiReplies(),
		},
		store: st, agent: agent, item: item, typed: filepath.Join(home, "typed.txt"),
	}
	if !running {
		return h
	}
	name := tmux.SessionName(agent.ID)
	script := "while IFS= read -r line; do " +
		`line=${line//$'\e[200~'/}; line=${line//$'\e[201~'/}; ` +
		`printf '%s\n' "$line" >> "$TYPED"; ` +
		`if [ -n "$REPLY_SESSION" ]; then esc=${line//\"/\\\"}; ts=$(date -u +%Y-%m-%dT%H:%M:%S.000Z); ` +
		`printf '{"type":"message","timestamp":"%s","message":{"role":"user","content":[{"type":"text","text":"%s"}]}}\n' "$ts" "$esc" >> "$REPLY_SESSION"; fi; ` +
		`done`
	env := []string{"TYPED=" + h.typed, "REPLY_SESSION=" + session}
	if err := manager.NewSessionEnv(context.Background(), name, cwd, env, "/bin/bash", "-c", script); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.KillSession(context.Background(), name) })
	time.Sleep(200 * time.Millisecond) // let the shell reach the read loop
	return h
}

// typedText waits briefly for the fake TUI to have received something.
func (h *answerHarness) typedText(wait time.Duration) string {
	deadline := time.Now().Add(wait)
	for {
		raw, _ := os.ReadFile(h.typed)
		if len(raw) > 0 || time.Now().After(deadline) {
			return string(raw)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (h *answerHarness) itemNow(t *testing.T) store.InboxItem {
	t.Helper()
	it, err := h.store.GetInboxItem(h.item.ID)
	if err != nil {
		t.Fatal(err)
	}
	return it
}

// Row: an asker is polling the item → the answer is recorded for it to read,
// and nothing is typed into the terminal, whatever the CLI.
func TestAnswerAgentQuestionWaitingAskerReadsIt(t *testing.T) {
	for _, cli := range []string{"claude-code", "pi", "omp"} {
		t.Run(cli, func(t *testing.T) {
			h := newAnswerHarness(t, cli, true, nil)
			h.deps.Replies.askPolled(h.item.ID)
			got, err := h.deps.AnswerAgentQuestion(context.Background(), h.item.ID, store.VerbRespond, "aprovado")
			if err != nil || got != answerRead {
				t.Fatalf("answer = %q, %v; want read", got, err)
			}
			it := h.itemNow(t)
			if it.State != store.InboxDone || it.Response == nil || *it.Response != "respond: aprovado" {
				t.Fatalf("item = %+v", it)
			}
			if typed := h.typedText(300 * time.Millisecond); typed != "" {
				t.Fatalf("a waiting asker's answer was also typed: %q", typed)
			}
			if tasks, _ := h.store.ListTasks(h.agent.ID, 5); len(tasks) != 0 {
				t.Fatalf("tasks = %+v", tasks)
			}
		})
	}
}

// Row: another CLI with its terminal running → recorded and typed into the
// TUI as the same payload every channel carries.
func TestAnswerAgentQuestionTypesIntoOtherCLIs(t *testing.T) {
	for _, cli := range []string{"claude-code", "codex", "grok"} {
		t.Run(cli, func(t *testing.T) {
			h := newAnswerHarness(t, cli, true, nil)
			got, err := h.deps.AnswerAgentQuestion(context.Background(), h.item.ID, store.VerbRespond, "aprovado")
			if err != nil || got != answerTyped {
				t.Fatalf("answer = %q, %v; want typed", got, err)
			}
			want := store.InboxForwardPayload(h.item, store.VerbRespond, "aprovado")
			if typed := h.typedText(3 * time.Second); !strings.Contains(typed, want) {
				t.Fatalf("typed = %q, want %q", typed, want)
			}
			if it := h.itemNow(t); it.State != store.InboxDone || strings.Contains(it.Body, "not told") {
				t.Fatalf("item = %+v", it)
			}
		})
	}
}

// Row: another CLI whose terminal is not running → recorded, and the item
// says the agent was not told.
func TestAnswerAgentQuestionOtherCLINotRunning(t *testing.T) {
	h := newAnswerHarness(t, "claude-code", false, nil)
	got, err := h.deps.AnswerAgentQuestion(context.Background(), h.item.ID, store.VerbRespond, "aprovado")
	if err != nil || got != answerUntold {
		t.Fatalf("answer = %q, %v; want untold", got, err)
	}
	it := h.itemNow(t)
	if it.State != store.InboxDone || !strings.Contains(it.Body, answerNotRunningNote) {
		t.Fatalf("item = %+v", it)
	}
	if !strings.Contains(got.Toast(), "not told") {
		t.Fatalf("toast = %q", got.Toast())
	}
}

// Row: the paste does not land → the answer stays recorded and the item says
// why nobody was told.
func TestAnswerAgentQuestionPasteFailureIsNoted(t *testing.T) {
	h := newAnswerHarness(t, "claude-code", false, nil)
	got, err := h.deps.typeAnswer(context.Background(), h.agent, h.item, store.VerbRespond, "aprovado")
	if err != nil || got != answerUntold {
		t.Fatalf("answer = %q, %v; want untold", got, err)
	}
	it := h.itemNow(t)
	if it.State != store.InboxDone || !strings.Contains(it.Body, "could not type it") {
		t.Fatalf("item = %+v", it)
	}
}

// Row: Omp with a stamped session and a running terminal → ADR-0060's
// delivery (here the paste fallback, no receiver hello), proven by the row.
func TestAnswerAgentQuestionOmpUsesTheReceiverDoor(t *testing.T) {
	h := newAnswerHarness(t, "omp", true, func(dataDir, agentID string) string {
		return filepath.Join(ompAgentSessionDir(dataDir, agentID), "exact.jsonl")
	})
	got, err := h.deps.AnswerAgentQuestion(context.Background(), h.item.ID, store.VerbRespond, "aprovado")
	if err != nil || got != answerDelivered {
		t.Fatalf("answer = %q, %v; want delivered", got, err)
	}
	var task store.Task
	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		tasks, _ := h.store.ListTasks(h.agent.ID, 5)
		if len(tasks) == 1 && tasks[0].Status == store.TaskDelivered {
			task = tasks[0]
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if task.Source != "inbox-tui:"+h.item.ID {
		tasks, _ := h.store.ListTasks(h.agent.ID, 5)
		t.Fatalf("delivery task never settled: %+v", tasks)
	}
}

// Row: Omp without a session (its receiver never said hello) → the
// any-CLI door: recorded and typed.
func TestAnswerAgentQuestionOmpWithoutSessionIsTyped(t *testing.T) {
	h := newAnswerHarness(t, "omp", true, nil)
	got, err := h.deps.AnswerAgentQuestion(context.Background(), h.item.ID, store.VerbRespond, "aprovado")
	if err != nil || got != answerTyped {
		t.Fatalf("answer = %q, %v; want typed", got, err)
	}
}

// Rows the item refuses before any door: a done item, a verb it does not allow.
func TestAnswerAgentQuestionRefusesInvalidAnswers(t *testing.T) {
	h := newAnswerHarness(t, "claude-code", false, nil)
	if _, err := h.deps.AnswerAgentQuestion(context.Background(), h.item.ID, store.VerbAccept, ""); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("disallowed verb = %v", err)
	}
	if _, err := h.store.RespondInboxItem(h.item.ID, store.VerbRespond, "first"); err != nil {
		t.Fatal(err)
	}
	if _, err := h.deps.AnswerAgentQuestion(context.Background(), h.item.ID, store.VerbRespond, "again"); err == nil || !strings.Contains(err.Error(), "already done") {
		t.Fatalf("done item = %v", err)
	}
}

// Only a poll that says it waits marks the item: the UI reading one item must
// not make a later answer skip the terminal.
func TestInboxGetMarksOnlyWaitingPolls(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	deps := Deps{Store: st, Replies: NewTuiReplies()}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	it, err := st.CreateInboxItem(store.InboxItemParams{Kind: store.InboxQuestion, SourceKind: store.InboxFromAgent, SourceID: "a", Reason: "r", Title: "q", Body: "?"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		query string
		want  bool
	}{{"", false}, {"?wait=1", true}} {
		res, err := http.Get(ts.URL + "/api/inbox/" + it.ID + c.query)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if got := deps.Replies.askWaiting(it.ID); got != c.want {
			t.Fatalf("GET%s: waiting = %v, want %v", c.query, got, c.want)
		}
	}
	askWaiterTTL = 0
	t.Cleanup(func() { askWaiterTTL = 30 * time.Second })
	time.Sleep(time.Millisecond)
	if deps.Replies.askWaiting(it.ID) {
		t.Fatal("a poll older than the TTL still counts as waiting")
	}
}

// An agent's question filed without a session path takes the one its
// receiver last reported, so Omp's receiver can answer into it.
func TestInboxCreateStampsTheReceiverSession(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	deps := Deps{Store: st, Replies: NewTuiReplies()}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	deps.Replies.HelloSession("omp-agent", "/data/omp-sessions/omp-agent/s.jsonl")

	_, out := inboxPost(t, ts, "/api/inbox", `{"kind":"question","sourceKind":"agent","sourceId":"omp-agent","reason":"r","title":"q","body":"?"}`)
	if got := out["sessionPath"]; got != "/data/omp-sessions/omp-agent/s.jsonl" {
		t.Fatalf("stamped session = %v", got)
	}
	_, out = inboxPost(t, ts, "/api/inbox", `{"kind":"question","sourceKind":"agent","sourceId":"other","reason":"r","title":"q","body":"?"}`)
	if got, _ := out["sessionPath"].(string); got != "" {
		t.Fatalf("an agent without a receiver got session %q", got)
	}
}

// Omp's launch loads Pi's reply receiver next to its activity extension, and
// the intercept writes the file the wrapper names.
func TestOmpPlanInjectsTheReplyReceiver(t *testing.T) {
	dir := t.TempDir()
	p := cliIntegrationPlan("omp", dir, "hook")
	if len(p.Branches) == 0 || !slices.Contains(p.Branches[0].Args, piReplyExtensionFile(dir)) {
		t.Fatalf("omp args = %+v", p.Branches)
	}
	if !slices.Contains(p.Files, piReplyExtensionFile(dir)) {
		t.Fatalf("omp files = %v", p.Files)
	}
	if err := writeOmpIntercept(dir, "hook"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(piReplyExtensionFile(dir)); err != nil {
		t.Fatalf("receiver file: %v", err)
	}
}
