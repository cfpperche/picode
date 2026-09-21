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

// An identified caller drives its session split without a stored grant
// (ADR-0172). A caller with no identity cannot.
func TestBrowserToolSessionDriveDoesNotNeedAGrant(t *testing.T) {
	ts, _, _ := browserServer(t)
	code, out := postTool(t, ts, `{"verb":"evaluate","params":{"expression":"1+1"}}`)
	if code != http.StatusForbidden || !strings.Contains(out, "no identity") {
		t.Fatalf("no identity = %d %s", code, out)
	}

	body, closeStream := openBrowserStream(t, ts)
	defer closeStream()
	readFrames(t, body, 1, 3*time.Second) // hello
	go func() {
		_, _ = postTool(t, ts, `{"agent":"agent-2","verb":"evaluate","params":{"expression":"1+1"}}`)
	}()
	cmd := readFrames(t, body, 1, 3*time.Second)[0]
	var pushed browser.Command
	if err := json.Unmarshal([]byte(cmd.Data), &pushed); err != nil {
		t.Fatal(err)
	}
	if !pushed.Session || pushed.Tier != "act" || pushed.Principal != "agent-2" || pushed.Method != "Runtime.evaluate" {
		t.Fatalf("command = %+v", pushed)
	}
	if code := postBrowserResult(t, ts, `{"id":"`+pushed.ID+`","output":{"result":{"value":2}}}`); code != http.StatusNoContent {
		t.Fatalf("result status = %d", code)
	}
}

// A stored domain list does not confine the session tab. file: still does.
// A navigate without a url, and params that are not an object, still fail
// before a command leaves.
func TestBrowserToolSessionNavigateIsNotDomainGated(t *testing.T) {
	ts, hub, st := browserServer(t)
	if err := browser.Save(st, "agent-3", browser.Policy{Tier: "act", Domains: []string{"example.com"}}); err != nil {
		t.Fatal(err)
	}

	code, out := postTool(t, ts, `{"agent":"agent-3","verb":"navigate","params":{"url":"file:///etc/passwd"}}`)
	if code != http.StatusForbidden || !strings.Contains(out, "file:///etc/passwd") {
		t.Fatalf("file url = %d %s", code, out)
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
		_, _ = postTool(t, ts, `{"agent":"agent-3","verb":"navigate","params":{"url":"https://evil.test/login"}}`)
	}()
	cmd := readFrames(t, body, 1, 3*time.Second)[0]
	var pushed browser.Command
	if err := json.Unmarshal([]byte(cmd.Data), &pushed); err != nil {
		t.Fatal(err)
	}
	if pushed.Method != "Page.navigate" || pushed.Tier != "act" || !pushed.Session {
		t.Fatalf("command = %+v", pushed)
	}
	if !strings.Contains(string(pushed.Params), "https://evil.test/login") {
		t.Fatalf("params = %s", pushed.Params)
	}
	if len(pushed.Domains) != 1 || pushed.Domains[0] != "*" {
		t.Fatalf("domains = %v, want * — a stored list must not confine the session tab", pushed.Domains)
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
