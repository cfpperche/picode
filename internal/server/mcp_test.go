package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/rpc"
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
	var added map[string]any
	if err := json.NewDecoder(add.Body).Decode(&added); err != nil {
		t.Fatal(err)
	}
	_ = add.Body.Close()
	srvs, _ := added["servers"].([]any)
	if len(srvs) != 1 {
		t.Fatalf("servers = %v", added["servers"])
	}
	row, _ := srvs[0].(map[string]any)
	if row["live"] != "idle" {
		t.Fatalf("live = %v", row["live"])
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

func TestMCPUnknownAgentStillReportsAdapter(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ts := newTestServer(t, "cat")
	settings := filepath.Join(home, ".pi", "agent", "settings.json")

	rows := []struct {
		name      string
		install   bool
		agent     string
		workspace string
		installed bool
	}{
		{"no adapter, no agent", false, "", "", false},
		{"no adapter, terminal tab", false, "t:terminal-4-027c76", "", false},
		{"adapter, no agent", true, "", "", true},
		{"adapter, terminal tab", true, "t:terminal-4-027c76", "", true},
		{"adapter, stale agent", true, "gone-agent", "", true},
		{"adapter, terminal as workspace", true, "", "t:terminal-4-027c76", true},
		{"adapter, workspace+terminal agent", true, "t:terminal-4-027c76", "t:terminal-4-027c76", true},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			if row.install {
				if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(settings, []byte(`{"packages":["npm:pi-mcp-adapter"]}`), 0o644); err != nil {
					t.Fatal(err)
				}
			} else {
				_ = os.Remove(settings)
			}
			q := ts.URL + "/api/mcp"
			sep := "?"
			if row.agent != "" {
				q += sep + "agent=" + row.agent
				sep = "&"
			}
			if row.workspace != "" {
				q += sep + "workspace=" + row.workspace
			}
			res, err := ts.Client().Get(q)
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusOK {
				t.Fatalf("GET status %d", res.StatusCode)
			}
			var body map[string]any
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			_ = res.Body.Close()
			ad, _ := body["adapter"].(map[string]any)
			if ad["installed"] != row.installed {
				t.Fatalf("adapter.installed = %v, want %v", ad["installed"], row.installed)
			}
		})
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
	hit := postJSON(t, ts, "/api/mcp/import", map[string]any{
		"picks": []any{map[string]any{"kind": "cursor", "servers": []any{"ext"}}},
	})
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

func TestMCPAuthNeedsName(t *testing.T) {
	ts := newTestServer(t, "cat")
	res := postJSON(t, ts, "/api/mcp/auth", map[string]any{})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
	_ = res.Body.Close()
}

func TestMCPAuthLogoutNeedsName(t *testing.T) {
	ts := newTestServer(t, "cat")
	res := postJSON(t, ts, "/api/mcp/auth/logout", map[string]any{})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
	_ = res.Body.Close()
}

func TestMCPAuthShortPi(t *testing.T) {
	rpc.AuthTestInstant = true
	t.Cleanup(func() { rpc.AuthTestInstant = false })
	ts := bashTestServer(t)
	home := os.Getenv("HOME")
	dir := filepath.Join(home, ".pi", "agent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mcp.json"), []byte(`{"mcpServers":{"docs":{"url":"https://example.test/mcp","auth":"oauth"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	res := postJSON(t, ts, "/api/mcp/auth", map[string]any{"name": "docs"})
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status %d body %s", res.StatusCode, body)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	id, _ := body["id"].(string)
	if id == "" {
		t.Fatalf("body = %v", body)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		st, err := ts.Client().Get(ts.URL + "/api/mcp/auth/status?id=" + id)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.NewDecoder(st.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_ = st.Body.Close()
		if got["ok"] == true {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("status = %v", got)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestMCPAddSecrets(t *testing.T) {
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

	ok := postJSON(t, ts, "/api/mcp", map[string]any{
		"scope": "user", "name": "docs", "url": "https://mcp.example/mcp",
		"auth": "bearer", "bearerToken": "tok",
		"headers": map[string]string{"X-Trace": "1"},
	})
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("add bearer = %d", ok.StatusCode)
	}
	_ = ok.Body.Close()

	bad := postJSON(t, ts, "/api/mcp", map[string]any{
		"scope": "user", "name": "oops", "url": "https://mcp.example/mcp", "auth": "bearer",
	})
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("bearer without token = %d", bad.StatusCode)
	}
	_ = bad.Body.Close()

	env := postJSON(t, ts, "/api/mcp", map[string]any{
		"scope": "user", "name": "cli", "command": "npx", "args": []string{"-y", "x"},
		"env": map[string]string{"API_KEY": "sekrit"},
	})
	if env.StatusCode != http.StatusOK {
		t.Fatalf("add env = %d", env.StatusCode)
	}
	_ = env.Body.Close()

	mixed := postJSON(t, ts, "/api/mcp", map[string]any{
		"scope": "user", "name": "mix", "command": "npx", "auth": "oauth",
	})
	if mixed.StatusCode != http.StatusBadRequest {
		t.Fatalf("auth on command = %d", mixed.StatusCode)
	}
	_ = mixed.Body.Close()
}

func mcpPatch(t *testing.T, ts *httptest.Server, body any) *http.Response {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/mcp", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return do(t, ts.Client(), req)
}
