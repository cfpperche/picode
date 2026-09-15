package server

// The browser history endpoints, table-tested: add → list, a missing
// delete is a 404, clear empties the table.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func historyServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return httptest.NewServer(New("127.0.0.1:0", Deps{Store: st}).Handler)
}

func TestBrowserHistoryEndpointsRoundTrip(t *testing.T) {
	ts := historyServer(t)
	add, _ := json.Marshal(map[string]any{"url": "https://example.com/page", "title": "Example", "typed": true})
	resp, err := http.Post(ts.URL+"/api/browser/history", "application/json", bytes.NewReader(add))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("add status %d", resp.StatusCode)
	}
	resp.Body.Close()

	list, err := http.Get(ts.URL + "/api/browser/history?limit=10")
	if err != nil {
		t.Fatal(err)
	}
	defer list.Body.Close()
	var body struct {
		Visits []struct {
			URL   string `json:"url"`
			Typed bool   `json:"typed"`
		} `json:"visits"`
	}
	if err := json.NewDecoder(list.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Visits) != 1 || body.Visits[0].URL != "https://example.com/page" || !body.Visits[0].Typed {
		t.Fatalf("list did not round trip: %+v", body.Visits)
	}

	bad, _ := http.Post(ts.URL+"/api/browser/history", "application/json", bytes.NewReader([]byte(`{"url":""}`)))
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("an empty URL must be a 400, got %d", bad.StatusCode)
	}
	bad.Body.Close()

	miss, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/browser/history/999", nil)
	missResp, err := http.DefaultClient.Do(miss)
	if err != nil {
		t.Fatal(err)
	}
	missResp.Body.Close()
	if missResp.StatusCode != http.StatusNotFound {
		t.Fatalf("a missing delete must be a 404, got %d", missResp.StatusCode)
	}

	clear, _ := http.Post(ts.URL+"/api/browser/history/clear", "application/json", nil)
	clear.Body.Close()
	after := listPoliciesHistory(t, ts)
	if len(after) != 0 {
		t.Fatalf("clear must empty the history, got %+v", after)
	}
}

func listPoliciesHistory(t *testing.T, ts *httptest.Server) []any {
	t.Helper()
	resp, err := http.Get(ts.URL + "/api/browser/history")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Visits []any `json:"visits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Visits
}
