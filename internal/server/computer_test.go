package server

// The computer tool's daemon half (ADR-0148): the grant is the whole gate,
// the command rides the shell line as a kind-tagged frame, and every call
// leaves a computer.step row. The helpers come from browser_test.go (the
// same line) and browser_developer_test.go (seeded agents).

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/browser"
	"github.com/cfpperche/picode/internal/computer"
	"github.com/cfpperche/picode/internal/store"
)

func postComputerTool(t *testing.T, ts *httptest.Server, body string) (int, string) {
	t.Helper()
	res, err := http.Post(ts.URL+"/api/computer/tool", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(out)
}

func computerSteps(t *testing.T, st *store.Store) []map[string]any {
	t.Helper()
	rows, err := st.EventsOfType("computer.step", 20)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]map[string]any, 0, len(rows))
	for _, ev := range rows {
		var data map[string]any
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			t.Fatalf("step data %s: %v", ev.Data, err)
		}
		if ev.AgentID != nil {
			data["_agent"] = *ev.AgentID
		}
		out = append(out, data)
	}
	return out
}

func TestComputerToolRefusesWithoutIdentityOrGrant(t *testing.T) {
	ts, _, st := browserServer(t)
	agent := seedAgent(t, st, "worker")

	code, body := postComputerTool(t, ts, `{"action":"screenshot"}`)
	if code != http.StatusForbidden || !strings.Contains(body, "no identity") {
		t.Fatalf("no identity: %d %s", code, body)
	}
	code, body = postComputerTool(t, ts, `{"agent":"`+agent+`","action":"screenshot"}`)
	if code != http.StatusForbidden || !strings.Contains(body, "Settings ▸ Computer") {
		t.Fatalf("no grant: %d %s", code, body)
	}
	code, body = postComputerTool(t, ts, `{"agent":"`+agent+`","action":"format_disk"}`)
	if code != http.StatusBadRequest || !strings.Contains(body, "unknown action") || !strings.Contains(body, "screenshot") {
		t.Fatalf("unknown action: %d %s", code, body)
	}
	code, body = postComputerTool(t, ts, `{"agent":"`+agent+`","action":"screenshot","params":[1]}`)
	if code != http.StatusBadRequest {
		t.Fatalf("params not an object: %d %s", code, body)
	}

	steps := computerSteps(t, st)
	if len(steps) != 2 {
		t.Fatalf("audit rows = %d, want the two refusals", len(steps))
	}
	// Newest first: the grant refusal carries the agent, the identity one no one.
	if steps[0]["outcome"] != "refused" || steps[0]["reason"] != "no grant" || steps[0]["_agent"] != agent {
		t.Fatalf("grant refusal row = %v", steps[0])
	}
	if steps[1]["reason"] != "no identity" || steps[1]["_agent"] != nil {
		t.Fatalf("identity refusal row = %v", steps[1])
	}
}

func TestComputerToolRidesTheShellLineAndIsAudited(t *testing.T) {
	ts, _, st := browserServer(t)
	agent := seedAgent(t, st, "worker")
	if err := computer.Save(st, agent, true); err != nil {
		t.Fatal(err)
	}
	body, closeStream := openBrowserStream(t, ts)
	defer closeStream()
	_ = readFrames(t, body, 1, 3*time.Second) // hello

	type answer struct {
		code int
		body string
	}
	done := make(chan answer, 1)
	go func() {
		code, out := postComputerTool(t, ts, `{"agent":"`+agent+`","call":"toolu_1","action":"Left_Click","params":{"coordinate":[412,88]}}`)
		done <- answer{code, out}
	}()
	frame := readFrames(t, body, 1, 3*time.Second)[0]
	var cmd browser.Command
	if err := json.Unmarshal([]byte(frame.Data), &cmd); err != nil {
		t.Fatal(err)
	}
	if cmd.Kind != "computer" || cmd.Method != "left_click" || cmd.Principal != agent || cmd.Tier != "" {
		t.Fatalf("frame = %+v", cmd)
	}
	if string(cmd.Params) != `{"coordinate":[412,88]}` {
		t.Fatalf("params = %s", cmd.Params)
	}
	output := `{"ok":true,"image":"iVBORw0KGgo=","mime":"image/png","meta":{"display":1,"window":{"exe":"notepad.exe"},"width":1280,"height":720}}`
	if code := postBrowserResult(t, ts, `{"id":"`+cmd.ID+`","output":`+output+`}`); code != http.StatusNoContent {
		t.Fatalf("result status %d", code)
	}
	got := <-done
	if got.code != http.StatusOK || !strings.Contains(got.body, `"action":"left_click"`) || !strings.Contains(got.body, `"image":"iVBORw0KGgo="`) {
		t.Fatalf("tool answer: %d %s", got.code, got.body)
	}
	steps := computerSteps(t, st)
	if len(steps) != 1 {
		t.Fatalf("audit rows = %d", len(steps))
	}
	row := steps[0]
	if row["outcome"] != "allowed" || row["action"] != "left_click" || row["call"] != "toolu_1" || row["_agent"] != agent {
		t.Fatalf("row = %v", row)
	}
	if sha, _ := row["imageSha256"].(string); len(sha) != 64 {
		t.Fatalf("image hash = %v", row["imageSha256"])
	}
	if row["display"] != float64(1) || !strings.Contains(string(mustJSON(row["window"])), "notepad.exe") {
		t.Fatalf("row meta = %v", row)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestComputerToolTerminalPrincipalAndNoShell(t *testing.T) {
	ts, hub, st := browserServer(t)
	hub.Timeout = 200 * time.Millisecond
	term, err := st.CreateTerminal("shell", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := computer.Save(st, "term:"+term.ID, true); err != nil {
		t.Fatal(err)
	}
	// Granted, but no shell on the line: the answer is immediate and audited.
	code, body := postComputerTool(t, ts, `{"term":"`+term.ID+`","action":"screenshot"}`)
	if code != http.StatusBadGateway || !strings.Contains(body, "not connected") {
		t.Fatalf("no shell: %d %s", code, body)
	}
	steps := computerSteps(t, st)
	if len(steps) != 1 || steps[0]["outcome"] != "failed" || steps[0]["termId"] != term.ID || steps[0]["_agent"] != nil {
		t.Fatalf("row = %v", steps)
	}
	if steps[0]["principal"] != "term:"+term.ID {
		t.Fatalf("principal = %v", steps[0]["principal"])
	}
}

func TestComputerPoliciesListAndSave(t *testing.T) {
	ts, _, st := browserServer(t)
	agent := seedAgent(t, st, "worker")
	term, err := st.CreateTerminal("shell", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	post := func(body string) (int, string) {
		res, err := http.Post(ts.URL+"/api/computer/policy", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		out, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(out)
	}
	if code, body := post(`{"agent":"nobody","enabled":true}`); code != http.StatusNotFound {
		t.Fatalf("unknown agent: %d %s", code, body)
	}
	if code, body := post(`{"agent":"` + agent + `","term":"` + term.ID + `","enabled":true}`); code != http.StatusBadRequest {
		t.Fatalf("both ids: %d %s", code, body)
	}
	if code, body := post(`{"agent":"` + agent + `","enabled":true}`); code != http.StatusOK || !strings.Contains(body, `"enabled":true`) {
		t.Fatalf("save agent: %d %s", code, body)
	}
	if code, body := post(`{"term":"` + term.ID + `","enabled":true}`); code != http.StatusOK {
		t.Fatalf("save term: %d %s", code, body)
	}
	if !computer.Resolve(st, agent).Enabled || !computer.Resolve(st, "term:"+term.ID).Enabled {
		t.Fatal("grants did not land")
	}
	res, err := http.Get(ts.URL + "/api/computer/policies")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var listed struct {
		Policies []struct {
			Kind    string `json:"kind"`
			AgentID string `json:"agentId"`
			TermID  string `json:"termId"`
			Enabled bool   `json:"enabled"`
			Saved   bool   `json:"saved"`
		} `json:"policies"`
	}
	if err := json.NewDecoder(res.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	var sawAgent, sawTerm bool
	for _, p := range listed.Policies {
		if p.Kind == "agent" && p.AgentID == agent && p.Enabled && p.Saved {
			sawAgent = true
		}
		if p.Kind == "terminal" && p.TermID == term.ID && p.Enabled && p.Saved {
			sawTerm = true
		}
	}
	if !sawAgent || !sawTerm {
		t.Fatalf("rows = %+v", listed.Policies)
	}
	if code, _ := post(`{"agent":"` + agent + `","enabled":false}`); code != http.StatusOK {
		t.Fatal("revoke failed")
	}
	if computer.Resolve(st, agent).Enabled {
		t.Fatal("revoke did not land")
	}
	// The audit reader returns what the tool route recorded.
	_, _ = postComputerTool(t, ts, `{"agent":"`+agent+`","action":"screenshot"}`)
	ares, err := http.Get(ts.URL + "/api/computer/audit?limit=5")
	if err != nil {
		t.Fatal(err)
	}
	defer ares.Body.Close()
	abody, _ := io.ReadAll(ares.Body)
	if !strings.Contains(string(abody), `"outcome":"refused"`) || !strings.Contains(string(abody), `"agentId":"`+agent+`"`) {
		t.Fatalf("audit = %s", abody)
	}
}
