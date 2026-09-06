package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

// A pi inside a PiCode terminal (no PICODE_AGENT_ID) publishes under the
// terminal id: same body semantics, terminal identity, view fold.
func TestTerminalChecklistRoutes(t *testing.T) {
	ts, st := newInboxServer(t)
	tm, err := st.CreateTerminalIn(store.FreeWorkspaceID, "agent cli", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// Unknown terminal → 404; bad status → 400.
	res, _ := inboxPost(t, ts, "/api/terminals/nope/checklist", `{"items":[{"text":"x"}]}`)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown terminal = %d", res.StatusCode)
	}
	res, _ = inboxPost(t, ts, "/api/terminals/"+tm.ID+"/checklist", `{"items":[{"text":"x","status":"done"}]}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad status = %d", res.StatusCode)
	}

	// A list, then a blocked marker (stale steps must not survive).
	res, out := inboxPost(t, ts, "/api/terminals/"+tm.ID+"/checklist", `{"sessionId":"s1","items":[{"text":"read the code","status":"completed"},{"text":"edit","status":"in-progress"}]}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("set = %d %v", res.StatusCode, out)
	}
	if out["termId"] != tm.ID {
		t.Fatalf("termId = %v", out["termId"])
	}
	res, out = inboxPost(t, ts, "/api/terminals/"+tm.ID+"/checklist", `{"items":[{"text":"stale"}],"blocked":true}`)
	if res.StatusCode != http.StatusOK || out["absent"] != true {
		t.Fatalf("blocked = %d %v", res.StatusCode, out)
	}

	// The terminal view (GET /api/terminals) carries the checklist, the way
	// it carries live state — the list is the boot fetch.
	type termView struct {
		ID        string                   `json:"id"`
		Checklist *store.TerminalChecklist `json:"checklist"`
	}
	var list struct {
		Terminals []termView `json:"terminals"`
	}
	r := do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	_ = json.NewDecoder(r.Body).Decode(&list)
	r.Body.Close()
	var view *termView
	for i := range list.Terminals {
		if list.Terminals[i].ID == tm.ID {
			view = &list.Terminals[i]
		}
	}
	if view == nil || view.Checklist == nil || !view.Checklist.Absent {
		t.Fatalf("view fold = %+v", view)
	}

	// A fresh session resets: the row is dropped, the event announces the
	// empty state, and an unknown terminal stays 404.
	res, _ = inboxPost(t, ts, "/api/terminals/"+tm.ID+"/checklist", `{"reset":true}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reset = %d", res.StatusCode)
	}
	res, _ = inboxPost(t, ts, "/api/terminals/nope/checklist", `{"reset":true}`)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("reset unknown = %d", res.StatusCode)
	}
	r = do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals"))
	var fresh struct {
		Terminals []termView `json:"terminals"`
	}
	_ = json.NewDecoder(r.Body).Decode(&fresh)
	r.Body.Close()
	for _, t2 := range fresh.Terminals {
		if t2.ID == tm.ID && t2.Checklist != nil {
			t.Fatalf("checklist survived reset = %+v", t2.Checklist)
		}
	}

	// Every set/blocked/reset POST above announced itself.
	evs, err := st.ListEventsSince(0, 100)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, ev := range evs {
		if ev.Type == "terminal.checklist" {
			n++
		}
	}
	if n != 3 {
		t.Fatalf("terminal.checklist events = %d, want 3", n)
	}

	// Unknown terminal's GET is an empty answer, not an error.
	r = do(t, ts.Client(), mustGet(t, ts.URL+"/api/terminals/nope/checklist"))
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatalf("get unknown = %d", r.StatusCode)
	}
}

func TestChecklistRoutes(t *testing.T) {
	ts, st := newInboxServer(t)
	ws, err := st.AddWorkspace("w", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ag, err := st.AddAgent(ws.ID, "planner", "")
	if err != nil {
		t.Fatal(err)
	}

	// Unknown agent → 404.
	res, _ := inboxPost(t, ts, "/api/agents/nope/checklist", `{"items":[{"text":"x"}]}`)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown agent = %d", res.StatusCode)
	}
	// Bad status → 400.
	res, out := inboxPost(t, ts, "/api/agents/"+ag.ID+"/checklist", `{"items":[{"text":"x","status":"done"}]}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad status = %d %v", res.StatusCode, out)
	}
	// A list.
	res, out = inboxPost(t, ts, "/api/agents/"+ag.ID+"/checklist", `{"sessionId":"s1","items":[{"text":"read  the code","status":"completed"},{"text":"edit","status":"in-progress"},{"text":"test"}]}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("set = %d %v", res.StatusCode, out)
	}
	items := out["items"].([]any)
	if len(items) != 3 || items[0].(map[string]any)["text"] != "read the code" || items[2].(map[string]any)["status"] != "pending" {
		t.Fatalf("items = %v", items)
	}
	if out["absent"] != false {
		t.Fatalf("absent = %v", out["absent"])
	}

	// GET one and the list.
	var one struct{ Checklist *store.Checklist }
	r := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+ag.ID+"/checklist"))
	_ = json.NewDecoder(r.Body).Decode(&one)
	r.Body.Close()
	if one.Checklist == nil || one.Checklist.SessionID != "s1" || len(one.Checklist.Items) != 3 {
		t.Fatalf("get = %+v", one.Checklist)
	}
	var all struct{ Checklists []store.Checklist }
	r = do(t, ts.Client(), mustGet(t, ts.URL+"/api/checklists"))
	_ = json.NewDecoder(r.Body).Decode(&all)
	r.Body.Close()
	if len(all.Checklists) != 1 || all.Checklists[0].AgentID != ag.ID {
		t.Fatalf("list = %+v", all.Checklists)
	}

	// A refused change is absence, whatever the body carries: the gate only
	// refuses unplanned tasks, so stale items must not survive as this
	// task's plan. Same for an explicit absent marker with items.
	res, out = inboxPost(t, ts, "/api/agents/"+ag.ID+"/checklist", `{"items":[],"blocked":true}`)
	if res.StatusCode != http.StatusOK || out["absent"] != true {
		t.Fatalf("blocked = %d %v", res.StatusCode, out)
	}
	res, out = inboxPost(t, ts, "/api/agents/"+ag.ID+"/checklist", `{"items":[{"text":"stale task"}],"blocked":true}`)
	if res.StatusCode != http.StatusOK || out["absent"] != true {
		t.Fatalf("blocked with items = %d %v", res.StatusCode, out)
	}
	if got := out["items"].([]any); len(got) != 0 {
		t.Fatalf("blocked kept items: %v", got)
	}
	res, out = inboxPost(t, ts, "/api/agents/"+ag.ID+"/checklist", `{"items":[{"text":"also stale"}],"absent":true}`)
	if res.StatusCode != http.StatusOK || out["absent"] != true {
		t.Fatalf("absent with items = %d %v", res.StatusCode, out)
	}
	if got := out["items"].([]any); len(got) != 0 {
		t.Fatalf("absent kept items: %v", got)
	}

	// A fresh session resets the row: the shells drop their entry and the
	// boot list no longer carries the agent.
	res, out = inboxPost(t, ts, "/api/agents/"+ag.ID+"/checklist", `{"sessionId":"s2","reset":true}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reset = %d %v", res.StatusCode, out)
	}
	if got := out["items"].([]any); len(got) != 0 {
		t.Fatalf("reset kept items: %v", got)
	}
	r = do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+ag.ID+"/checklist"))
	_ = json.NewDecoder(r.Body).Decode(&one)
	r.Body.Close()
	if one.Checklist != nil {
		t.Fatalf("after reset get = %+v, want nil", one.Checklist)
	}
	r = do(t, ts.Client(), mustGet(t, ts.URL+"/api/checklists"))
	_ = json.NewDecoder(r.Body).Decode(&all)
	r.Body.Close()
	if len(all.Checklists) != 0 {
		t.Fatalf("after reset list = %+v", all.Checklists)
	}
	// Reset is idempotent and unknown agents are still 404.
	res, _ = inboxPost(t, ts, "/api/agents/"+ag.ID+"/checklist", `{"reset":true}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("second reset = %d", res.StatusCode)
	}
	res, _ = inboxPost(t, ts, "/api/agents/nope/checklist", `{"reset":true}`)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("reset unknown = %d", res.StatusCode)
	}

	// Every set/absent/blocked/reset POST above announced itself.
	evs, err := st.ListEventsSince(0, 100)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, ev := range evs {
		if ev.Type == "agent.checklist" {
			n++
		}
	}
	if n != 6 {
		t.Fatalf("agent.checklist events = %d, want 6", n)
	}

	// Unknown agent's GET is an empty answer, not an error.
	r = do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/nope/checklist"))
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatalf("get unknown = %d", r.StatusCode)
	}

	// The level rides PATCH and the spawn env; read-only wins.
	res, out = inboxPatch(t, ts, "/api/agents/"+ag.ID, `{"checklist":"always"}`)
	if res.StatusCode != http.StatusOK || out["checklist"] != "always" {
		t.Fatalf("patch = %d %v", res.StatusCode, out)
	}
	res, _ = inboxPatch(t, ts, "/api/agents/"+ag.ID, `{"checklist":"sometimes"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad level = %d", res.StatusCode)
	}
	got, _ := st.GetAgent(ag.ID)
	if !hasEnv(got.SpawnEnv(), "PICODE_CHECKLIST=always") {
		t.Fatalf("spawn env = %v", got.SpawnEnv())
	}
	ro := store.OpModeReadonly
	if _, err := st.UpdateAgent(ag.ID, store.AgentPatch{OpMode: &ro}); err != nil {
		t.Fatal(err)
	}
	got, _ = st.GetAgent(ag.ID)
	if !hasEnv(got.SpawnEnv(), "PICODE_CHECKLIST=never") {
		t.Fatalf("read-only spawn env = %v", got.SpawnEnv())
	}

	// Deleting the agent drops its checklist.
	if err := st.DeleteAgent(ag.ID); err != nil {
		t.Fatal(err)
	}
	r = do(t, ts.Client(), mustGet(t, ts.URL+"/api/checklists"))
	_ = json.NewDecoder(r.Body).Decode(&all)
	r.Body.Close()
	if len(all.Checklists) != 0 {
		t.Fatalf("after delete = %+v", all.Checklists)
	}
}

func hasEnv(env []string, kv string) bool {
	for _, e := range env {
		if e == kv {
			return true
		}
	}
	return false
}

func inboxPatch(t *testing.T, ts *httptest.Server, path, body string) (*http.Response, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPatch, ts.URL+path, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("PATCH %s: %v", path, err)
	}
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	return res, out
}
