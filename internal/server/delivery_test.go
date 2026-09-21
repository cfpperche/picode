package server

import (
	"encoding/json"
	"fmt"
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
	if res.StatusCode != 200 || out["identityScope"] != "launch" || out["integrationQueue"] != false {
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
		{"queue unavailable", func(p map[string]any) { p["action"] = "request-integration" }, 409},
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
