package server

// ADR-0146: the history verb is its own gate, answered by the daemon. One row
// per condition that changes the outcome — the setting unset (never), allow,
// allow with a query, a limit above the cap, and the master switch off.

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBrowserHistoryVerbGate(t *testing.T) {
	ts, _, st := browserServer(t)
	if _, err := st.AddBrowserVisit("https://example.com/alpha", "Alpha", true); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddBrowserVisit("https://other.test/beta", "Beta", false); err != nil {
		t.Fatal(err)
	}
	call := func(params map[string]any) (int, string) {
		body, _ := json.Marshal(map[string]any{
			"agent": "", "term": "term-1", "verb": "history", "params": params,
		})
		res, err := http.Post(ts.URL+"/api/browser/tool", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(raw)
	}

	// Unset is never, and the refusal names where to change it.
	if code, text := call(map[string]any{}); code != http.StatusForbidden || !strings.Contains(text, "Agent history access") {
		t.Fatalf("unset: %d %s", code, text)
	}

	if err := st.SetSetting("browser.historyAccess", "allow"); err != nil {
		t.Fatal(err)
	}
	code, text := call(map[string]any{})
	if code != http.StatusOK || !strings.Contains(text, "example.com/alpha") || !strings.Contains(text, "other.test/beta") {
		t.Fatalf("allow: %d %s", code, text)
	}

	// A query narrows the answer to what matched, and a limit is honoured as a
	// row count — the newest first, so one row is the newest one.
	if _, text := call(map[string]any{"query": "alpha"}); strings.Contains(text, "other.test") {
		t.Fatalf("query leaked a row it should not: %s", text)
	}
	if _, text := call(map[string]any{"limit": 1}); strings.Count(text, "\"visitedAt\"") != 1 {
		t.Fatalf("limit 1 returned a different number of rows: %s", text)
	}
	if code, _ := call(map[string]any{"limit": 9999}); code != http.StatusOK {
		t.Fatalf("limit above the cap refused: %d", code)
	}

	// The master switch is checked first and says so.
	if err := st.SetSetting("browser.agentAccess", "0"); err != nil {
		t.Fatal(err)
	}
	if code, text := call(map[string]any{}); code != http.StatusForbidden || !strings.Contains(text, "turn it back on") {
		t.Fatalf("master switch off: %d %s", code, text)
	}
}
