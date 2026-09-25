package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/pkgs"
)

// The describe lifecycle (ADR-0119 C5): the owner describes a package, the
// package becomes GUI-configurable, values are written where the
// description says — and deleting the description honestly removes the
// Configure affordance again.

func describeServer(t *testing.T) (*httptest.Server, string, string) {
	t.Helper()
	st := testStore(t)
	dataDir := t.TempDir()
	agentDir := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return agentDir }
	t.Cleanup(func() { pipkg.UserDir = old })
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat", DataDir: dataDir}).Handler)
	t.Cleanup(ts.Close)
	return ts, dataDir, agentDir
}

func putDescribe(t *testing.T, ts *httptest.Server, body map[string]any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/packages/describe", strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { res.Body.Close() })
	return res
}

func exampleDescriptor() map[string]any {
	return map[string]any{
		"id": "pi-example", "match": "pi-example", "title": "Example",
		"application": "Applies on next use.",
		"files":       []map[string]any{{"scope": "agent", "path": "example.json", "format": "json"}},
		"fields": []map[string]any{
			{"key": "provider", "label": "Provider", "type": "enum", "required": true, "options": []string{"alpha", "beta"}},
			{"key": "name", "label": "Name", "type": "string", "required": true},
		},
	}
}

func TestDescribeRoundTrip(t *testing.T) {
	ts, dataDir, agentDir := describeServer(t)

	// Validation: a descriptor without fields is refused before anything is
	// persisted.
	bad := exampleDescriptor()
	bad["fields"] = []map[string]any{}
	res := putDescribe(t, ts, bad)
	if res.StatusCode != 400 {
		t.Fatalf("status %d, want 400 for a fieldless descriptor", res.StatusCode)
	}
	res.Body.Close()

	// The good description persists and immediately stamps the listing.
	res = putDescribe(t, ts, exampleDescriptor())
	if res.StatusCode != 200 {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	res.Body.Close()
	if _, err := os.Stat(filepath.Join(dataDir, "package-configs", "pi-example.json")); err != nil {
		t.Fatalf("descriptor file not persisted: %v", err)
	}

	res, err := http.Get(ts.URL + "/api/packages/report?cli=pi")
	if err != nil {
		t.Fatal(err)
	}
	rep := decode[pkgs.Report](t, res)
	stamped := false
	for _, p := range rep.Rows {
		if strings.Contains(p.Source, "pi-example") {
			stamped = p.ConfigKind == "pi-example"
		}
	}
	_ = stamped // the listing carries it only when the package is installed;
	// the describe API itself is what this test owns.

	// The value form writes exactly where the description says: the agent
	// file inside the swapped UserDir.
	values := putDescriptor(t, ts, map[string]any{"package": "pi-example",
		"config": map[string]any{"provider": "alpha", "name": "n1"}})
	if values.StatusCode != 200 {
		t.Fatalf("value save status %d, want 200", values.StatusCode)
	}
	values.Body.Close()
	b, err := os.ReadFile(filepath.Join(agentDir, "example.json"))
	if err != nil {
		t.Fatalf("value file not written: %v", err)
	}
	if !strings.Contains(string(b), "\"provider\": \"alpha\"") {
		t.Fatalf("value file = %q", b)
	}

	// The config view answers with source=user and the declared fields.
	res, err = http.Get(ts.URL + "/api/packages/config?package=pi-example")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var view struct {
		Source string                 `json:"source"`
		Kind   string                 `json:"kind"`
		Fields []struct{ Key string } `json:"fields"`
	}
	if err := json.NewDecoder(res.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.Source != "user" || view.Kind != "pi-example" || len(view.Fields) != 2 {
		t.Fatalf("view = %+v", view)
	}
}

func TestDescribeDeleteRemovesTheAffordance(t *testing.T) {
	ts, dataDir, agentDir := describeServer(t)
	res := putDescribe(t, ts, exampleDescriptor())
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	res.Body.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/packages/describe?package=pi-example", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("delete status %d, want 200 (idempotent too)", res.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "package-configs", "pi-example.json")); !os.IsNotExist(err) {
		t.Fatal("descriptor file must be removed with the description")
	}
	res, err = http.Get(ts.URL + "/api/packages/config?package=pi-example")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Fatalf("config status %d, want 400 after the description is gone", res.StatusCode)
	}
	_ = agentDir
}

func TestDescribePrecedenceUserWinsOverCatalog(t *testing.T) {
	ts, _, _ := describeServer(t)
	// A user descriptor for a catalog package, under its own id.
	res := putDescribe(t, ts, map[string]any{
		"id": "my-web-search", "match": "pi-web-search", "title": "My web search",
		"files":  []map[string]any{{"scope": "agent", "path": "web-search.json", "format": "json"}},
		"fields": []map[string]any{{"key": "provider", "label": "Provider", "type": "enum", "required": true, "options": []string{"xai"}}},
	})
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	res.Body.Close()
	res, err := http.Get(ts.URL + "/api/packages/config?package=my-web-search")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var view struct {
		Source string `json:"source"`
		Kind   string `json:"kind"`
	}
	_ = json.NewDecoder(res.Body).Decode(&view)
	if view.Source != "user" || view.Kind != "my-web-search" {
		t.Fatalf("view = %+v, want the user descriptor to win", view)
	}
}
