package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/apps"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func newInboxServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)
	return ts, st
}

func inboxPost(t *testing.T, ts *httptest.Server, path, body string) (*http.Response, map[string]any) {
	t.Helper()
	res, err := http.Post(ts.URL+path, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	return res, out
}

func TestInboxCreateAndList(t *testing.T) {
	ts, _ := newInboxServer(t)
	for _, body := range []string{
		`{"kind":"fyi","sourceKind":"system","reason":"qa","title":"note"}`,
		`{"kind":"result","sourceKind":"agent","sourceId":"a1","reason":"run finished","title":"done","body":"all good"}`,
		`{"kind":"question","sourceKind":"agent","sourceId":"a1","reason":"needs input","title":"which db?","body":"sqlite or pg?"}`,
	} {
		res, out := inboxPost(t, ts, "/api/inbox", body)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("create %s = %d %v", body, res.StatusCode, out)
		}
	}
	// question without body → 400
	res, _ := inboxPost(t, ts, "/api/inbox", `{"kind":"question","sourceKind":"system","reason":"r","title":"q"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("question sans body = %d", res.StatusCode)
	}

	get, err := http.Get(ts.URL + "/api/inbox")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	var list struct {
		Items []store.InboxItem `json:"items"`
	}
	_ = json.NewDecoder(get.Body).Decode(&list)
	get.Body.Close()
	if len(list.Items) != 3 {
		t.Fatalf("list = %d items", len(list.Items))
	}
	blocking, err0 := http.Get(ts.URL + "/api/inbox?blocking=1")
	if err0 != nil {
		t.Fatalf("GET blocking: %v", err0)
	}
	list.Items = nil
	_ = json.NewDecoder(blocking.Body).Decode(&list)
	blocking.Body.Close()
	if len(list.Items) != 1 || list.Items[0].Kind != store.InboxQuestion {
		t.Fatalf("blocking filter = %+v", list.Items)
	}
}

func TestInboxRespondForwardsToAgent(t *testing.T) {
	ts, st := newInboxServer(t)
	ws, _ := st.AddWorkspace("wsx", t.TempDir())
	ag, err := st.AddAgent(ws.ID, "helper", "")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	_, out := inboxPost(t, ts, "/api/inbox",
		`{"kind":"question","sourceKind":"agent","sourceId":"`+ag.ID+`","reason":"needs input","title":"port?","body":"8080?"}`)
	id, _ := out["id"].(string)

	// text required for respond
	res, _ := inboxPost(t, ts, "/api/inbox/"+id+"/respond", `{"verb":"respond","text":" "}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty text = %d", res.StatusCode)
	}
	// disallowed verb
	res, _ = inboxPost(t, ts, "/api/inbox/"+id+"/respond", `{"verb":"accept"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("disallowed verb = %d", res.StatusCode)
	}
	// happy path
	res, out = inboxPost(t, ts, "/api/inbox/"+id+"/respond", `{"verb":"respond","text":"8445"}`)
	if res.StatusCode != http.StatusOK || out["state"] != store.InboxDone {
		t.Fatalf("respond = %d %v", res.StatusCode, out)
	}
	tasks, _ := st.ListTasks(ag.ID, 5)
	if len(tasks) != 1 || tasks[0].Kind != store.TaskFollowUp || tasks[0].Source != "inbox" || !strings.Contains(tasks[0].Payload, "8445") {
		t.Fatalf("forwarded task = %+v", tasks)
	}
}

func TestInboxRespondDeadAgent(t *testing.T) {
	ts, st := newInboxServer(t)
	_, out := inboxPost(t, ts, "/api/inbox",
		`{"kind":"question","sourceKind":"agent","sourceId":"ghost","reason":"r","title":"q","body":"?"}`)
	id, _ := out["id"].(string)
	res, body := inboxPost(t, ts, "/api/inbox/"+id+"/respond", `{"verb":"respond","text":"hi"}`)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("dead agent = %d %v", res.StatusCode, body)
	}
	after, _ := st.GetInboxItem(id)
	if after.State == store.InboxDone || !strings.Contains(after.Body, "agent no longer exists") {
		t.Fatalf("item after failure = %+v", after)
	}
	// unknown item id → 404
	res, _ = inboxPost(t, ts, "/api/inbox/nope/respond", `{"verb":"ignore"}`)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown item = %d", res.StatusCode)
	}
}

// TestInboxRespondInteractiveAgent covers the gap found live: replying to
// an agent parked in a TUI/tmux session queued a follow_up task nothing
// ever drains (deliverLoop only runs for the RPC runtime). The route must
// refuse before enqueueing, not silently accept a reply that never lands.
func TestInboxRespondInteractiveAgent(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	tm := tmux.New()
	ws, _ := st.AddWorkspace("wsx", t.TempDir())
	ag, err := st.AddAgent(ws.ID, "helper", "")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	name := tmux.SessionName(ag.ID)
	if err := tm.NewSession(context.Background(), name, ws.Path, "cat"); err != nil {
		t.Skipf("tmux session unavailable in this environment: %v", err)
	}
	t.Cleanup(func() { _ = tm.KillSession(context.Background(), name) })

	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tm, AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)

	_, out := inboxPost(t, ts, "/api/inbox",
		`{"kind":"question","sourceKind":"agent","sourceId":"`+ag.ID+`","reason":"r","title":"q","body":"?"}`)
	id, _ := out["id"].(string)
	res, body := inboxPost(t, ts, "/api/inbox/"+id+"/respond", `{"verb":"respond","text":"hi"}`)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("interactive agent = %d %v", res.StatusCode, body)
	}
	after, _ := st.GetInboxItem(id)
	if after.State == store.InboxDone || !strings.Contains(after.Body, "interactive terminal") {
		t.Fatalf("item after refusal = %+v", after)
	}
	if tasks, _ := st.ListTasks(ag.ID, 10); len(tasks) != 0 {
		t.Fatalf("a task was queued for an undeliverable agent: %+v", tasks)
	}
}

// TestInboxAppActionRespondInteractiveAgent is the regression test for the
// bug the above test's sibling did NOT catch: the raw /api/inbox/respond
// route and the Inbox app's /api/apps/inbox/action route each build their
// own AgentDeliverable closure (internal/server/inbox.go vs
// internal/server/apps.go's appsHost). The first negated
// deps.agentInteractive correctly; the second passed it straight through
// under the old, differently-named field — silently inverted, so a reply
// to a TUI agent was accepted and queued forever. A unit test that sets
// apps.Host.AgentDeliverable by hand (internal/apps/inbox_test.go) cannot
// see this: the bug lived entirely in appsHost()'s wiring. Only a real
// HTTP round trip through the apps route, against a real tmux session,
// exercises it.
func TestInboxAppActionRespondInteractiveAgent(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available")
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	tm := tmux.New()
	ws, _ := st.AddWorkspace("wsx", t.TempDir())
	ag, err := st.AddAgent(ws.ID, "helper", "")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	name := tmux.SessionName(ag.ID)
	if err := tm.NewSession(context.Background(), name, ws.Path, "cat"); err != nil {
		t.Skipf("tmux session unavailable in this environment: %v", err)
	}
	t.Cleanup(func() { _ = tm.KillSession(context.Background(), name) })

	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tm, AgentCmd: "cat", Apps: apps.NewRegistry(apps.BuiltIns(false)...),
	}).Handler)
	t.Cleanup(ts.Close)

	_, out := inboxPost(t, ts, "/api/inbox",
		`{"kind":"question","sourceKind":"agent","sourceId":"`+ag.ID+`","reason":"r","title":"q","body":"?"}`)
	id, _ := out["id"].(string)

	res, err := http.Post(ts.URL+"/api/apps/inbox/action", "application/json",
		bytes.NewBufferString(`{"action":"respond","path":"item/`+id+`","args":{"reply":"hi"}}`))
	if err != nil {
		t.Fatalf("POST action: %v", err)
	}
	body, _ := json.Marshal(map[string]any{})
	_ = json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("app action to an interactive agent = %d, want 400 (refused)", res.StatusCode)
	}
	after, _ := st.GetInboxItem(id)
	if after.State == store.InboxDone {
		t.Fatalf("interactive-agent item was closed by the app action route")
	}
	if !strings.Contains(after.Body, "interactive terminal") {
		t.Fatalf("no actionable annotation: %q", after.Body)
	}
	if tasks, _ := st.ListTasks(ag.ID, 10); len(tasks) != 0 {
		t.Fatalf("app action queued a task for an undeliverable agent: %+v", tasks)
	}
}

func TestInboxStateAndSnooze(t *testing.T) {
	ts, _ := newInboxServer(t)
	_, out := inboxPost(t, ts, "/api/inbox", `{"kind":"fyi","sourceKind":"system","reason":"r","title":"n"}`)
	id, _ := out["id"].(string)

	res, body := inboxPost(t, ts, "/api/inbox/"+id+"/state", `{"state":"read"}`)
	if res.StatusCode != http.StatusOK || body["state"] != store.InboxRead {
		t.Fatalf("read = %d %v", res.StatusCode, body)
	}
	res, _ = inboxPost(t, ts, "/api/inbox/"+id+"/state", `{"snoozedUntil":"2999-01-01T00:00:00Z"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("snooze = %d", res.StatusCode)
	}
	get, err := http.Get(ts.URL + "/api/inbox")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	var list struct {
		Items []store.InboxItem `json:"items"`
	}
	_ = json.NewDecoder(get.Body).Decode(&list)
	get.Body.Close()
	if len(list.Items) != 0 {
		t.Fatalf("snoozed item listed: %+v", list.Items)
	}
	res, _ = inboxPost(t, ts, "/api/inbox/"+id+"/state", `{"state":"sleeping"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad state = %d", res.StatusCode)
	}
}
