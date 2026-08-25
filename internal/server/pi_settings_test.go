package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func TestPiSettingsGlobalRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".pi", "agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	ts := newTestServer(t, "cat")

	res, err := ts.Client().Get(ts.URL + "/api/pi-settings")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET status %d", res.StatusCode)
	}
	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	g, _ := got["global"].(map[string]any)
	if g["compactionEnabled"] != true {
		t.Fatalf("default compact = %v", g["compactionEnabled"])
	}

	body, _ := json.Marshal(map[string]any{
		"layer": "global",
		"patch": map[string]any{"compactionEnabled": false, "steeringMode": "all"},
	})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/pi-settings", bytes.NewReader(body))
	put, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer put.Body.Close()
	if put.StatusCode != http.StatusOK {
		t.Fatalf("PUT status %d", put.StatusCode)
	}
	var saved map[string]any
	if err := json.NewDecoder(put.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	sg, _ := saved["global"].(map[string]any)
	if sg["compactionEnabled"] != false || sg["steeringMode"] != "all" {
		t.Fatalf("%v", sg)
	}

	raw, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["steeringMode"] != "all" {
		t.Fatalf("file = %s", raw)
	}
}

func TestPiSettingsProjectUntrusted(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := t.TempDir()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	_, agent, err := st.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]any{
		"agentId": agent.ID, "layer": "project",
		"patch": map[string]any{"compactionEnabled": false},
	})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/pi-settings", bytes.NewReader(body))
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestPiSettingsProjectWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".pi", "agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	trust, _ := json.Marshal(map[string]bool{proj: true})
	if err := os.WriteFile(filepath.Join(home, ".pi", "agent", "trust.json"), trust, 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	_, agent, err := st.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)
	body, _ := json.Marshal(map[string]any{
		"agentId": agent.ID, "layer": "project",
		"patch": map[string]any{"steeringMode": "all"},
	})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/pi-settings", bytes.NewReader(body))
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	raw, err := os.ReadFile(filepath.Join(proj, ".pi", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["steeringMode"] != "all" {
		t.Fatalf("%s", raw)
	}
}

func TestPiSettingsRejectsProjectWrite(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ts := newTestServer(t, "cat")
	body, _ := json.Marshal(map[string]any{"layer": "project", "patch": map[string]any{}})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/pi-settings", bytes.NewReader(body))
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
}
