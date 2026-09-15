package server

// The site-permission endpoints: the shell's report inserts, the dialog's
// edit changes it, the list reads back, a per-kind clear empties one kind.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestBrowserPermissionsEndpoints(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptestNewServerForHistoryTest(t, st)

	post := func(path string, body map[string]any) (int, map[string]any) {
		t.Helper()
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, ts.URL+path, bytes.NewReader(raw))
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

	if code, _ := post("/api/browser/permissions", map[string]any{"origin": "meet.example.com", "kind": "camera", "decision": "allow"}); code != http.StatusOK {
		t.Fatalf("a decision must be accepted, got %d", code)
	}
	if code, _ := post("/api/browser/permissions", map[string]any{"origin": "meet.example.com", "kind": "camera", "decision": "sometimes"}); code != http.StatusBadRequest {
		t.Fatalf("an unknown decision must be refused, got %d", code)
	}
	if code, _ := post("/api/browser/permissions", map[string]any{"origin": "meet.example.com", "kind": "telepathy", "decision": "allow"}); code != http.StatusBadRequest {
		t.Fatalf("an unknown kind must be refused, got %d", code)
	}
	if code, _ := post("/api/browser/permissions", map[string]any{"origin": "meet.example.com", "kind": "microphone", "decision": "deny"}); code != http.StatusOK {
		t.Fatalf("a second kind must be accepted, got %d", code)
	}

	res, err := http.Get(ts.URL + "/api/browser/permissions?kind=camera")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var body struct {
		Permissions []store.BrowserPermission `json:"permissions"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Permissions) != 1 || body.Permissions[0].Kind != "camera" || body.Permissions[0].Decision != "allow" {
		t.Fatalf("the list must carry the camera standing only, got %+v", body.Permissions)
	}

	if code, _ := post("/api/browser/permissions/clear", map[string]any{"kind": "camera"}); code != http.StatusNoContent {
		t.Fatalf("a per-kind clear must be a 204, got %d", code)
	}
	left, err := st.ListBrowserPermissions(10, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 || left[0].Kind != "microphone" {
		t.Fatalf("only the camera standing must go, got %+v", left)
	}
}
