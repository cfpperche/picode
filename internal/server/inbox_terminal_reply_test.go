package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

// An inbox question filed by pi running in an Agent CLI terminal: the
// sourceKind pi-inbox stamps from PICODE_TERM_ID, carrying the exact
// session that asked (ADR-0059).
func terminalQuestion(t *testing.T, st *store.Store, termID, sessionPath string) store.InboxItem {
	t.Helper()
	it, err := st.CreateInboxItem(store.InboxItemParams{
		Kind: store.InboxQuestion, SourceKind: store.InboxFromTerminal, SourceID: termID,
		Reason: "agent needs your input", Title: "fix in english?", Body: "fix in english?",
		SessionPath: sessionPath, Blocking: true, Allowed: []string{store.VerbRespond, store.VerbIgnore},
	})
	if err != nil {
		t.Fatal(err)
	}
	return it
}

// The reply reaches the terminal's receiver as a one-shot file naming the
// item's exact session; the ack parks the item done. No task row — the
// queue belongs to agents (ADR-0089).
func TestInboxTerminalReplyDeliversThroughTheReceiver(t *testing.T) {
	ts, deps, st, term, _, sessionPath := piTerminalFixture(t)
	deps.Replies.HelloSession(termReplyKey(term.ID), sessionPath)
	it := terminalQuestion(t, st, term.ID, sessionPath)

	type result struct {
		itemID string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		id, err := deps.DeliverTerminalReply(it.ID, store.VerbRespond, "yes, english is fine")
		done <- result{id, err}
	}()

	dir := replyDir(deps.DataDir, termReplyKey(term.ID))
	var file string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && file == "" {
		if ents, err := os.ReadDir(dir); err == nil {
			for _, e := range ents {
				if filepath.Ext(e.Name()) == ".json" {
					file = filepath.Join(dir, e.Name())
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if file == "" {
		t.Fatal("no reply file reached the terminal's receiver directory")
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc replyFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SessionPath != sessionPath || !strings.Contains(doc.Payload, "yes, english is fine") {
		t.Errorf("reply file = %+v", doc)
	}
	if code, _ := postRaw(t, ts, "/api/terminals/"+term.ID+"/tui-ack", `{"nonce":`+jsonString(doc.Nonce)+`,"ok":true}`); code != http.StatusNoContent {
		t.Fatalf("ack: %d", code)
	}
	res := <-done
	if res.err != nil {
		t.Fatalf("DeliverTerminalReply: %v", res.err)
	}
	if res.itemID != term.ID {
		t.Errorf("returned terminal = %q; want %q", res.itemID, term.ID)
	}
	after, _ := st.GetInboxItem(it.ID)
	if after.State != store.InboxDone || after.Response == nil || !strings.Contains(*after.Response, "yes, english is fine") {
		t.Fatalf("item after delivery = %+v", after)
	}
	if tasks, _ := st.ListTasks(term.ID, 10); len(tasks) != 0 {
		t.Errorf("a terminal reply must not enqueue an agent task: %+v", tasks)
	}
}

// The raw respond route answers a terminal-sourced question the same way,
// end to end over HTTP.
func TestInboxRespondRouteDeliversToTerminal(t *testing.T) {
	ts, deps, st, term, _, sessionPath := piTerminalFixture(t)
	deps.Replies.HelloSession(termReplyKey(term.ID), sessionPath)
	it := terminalQuestion(t, st, term.ID, sessionPath)

	go func() {
		// The receiver's file is consumed by acking over HTTP.
		dir := replyDir(deps.DataDir, termReplyKey(term.ID))
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			ents, err := os.ReadDir(dir)
			if err == nil && len(ents) > 0 {
				raw, err := os.ReadFile(filepath.Join(dir, ents[0].Name()))
				if err != nil {
					continue
				}
				var doc replyFile
				if json.Unmarshal(raw, &doc) == nil {
					postRaw(t, ts, "/api/terminals/"+term.ID+"/tui-ack", `{"nonce":`+jsonString(doc.Nonce)+`,"ok":true}`)
					return
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()
	code, body := postRaw(t, ts, "/api/inbox/"+it.ID+"/respond", `{"verb":"respond","text":"fixed"}`)
	if code != http.StatusOK {
		t.Fatalf("respond to terminal question: %d %v", code, body)
	}
	after, _ := st.GetInboxItem(it.ID)
	if after.State != store.InboxDone {
		t.Fatalf("item state = %s; want done", after.State)
	}
}

// Every refusal names itself, leaves the item open, and writes no file.
func TestInboxTerminalReplyRefusals(t *testing.T) {
	_, deps, st, term, repo, sessionPath := piTerminalFixture(t)
	it := terminalQuestion(t, st, term.ID, sessionPath)
	refused := func(want string) {
		t.Helper()
		if _, err := deps.DeliverTerminalReply(it.ID, store.VerbRespond, "hi"); err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal = %v; want it to name %q", err, want)
		}
		after, _ := st.GetInboxItem(it.ID)
		if after.State == store.InboxDone {
			t.Fatalf("item closed despite the refusal")
		}
		ents, _ := os.ReadDir(replyDir(deps.DataDir, termReplyKey(term.ID)))
		if len(ents) != 0 {
			t.Fatalf("a reply file was written despite the refusal: %v", ents)
		}
	}

	// No hello yet: no receiver.
	refused("no receiver is listening")
	// Not running pi right now.
	deps.TermRuntimes.Drop(term.ID)
	refused("not running pi right now")
	deps.TermRuntimes.Start(term.ID, TermRuntime{CLI: "pi", Source: "test", RunID: "r2", StartedAt: time.Now()})
	// Not a pi terminal at all: neither the launch record nor the running
	// process may say pi.
	deps.TermRuntimes.Drop(term.ID)
	if err := st.SetTerminalLaunch(term.ID, "claude-code", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	refused("is not running pi")
	if err := st.SetTerminalLaunch(term.ID, "pi", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	deps.TermRuntimes.Start(term.ID, TermRuntime{CLI: "pi", Source: "test", RunID: "r3", StartedAt: time.Now()})
	// A fresh receiver that names no conversation at all — a nested pi in
	// this terminal, or a TUI before its session starts — is not a channel:
	// the reply would be parked and then refused by a process with no
	// session, which is how it read "showing a different session" with no
	// session in sight (2026-09-11).
	deps.Replies.HelloSession(termReplyKey(term.ID), "")
	refused("has not opened a conversation yet")
	// A hello naming another session: the exact-session rule wins.
	deps.Replies.HelloSession(termReplyKey(term.ID), filepath.Join(session.Dir(repo), "other.jsonl"))
	refused("different session")
	deps.Replies.HelloSession(termReplyKey(term.ID), sessionPath)
	// An item with no session path cannot be routed (legacy rows).
	bare := terminalQuestion(t, st, term.ID, "")
	if _, err := deps.DeliverTerminalReply(bare.ID, store.VerbRespond, "hi"); err == nil || !strings.Contains(err.Error(), "predates session tracking") {
		t.Fatalf("sessionless item = %v", err)
	}
	// A session file that vanished.
	if err := os.Remove(sessionPath); err != nil {
		t.Fatal(err)
	}
	if _, err := deps.DeliverTerminalReply(it.ID, store.VerbRespond, "hi"); err == nil {
		t.Fatal("a vanished session file was accepted")
	} else if !strings.Contains(err.Error(), "no longer exists") {
		t.Fatalf("vanished session = %v", err)
	}
	// A gone terminal annotates and stays open.
	gone := terminalQuestion(t, st, "term-ghost", sessionPath)
	if _, err := deps.DeliverTerminalReply(gone.ID, store.VerbRespond, "hi"); err == nil || !strings.Contains(err.Error(), "no longer exists") {
		t.Fatalf("gone terminal = %v", err)
	}
	if got, _ := st.GetInboxItem(gone.ID); got.State == store.InboxDone || !strings.Contains(got.Body, "terminal no longer exists") {
		t.Fatalf("gone-terminal item = %+v", got)
	}
}

// Ignore is not a reply: it must close the item wherever the terminal is,
// even with no receiver at all — the decision sends nothing, and routing it
// through the receiver left a question undismissable.
func TestInboxTerminalIgnoreClosesWithoutTheReceiver(t *testing.T) {
	ts, deps, st, term, _, sessionPath := piTerminalFixture(t)
	// No hello, and the terminal is not even running pi: nothing can be
	// delivered anywhere, and ignore still completes.
	deps.TermRuntimes.Drop(term.ID)
	if err := st.SetTerminalLaunch(term.ID, "claude-code", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	it := terminalQuestion(t, st, term.ID, sessionPath)
	code, body := postRaw(t, ts, "/api/inbox/"+it.ID+"/respond", `{"verb":"ignore"}`)
	if code != http.StatusOK {
		t.Fatalf("ignore on a terminal question: %d %v", code, body)
	}
	after, _ := st.GetInboxItem(it.ID)
	if after.State != store.InboxDone {
		t.Fatalf("item state = %s; want done", after.State)
	}
	if after.Response == nil || *after.Response != store.VerbIgnore {
		t.Fatalf("item response = %v; want %q", after.Response, store.VerbIgnore)
	}
	if strings.Contains(after.Body, "could not be delivered") {
		t.Fatalf("ignore annotated the item as a failed delivery: %q", after.Body)
	}
	if ents, _ := os.ReadDir(replyDir(deps.DataDir, termReplyKey(term.ID))); len(ents) != 0 {
		t.Fatalf("ignore wrote reply files: %v", ents)
	}
	if tasks, _ := st.ListTasks(term.ID, 5); len(tasks) != 0 {
		t.Fatalf("ignore enqueued a task: %+v", tasks)
	}
}

// The reply file names the process whose hello the daemon accepted: every pi
// that inherited the terminal id watches the same directory, and only the
// addressee may consume the file.
func TestInboxTerminalReplyFileNamesTheReceivingProcess(t *testing.T) {
	ts, deps, st, term, _, sessionPath := piTerminalFixture(t)
	if code, _ := postRaw(t, ts, "/api/terminals/"+term.ID+"/tui-hello", `{"session":`+jsonString(sessionPath)+`,"pid":4242}`); code != http.StatusNoContent {
		t.Fatalf("hello: %d", code)
	}
	if got := deps.Replies.receiverPID(termReplyKey(term.ID)); got != 4242 {
		t.Fatalf("recorded receiver pid = %d; want 4242", got)
	}
	it := terminalQuestion(t, st, term.ID, sessionPath)

	done := make(chan error, 1)
	go func() {
		_, err := deps.DeliverTerminalReply(it.ID, store.VerbRespond, "yes")
		done <- err
	}()

	dir := replyDir(deps.DataDir, termReplyKey(term.ID))
	var doc replyFile
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && doc.Nonce == "" {
		if ents, err := os.ReadDir(dir); err == nil {
			for _, e := range ents {
				if filepath.Ext(e.Name()) != ".json" {
					continue
				}
				raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
				if err != nil || json.Unmarshal(raw, &doc) != nil {
					continue
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if doc.Nonce == "" {
		t.Fatal("no reply file reached the terminal's receiver directory")
	}
	if doc.PID != 4242 {
		t.Errorf("reply file pid = %d; want the hello's 4242", doc.PID)
	}
	if code, _ := postRaw(t, ts, "/api/terminals/"+term.ID+"/tui-ack", `{"nonce":`+jsonString(doc.Nonce)+`,"ok":true}`); code != http.StatusNoContent {
		t.Fatalf("ack: %d", code)
	}
	if err := <-done; err != nil {
		t.Fatalf("DeliverTerminalReply: %v", err)
	}
}

// When the receiver takes the message but pi never processes it into a
// JSONL row, the item reopens with the response preserved for prefill.
func TestInboxTerminalReplyReopensWhenTheRowNeverAppears(t *testing.T) {
	ts, deps, st, term, _, sessionPath := piTerminalFixture(t)
	deps.Replies.HelloSession(termReplyKey(term.ID), sessionPath)
	it := terminalQuestion(t, st, term.ID, sessionPath)

	prevWait, prevRow := receiverAckWait, replyRowWait
	receiverAckWait = 500 * time.Millisecond
	replyRowWait = 30 * time.Millisecond
	t.Cleanup(func() { receiverAckWait, replyRowWait = prevWait, prevRow })

	// The receiver takes the message and acks ok, but pi never turns it
	// into a JSONL row: the background row wait must reopen the item.
	go func() {
		dir := replyDir(deps.DataDir, termReplyKey(term.ID))
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			ents, err := os.ReadDir(dir)
			if err == nil && len(ents) > 0 {
				raw, err := os.ReadFile(filepath.Join(dir, ents[0].Name()))
				if err != nil {
					continue
				}
				var doc replyFile
				if json.Unmarshal(raw, &doc) == nil {
					_ = os.Remove(filepath.Join(dir, ents[0].Name()))
					postRaw(t, ts, "/api/terminals/"+term.ID+"/tui-ack", `{"nonce":`+jsonString(doc.Nonce)+`,"ok":true}`)
					return
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	if _, err := deps.DeliverTerminalReply(it.ID, store.VerbRespond, "will not land"); err != nil {
		t.Fatalf("delivery: %v", err)
	}
	// No ack will come; the row wait expires and the item must reopen.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		after, _ := st.GetInboxItem(it.ID)
		if after.State == store.InboxUnread {
			if after.Response == nil || !strings.Contains(*after.Response, "will not land") {
				t.Fatalf("reopened without the response preserved: %+v", after)
			}
			if !strings.Contains(after.Body, "never reached the terminal") {
				t.Fatalf("reopen note missing: %q", after.Body)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the item never reopened after the terminal ignored the reply")
}
