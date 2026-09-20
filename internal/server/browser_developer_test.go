package server

// The raw-CDP route (ADR-0144): the decision table the owner agreed to, the
// audit row that makes the door defensible, and the reader the settings card
// shows. The catalog verbs keep their own tests in browser_test.go — this
// file is only about the path that goes around the catalog.

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/browser"
	"github.com/cfpperche/picode/internal/store"
)

func postBrowserTool(t *testing.T, url, body string) (int, string) {
	t.Helper()
	res, err := http.Post(url+"/api/browser/tool", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(out)
}

// rawCalls reads the audit rows the way the settings card does.
func rawCalls(t *testing.T, st *store.Store) []store.Event {
	t.Helper()
	rows, err := st.EventsOfType("browser.cdp", 20)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func rawCallData(t *testing.T, ev store.Event) map[string]any {
	t.Helper()
	var data map[string]any
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		t.Fatalf("audit data %s: %v", ev.Data, err)
	}
	return data
}

// seedAgent makes an agent real and returns its id: audit events carry an
// agent id, and the store refuses a record about someone who does not exist.
func seedAgent(t *testing.T, st *store.Store, name string) string {
	t.Helper()
	ws, err := st.AddWorkspace("ws-"+name, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, err := st.AddAgent(ws.ID, name, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return a.ID
}

func TestRawCDPDecisionTable(t *testing.T) {
	// The rows are the ADR's table, one test each:
	// developer mode off + any tier   → refused
	// developer mode on  + read       → refused
	// developer mode on  + act        → refused
	// developer mode on  + full       → allowed (and recorded)
	cases := []struct {
		name    string
		devMode bool
		tier    string
		want    int
		why     string
	}{
		{"off + full", false, "full", http.StatusForbidden, "Developer mode"},
		{"on + read", true, "read", http.StatusForbidden, "full tier"},
		{"on + act", true, "act", http.StatusForbidden, "full tier"},
		{"on + full", true, "full", http.StatusOK, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts, _, st := browserServer(t)
			agentID := seedAgent(t, st, "raw")
			body, closeStream := openBrowserStream(t, ts)
			defer closeStream()
			readFrames(t, body, 1, 3*time.Second) // hello
			if tc.devMode {
				if err := st.SetSetting("browser.developerMode", "1"); err != nil {
					t.Fatal(err)
				}
			}
			if err := browser.Save(st, agentID, browser.Policy{Tier: tc.tier}); err != nil {
				t.Fatal(err)
			}

			type answer struct {
				status int
				body   string
			}
			done := make(chan answer, 1)
			payload := `{"agent":"` + agentID + `","verb":"cdp","params":{"method":"Network.getAllCookies"}}`
			go func() {
				status, out := postBrowserTool(t, ts.URL, payload)
				done <- answer{status, out}
			}()

			if tc.want == http.StatusOK {
				cmd := readFrames(t, body, 1, 3*time.Second)[0]
				var pushed browser.Command
				if err := json.Unmarshal([]byte(cmd.Data), &pushed); err != nil {
					t.Fatal(err)
				}
				// The method the caller named is the method the shell runs,
				// and it travels with the raw flag — that flag is what the
				// shell lets past its catalog after its own checks.
				if pushed.Method != "Network.getAllCookies" || !pushed.Raw || pushed.Tier != "full" {
					t.Fatalf("command = %+v", pushed)
				}
				if code := postBrowserResult(t, ts, `{"id":"`+pushed.ID+`","output":{"cookies":[]}}`); code != http.StatusNoContent {
					t.Fatalf("result status = %d", code)
				}
			}

			got := <-done
			if got.status != tc.want {
				t.Fatalf("status = %d %s, want %d", got.status, got.body, tc.want)
			}
			if tc.why != "" && !strings.Contains(got.body, tc.why) {
				t.Fatalf("refusal = %q, want it to name %q", got.body, tc.why)
			}
			// Every row leaves exactly one record, allowed or refused: that is
			// what makes the switch defensible.
			rows := rawCalls(t, st)
			if len(rows) != 1 {
				t.Fatalf("audit rows = %d, want 1", len(rows))
			}
			data := rawCallData(t, rows[0])
			if data["method"] != "Network.getAllCookies" {
				t.Fatalf("audit method = %v", data["method"])
			}
			wantOutcome := "allowed"
			if tc.want != http.StatusOK {
				wantOutcome = "refused"
			}
			if data["outcome"] != wantOutcome {
				t.Fatalf("audit outcome = %v, want %s", data["outcome"], wantOutcome)
			}
			if rows[0].AgentID == nil || *rows[0].AgentID != agentID {
				t.Fatalf("audit agent = %v, want %s", rows[0].AgentID, agentID)
			}
		})
	}
}

// A refusal below full tier must not need a shell: the decision is the
// daemon's, and the owner hears about it before anything travels. The stream
// is never opened here on purpose.
func TestRawCDPRefusesWithoutAShell(t *testing.T) {
	ts, _, st := browserServer(t)
	agentID := seedAgent(t, st, "raw")
	if err := st.SetSetting("browser.developerMode", "1"); err != nil {
		t.Fatal(err)
	}
	if err := browser.Save(st, agentID, browser.Policy{Tier: "read"}); err != nil {
		t.Fatal(err)
	}
	status, body := postBrowserTool(t, ts.URL, `{"agent":"`+agentID+`","verb":"cdp","params":{"method":"Page.navigate"}}`)
	if status != http.StatusForbidden || !strings.Contains(body, "full") {
		t.Fatalf("answer = %d %s", status, body)
	}
}

func TestRawCDPNeedsAMethod(t *testing.T) {
	ts, _, st := browserServer(t)
	agentID := seedAgent(t, st, "raw")
	if err := st.SetSetting("browser.developerMode", "1"); err != nil {
		t.Fatal(err)
	}
	if err := browser.Save(st, agentID, browser.Policy{Tier: "full"}); err != nil {
		t.Fatal(err)
	}
	for _, params := range []string{`{}`, `{"method":""}`, `{"method":42}`} {
		status, body := postBrowserTool(t, ts.URL, `{"agent":"`+agentID+`","verb":"cdp","params":`+params+`}`)
		if status != http.StatusBadRequest || !strings.Contains(body, "method") {
			t.Fatalf("params %s → %d %s", params, status, body)
		}
	}
	// An unnamed method never leaves, so it is never audited either: there is
	// nothing to record.
	if rows := rawCalls(t, st); len(rows) != 0 {
		t.Fatalf("audit rows = %d, want 0", len(rows))
	}
}

// The card's reader: newest first, only the raw calls, and the switch's state.
func TestRawCDPAuditEndpoint(t *testing.T) {
	ts, _, st := browserServer(t)
	agentID := seedAgent(t, st, "raw")
	if err := st.SetSetting("browser.developerMode", "1"); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"Network.enable", "Storage.getCookies"} {
		if err := st.AppendEvent("browser.cdp", &agentID, nil, map[string]any{"method": method, "outcome": "allowed"}); err != nil {
			t.Fatal(err)
		}
	}
	// A different event type must not appear in the audit.
	other := seedAgent(t, st, "other")
	if err := st.AppendEvent("agent_started", &other, nil, nil); err != nil {
		t.Fatal(err)
	}
	res, err := http.Get(ts.URL + "/api/browser/developer/audit?limit=10")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	var out struct {
		DeveloperMode bool `json:"developerMode"`
		Calls         []struct {
			Method  string `json:"method"`
			Outcome string `json:"outcome"`
			AgentID string `json:"agentId"`
		} `json:"calls"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Calls) != 2 {
		t.Fatalf("calls = %+v, want the two raw ones", out.Calls)
	}
	if out.Calls[0].Method != "Storage.getCookies" {
		t.Fatalf("newest first expected, got %+v", out.Calls)
	}
	if out.Calls[1].AgentID != agentID {
		t.Fatalf("agent = %q, want %s", out.Calls[1].AgentID, agentID)
	}
	if !out.DeveloperMode {
		t.Fatal("the reader must report the switch it is describing")
	}
}

// Developer mode is fail-closed, unlike the other browser prefs.
func TestDeveloperModeDefaultsOff(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/picode.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	prefs, err := browserPrefsRead(st)
	if err != nil {
		t.Fatal(err)
	}
	if prefs.DeveloperMode {
		t.Fatal("developer mode defaults on")
	}
	if browserDeveloperMode(nil) || browserDeveloperMode(st) {
		t.Fatal("missing setting must read as off")
	}
	if err := st.SetSetting("browser.developerMode", "1"); err != nil {
		t.Fatal(err)
	}
	prefs, _ = browserPrefsRead(st)
	if !prefs.DeveloperMode || !browserDeveloperMode(st) {
		t.Fatal("setting on did not read back")
	}
}
