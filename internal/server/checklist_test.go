package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

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

	// A refused change with no list marks absence; the event is on the feed.
	res, out = inboxPost(t, ts, "/api/agents/"+ag.ID+"/checklist", `{"items":[],"blocked":true}`)
	if res.StatusCode != http.StatusOK || out["absent"] != true {
		t.Fatalf("blocked = %d %v", res.StatusCode, out)
	}
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
	if n != 2 {
		t.Fatalf("agent.checklist events = %d, want 2", n)
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
