package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	// Order, authorize, run, finish — the whole life of an entry.
	for _, step := range []struct {
		payload string
		want    string
	}{
		{fmt.Sprintf(`{"action":"order","requestId":"o1","id":%q,"expectedVersion":1,"orderKey":0}`, id), "waiting"},
		{fmt.Sprintf(`{"action":"authorize","requestId":"a1","id":%q,"expectedVersion":2}`, id), "authorized"},
		{fmt.Sprintf(`{"action":"start","requestId":"s1","id":%q,"expectedVersion":3}`, id), "running"},
		{fmt.Sprintf(`{"action":"finish","requestId":"f1","id":%q,"expectedVersion":4,"note":"integrated"}`, id), "done"},
	} {
		res, out = post(step.payload)
		if res != 200 {
			t.Fatalf("%s = %d %v", step.payload, res, out)
		}
		if got := out["entry"].(map[string]any)["state"]; got != step.want {
			t.Fatalf("state = %v, want %s", got, step.want)
		}
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
