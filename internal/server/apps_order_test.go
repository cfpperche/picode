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

func TestAppsGridOrder(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st}).Handler)
	t.Cleanup(ts.Close)

	res, err := ts.Client().Get(ts.URL + "/api/apps/order")
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 || len(got.IDs) != 0 {
		t.Fatalf("empty order = %d %#v", res.StatusCode, got.IDs)
	}

	raw, _ := json.Marshal(map[string]any{"ids": []string{"tmux", "inbox"}})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/apps/order", bytes.NewReader(raw))
	res, err = ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("put = %d", res.StatusCode)
	}
	ids, err := readAppsOrder(st)
	if err != nil || len(ids) != 2 || ids[0] != "tmux" || ids[1] != "inbox" {
		t.Fatalf("stored = %#v %v", ids, err)
	}

	raw, _ = json.Marshal(map[string]any{"ids": []string{"inbox", "inbox"}})
	req, _ = http.NewRequest(http.MethodPut, ts.URL+"/api/apps/order", bytes.NewReader(raw))
	res, err = ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("duplicate = %d", res.StatusCode)
	}
}
