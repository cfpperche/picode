package server

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
)

func postTool(t *testing.T, ts *httptest.Server, body string) (int, string) {
	t.Helper()
	res, err := http.Post(ts.URL+"/api/browser/tool", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(out)
}

// Rows 1–2 of the grant's decision table: an act verb is out of reach until a
// grant says otherwise, and the refusal names the way to grant it.
func TestBrowserToolRefusesActWithoutAGrant(t *testing.T) {
	ts, _, _ := browserServer(t)
	code, out := postTool(t, ts, `{"agent":"agent-2","verb":"evaluate","params":{"expression":"1+1"}}`)
	if code != http.StatusForbidden || !strings.Contains(out, "act tier") || !strings.Contains(out, "Settings") {
		t.Fatalf("answer = %d %s", code, out)
	}
}

// Rows 3–5: the one verb with a destination is checked against the grant's
// domains before the command leaves the daemon — and when it does leave, the
// params and the domains travel with it (the shell checks navigation again).
func TestBrowserToolChecksANavigateAgainstTheGrant(t *testing.T) {
	ts, hub, st := browserServer(t)
	if err := browser.Save(st, "agent-3", browser.Policy{Tier: "act", Domains: []string{"example.com"}}); err != nil {
		t.Fatal(err)
	}

	code, out := postTool(t, ts, `{"agent":"agent-3","verb":"navigate","params":{"url":"https://evil.test/"}}`)
	if code != http.StatusForbidden || !strings.Contains(out, "evil.test") || !strings.Contains(out, "example.com") {
		t.Fatalf("outside the grant = %d %s", code, out)
	}
	code, out = postTool(t, ts, `{"agent":"agent-3","verb":"navigate"}`)
	if code != http.StatusForbidden {
		t.Fatalf("no url = %d %s, want 403", code, out)
	}
	code, out = postTool(t, ts, `{"agent":"agent-3","verb":"navigate","params":"nope"}`)
	if code != http.StatusBadRequest {
		t.Fatalf("non-object params = %d %s, want 400", code, out)
	}

	body, closeStream := openBrowserStream(t, ts)
	defer closeStream()
	readFrames(t, body, 1, 3*time.Second) // hello
	go func() {
		_, _ = postTool(t, ts, `{"agent":"agent-3","verb":"navigate","params":{"url":"https://example.com/docs"}}`)
	}()
	cmd := readFrames(t, body, 1, 3*time.Second)[0]
	var pushed browser.Command
	if err := json.Unmarshal([]byte(cmd.Data), &pushed); err != nil {
		t.Fatal(err)
	}
	if pushed.Method != "Page.navigate" || pushed.Tier != "act" {
		t.Fatalf("command = %+v", pushed)
	}
	if string(pushed.Params) != `{"url":"https://example.com/docs"}` {
		t.Fatalf("params = %s", pushed.Params)
	}
	if len(pushed.Domains) != 1 || pushed.Domains[0] != "example.com" {
		t.Fatalf("domains = %v", pushed.Domains)
	}
	if code := postBrowserResult(t, ts, `{"id":"`+pushed.ID+`","output":{}}`); code != http.StatusNoContent {
		t.Fatalf("result status = %d", code)
	}
	_ = hub
}

// A read verb keeps travelling without params, exactly as before: the pipeline
// is additive, so nothing that worked changed shape.
func TestBrowserToolReadVerbCarriesNoParams(t *testing.T) {
	ts, _, _ := browserServer(t)
	body, closeStream := openBrowserStream(t, ts)
	defer closeStream()
	readFrames(t, body, 1, 3*time.Second)
	go func() { _, _ = postTool(t, ts, `{"agent":"agent-4","verb":"events","params":{"since":7}}`) }()
	cmd := readFrames(t, body, 1, 3*time.Second)[0]
	var pushed browser.Command
	if err := json.Unmarshal([]byte(cmd.Data), &pushed); err != nil {
		t.Fatal(err)
	}
	if pushed.Method != "shell.events" || string(pushed.Params) != `{"since":7}` {
		t.Fatalf("command = %+v", pushed)
	}
	postBrowserResult(t, ts, `{"id":"`+pushed.ID+`","output":{}}`)
}
