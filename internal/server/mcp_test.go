package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMCPMatrix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")

	res, err := ts.Client().Get(ts.URL + "/api/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET status %d", res.StatusCode)
	}
	var empty map[string]any
	if err := json.NewDecoder(res.Body).Decode(&empty); err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	ad, _ := empty["adapter"].(map[string]any)
	if ad["installed"] != false {
		t.Fatalf("adapter = %v", ad)
	}

	add := postJSON(t, ts, "/api/mcp", map[string]any{
		"scope": "user", "name": "deepwiki", "url": "https://mcp.deepwiki.com/mcp",
	})
	if add.StatusCode != http.StatusConflict {
		t.Fatalf("add without adapter = %d", add.StatusCode)
	}

	settings := filepath.Join(home, ".pi", "agent", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"packages":["npm:pi-mcp-adapter"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	add = postJSON(t, ts, "/api/mcp", map[string]any{
		"scope": "user", "name": "deepwiki", "url": "https://mcp.deepwiki.com/mcp",
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("add = %d", add.StatusCode)
	}

	bad := postJSON(t, ts, "/api/mcp", map[string]any{"scope": "user", "name": "x"})
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid add = %d", bad.StatusCode)
	}

	off := mcpPatch(t, ts, map[string]any{"scope": "user", "name": "deepwiki", "disabled": true})
	if off.StatusCode != http.StatusOK {
		t.Fatalf("toggle = %d", off.StatusCode)
	}

	del, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/mcp?scope=user&name=deepwiki", nil)
	if err != nil {
		t.Fatal(err)
	}
	rm := do(t, ts.Client(), del)
	if rm.StatusCode != http.StatusOK {
		t.Fatalf("remove = %d", rm.StatusCode)
	}
}

func TestMCPProjectAndAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")
	settings := filepath.Join(home, ".pi", "agent", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"packages":["npm:pi-mcp-adapter"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	wsDir := t.TempDir()
	res := postJSON(t, ts, "/api/workspaces", map[string]any{"name": "ws", "path": wsDir})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("workspace = %d", res.StatusCode)
	}
	var wk workspaceView
	if err := json.NewDecoder(res.Body).Decode(&wk); err != nil {
		t.Fatal(err)
	}

	add := postJSON(t, ts, "/api/mcp", map[string]any{
		"scope": "project", "workspaceId": wk.ID, "name": "folder", "url": "https://f.example/mcp",
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("project add = %d", add.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(wsDir, ".mcp.json")); err != nil {
		t.Fatalf("project file: %v", err)
	}

	agentDir := t.TempDir()
	ares := postJSON(t, ts, "/api/agents", map[string]any{"name": "solo", "path": agentDir})
	if ares.StatusCode != http.StatusCreated {
		t.Fatalf("free agent = %d", ares.StatusCode)
	}
	var agent agentView
	if err := json.NewDecoder(ares.Body).Decode(&agent); err != nil {
		t.Fatal(err)
	}
	add = postJSON(t, ts, "/api/mcp", map[string]any{
		"scope": "agent", "agentId": agent.ID, "name": "only-me", "command": "npx", "args": []string{"-y", "x"},
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("agent add = %d", add.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(agentDir, ".pi", "mcp.json")); err != nil {
		t.Fatalf("agent file: %v", err)
	}
}

func TestMCPImport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")

	no := postJSON(t, ts, "/api/mcp/import", map[string]any{})
	if no.StatusCode != http.StatusConflict {
		t.Fatalf("no adapter = %d", no.StatusCode)
	}

	settings := filepath.Join(home, ".pi", "agent", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"packages":["npm:pi-mcp-adapter"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	missing := postJSON(t, ts, "/api/mcp/import", map[string]any{})
	if missing.StatusCode != http.StatusBadRequest {
		t.Fatalf("no kinds = %d", missing.StatusCode)
	}

	if err := os.MkdirAll(filepath.Join(home, ".cursor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".cursor", "mcp.json"), []byte(`{"mcpServers":{"ext":{"url":"https://c.example/mcp"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	hit := postJSON(t, ts, "/api/mcp/import", map[string]any{"kinds": []string{"cursor"}})
	if hit.StatusCode != http.StatusOK {
		t.Fatalf("import = %d", hit.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(hit.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	imp, _ := body["import"].(map[string]any)
	added, _ := imp["added"].([]any)
	if len(added) != 1 || added[0] != "cursor" {
		t.Fatalf("import body = %v", body["import"])
	}
}

func mcpPatch(t *testing.T, ts *httptest.Server, body any) *http.Response {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/mcp", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return do(t, ts.Client(), req)
}
