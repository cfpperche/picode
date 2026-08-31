package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/apps"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// newAppsServer builds a server whose registry holds the demo app
// directly — no env var, per the plan (env is cmd/picode's business).
func newAppsServer(t *testing.T, reg *apps.Registry) *httptest.Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), AgentCmd: "cat", Apps: reg}).Handler)
	t.Cleanup(ts.Close)
	return ts
}

func getJSON(t *testing.T, ts *httptest.Server, path string, out any) int {
	t.Helper()
	res, err := http.Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer res.Body.Close()
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			t.Fatalf("GET %s decode: %v", path, err)
		}
	}
	return res.StatusCode
}

func TestListAppsWithBadge(t *testing.T) {
	ts := newAppsServer(t, apps.NewRegistry(apps.BuiltIns(true)...))
	var body struct {
		APIVersion int `json:"apiVersion"`
		Apps       []struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Icon       string `json:"icon"`
			APIVersion int    `json:"apiVersion"`
			Badge      struct {
				Count int  `json:"count"`
				Dot   bool `json:"dot"`
			} `json:"badge"`
		} `json:"apps"`
	}
	if code := getJSON(t, ts, "/api/apps", &body); code != http.StatusOK {
		t.Fatalf("GET /api/apps = %d", code)
	}
	if body.APIVersion != apps.APIVersion || len(body.Apps) != 1 {
		t.Fatalf("list = %+v", body)
	}
	a := body.Apps[0]
	if a.ID != "demo" || a.Icon == "" || a.APIVersion != apps.APIVersion || a.Badge.Count != 3 {
		t.Fatalf("demo row = %+v", a)
	}
}

func TestListAppsEmptyRegistry(t *testing.T) {
	ts := newAppsServer(t, nil) // nil registry must serve, not panic
	var body struct {
		Apps []any `json:"apps"`
	}
	if code := getJSON(t, ts, "/api/apps", &body); code != http.StatusOK {
		t.Fatalf("GET /api/apps = %d", code)
	}
	if body.Apps == nil || len(body.Apps) != 0 {
		t.Fatalf("apps = %v, want []", body.Apps)
	}
}

func TestAppView(t *testing.T) {
	ts := newAppsServer(t, apps.NewRegistry(apps.BuiltIns(true)...))

	var v apps.View
	if code := getJSON(t, ts, "/api/apps/demo/view", &v); code != http.StatusOK {
		t.Fatalf("view = %d", code)
	}
	if v.APIVersion != apps.APIVersion || len(v.Blocks) == 0 {
		t.Fatalf("root view = %+v", v)
	}
	if code := getJSON(t, ts, "/api/apps/demo/view?path=item%2F1", &v); code != http.StatusOK {
		t.Fatalf("item view = %d", code)
	}
	if !strings.Contains(v.Title, "Item 1") {
		t.Fatalf("item view title = %q", v.Title)
	}
	if code := getJSON(t, ts, "/api/apps/nope/view", nil); code != http.StatusNotFound {
		t.Fatalf("unknown app view = %d, want 404", code)
	}
	if code := getJSON(t, ts, "/api/apps/demo/view?path=nope", nil); code != http.StatusInternalServerError {
		t.Fatalf("bad path = %d, want 500", code)
	}
}

func TestAppAction(t *testing.T) {
	ts := newAppsServer(t, apps.NewRegistry(apps.BuiltIns(true)...))
	post := func(path, body string) (*http.Response, error) {
		return http.Post(ts.URL+path, "application/json", bytes.NewBufferString(body))
	}

	res, err := post("/api/apps/demo/action", `{"action":"toast","args":{"item":"1"}}`)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	var out apps.ActionResult
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK || out.Toast == "" {
		t.Fatalf("toast action = %d %+v", res.StatusCode, out)
	}

	res, _ = post("/api/apps/demo/action", `{"action":"reset","args":{"item":"2"}}`)
	out = apps.ActionResult{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || out.View == nil {
		t.Fatalf("reset action = %d %+v (want view)", res.StatusCode, out)
	}

	res, _ = post("/api/apps/nope/action", `{"action":"x"}`)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown app action = %d, want 404", res.StatusCode)
	}
	res, _ = post("/api/apps/demo/action", `{`)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad JSON = %d, want 400", res.StatusCode)
	}
	res, _ = post("/api/apps/demo/action", `{"action":""}`)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty action = %d, want 400", res.StatusCode)
	}
	res, _ = post("/api/apps/demo/action", `{"action":"unknown-thing"}`)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown action = %d, want 400", res.StatusCode)
	}
}
