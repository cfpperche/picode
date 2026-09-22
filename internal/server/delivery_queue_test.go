package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/store"
)

// queueRequest performs one request against the test server and returns the
// status with the decoded JSON object; the test server needs no credential.
func queueRequest(t *testing.T, ts *httptest.Server, method, path, payload string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, ts.URL+path, bytes.NewBufferString(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// The owner's half of ADR-0182: the queue read inside the Delivery read, the
// owner's actions on it, and the declaration with its workspace-first fallback.
func TestDeliveryQueueOwnerDoors(t *testing.T) {
	ts, st := newInboxServer(t)
	repo := gitRepo(t)
	gitRun(t, repo, "branch", "feature")
	head := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	ws, err := st.AddWorkspace("queue", repo)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := st.AddAgentWithCLI(ws.ID, "claude-code", "Claude", "")
	if err != nil {
		t.Fatal(err)
	}
	regRes, out := inboxPost(t, ts, "/api/delivery/tool",
		fmt.Sprintf(`{"agent":%q,"action":"register","requestId":"create","title":"Fix","branch":"feature","revision":%q,"target":"main"}`, agent.ID, head))
	if regRes.StatusCode != 200 {
		t.Fatalf("register = %d %v", regRes.StatusCode, out)
	}
	res := 0
	delivery := out["delivery"].(map[string]any)["id"].(string)

	queuePath := "/api/workspaces/" + ws.ID + "/delivery/queue"
	post := func(payload string) (int, map[string]any) { return queueRequest(t, ts, "POST", queuePath, payload) }

	// The owner may write an entry down for a delivery an agent declared; the
	// entry belongs to that agent.
	res, out = post(fmt.Sprintf(`{"action":"enqueue","requestId":"q1","deliveryId":%q,"revision":%q,"target":"main"}`, delivery, head))
	if res != 200 {
		t.Fatalf("enqueue = %d %v", res, out)
	}
	entry := out["entry"].(map[string]any)
	id := entry["id"].(string)
	if entry["state"] != "waiting" || entry["principal"] != agent.ID || entry["version"].(float64) != 1 {
		t.Fatalf("entry = %v", entry)
	}

	// A second active entry for the same delivery is a refusal, not a duplicate.
	if res, out = post(fmt.Sprintf(`{"action":"enqueue","requestId":"q2","deliveryId":%q,"revision":%q,"target":"main"}`, delivery, head)); res != 409 {
		t.Fatalf("second enqueue = %d %v", res, out)
	}
	// A shape the action cannot carry is a 400 before any state is touched.
	if res, _ = post(fmt.Sprintf(`{"action":"withdraw","requestId":"w0","id":%q,"expectedVersion":1,"note":"x"}`, id)); res != 400 {
		t.Fatalf("withdraw with a note = %d", res)
	}
	// Skipping authorization is a 409 that names the way out.
	if res, out = post(fmt.Sprintf(`{"action":"start","requestId":"s0","id":%q,"expectedVersion":1}`, id)); res != 409 || !strings.Contains(fmt.Sprint(out), "must be authorized first") {
		t.Fatalf("start before authorize = %d %v", res, out)
	}
	// A stale version never overwrites a writer.
	if res, _ = post(fmt.Sprintf(`{"action":"order","requestId":"o0","id":%q,"expectedVersion":9,"orderKey":1}`, id)); res != 409 {
		t.Fatalf("stale version = %d", res)
	}
	if res, _ = post(`{"action":"authorize","requestId":"a0","id":"queue_nope","expectedVersion":1}`); res != 404 {
		t.Fatalf("unknown entry = %d", res)
	}

	// The owner orders and authorizes; authorizing is execution authority, so
	// the entry runs. This repository declares no rules, which is a named
	// blocker rather than a guess: the run fails it and says why.
	if res, out = post(fmt.Sprintf(`{"action":"order","requestId":"o1","id":%q,"expectedVersion":1,"orderKey":0}`, id)); res != 200 ||
		out["entry"].(map[string]any)["state"] != "waiting" {
		t.Fatalf("order = %d %v", res, out)
	}
	if res, out = post(fmt.Sprintf(`{"action":"authorize","requestId":"a1","id":%q,"expectedVersion":2}`, id)); res != 200 ||
		out["entry"].(map[string]any)["state"] != "authorized" {
		t.Fatalf("authorize = %d %v", res, out)
	}
	settled := waitForQueueState(t, ts, ws.ID, id)
	if settled["state"] != "failed" || !strings.HasPrefix(settled["note"].(string), "not run: the project declares no integration rules") {
		t.Fatalf("settled = %v", settled)
	}

	// The Delivery read carries the queue and the effective declaration.
	res, out = queueRequest(t, ts, "GET", "/api/workspaces/"+ws.ID+"/delivery", "")
	if res != 200 {
		t.Fatalf("read = %d", res)
	}
	queue, _ := out["queue"].([]any)
	if len(queue) != 1 {
		t.Fatalf("queue = %v", out["queue"])
	}
	integration, _ := out["integration"].(map[string]any)
	if integration == nil || integration["ffOnly"] != true || integration["fromScope"] != "default" {
		t.Fatalf("integration = %v", out["integration"])
	}
}

func TestDeliveryIntegrationDeclaration(t *testing.T) {
	ts, st := newInboxServer(t)
	root := gitRepo(t)
	ws, err := st.AddWorkspace("declared", root)
	if err != nil {
		t.Fatal(err)
	}
	// A second workspace on its own folder: AddWorkspace is idempotent by path,
	// so sharing the repository would return the same row.
	other, err := st.AddWorkspace("silent", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	put := func(query, payload string) (int, map[string]any) {
		return queueRequest(t, ts, "PUT", "/api/delivery/integration"+query, payload)
	}
	get := func(query string) map[string]any {
		_, out := queueRequest(t, ts, "GET", "/api/delivery/integration"+query, "")
		return out
	}

	// A workspace declares its own operation.
	if code, out := put("?workspace="+ws.ID, `{"ffOnly":true,"checks":["make ci"]}`); code != 200 {
		t.Fatalf("put workspace = %d %v", code, out)
	}
	if out := get("?workspace=" + ws.ID); out["declared"] != true || out["effective"].(map[string]any)["fromScope"] != ws.ID {
		t.Fatalf("declared = %v", out)
	}
	// A workspace that declared nothing falls back to the machine default.
	if code, out := put("", `{"ffOnly":false,"checks":["go test ./..."]}`); code != 200 {
		t.Fatalf("put machine = %d %v", code, out)
	}
	out := get("?workspace=" + other.ID)
	eff := out["effective"].(map[string]any)
	checks, _ := eff["checks"].([]any)
	if out["declared"] != false || eff["ffOnly"] != false || len(checks) != 1 || checks[0] != "go test ./..." {
		t.Fatalf("fallback = %v", out)
	}
	// The workspace layer still wins where it exists.
	if eff := get("?workspace=" + ws.ID)["effective"].(map[string]any); eff["fromScope"] != ws.ID || eff["ffOnly"] != true {
		t.Fatalf("workspace layer lost: %v", eff)
	}
	// Shape refusals: too many commands, and a command that is not one line.
	if code, _ := put("?workspace="+ws.ID, `{"ffOnly":true,"checks":["1","2","3","4","5","6","7","8","9"]}`); code != 400 {
		t.Fatalf("nine checks = %d", code)
	}
	if code, _ := put("?workspace="+ws.ID, `{"ffOnly":true,"checks":["make ci\ngo vet"]}`); code != 400 {
		t.Fatalf("two-line check = %d", code)
	}
	// A unknown workspace is refused instead of writing a row nobody reads.
	if code, _ := put("?workspace=nope", `{"ffOnly":true}`); code != 404 {
		t.Fatalf("unknown workspace = %d", code)
	}
}

// The Settings dialog's doors: a stale save conflicts, "use the machine's"
// drops the workspace layer, and the machine layer cannot be deleted here.
func TestIntegrationDeclarationConflictAndInherit(t *testing.T) {
	ts, st := newInboxServer(t)
	ws, err := st.AddWorkspace("w", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	q := "/api/delivery/integration?workspace=" + ws.ID
	if code, out := queueRequest(t, ts, "PUT", q, `{"ffOnly":true,"checks":["make ci"]}`); code != 200 {
		t.Fatalf("first save = %d %v", code, out)
	}
	if code, _ := queueRequest(t, ts, "PUT", q, `{"ffOnly":false,"expectedVersion":1}`); code != 200 {
		t.Fatalf("save at the read version = %d", code)
	}
	if code, _ := queueRequest(t, ts, "PUT", q, `{"ffOnly":true,"expectedVersion":1}`); code != 409 {
		t.Fatalf("stale save = %d, want 409", code)
	}
	if code, _ := queueRequest(t, ts, "DELETE", q, ""); code != 204 {
		t.Fatalf("inherit = %d, want 204", code)
	}
	if _, out := queueRequest(t, ts, "GET", q, ""); out["declared"] != false {
		t.Fatalf("after inherit = %v", out)
	}
	for _, c := range []struct {
		path string
		code int
	}{
		{"/api/delivery/integration", 400},
		{"/api/delivery/integration?workspace=nope", 404},
	} {
		if code, _ := queueRequest(t, ts, "DELETE", c.path, ""); code != c.code {
			t.Errorf("DELETE %s = %d, want %d", c.path, code, c.code)
		}
	}
}

// Authorizing is execution authority (ADR-0182): the declared operation runs
// against the repository, and the answer never waits for it.
func TestDeliveryQueueAuthorizeRunsTheDeclaration(t *testing.T) {
	ts, st := newInboxServer(t)
	repo := gitRepo(t)
	gitRun(t, repo, "checkout", "-q", "-b", "feature")
	gitRun(t, repo, "commit", "--allow-empty", "-m", "feature work")
	head := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	gitRun(t, repo, "checkout", "-q", "main")
	ws, agent, err := storeWorkspaceWithAgent(st, "queue", repo)
	if err != nil {
		t.Fatal(err)
	}
	if code, out := queueRequest(t, ts, "PUT", "/api/delivery/integration?workspace="+ws.ID,
		`{"ffOnly":true,"checks":["true"]}`); code != 200 {
		t.Fatalf("declare = %d %v", code, out)
	}
	code, out := queueRequest(t, ts, "POST", "/api/delivery/tool", fmt.Sprintf(
		`{"agent":%q,"action":"register","requestId":"create","title":"Fix","branch":"feature","revision":%q,"target":"main"}`, agent.ID, head))
	if code != 200 {
		t.Fatalf("register = %d %v", code, out)
	}
	delivery := out["delivery"].(map[string]any)["id"].(string)
	code, out = queueRequest(t, ts, "POST", "/api/delivery/tool", fmt.Sprintf(
		`{"agent":%q,"action":"request-integration","requestId":"q1","id":%q,"revision":%q,"target":"main"}`, agent.ID, delivery, head))
	if code != 200 {
		t.Fatalf("request = %d %v", code, out)
	}
	queueID := out["queue"].(map[string]any)["id"].(string)
	if code, out = queueRequest(t, ts, "POST", "/api/workspaces/"+ws.ID+"/delivery/queue", fmt.Sprintf(
		`{"action":"authorize","requestId":"a1","id":%q,"expectedVersion":1}`, queueID)); code != 200 {
		t.Fatalf("authorize = %d %v", code, out)
	}
	entry := waitForQueueState(t, ts, ws.ID, queueID)
	if entry["state"] != "done" || !strings.Contains(entry["note"].(string), "integrated") {
		t.Fatalf("entry = %v", entry)
	}
	if got := strings.TrimSpace(gitOut(t, repo, "rev-parse", "main")); got != head {
		t.Fatalf("main = %s, want %s", got, head)
	}
}

// A declared command that fails stops the operation, names itself, and leaves
// the target where it was.
func TestDeliveryQueueFailedCheckLeavesTheTarget(t *testing.T) {
	ts, st := newInboxServer(t)
	repo := gitRepo(t)
	gitRun(t, repo, "checkout", "-q", "-b", "feature")
	gitRun(t, repo, "commit", "--allow-empty", "-m", "feature work")
	head := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	gitRun(t, repo, "checkout", "-q", "main")
	before := strings.TrimSpace(gitOut(t, repo, "rev-parse", "main"))
	ws, agent, err := storeWorkspaceWithAgent(st, "queue", repo)
	if err != nil {
		t.Fatal(err)
	}
	if code, out := queueRequest(t, ts, "PUT", "/api/delivery/integration?workspace="+ws.ID,
		`{"ffOnly":true,"checks":["echo nope >&2; false"]}`); code != 200 {
		t.Fatalf("declare = %d %v", code, out)
	}
	code, out := queueRequest(t, ts, "POST", "/api/delivery/tool", fmt.Sprintf(
		`{"agent":%q,"action":"register","requestId":"create","title":"Fix","branch":"feature","revision":%q,"target":"main"}`, agent.ID, head))
	if code != 200 {
		t.Fatalf("register = %d %v", code, out)
	}
	delivery := out["delivery"].(map[string]any)["id"].(string)
	code, out = queueRequest(t, ts, "POST", "/api/delivery/tool", fmt.Sprintf(
		`{"agent":%q,"action":"request-integration","requestId":"q1","id":%q,"revision":%q,"target":"main"}`, agent.ID, delivery, head))
	if code != 200 {
		t.Fatalf("request = %d %v", code, out)
	}
	queueID := out["queue"].(map[string]any)["id"].(string)
	if code, out = queueRequest(t, ts, "POST", "/api/workspaces/"+ws.ID+"/delivery/queue", fmt.Sprintf(
		`{"action":"authorize","requestId":"a1","id":%q,"expectedVersion":1}`, queueID)); code != 200 {
		t.Fatalf("authorize = %d %v", code, out)
	}
	entry := waitForQueueState(t, ts, ws.ID, queueID)
	if entry["state"] != "failed" || !strings.Contains(entry["note"].(string), "check 1 of 1 failed") {
		t.Fatalf("entry = %v", entry)
	}
	if got := strings.TrimSpace(gitOut(t, repo, "rev-parse", "main")); got != before {
		t.Fatalf("main moved to %s although the declared check failed", got)
	}
}

// waitForQueueState polls the Delivery read until the entry reaches a terminal
// state: the run is asynchronous by design, so the test waits for the queue,
// not for an arbitrary sleep.
func waitForQueueState(t *testing.T, ts *httptest.Server, workspaceID, entryID string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for {
		_, read := queueRequest(t, ts, "GET", "/api/workspaces/"+workspaceID+"/delivery", "")
		if entries, ok := read["queue"].([]any); ok {
			for _, raw := range entries {
				entry := raw.(map[string]any)
				if entry["id"] != entryID {
					continue
				}
				switch entry["state"] {
				case "done", "failed", "withdrawn":
					return entry
				}
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("entry %s never finished: %v", entryID, read)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// The daemon's own start path (ResumeDeliveryQueue): an entry left running by a
// previous process is marked unknown — never failed, never retried — and every
// authorized entry runs. This is what a restart owes the repository.
func TestDeliveryQueueResumeMarksUnknownAndDrains(t *testing.T) {
	ts, st := newInboxServer(t)
	repo := gitRepo(t)
	gitRun(t, repo, "checkout", "-q", "-b", "feature")
	gitRun(t, repo, "commit", "--allow-empty", "-m", "feature work")
	head := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	gitRun(t, repo, "checkout", "-q", "main")
	ws, agent, err := storeWorkspaceWithAgent(st, "queue", repo)
	if err != nil {
		t.Fatal(err)
	}
	// Two deliveries: one whose run died with the previous daemon, one the owner
	// had authorized and never got to see.
	interrupted := queueEntryIn(t, st, ws.ID, agent.ID, repo, head, "one")
	running, err := st.ApplyQueueMutation(repoKeyOf(t, repo), store.OwnerActor, store.QueueMutation{Action: "start",
		RequestID: "start-interrupted", ID: interrupted.ID, ExpectedVersion: interrupted.Version})
	if err != nil {
		t.Fatal(err)
	}
	authorized := queueEntryIn(t, st, ws.ID, agent.ID, repo, head, "two")

	newQueueRuns(Deps{Store: st}).resume()

	marked, err := st.GetQueueEntry(repoKeyOf(t, repo), running.ID)
	if err != nil || marked.State != store.QueueRunning || !strings.Contains(marked.Note, "outcome is unknown") {
		t.Fatalf("the interrupted entry = %+v (%v)", marked, err)
	}
	settled := waitForQueueState(t, ts, ws.ID, authorized.ID)
	if settled["state"] != "failed" || !strings.HasPrefix(settled["note"].(string), "not run: the project declares no integration rules") {
		t.Fatalf("the authorized entry = %v", settled)
	}
	if marked2, err := st.GetQueueEntry(repoKeyOf(t, repo), running.ID); err != nil || marked2.Note != marked.Note {
		t.Fatalf("the interrupted entry was touched again: %+v (%v)", marked2, err)
	}
}

// queueEntryIn registers a delivery and leaves an entry waiting, through the
// store: the door's authorize would run it, and these tests stage states.
func queueEntryIn(t *testing.T, st *store.Store, workspaceID, agentID, repo, revision, tag string) store.QueueEntry {
	t.Helper()
	key := repoKeyOf(t, repo)
	d, err := st.ApplyDelivery(key, agentID, store.DeliveryMutation{Action: "register", RequestID: "create-" + tag,
		Title: "Fix " + tag, Branch: "feature", Revision: revision, Target: "main"})
	if err != nil {
		t.Fatal(err)
	}
	e, err := st.ApplyQueueMutation(key, agentID, store.QueueMutation{Action: "enqueue", RequestID: "enqueue-" + tag,
		DeliveryID: d.ID, Revision: revision, Target: "main"})
	if err != nil {
		t.Fatal(err)
	}
	authorized, err := st.ApplyQueueMutation(key, store.OwnerActor, store.QueueMutation{Action: "authorize",
		RequestID: "authorize-" + tag, ID: e.ID, ExpectedVersion: e.Version})
	if err != nil {
		t.Fatal(err)
	}
	return authorized
}

func repoKeyOf(t *testing.T, repo string) string {
	t.Helper()
	key := gitgraph.Key(repo)
	if key == "" {
		t.Fatalf("no repository key for %s", repo)
	}
	return key
}
