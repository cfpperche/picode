package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// newAutomationServer uses a missing agent command so real runs resolve
// to the "pi missing" row without spawning anything.
func newAutomationServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	rt := rpc.NewRuntime("picode-test-no-such-binary", st, nil)
	deps := Deps{Store: st, Tmux: tmux.New(), Runtime: rt, AgentCmd: "picode-test-no-such-binary"}
	ts := httptest.NewServer(New("127.0.0.1:0", deps).Handler)
	t.Cleanup(ts.Close)
	return ts, st
}

func doJSON(t *testing.T, method, url, body string, hdr map[string]string) (*http.Response, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(method, url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	return res, out
}

func TestDecideFire(t *testing.T) {
	cases := []struct {
		name string
		in   fireInput
		want fireDecision
	}{
		{"disabled", fireInput{Trigger: store.TriggerSchedule, Action: store.AutomationStart}, fireDecision{Status: "none"}},
		{"disabled_manual", fireInput{Trigger: store.TriggerManual, Action: store.AutomationStart}, fireDecision{}},
		{"busy", fireInput{Enabled: true, Busy: true, Action: store.AutomationStart}, fireDecision{Status: store.RunSkipped, Reason: reasonBusy}},
		{"rate_cap", fireInput{Enabled: true, RateHit: true, Action: store.AutomationStart}, fireDecision{Status: store.RunSkipped, Reason: reasonRateCap}},
		{"pi_missing", fireInput{Enabled: true, PiMissing: true, Action: store.AutomationStart}, fireDecision{Status: store.RunFailed, Reason: reasonPiMissing, Notify: true}},
		{"agent_in_terminal", fireInput{Enabled: true, Action: store.AutomationStart, AgentMode: modeInteractive}, fireDecision{Status: store.RunSkipped, Reason: reasonInTerminal}},
		{"start_ok", fireInput{Enabled: true, Action: store.AutomationStart, AgentMode: modeStopped}, fireDecision{}},
		{"target_gone", fireInput{Enabled: true, Action: store.AutomationMessage}, fireDecision{Status: store.RunFailed, Reason: reasonTargetGone, Notify: true}},
		{"target_interactive", fireInput{Enabled: true, Action: store.AutomationMessage, TargetExists: true, AgentMode: modeInteractive}, fireDecision{Status: store.RunSkipped, Reason: reasonInTerminal}},
		{"message_needs_pi_to_start_the_agent", fireInput{Enabled: true, PiMissing: true, Action: store.AutomationMessage, TargetExists: true, AgentMode: modeStopped}, fireDecision{Status: store.RunFailed, Reason: reasonPiMissing, Notify: true}},
		{"message_to_a_running_agent_needs_no_pi", fireInput{Enabled: true, PiMissing: true, Action: store.AutomationMessage, TargetExists: true, AgentMode: modeManaged}, fireDecision{}},
	}
	for _, c := range cases {
		if got := decideFire(c.in); got != c.want {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
	}
}

func TestShouldNotifySkip(t *testing.T) {
	if !shouldNotifySkip(nil, store.RunSkipped, reasonBusy) {
		t.Fatal("first skip notifies")
	}
	if shouldNotifySkip(&store.Run{Status: store.RunSkipped, Reason: reasonBusy}, store.RunSkipped, reasonBusy) {
		t.Fatal("same skip twice is silent")
	}
	if !shouldNotifySkip(&store.Run{Status: store.RunDone}, store.RunSkipped, reasonBusy) {
		t.Fatal("skip after a run notifies")
	}
}

func TestComposeAutomationPrompt(t *testing.T) {
	now := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	if got := composeAutomationPrompt(" hi ", store.TriggerWebhook, "", now); got != "hi" {
		t.Fatalf("no payload: %q", got)
	}
	got := composeAutomationPrompt("hi", store.TriggerWebhook, `{"a":1}`, now)
	if !strings.Contains(got, "[webhook]\nreceived: 2026-09-01T09:00:00Z\npayload:\n{\"a\":1}") {
		t.Fatalf("payload block: %q", got)
	}
}

func TestAutomationCRUDAndSecret(t *testing.T) {
	ts, _ := newAutomationServer(t)
	res, out := doJSON(t, "POST", ts.URL+"/api/automations", `{"name":"Nightly","action":"start","prompt":"p","cron":"0 9 * * 1-5","webhook":true,"maxCostUsd":0.5}`, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create = %d %v", res.StatusCode, out)
	}
	secret, _ := out["webhookSecret"].(string)
	if len(secret) != 64 {
		t.Fatalf("secret = %q", secret)
	}
	a := out["automation"].(map[string]any)
	id := a["id"].(string)
	if a["nextFireAt"] == nil || a["webhook"] != true {
		t.Fatalf("view = %v", a)
	}
	res, _ = doJSON(t, "POST", ts.URL+"/api/automations", `{"name":"x","action":"start","prompt":"p"}`, nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("no trigger = %d", res.StatusCode)
	}
	res, out = doJSON(t, "GET", ts.URL+"/api/automations", "", nil)
	if res.StatusCode != 200 || len(out["items"].([]any)) != 1 {
		t.Fatalf("list = %d %v", res.StatusCode, out)
	}
	res, out = doJSON(t, "PATCH", ts.URL+"/api/automations/"+id, `{"enabled":false,"name":"Nightly 2"}`, nil)
	if res.StatusCode != 200 || out["automation"].(map[string]any)["enabled"] != false {
		t.Fatalf("patch = %d %v", res.StatusCode, out)
	}
	if _, ok := out["webhookSecret"]; ok {
		t.Fatal("patch without webhook change must not mint a secret")
	}
	res, out = doJSON(t, "POST", ts.URL+"/api/automations/"+id+"/secret", "", nil)
	if res.StatusCode != 200 || out["webhookSecret"] == secret {
		t.Fatalf("rotate = %d %v", res.StatusCode, out)
	}
	res, _ = doJSON(t, "GET", ts.URL+"/api/automations/nope", "", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing = %d", res.StatusCode)
	}
	res, _ = doJSON(t, "DELETE", ts.URL+"/api/automations/"+id, "", nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", res.StatusCode)
	}
}

func TestAutomationFireAuthAndSize(t *testing.T) {
	ts, st := newAutomationServer(t)
	_, out := doJSON(t, "POST", ts.URL+"/api/automations", `{"name":"Hook","action":"start","prompt":"p","webhook":true}`, nil)
	id := out["automation"].(map[string]any)["id"].(string)
	secret := out["webhookSecret"].(string)
	fire := ts.URL + "/api/automations/" + id + "/fire"

	if res, _ := doJSON(t, "POST", fire, "{}", nil); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no secret = %d", res.StatusCode)
	}
	if res, _ := doJSON(t, "POST", fire, "{}", map[string]string{"Authorization": "Bearer nope"}); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad secret = %d", res.StatusCode)
	}
	big := strings.Repeat("x", maxWebhookPayload+1)
	if res, _ := doJSON(t, "POST", fire, big, map[string]string{"X-Webhook-Secret": secret}); res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized = %d", res.StatusCode)
	}
	// Good secret: pi is missing in this harness, so the run is recorded
	// as failed / pi missing and one Inbox fyi is filed.
	res, out := doJSON(t, "POST", fire, `{"hello":1}`, map[string]string{"Authorization": "Bearer " + secret})
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("fire = %d %v", res.StatusCode, out)
	}
	run := out["run"].(map[string]any)
	if run["status"] != store.RunFailed || run["reason"] != reasonPiMissing || run["trigger"] != store.TriggerWebhook {
		t.Fatalf("run = %v", run)
	}
	items, _ := st.ListInboxItems(store.InboxFilter{})
	if len(items) != 1 || items[0].SourceKind != store.InboxFromAutomation || items[0].SourceID != id {
		t.Fatalf("inbox = %+v", items)
	}
	// Disabled: the webhook answers 409 and records nothing.
	doJSON(t, "PATCH", ts.URL+"/api/automations/"+id, `{"enabled":false}`, nil)
	if res, _ := doJSON(t, "POST", fire, "{}", map[string]string{"Authorization": "Bearer " + secret}); res.StatusCode != http.StatusConflict {
		t.Fatalf("disabled fire = %d", res.StatusCode)
	}
	if runs, _ := st.ListRuns(id, 0); len(runs) != 1 {
		t.Fatalf("disabled fire wrote a row: %+v", runs)
	}
	// No webhook configured → 404 (same as a missing id).
	_, out = doJSON(t, "POST", ts.URL+"/api/automations", `{"name":"Cron","action":"start","prompt":"p","cron":"0 9 * * *"}`, nil)
	cronID := out["automation"].(map[string]any)["id"].(string)
	if res, _ := doJSON(t, "POST", ts.URL+"/api/automations/"+cronID+"/fire", "{}", map[string]string{"Authorization": "Bearer x"}); res.StatusCode != http.StatusNotFound {
		t.Fatalf("no webhook = %d", res.StatusCode)
	}
}

func TestAutomationRunNowBusyAndMessage(t *testing.T) {
	ts, st := newAutomationServer(t)
	_, out := doJSON(t, "POST", ts.URL+"/api/automations", `{"name":"A","action":"start","prompt":"p","cron":"0 9 * * *","enabled":false}`, nil)
	id := out["automation"].(map[string]any)["id"].(string)
	// Run now ignores the toggle; with pi missing it fails honestly.
	res, out := doJSON(t, "POST", ts.URL+"/api/automations/"+id+"/run", "", nil)
	if res.StatusCode != http.StatusAccepted || out["run"].(map[string]any)["reason"] != reasonPiMissing {
		t.Fatalf("run now = %d %v", res.StatusCode, out)
	}
	// A running row makes the next Run now a 409 + skipped/busy.
	_, _ = st.CreateRun(id, store.TriggerSchedule, store.RunRunning, "")
	res, out = doJSON(t, "POST", ts.URL+"/api/automations/"+id+"/run", "", nil)
	if res.StatusCode != http.StatusConflict || out["run"].(map[string]any)["reason"] != reasonBusy {
		t.Fatalf("busy = %d %v", res.StatusCode, out)
	}
	res, out = doJSON(t, "GET", ts.URL+"/api/automations/"+id+"/runs?limit=10", "", nil)
	if res.StatusCode != 200 || len(out["items"].([]any)) != 3 {
		t.Fatalf("runs = %d %v", res.StatusCode, out)
	}

	// Message action delivers by running the agent, so without pi it fails
	// like a start run — and nothing is left in the agent's task queue.
	w, err := st.AddWorkspace("W", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ag, _ := st.AddAgent(w.ID, "target", "")
	_, out = doJSON(t, "POST", ts.URL+"/api/automations", `{"name":"M","action":"message","targetAgentId":"`+ag.ID+`","prompt":"ping","cron":"0 9 * * *"}`, nil)
	mid := out["automation"].(map[string]any)["id"].(string)
	res, out = doJSON(t, "POST", ts.URL+"/api/automations/"+mid+"/run", "", nil)
	if res.StatusCode != http.StatusAccepted || out["run"].(map[string]any)["reason"] != reasonPiMissing {
		t.Fatalf("message run without pi = %d %v", res.StatusCode, out)
	}
	if tasks, _ := st.ListTasks(ag.ID, 5); len(tasks) != 0 {
		t.Fatalf("a message run must not leave a queued task: %+v", tasks)
	}
	// Target deleted → failed / target gone.
	_ = st.DeleteAgent(ag.ID)
	res, out = doJSON(t, "POST", ts.URL+"/api/automations/"+mid+"/run", "", nil)
	if res.StatusCode != http.StatusAccepted || out["run"].(map[string]any)["reason"] != reasonTargetGone {
		t.Fatalf("target gone = %d %v", res.StatusCode, out)
	}
}

func TestAutomationTemplates(t *testing.T) {
	ts, _ := newAutomationServer(t)
	res, out := doJSON(t, "GET", ts.URL+"/api/automations/templates", "", nil)
	items, _ := out["items"].([]any)
	if res.StatusCode != 200 || len(items) < 6 {
		t.Fatalf("templates = %d %v", res.StatusCode, out)
	}
	first := items[0].(map[string]any)
	if first["id"] == "" || first["cron"] == "" || first["prompt"] == "" {
		t.Fatalf("template shape: %v", first)
	}
	// The templates route must not be shadowed by the {id} route.
	if res, _ := doJSON(t, "GET", ts.URL+"/api/automations/templates", "", nil); res.StatusCode == http.StatusNotFound {
		t.Fatal("templates route shadowed")
	}
}

// A managed process stopped under a run — the agent opened in a terminal,
// or stopped by hand — fails the run instead of leaving it running
// forever; the run's own stop stays silent.
func TestRunWatchExitedUnderTheRun(t *testing.T) {
	_, st := newAutomationServer(t)
	a, _, _ := st.CreateAutomation(store.AutomationParams{Name: "x", Action: "start", Prompt: "p", Cron: "0 9 * * *"})
	run, _ := st.CreateRun(a.ID, store.TriggerManual, store.RunRunning, "")
	w := &runWatch{runner: automationRunner{deps: Deps{Store: st}}, a: a, run: run, agentID: "none"}
	w.exited(true)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if r, _ := st.GetRun(run.ID); r.Status == store.RunFailed {
			if r.Reason != reasonStopped {
				t.Fatalf("reason %q", r.Reason)
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if r, _ := st.GetRun(run.ID); r.Status != store.RunFailed {
		t.Fatal("run not failed after an unexpected stop")
	}

	run2, _ := st.CreateRun(a.ID, store.TriggerManual, store.RunRunning, "")
	w2 := &runWatch{runner: automationRunner{deps: Deps{Store: st}}, a: a, run: run2, agentID: "none"}
	w2.letGo()
	w2.exited(true)
	time.Sleep(100 * time.Millisecond)
	if r, _ := st.GetRun(run2.ID); r.Status != store.RunRunning {
		t.Fatalf("our own stop must stay silent, got %q", r.Status)
	}
}
