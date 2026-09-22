package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestDeliveryToolNativePrincipals(t *testing.T) {
	ts, st := newInboxServer(t)
	repo := gitRepo(t)
	gitRun(t, repo, "branch", "feature")
	head := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	ws, err := st.AddWorkspace("delivery", repo)
	if err != nil {
		t.Fatal(err)
	}
	// No vendor process or credentials are used: this verifies the common boundary.
	for _, cli := range []string{"pi", "claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy", "omp"} {
		t.Run(cli, func(t *testing.T) {
			a, err := st.AddAgentWithCLI(ws.ID, cli, cli, "")
			if err != nil {
				t.Fatal(err)
			}
			payload := fmt.Sprintf(`{"agent":%q,"action":"register","requestId":"create","title":"Fix","branch":"feature","revision":%q,"target":"main"}`, a.ID, head)
			res, out := inboxPost(t, ts, "/api/delivery/tool", payload)
			if res.StatusCode != 200 {
				t.Fatalf("%d %v", res.StatusCode, out)
			}
			d := out["delivery"].(map[string]any)
			id := d["id"].(string)
			res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"agent":%q,"action":"request-review","id":%q,"expectedVersion":1,"requestId":"review"}`, a.ID, id))
			if res.StatusCode != 200 || out["delivery"].(map[string]any)["review"] != "requested" {
				t.Fatalf("%d %v", res.StatusCode, out)
			}
			res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"agent":%q,"action":"show","id":%q}`, a.ID, id))
			if res.StatusCode != 200 || out["validation"] != "unknown" {
				t.Fatalf("%d %v", res.StatusCode, out)
			}
		})
	}
	tm, err := st.CreateTerminalIn(ws.ID, "generic", repo)
	if err != nil {
		t.Fatal(err)
	}
	res, out := inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"term":%q,"action":"capabilities"}`, tm.ID))
	if res.StatusCode != 200 || out["identityScope"] != "launch" || out["integrationQueue"] != true {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"term":%q,"action":"register","requestId":"t","title":"Terminal","branch":"feature","revision":%q,"target":"main"}`, tm.ID, head))
	if res.StatusCode != 200 {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	id := out["delivery"].(map[string]any)["id"].(string)
	gitRun(t, repo, "checkout", "feature")
	gitRun(t, repo, "commit", "--allow-empty", "-m", "later")
	res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"term":%q,"action":"show","id":%q}`, tm.ID, id))
	if res.StatusCode != 200 || out["source"].(map[string]any)["status"] != "changed" {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	// Exact retry remains available after the branch moves, with no new record.
	res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"term":%q,"action":"register","requestId":"t","title":"Terminal","branch":"feature","revision":%q,"target":"main"}`, tm.ID, head))
	if res.StatusCode != 200 || out["replayed"] != true {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
}

func TestDeliveryToolRefusalTable(t *testing.T) {
	ts, st := newInboxServer(t)
	repo := gitRepo(t)
	ws, a, err := storeWorkspaceWithAgent(st, "delivery", repo)
	if err != nil {
		t.Fatal(err)
	}
	tm, _ := st.CreateTerminalIn(ws.ID, "other", repo)
	head := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	base := map[string]any{"agent": a.ID, "action": "register", "requestId": "create", "title": "Fix", "branch": "main", "revision": head, "target": "main"}
	rows := []struct {
		name   string
		edit   func(map[string]any)
		status int
	}{
		{"no identity", func(p map[string]any) { delete(p, "agent") }, 403},
		{"unknown identity", func(p map[string]any) { p["agent"] = "missing" }, 404},
		{"conflicting identity", func(p map[string]any) { p["term"] = tm.ID }, 403},
		{"unknown field", func(p map[string]any) { p["approved"] = true }, 400},
		{"unknown action", func(p map[string]any) { p["action"] = "force" }, 400},
		{"queue without a delivery", func(p map[string]any) { p["action"] = "request-integration" }, 404},
		{"withdraw of an unknown entry", func(p map[string]any) {
			p["action"] = "withdraw-integration"
			p["id"] = "queue_missing"
			p["expectedVersion"] = 1
		}, 404},
		{"deploy unavailable", func(p map[string]any) { p["action"] = "request-deployment" }, 409},
		{"missing retry", func(p map[string]any) { delete(p, "requestId") }, 400},
		{"wrong revision", func(p map[string]any) { p["revision"] = strings.Repeat("a", 40) }, 409},
		{"invalid ref", func(p map[string]any) { p["branch"] = "../../other" }, 400},
		{"missing target", func(p map[string]any) { p["target"] = "absent" }, 409},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			p := map[string]any{}
			for k, v := range base {
				p[k] = v
			}
			row.edit(p)
			raw, _ := json.Marshal(p)
			res, out := inboxPost(t, ts, "/api/delivery/tool", string(raw))
			if res.StatusCode != row.status {
				t.Fatalf("%d %v", res.StatusCode, out)
			}
		})
	}
	res, _ := inboxPost(t, ts, "/api/delivery/tool", `{} {}`)
	if res.StatusCode != 400 {
		t.Fatal(res.StatusCode)
	}
}

func TestDeliveryScopeAndSourceChanges(t *testing.T) {
	ts, st := newInboxServer(t)
	repo := gitRepo(t)
	ws, a, err := storeWorkspaceWithAgent(st, "source", repo)
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.AddAgent(ws.ID, "other", "")
	if err != nil {
		t.Fatal(err)
	}
	_, foreign, err := storeWorkspaceWithAgent(st, "foreign", gitRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	_, nongit, err := storeWorkspaceWithAgent(st, "nongit", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	head := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	res, out := inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"agent":%q,"action":"register","requestId":"create","title":"Fix","branch":"main","revision":%q,"target":"main"}`, a.ID, head))
	if res.StatusCode != 200 {
		t.Fatal(out)
	}
	id := out["delivery"].(map[string]any)["id"].(string)
	for _, row := range []struct {
		name, agent, action string
		status              int
	}{
		{"peer reads", other.ID, "show", 200},
		{"peer cannot write", other.ID, "withdraw-review", 404},
		{"foreign cannot read", foreign.ID, "show", 404},
		{"no Git repository", nongit.ID, "capabilities", 409},
	} {
		t.Run(row.name, func(t *testing.T) {
			res, out := inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"agent":%q,"action":%q,"id":%q,"expectedVersion":1,"requestId":"attempt"}`, row.agent, row.action, id))
			if res.StatusCode != row.status {
				t.Fatalf("%d %v", res.StatusCode, out)
			}
		})
	}
	gitRun(t, repo, "commit", "--allow-empty", "-m", "move")
	res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"agent":%q,"action":"request-review","id":%q,"expectedVersion":1,"requestId":"review"}`, a.ID, id))
	if res.StatusCode != 409 {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	if err := st.DeleteAgent(a.ID); err != nil {
		t.Fatal(err)
	}
	res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"agent":%q,"action":"show","id":%q}`, a.ID, id))
	if res.StatusCode != 404 {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"agent":%q,"action":"show","id":%q}`, other.ID, id))
	if res.StatusCode != 200 {
		t.Fatalf("history lost: %d %v", res.StatusCode, out)
	}
}

func TestDeliveryToolBoundPrincipal(t *testing.T) {
	ts, st := newInboxServer(t)
	repo := gitRepo(t)
	ws, a, err := storeWorkspaceWithAgent(st, "delivery", repo)
	if err != nil {
		t.Fatal(err)
	}
	tm, err := st.CreateTerminalIn(ws.ID, "bound", repo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = st.UpdateAgent(a.ID, store.AgentPatch{TerminalID: &tm.ID}); err != nil {
		t.Fatal(err)
	}
	for _, identity := range []string{fmt.Sprintf(`"agent":%q,"term":%q`, a.ID, tm.ID), fmt.Sprintf(`"term":%q`, tm.ID)} {
		res, out := inboxPost(t, ts, "/api/delivery/tool", `{`+identity+`,"action":"capabilities"}`)
		if res.StatusCode != 200 || out["principal"] != a.ID {
			t.Fatalf("%d %v", res.StatusCode, out)
		}
	}
	head := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	payload := fmt.Sprintf(`"action":"register","requestId":"create","title":"Bound","branch":"main","revision":%q,"target":"main"`, head)
	res, out := inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"agent":%q,"term":%q,%s}`, a.ID, tm.ID, payload))
	if res.StatusCode != 200 {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	d := out["delivery"].(map[string]any)
	id := d["id"].(string)
	if d["principal"] != a.ID {
		t.Fatalf("%v", d)
	}
	res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"term":%q,%s}`, tm.ID, payload))
	if res.StatusCode != 200 || out["replayed"] != true || out["delivery"].(map[string]any)["id"] != id {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"term":%q,"action":"request-review","id":%q,"expectedVersion":1,"requestId":"review"}`, tm.ID, id))
	if res.StatusCode != 200 || out["delivery"].(map[string]any)["review"] != "requested" {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
	res, out = inboxPost(t, ts, "/api/delivery/tool", fmt.Sprintf(`{"agent":%q,"term":%q,"action":"withdraw-review","id":%q,"expectedVersion":2,"requestId":"withdraw"}`, a.ID, tm.ID, id))
	if res.StatusCode != 200 || out["delivery"].(map[string]any)["review"] != "not-requested" {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
}

func TestDeliveryObservationOwnerAndRoot(t *testing.T) {
	ts, st := newInboxServer(t)
	repo := gitRepo(t)
	gitRun(t, repo, "checkout", "-b", "feature")
	ws, a, err := storeWorkspaceWithAgent(st, "observer", repo)
	if err != nil {
		t.Fatal(err)
	}
	tm, err := st.CreateTerminalIn(ws.ID, "terminal", repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/workspaces/" + ws.ID, "/api/agents/" + a.ID, "/api/terminals/" + tm.ID} {
		res, e := http.Get(ts.URL + path + "/delivery?target=main")
		if e != nil {
			t.Fatal(e)
		}
		var out map[string]any
		json.NewDecoder(res.Body).Decode(&out)
		res.Body.Close()
		if res.StatusCode != 200 || out["targetOid"] == "" || out["schemaVersion"] != float64(1) {
			t.Fatalf("%d %v", res.StatusCode, out)
		}
		associated := false
		for _, item := range out["changes"].([]any) {
			change := item.(map[string]any)
			if change["branch"] != "feature" {
				continue
			}
			for _, name := range change["agents"].([]any) {
				associated = associated || name == a.Name
			}
		}
		if !associated {
			t.Fatalf("current checkout agent missing: %v", out)
		}
		res, e = http.Get(ts.URL + path + "/delivery?root=wrong")
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode != 409 {
			t.Fatal(res.StatusCode)
		}
	}
}

// The agent's half of ADR-0182: a launch asks for its own change, sees the
// entry it got, and can step out of the queue while it waits.
// The agent's half of ADR-0182: a launch asks for its own change, sees the
// entry it got, and can step out of the queue while it waits.
func TestDeliveryIntegrationQueueAgentDoor(t *testing.T) {
	ts, st := newInboxServer(t)
	repo := gitRepo(t)
	gitRun(t, repo, "branch", "feature")
	head := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	ws, agent, err := storeWorkspaceWithAgent(st, "queue", repo)
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.AddAgentWithCLI(ws.ID, "codex", "Other", "")
	if err != nil {
		t.Fatal(err)
	}
	call := func(agentID string, payload map[string]any) (int, map[string]any) {
		payload["agent"] = agentID
		raw, _ := json.Marshal(payload)
		return queueRequest(t, ts, "POST", "/api/delivery/tool", string(raw))
	}
	if code, out := call(agent.ID, map[string]any{"action": "capabilities"}); code != 200 || out["integrationQueue"] != true {
		t.Fatalf("capabilities = %d %v", code, out)
	} else {
		actions, _ := out["actions"].([]any)
		found := 0
		for _, a := range actions {
			if a == "request-integration" || a == "withdraw-integration" {
				found++
			}
		}
		if found != 2 {
			t.Fatalf("the queue actions are not offered: %v", actions)
		}
	}
	code, out := call(agent.ID, map[string]any{"action": "register", "requestId": "create", "title": "Fix", "branch": "feature", "revision": head, "target": "main"})
	if code != 200 {
		t.Fatalf("register = %d %v", code, out)
	}
	delivery := out["delivery"].(map[string]any)["id"].(string)
	request := func(key, revision string) map[string]any {
		return map[string]any{"action": "request-integration", "requestId": key, "id": delivery, "revision": revision, "target": "main"}
	}
	// A revision that is not the declared one is refused before anything waits.
	if code, out := call(agent.ID, request("q0", strings.Repeat("b", 40))); code != 409 {
		t.Fatalf("drifted revision = %d %v", code, out)
	}
	code, out = call(agent.ID, request("q1", head))
	if code != 200 {
		t.Fatalf("request = %d %v", code, out)
	}
	entry := out["queue"].(map[string]any)
	if entry["state"] != "waiting" || entry["principal"] != agent.ID || out["replayed"] != false {
		t.Fatalf("entry = %v", entry)
	}
	queueID := entry["id"].(string)
	// A retry with the same key answers with the same entry, not a second one.
	if code, out = call(agent.ID, request("q1", head)); code != 200 || out["replayed"] != true || out["queue"].(map[string]any)["id"] != queueID {
		t.Fatalf("retry = %d %v", code, out)
	}
	if code, out = call(agent.ID, request("q2", head)); code != 409 {
		t.Fatalf("second request = %d %v", code, out)
	}
	// show carries the entry and the version a withdraw must name.
	if code, out = call(agent.ID, map[string]any{"action": "show", "id": delivery}); code != 200 {
		t.Fatalf("show = %d %v", code, out)
	} else if entries, ok := out["queue"].([]any); !ok || len(entries) != 1 || entries[0].(map[string]any)["id"] != queueID {
		t.Fatalf("show queue = %v", out["queue"])
	}
	// Another launch in the same repository may not touch it.
	if code, out = call(other.ID, request("x1", head)); code != 403 {
		t.Fatalf("foreign request = %d %v", code, out)
	}
	if code, out = call(other.ID, map[string]any{"action": "withdraw-integration", "requestId": "x2", "id": queueID, "expectedVersion": 1}); code != 403 {
		t.Fatalf("foreign withdraw = %d %v", code, out)
	}
	// A stale version is a conflict, whatever the actor.
	if code, out = call(agent.ID, map[string]any{"action": "withdraw-integration", "requestId": "w0", "id": queueID, "expectedVersion": 9}); code != 409 {
		t.Fatalf("stale withdraw = %d %v", code, out)
	}
	// The agent's door has no authority: authorizing is the owner's alone, and
	// this face does not offer it at all.
	if code, out = call(agent.ID, map[string]any{"action": "authorize", "requestId": "a1", "id": queueID, "expectedVersion": 1}); code != 400 {
		t.Fatalf("the agent tried to authorize = %d %v", code, out)
	}
	// Withdrawing while it waits is the agent's own move, and it frees the
	// delivery for a fresh request.
	if code, out = call(agent.ID, map[string]any{"action": "withdraw-integration", "requestId": "w1", "id": queueID, "expectedVersion": 1}); code != 200 || out["queue"].(map[string]any)["state"] != "withdrawn" {
		t.Fatalf("withdraw = %d %v", code, out)
	}
	// Withdrawn frees the delivery: a fresh request takes its own place.
	if code, out = call(agent.ID, request("q3", head)); code != 200 {
		t.Fatalf("re-request = %d %v", code, out)
	} else if out["queue"].(map[string]any)["state"] != "waiting" {
		t.Fatalf("re-request state = %v", out["queue"])
	}
	if code, out = call(agent.ID, map[string]any{"action": "request-deployment", "requestId": "d1"}); code != 409 {
		t.Fatalf("deployment = %d %v", code, out)
	}
}
