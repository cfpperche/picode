package server

// The download endpoints: the start report inserts, the outcome lands on it,
// the list reads back, and a delete forgets the row.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestBrowserDownloadsEndpoints(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptestNewServerForHistoryTest(t, st)

	post := func(path string, body map[string]any) int {
		t.Helper()
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		return res.StatusCode
	}

	if code := post("/api/browser/downloads", map[string]any{"url": "https://files.example/a.zip", "path": `C:\dl\a.zip`, "total": 7}); code != http.StatusOK {
		t.Fatalf("the start report must be accepted, got %d", code)
	}
	if code := post("/api/browser/downloads/status", map[string]any{"path": `C:\dl\a.zip`, "status": "completed", "received": 7}); code != http.StatusNoContent {
		t.Fatalf("the outcome must be accepted, got %d", code)
	}
	if code := post("/api/browser/downloads/status", map[string]any{"path": `C:\gone\b.zip`, "status": "completed", "received": 1}); code != http.StatusNoContent {
		t.Fatalf("an outcome for an unknown path must be accepted and ignored, got %d", code)
	}
	if code := post("/api/browser/downloads/status", map[string]any{"path": `C:\dl\a.zip`, "status": "melted"}); code != http.StatusBadRequest {
		t.Fatalf("an unknown outcome word must be refused, got %d", code)
	}

	res, err := http.Get(ts.URL + "/api/browser/downloads")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body struct {
		Downloads []store.BrowserDownload `json:"downloads"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Downloads) != 1 || body.Downloads[0].Status != "completed" {
		t.Fatalf("the list must carry the finished download, got %+v", body.Downloads)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/browser/downloads/"+strconv.FormatInt(body.Downloads[0].ID, 10), nil)
	if res, err := http.DefaultClient.Do(req); err != nil || res.StatusCode != http.StatusNoContent {
		t.Fatalf("deleting the row must be a 204, got %v %v", res, err)
	}
}
