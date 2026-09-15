package server

// The history dialog's bulk remove: one request, the named visits go, and
// a selection with nothing real in it is refused.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"net/http/httptest"

	"github.com/cfpperche/picode/internal/store"
)

func httptestNewServerForHistoryTest(t *testing.T, st *store.Store) *httptest.Server {
	t.Helper()
	return httptest.NewServer(New("127.0.0.1:0", Deps{Store: st}).Handler)
}

func TestBrowserHistoryDeleteMany(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptestNewServerForHistoryTest(t, st)

	a, err := st.AddBrowserVisit("https://a.example/1", "A", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddBrowserVisit("https://b.example/2", "B", false); err != nil {
		t.Fatal(err)
	}

	post := func(body map[string]any) (int, map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/browser/history/delete", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}

	if code, _ := post(map[string]any{"ids": []int64{}}); code != http.StatusBadRequest {
		t.Fatalf("an empty selection must be a 400, got %d", code)
	}
	code, body := post(map[string]any{"ids": []int64{a.ID}})
	if code != http.StatusOK || body["removed"] != float64(1) {
		t.Fatalf("one id must remove one visit, got %d %v", code, body)
	}
	visits, err := st.ListBrowserHistory(50, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(visits) != 1 || visits[0].Host != "b.example" {
		t.Fatalf("only b.example must survive, got %+v", visits)
	}
}
