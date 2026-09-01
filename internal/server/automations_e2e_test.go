package server

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func waitRun(t *testing.T, url string, want func(map[string]any) bool, d time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(d)
	var last map[string]any
	for time.Now().Before(deadline) {
		_, out := doJSON(t, "GET", url, "", nil)
		items, _ := out["items"].([]any)
		if len(items) > 0 {
			last = items[0].(map[string]any)
			if want(last) {
				return last
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("run never reached the wanted state; last = %v", last)
	return nil
}

// A real `start` run against the fake pi: the automation's agent is
// created, the turn settles, the run is done with the fake's cost, the
// Inbox carries the agent's final text, and the agent is stopped again.
// Then a cost cap below the fake's per-message usage fails the next run
// at message granularity — no 30 s poll involved.
func TestAutomationStartRunEndToEnd(t *testing.T) {
	t.Setenv("PICODE_FAKE_RPC", "1")
	ts := newTestServer(t, os.Args[0])
	proj := t.TempDir()
	wsv := addWorkspaceWithAgent(t, ts, "Auto", proj)

	res, out := doJSON(t, "POST", ts.URL+"/api/automations", `{"name":"E2E","workspaceId":"`+wsv.ID+`","action":"start","prompt":"say hi","cron":"0 9 * * *"}`, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d %v", res.StatusCode, out)
	}
	id := out["automation"].(map[string]any)["id"].(string)

	res, out = doJSON(t, "POST", ts.URL+"/api/automations/"+id+"/run", "", nil)
	if res.StatusCode != http.StatusAccepted || out["run"].(map[string]any)["status"] != "running" {
		t.Fatalf("run now = %d %v", res.StatusCode, out)
	}
	runs := ts.URL + "/api/automations/" + id + "/runs?limit=1"
	done := waitRun(t, runs, func(r map[string]any) bool { return r["status"] == "done" }, 10*time.Second)
	if done["costUsd"].(float64) < 0.01 {
		t.Fatalf("done run lost the cost: %v", done)
	}

	// Inbox: one result with the agent's real final text.
	_, inbox := doJSON(t, "GET", ts.URL+"/api/inbox", "", nil)
	items, _ := inbox["items"].([]any)
	found := false
	for _, it := range items {
		m := it.(map[string]any)
		if m["kind"] == "result" && m["sourceKind"] == "automation" && strings.Contains(m["body"].(string), "hello from fake") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no automation result in inbox: %v", inbox)
	}

	// The automation's agent exists, is named after it, and is stopped.
	_, view := doJSON(t, "GET", ts.URL+"/api/automations/"+id, "", nil)
	a := view["automation"].(map[string]any)
	if a["agentId"] == nil || a["agentName"] != "E2E" || a["running"] != false {
		t.Fatalf("automation view after run: %v", a)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, ag := doJSON(t, "GET", ts.URL+"/api/agents/"+a["agentId"].(string), "", nil)
		if ag["running"] == false || ag["mode"] == "stopped" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Cost cap below one fake message (0.01): the run fails at the event.
	doJSON(t, "PATCH", ts.URL+"/api/automations/"+id, `{"maxCostUsd":0.005}`, nil)
	res, out = doJSON(t, "POST", ts.URL+"/api/automations/"+id+"/run", "", nil)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("second run = %d %v", res.StatusCode, out)
	}
	capped := waitRun(t, runs, func(r map[string]any) bool { return r["status"] != "running" }, 10*time.Second)
	if capped["status"] != "failed" || capped["reason"] != reasonCostCap {
		t.Fatalf("capped run = %v", capped)
	}
}
