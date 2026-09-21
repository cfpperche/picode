package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestReorderWorkspacesHTTP(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	b, err := st.AddWorkspace("B", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, err := st.AddWorkspace("A", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st}).Handler)
	t.Cleanup(ts.Close)

	put := func(body any) *http.Response {
		t.Helper()
		raw, _ := json.Marshal(body)
		req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/workspaces/order", bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		res, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	res := put(map[string]any{"ids": []string{a.ID, b.ID}})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("reorder status = %d", res.StatusCode)
	}
	got := listWorkspaceIDs(t, ts)
	if len(got) != 2 || got[0] != a.ID || got[1] != b.ID {
		t.Fatalf("list = %v", got)
	}

	res = put(map[string]any{"ids": []string{a.ID}})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("partial list status = %d", res.StatusCode)
	}
	if after := listWorkspaceIDs(t, ts); after[0] != a.ID || after[1] != b.ID {
		t.Fatalf("rejected reorder changed the list to %v", after)
	}

	res = put(map[string]any{"nope": true})
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d", res.StatusCode)
	}
}

func listWorkspaceIDs(t *testing.T, ts *httptest.Server) []string {
	t.Helper()
	res, err := ts.Client().Get(ts.URL + "/api/workspaces")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var views []struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&views); err != nil {
		t.Fatal(err)
	}
	out := make([]string, len(views))
	for i, v := range views {
		out[i] = v.ID
	}
	return out
}
