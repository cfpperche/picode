package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/mcp"
	"github.com/cfpperche/picode/internal/rpc"
)

func TestMCPCanvas(t *testing.T) {
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

// ADR-0157: the definition-import endpoint is gone. Routes() is the
// server's own registry, so this holds regardless of how the fallback
// answers the unmatched path: the unbuilt-UI handler serves 503 and the
// built UI's file server 405 — neither is the old API contract.
func TestMCPImportGone(t *testing.T) {
	for _, r := range Routes() {
		if r.Method == http.MethodPost && r.Pattern == "/api/mcp/import" {
			t.Fatal("POST /api/mcp/import is still registered")
		}
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

// ADR-0150: a request naming a CLI without a driver must fail loudly
// instead of writing Pi's files. claude-code, codex, omp, agy, opencode and
// grok ship drivers; every other CLI agent still has none.
func TestMCPRejectsCLIWithoutDriver(t *testing.T) {
	ts := newTestServer(t, "cat")

	get, err := ts.Client().Get(ts.URL + "/api/mcp?cli=cursor")
	if err != nil {
		t.Fatal(err)
	}
	if get.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET cli=cursor status %d", get.StatusCode)
	}
	_ = get.Body.Close()

	add := postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "cursor", "scope": "user", "name": "deepwiki", "url": "https://mcp.deepwiki.com/mcp",
	})
	if add.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST cli=cursor status %d", add.StatusCode)
	}
	_ = add.Body.Close()

	del, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/mcp?scope=user&name=deepwiki&cli=cursor", nil)
	if err != nil {
		t.Fatal(err)
	}
	rm := do(t, ts.Client(), del)
	if rm.StatusCode != http.StatusBadRequest {
		t.Fatalf("DELETE cli=cursor status %d", rm.StatusCode)
	}
	_ = rm.Body.Close()

	// An empty cli means Pi: the pre-existing contract is unchanged.
	empty, err := ts.Client().Get(ts.URL + "/api/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if empty.StatusCode != http.StatusOK {
		t.Fatalf("GET without cli status %d", empty.StatusCode)
	}
	_ = empty.Body.Close()

	pi, err := ts.Client().Get(ts.URL + "/api/mcp?cli=pi")
	if err != nil {
		t.Fatal(err)
	}
	if pi.StatusCode != http.StatusOK {
		t.Fatalf("GET cli=pi status %d", pi.StatusCode)
	}
	// Pi has the packages: the PiCode tool cards (ADR-0154) are not in its catalog.
	var piRep struct {
		Presets []struct {
			ID string `json:"id"`
		} `json:"presets"`
	}
	if err := json.NewDecoder(pi.Body).Decode(&piRep); err != nil {
		t.Fatal(err)
	}
	_ = pi.Body.Close()
	for _, p := range piRep.Presets {
		if mcp.IsToolPreset(p.ID) {
			t.Fatalf("pi catalog carries %s", p.ID)
		}
	}
	if len(piRep.Presets) == 0 {
		t.Fatal("pi catalog is empty")
	}
}

// writeFakeCLI plants a shell script that logs argv and answers `mcp list`
// from a file — the fake-binary pattern (term_wiring_test.go). Tests must
// never depend on a real claude/codex install.
func writeFakeCLI(t *testing.T, name, listOut string) string {
	t.Helper()
	dir := t.TempDir()
	fixture := filepath.Join(dir, "list.out")
	if err := os.WriteFile(fixture, []byte(listOut), 0o644); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nPATH=\"$PATH:/usr/bin:/bin\"\n" +
		"if [ \"$1\" = mcp ] && [ \"$2\" = list ]; then cat \"" + fixture + "\"; exit 0; fi\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	// Prepend, so several fakes (claude + codex) coexist on one PATH.
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

// ADR-0150 phase 1: ?cli=claude-code|codex answers with the same Report shape as
// Pi's /api/mcp, CLI agent auth refuses with instructions, and CLI agent
// mutations land in the CLI's native config.
func TestMCPCliAgentDriversDispatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	writeFakeCLI(t, "claude", `{"mcpServers":{"relay":{"type":"http","url":"https://relay.example/mcp"}}}`)
	writeFakeCLI(t, "codex", "")
	ts := newTestServer(t, "cat")

	// GET ?cli=claude-code: Report shape with the CLI adapter source.
	res, err := ts.Client().Get(ts.URL + "/api/mcp?cli=claude-code")
	if err != nil {
		t.Fatal(err)
	}
	var rep struct {
		Adapter struct {
			Installed bool   `json:"installed"`
			Source    string `json:"source"`
		} `json:"adapter"`
		Servers []struct {
			Name  string `json:"name"`
			Scope string `json:"scope"`
			Live  string `json:"live"`
		} `json:"servers"`
		Presets []any             `json:"presets"`
		Layers  []any             `json:"layers"`
		Packs   []any             `json:"connectorPackages"`
		Envs    map[string]string `json:"-"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK || rep.Adapter.Source != "native:claude" || !rep.Adapter.Installed {
		t.Fatalf("GET cli=claude-code = %d %+v", res.StatusCode, rep.Adapter)
	}
	if len(rep.Servers) != 1 || rep.Servers[0].Name != "relay" || rep.Servers[0].Scope != "user" || rep.Servers[0].Live != "" {
		t.Fatalf("servers = %+v", rep.Servers)
	}
	if len(rep.Presets) == 0 || len(rep.Packs) != 0 {
		t.Fatalf("report shape = presets %d packs %v", len(rep.Presets), rep.Packs)
	}

	// CLI agent add (project scope, no binary needed) writes the native file and
	// answers with the full Report shape.
	wsDir := t.TempDir()
	wres := postJSON(t, ts, "/api/workspaces", map[string]any{"name": "ws", "path": wsDir})
	if wres.StatusCode != http.StatusCreated {
		t.Fatalf("workspace = %d", wres.StatusCode)
	}
	var wk workspaceView
	if err := json.NewDecoder(wres.Body).Decode(&wk); err != nil {
		t.Fatal(err)
	}
	add := postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "claude-code", "scope": "project", "workspaceId": wk.ID, "name": "docs", "url": "https://docs.example/mcp",
	})
	if add.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(add.Body)
		t.Fatalf("CLI agent add = %d body %s", add.StatusCode, body)
	}
	var added map[string]any
	if err := json.NewDecoder(add.Body).Decode(&added); err != nil {
		t.Fatal(err)
	}
	_ = add.Body.Close()
	if added["adapter"] == nil || added["servers"] == nil || added["presets"] == nil {
		t.Fatalf("mutation response is not a Report: %v", added)
	}
	raw, err := os.ReadFile(filepath.Join(wsDir, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"https://docs.example/mcp"`) {
		t.Fatalf("project file: %s", raw)
	}

	// Codex keeps one config file: project scope refuses, and the workspace
	// stays untouched (the vendor never loads a per-workspace file).
	add = postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "codex", "scope": "project", "workspaceId": wk.ID, "name": "docs", "command": "npx", "args": []string{"-y", "x"},
	})
	if add.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(add), "keeps one config file") {
		t.Fatalf("codex project add = %d %s", add.StatusCode, thisBody(add))
	}
	add.Body.Close()
	if _, err := os.Stat(filepath.Join(wsDir, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("codex project add touched the workspace: %v", err)
	}

	// The user-scope path still writes its TOML table.
	add = postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "codex", "scope": "user", "name": "docs", "command": "npx", "args": []string{"-y", "x"},
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("codex add = %d %s", add.StatusCode, thisBody(add))
	}
	add.Body.Close()
	tomlRaw, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tomlRaw), "[mcp_servers.docs]") {
		t.Fatalf("codex user file: %s", tomlRaw)
	}

	// CLI agent toggle of the codex server flips enabled in place.
	off := mcpPatch(t, ts, map[string]any{"cli": "codex", "scope": "user", "name": "docs", "disabled": true})
	if off.StatusCode != http.StatusOK {
		t.Fatalf("codex toggle = %d", off.StatusCode)
	}
	off.Body.Close()
	tomlRaw, _ = os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if !strings.Contains(string(tomlRaw), "enabled = false") {
		t.Fatalf("codex toggle: %s", tomlRaw)
	}

	// Claude has no per-server switch: the refusal is the observable result.
	off = mcpPatch(t, ts, map[string]any{"cli": "claude-code", "scope": "user", "name": "docs", "disabled": true})
	if off.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(off), "turn on and off in Claude Code") {
		t.Fatalf("claude toggle = %d %s", off.StatusCode, thisBody(off))
	}
	off.Body.Close()

	// CLI agent remove (user) deletes the table again.
	del, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/mcp?cli=codex&scope=user&name=docs", nil)
	if err != nil {
		t.Fatal(err)
	}
	rmv := do(t, ts.Client(), del)
	if rmv.StatusCode != http.StatusOK {
		t.Fatalf("codex remove = %d", rmv.StatusCode)
	}
	rmv.Body.Close()
	tomlRaw, _ = os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if strings.Contains(string(tomlRaw), "mcp_servers.docs") {
		t.Fatalf("codex remove: %s", tomlRaw)
	}
}

func thisBody(res *http.Response) string {
	raw, _ := io.ReadAll(res.Body)
	return string(raw)
}

// ADR-0150 phase 2: ?cli=omp|agy answer through their CLI agent drivers — the
// same Report shape, native files edited in place, the AGY legacy url-only
// entry reported unowned and refused on write.
func TestMCPCliAgentDriversPhase2(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	writeFakeCLI(t, "omp", "")
	writeFakeCLI(t, "agy", "")
	ts := newTestServer(t, "cat")

	for _, tc := range []struct{ cli, source string }{{"omp", "native:omp"}, {"agy", "native:agy"}} {
		res, err := ts.Client().Get(ts.URL + "/api/mcp?cli=" + tc.cli)
		if err != nil {
			t.Fatal(err)
		}
		var rep struct {
			Adapter struct {
				Installed bool   `json:"installed"`
				Source    string `json:"source"`
			} `json:"adapter"`
		}
		if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK || rep.Adapter.Source != tc.source || !rep.Adapter.Installed {
			t.Fatalf("GET cli=%s = %d %+v", tc.cli, res.StatusCode, rep.Adapter)
		}
	}

	wsDir := t.TempDir()
	wres := postJSON(t, ts, "/api/workspaces", map[string]any{"name": "ws", "path": wsDir})
	if wres.StatusCode != http.StatusCreated {
		t.Fatalf("workspace = %d", wres.StatusCode)
	}
	var wk workspaceView
	if err := json.NewDecoder(wres.Body).Decode(&wk); err != nil {
		t.Fatal(err)
	}

	// Omp add lands in .omp/mcp.json; toggle flips the entry's enabled flag.
	add := postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "omp", "scope": "project", "workspaceId": wk.ID, "name": "docs", "url": "https://docs.example/mcp",
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("omp add = %d %s", add.StatusCode, thisBody(add))
	}
	_ = add.Body.Close()
	ompRaw, err := os.ReadFile(filepath.Join(wsDir, ".omp", "mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ompRaw), `"https://docs.example/mcp"`) {
		t.Fatalf("omp project file: %s", ompRaw)
	}
	off := mcpPatch(t, ts, map[string]any{"cli": "omp", "scope": "project", "workspaceId": wk.ID, "name": "docs", "disabled": true})
	if off.StatusCode != http.StatusOK {
		t.Fatalf("omp toggle = %d %s", off.StatusCode, thisBody(off))
	}
	_ = off.Body.Close()
	ompRaw, _ = os.ReadFile(filepath.Join(wsDir, ".omp", "mcp.json"))
	if !strings.Contains(string(ompRaw), `"enabled": false`) {
		t.Fatalf("omp toggle: %s", ompRaw)
	}

	// AGY keeps one config file: project scope refuses, the workspace stays
	// untouched (the vendor CLI resolves only the user file).
	add = postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "agy", "scope": "project", "workspaceId": wk.ID, "name": "relay", "url": "https://relay.example/mcp",
	})
	if add.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(add), "keeps one config file") {
		t.Fatalf("agy project add = %d %s", add.StatusCode, thisBody(add))
	}
	add.Body.Close()
	if _, err := os.Stat(filepath.Join(wsDir, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("agy project add touched the workspace: %v", err)
	}

	// The user file takes the add: serverUrl, never the legacy url key.
	add = postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "agy", "scope": "user", "name": "relay", "url": "https://relay.example/mcp",
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("agy add = %d %s", add.StatusCode, thisBody(add))
	}
	add.Body.Close()
	agyPath := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	agyRaw, err := os.ReadFile(agyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agyRaw), `"serverUrl"`) || strings.Contains(string(agyRaw), `"url"`) {
		t.Fatalf("agy user file: %s", agyRaw)
	}

	// A legacy url-only AGY entry displays unowned; Toggle refuses and the
	// file survives untouched.
	seed := `{"mcpServers":{"old":{"url":"https://old.example/mcp"},"relay":{"serverUrl":"https://relay.example/mcp"}}}`
	if err := os.WriteFile(agyPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	off = mcpPatch(t, ts, map[string]any{"cli": "agy", "scope": "user", "name": "old", "disabled": true})
	if off.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(off), "legacy entry managed in Antigravity") {
		t.Fatalf("agy legacy toggle = %d %s", off.StatusCode, thisBody(off))
	}
	_ = off.Body.Close()
	if got, _ := os.ReadFile(agyPath); string(got) != seed {
		t.Fatalf("legacy file modified: %q", got)
	}
	res, err := ts.Client().Get(ts.URL + "/api/mcp?cli=agy&workspace=" + wk.ID)
	if err != nil {
		t.Fatal(err)
	}
	var rep struct {
		Servers []struct {
			Name  string `json:"name"`
			URL   string `json:"url"`
			Owned bool   `json:"owned"`
		} `json:"servers"`
	}
	if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	owned := map[string]bool{}
	for _, s := range rep.Servers {
		owned[s.Name] = s.Owned
		if s.Name == "old" && s.Owned {
			t.Fatalf("legacy row reported owned: %+v", rep.Servers)
		}
		if s.Name == "old" && s.URL != "https://old.example/mcp" {
			t.Fatalf("legacy row url = %q", s.URL)
		}
	}
	if len(rep.Servers) != 2 || !owned["relay"] || owned["old"] {
		t.Fatalf("agy rows = %+v", rep.Servers)
	}

	// CLI agent remove of the managed AGY entry works; the legacy row is kept.
	del, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/mcp?cli=agy&scope=user&name=relay", nil)
	if err != nil {
		t.Fatal(err)
	}
	rmv := do(t, ts.Client(), del)
	if rmv.StatusCode != http.StatusOK {
		t.Fatalf("agy remove = %d %s", rmv.StatusCode, thisBody(rmv))
	}
	_ = rmv.Body.Close()
	agyRaw, _ = os.ReadFile(agyPath)
	if !strings.Contains(string(agyRaw), "old") || strings.Contains(string(agyRaw), "relay") {
		t.Fatalf("agy remove: %s", agyRaw)
	}
}

// CLI agent auth refuses with the vendor instruction; logout points at
// the CLI. The pi-mcp-adapter gate must not apply to CLI agents.
func TestMCPCliAgentAuthRefusals(t *testing.T) {
	ts := newTestServer(t, "cat")

	auth := postJSON(t, ts, "/api/mcp/auth", map[string]any{"cli": "claude-code", "name": "docs"})
	if auth.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(auth), "claude mcp login docs") {
		t.Fatalf("claude auth = %d %s", auth.StatusCode, thisBody(auth))
	}
	_ = auth.Body.Close()

	auth = postJSON(t, ts, "/api/mcp/auth", map[string]any{"cli": "codex", "name": "docs"})
	if auth.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(auth), "codex mcp login docs") {
		t.Fatalf("codex auth = %d %s", auth.StatusCode, thisBody(auth))
	}
	_ = auth.Body.Close()

	// Omp's OAuth is TUI-only; Antigravity signs in through its own
	// settings — the hints are driver-specific, never PiCode's flow.
	auth = postJSON(t, ts, "/api/mcp/auth", map[string]any{"cli": "omp", "name": "docs"})
	if auth.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(auth), "/mcp reauth docs") {
		t.Fatalf("omp auth = %d %s", auth.StatusCode, thisBody(auth))
	}
	_ = auth.Body.Close()

	auth = postJSON(t, ts, "/api/mcp/auth", map[string]any{"cli": "agy", "name": "docs"})
	if auth.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(auth), "Agent Settings") {
		t.Fatalf("agy auth = %d %s", auth.StatusCode, thisBody(auth))
	}
	_ = auth.Body.Close()

	// Phase 3: OpenCode signs in through its own terminal command; Grok
	// signs in on first use inside the CLI — plain text, nothing to copy.
	auth = postJSON(t, ts, "/api/mcp/auth", map[string]any{"cli": "opencode", "name": "docs"})
	if auth.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(auth), "opencode mcp auth docs") {
		t.Fatalf("opencode auth = %d %s", auth.StatusCode, thisBody(auth))
	}
	_ = auth.Body.Close()

	auth = postJSON(t, ts, "/api/mcp/auth", map[string]any{"cli": "grok", "name": "docs"})
	if auth.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(auth), "inside Grok") {
		t.Fatalf("grok auth = %d %s", auth.StatusCode, thisBody(auth))
	}
	_ = auth.Body.Close()

	// Phase 4: muse and hermes sign in through their own terminal commands.
	auth = postJSON(t, ts, "/api/mcp/auth", map[string]any{"cli": "muse", "name": "docs"})
	if auth.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(auth), "muse mcp login docs") {
		t.Fatalf("muse auth = %d %s", auth.StatusCode, thisBody(auth))
	}
	_ = auth.Body.Close()

	auth = postJSON(t, ts, "/api/mcp/auth", map[string]any{"cli": "hermes", "name": "docs"})
	if auth.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(auth), "hermes mcp login docs") {
		t.Fatalf("hermes auth = %d %s", auth.StatusCode, thisBody(auth))
	}
	_ = auth.Body.Close()

	logout := postJSON(t, ts, "/api/mcp/auth/logout", map[string]any{"cli": "codex", "name": "docs"})
	if logout.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(logout), "codex") {
		t.Fatalf("codex logout = %d %s", logout.StatusCode, thisBody(logout))
	}
	_ = logout.Body.Close()
}

// ADR-0150 phase 3: ?cli=opencode|grok answer through their CLI agent drivers —
// the OpenCode `mcp` block of opencode.json (command arrays, per-entry
// enabled flag) and Grok's TOML tables (toggle refused: no per-server
// switch).
func TestMCPCliAgentDriversPhase3(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "") // resolve OpenCode's user file from HOME
	writeFakeCLI(t, "opencode", "")
	writeFakeCLI(t, "grok", "")
	ts := newTestServer(t, "cat")

	for _, tc := range []struct{ cli, source string }{{"opencode", "native:opencode"}, {"grok", "native:grok"}} {
		res, err := ts.Client().Get(ts.URL + "/api/mcp?cli=" + tc.cli)
		if err != nil {
			t.Fatal(err)
		}
		var rep struct {
			Adapter struct {
				Installed bool   `json:"installed"`
				Source    string `json:"source"`
			} `json:"adapter"`
		}
		if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK || rep.Adapter.Source != tc.source || !rep.Adapter.Installed {
			t.Fatalf("GET cli=%s = %d %+v", tc.cli, res.StatusCode, rep.Adapter)
		}
	}

	wsDir := t.TempDir()
	wres := postJSON(t, ts, "/api/workspaces", map[string]any{"name": "ws", "path": wsDir})
	if wres.StatusCode != http.StatusCreated {
		t.Fatalf("workspace = %d", wres.StatusCode)
	}
	var wk workspaceView
	if err := json.NewDecoder(wres.Body).Decode(&wk); err != nil {
		t.Fatal(err)
	}

	// OpenCode add lands in the workspace's opencode.json as a local entry
	// with a command array; toggle flips the entry's enabled flag.
	add := postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "opencode", "scope": "project", "workspaceId": wk.ID, "name": "docs", "command": "npx", "args": []string{"-y", "docs-mcp"},
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("opencode add = %d %s", add.StatusCode, thisBody(add))
	}
	_ = add.Body.Close()
	ocPath := filepath.Join(wsDir, "opencode.json")
	ocRaw, err := os.ReadFile(ocPath)
	if err != nil {
		t.Fatal(err)
	}
	var oc map[string]any
	if err := json.Unmarshal(ocRaw, &oc); err != nil {
		t.Fatalf("opencode project file: %s", ocRaw)
	}
	docs, _ := oc["mcp"].(map[string]any)["docs"].(map[string]any)
	if docs == nil || docs["type"] != "local" {
		t.Fatalf("opencode entry = %v", docs)
	}
	cmd, _ := docs["command"].([]any)
	if len(cmd) != 3 || cmd[0] != "npx" || cmd[2] != "docs-mcp" {
		t.Fatalf("command array = %v", docs["command"])
	}
	off := mcpPatch(t, ts, map[string]any{"cli": "opencode", "scope": "project", "workspaceId": wk.ID, "name": "docs", "disabled": true})
	if off.StatusCode != http.StatusOK {
		t.Fatalf("opencode toggle = %d %s", off.StatusCode, thisBody(off))
	}
	_ = off.Body.Close()
	ocRaw, _ = os.ReadFile(ocPath)
	if err := json.Unmarshal(ocRaw, &oc); err != nil {
		t.Fatal(err)
	}
	docs, _ = oc["mcp"].(map[string]any)["docs"].(map[string]any)
	if docs["enabled"] != false {
		t.Fatalf("opencode toggle: %v", docs)
	}

	// Grok add lands in .grok/config.toml with the vendor's enabled flag;
	// toggle flips it in place and maintains the personal overlay array.
	add = postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "grok", "scope": "project", "workspaceId": wk.ID, "name": "docs", "command": "npx", "args": []string{"-y", "x"},
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("grok add = %d %s", add.StatusCode, thisBody(add))
	}
	add.Body.Close()
	grokPath := filepath.Join(wsDir, ".grok", "config.toml")
	grokRaw, err := os.ReadFile(grokPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(grokRaw), "[mcp_servers.docs]") || !strings.Contains(string(grokRaw), "enabled = true") {
		t.Fatalf("grok project file: %s", grokRaw)
	}
	off = mcpPatch(t, ts, map[string]any{"cli": "grok", "scope": "project", "workspaceId": wk.ID, "name": "docs", "disabled": true})
	if off.StatusCode != http.StatusOK {
		t.Fatalf("grok toggle = %d %s", off.StatusCode, thisBody(off))
	}
	off.Body.Close()
	grokRaw, _ = os.ReadFile(grokPath)
	if !strings.Contains(string(grokRaw), "enabled = false") {
		t.Fatalf("grok toggle: %s", grokRaw)
	}
	// A project toggle is the sticky project flag only: no overlay array
	// anywhere near the user file.
	if _, err := os.Stat(filepath.Join(home, ".grok")); !os.IsNotExist(err) {
		t.Fatalf("grok project toggle touched the user config: %v", err)
	}

	// CLI agent remove deletes each entry again.
	del, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/mcp?cli=opencode&scope=project&workspace="+wk.ID+"&name=docs", nil)
	if err != nil {
		t.Fatal(err)
	}
	rmv := do(t, ts.Client(), del)
	if rmv.StatusCode != http.StatusOK {
		t.Fatalf("opencode remove = %d %s", rmv.StatusCode, thisBody(rmv))
	}
	_ = rmv.Body.Close()
	ocRaw, _ = os.ReadFile(ocPath)
	if strings.Contains(string(ocRaw), "docs") {
		t.Fatalf("opencode remove: %s", ocRaw)
	}
	del, err = http.NewRequest(http.MethodDelete, ts.URL+"/api/mcp?cli=grok&scope=project&workspace="+wk.ID+"&name=docs", nil)
	if err != nil {
		t.Fatal(err)
	}
	rmv = do(t, ts.Client(), del)
	if rmv.StatusCode != http.StatusOK {
		t.Fatalf("grok remove = %d %s", rmv.StatusCode, thisBody(rmv))
	}
	_ = rmv.Body.Close()
	grokRaw, _ = os.ReadFile(grokPath)
	if strings.Contains(string(grokRaw), "mcp_servers.docs") {
		t.Fatalf("grok remove: %s", grokRaw)
	}
}

// ADR-0150 phase 4: ?cli=muse|hermes answer through their CLI agent drivers —
// Muse's settings.json mcpServers block (transport stdio|streamable_http,
// per-entry enabled flag, schema_version and mode preserved) and Hermes's
// config.yaml mcp_servers block (yaml.Node edits: comments and key order
// survive). Both CLIs keep one config file, so project scope refuses.
func TestMCPCliAgentDriversPhase4(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	writeFakeCLI(t, "muse", "")
	writeFakeCLI(t, "hermes", "")
	ts := newTestServer(t, "cat")

	for _, tc := range []struct{ cli, source string }{{"muse", "native:muse"}, {"hermes", "native:hermes"}} {
		res, err := ts.Client().Get(ts.URL + "/api/mcp?cli=" + tc.cli)
		if err != nil {
			t.Fatal(err)
		}
		var rep struct {
			Adapter struct {
				Installed bool   `json:"installed"`
				Source    string `json:"source"`
			} `json:"adapter"`
		}
		if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK || rep.Adapter.Source != tc.source || !rep.Adapter.Installed {
			t.Fatalf("GET cli=%s = %d %+v", tc.cli, res.StatusCode, rep.Adapter)
		}
	}

	// Muse add lands in ~/.config/muse/settings.json with the
	// streamable_http transport; toggle flips enabled; remove deletes.
	add := postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "muse", "scope": "user", "name": "docs", "url": "https://docs.example/mcp",
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("muse add = %d %s", add.StatusCode, thisBody(add))
	}
	_ = add.Body.Close()
	musePath := filepath.Join(home, ".config", "muse", "settings.json")
	museRaw, err := os.ReadFile(musePath)
	if err != nil {
		t.Fatal(err)
	}
	var museDoc map[string]any
	if err := json.Unmarshal(museRaw, &museDoc); err != nil {
		t.Fatalf("muse settings: %s", museRaw)
	}
	docs, _ := museDoc["mcpServers"].(map[string]any)["docs"].(map[string]any)
	if docs == nil || docs["transport"] != "streamable_http" {
		t.Fatalf("muse entry = %v", docs)
	}
	off := mcpPatch(t, ts, map[string]any{"cli": "muse", "scope": "user", "name": "docs", "disabled": true})
	if off.StatusCode != http.StatusOK {
		t.Fatalf("muse toggle = %d %s", off.StatusCode, thisBody(off))
	}
	_ = off.Body.Close()
	museRaw, _ = os.ReadFile(musePath)
	if err := json.Unmarshal(museRaw, &museDoc); err != nil {
		t.Fatal(err)
	}
	docs, _ = museDoc["mcpServers"].(map[string]any)["docs"].(map[string]any)
	if docs["enabled"] != false {
		t.Fatalf("muse toggle: %v", docs)
	}

	// Hermes add lands in ~/.hermes/config.yaml; toggle flips enabled;
	// a comment above the entry survives the whole round trip.
	add = postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "hermes", "scope": "user", "name": "docs", "command": "npx", "args": []string{"-y", "docs-mcp"},
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("hermes add = %d %s", add.StatusCode, thisBody(add))
	}
	_ = add.Body.Close()
	hermesPath := filepath.Join(home, ".hermes", "config.yaml")
	hermesRaw, err := os.ReadFile(hermesPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hermesRaw), "command: npx") {
		t.Fatalf("hermes config: %s", hermesRaw)
	}
	off = mcpPatch(t, ts, map[string]any{"cli": "hermes", "scope": "user", "name": "docs", "disabled": true})
	if off.StatusCode != http.StatusOK {
		t.Fatalf("hermes toggle = %d %s", off.StatusCode, thisBody(off))
	}
	_ = off.Body.Close()
	hermesRaw, _ = os.ReadFile(hermesPath)
	if !strings.Contains(string(hermesRaw), "enabled: false") {
		t.Fatalf("hermes toggle: %s", hermesRaw)
	}

	// Both CLIs keep one config file: project scope refuses loudly.
	proj := postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "hermes", "scope": "project", "name": "docs", "command": "npx",
	})
	if proj.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(proj), "no per-workspace file") {
		t.Fatalf("hermes project add = %d %s", proj.StatusCode, thisBody(proj))
	}
	_ = proj.Body.Close()
	proj = postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "muse", "scope": "project", "name": "docs", "url": "https://docs.example/mcp",
	})
	if proj.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(proj), "no per-workspace file") {
		t.Fatalf("muse project add = %d %s", proj.StatusCode, thisBody(proj))
	}
	_ = proj.Body.Close()

	// CLI agent removes clean up again.
	for _, cli := range []string{"hermes", "muse"} {
		del, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/mcp?cli="+cli+"&scope=user&name=docs", nil)
		if err != nil {
			t.Fatal(err)
		}
		rmv := do(t, ts.Client(), del)
		if rmv.StatusCode != http.StatusOK {
			t.Fatalf("%s remove = %d %s", cli, rmv.StatusCode, thisBody(rmv))
		}
		_ = rmv.Body.Close()
	}
	hermesRaw, _ = os.ReadFile(hermesPath)
	if strings.Contains(string(hermesRaw), "docs") {
		t.Fatalf("hermes remove: %s", hermesRaw)
	}
	museRaw, _ = os.ReadFile(musePath)
	if strings.Contains(string(museRaw), "docs") {
		t.Fatalf("muse remove: %s", museRaw)
	}
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

// A CLI config file PiCode cannot parse must not fail GET /api/mcp: the
// report still answers 200 with the layer blocked (exists + reason, no
// servers from it) while a healthy layer keeps listing. The owner's actual
// repro — OpenCode's JSONC-ish trailing comma — reads without blocking at
// all. The pane's Open action reveals the layer's own path, never a
// client-supplied one.
type mcpLayerView struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Error  string `json:"error"`
}

type mcpReportView struct {
	Layers  []mcpLayerView `json:"layers"`
	Servers []any          `json:"servers"`
}

func TestMCPCliAgentMalformedLayerDegrades(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	writeFakeCLI(t, "opencode", "")
	ts := newTestServer(t, "cat")

	userFile := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		t.Fatal(err)
	}

	get := func() (int, mcpReportView) {
		var rep mcpReportView
		res, err := ts.Client().Get(ts.URL + "/api/mcp?cli=opencode")
		if err != nil {
			t.Fatal(err)
		}
		if err := json.NewDecoder(res.Body).Decode(&rep); err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		return res.StatusCode, rep
	}
	blocked := func(rep mcpReportView) *mcpLayerView {
		for i := range rep.Layers {
			if rep.Layers[i].Error != "" {
				return &rep.Layers[i]
			}
		}
		return nil
	}

	// The owner's repro: JSONC-ish trailing comma after a schema-only
	// config. Tolerated on read — 200, no blocked layer.
	if err := os.WriteFile(userFile, []byte("{\n  \"$schema\": \"https://opencode.ai/config.json\",\n}"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, rep := get()
	if code != http.StatusOK {
		t.Fatalf("JSONC user file: GET = %d", code)
	}
	if b := blocked(rep); b != nil {
		t.Fatalf("JSONC user file blocked: %+v", b)
	}

	// Genuinely malformed: 200 with the layer blocked and no servers.
	if err := os.WriteFile(userFile, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, rep = get()
	if code != http.StatusOK {
		t.Fatalf("malformed user file: GET = %d", code)
	}
	if len(rep.Servers) != 0 {
		t.Fatalf("servers from a broken layer: %v", rep.Servers)
	}
	b := blocked(rep)
	if b == nil || !b.Exists || b.Error != "is not valid JSON" || !strings.HasSuffix(b.Path, "opencode.json") {
		t.Fatalf("blocked layer = %+v", b)
	}

	// The Open action: reveal resolves the layer server-side.
	opened := ""
	oldReveal := revealFn
	revealFn = func(p string) error { opened = p; return nil }
	t.Cleanup(func() { revealFn = oldReveal })
	res := postJSON(t, ts, "/api/mcp/reveal", map[string]any{"cli": "opencode", "scope": "user"})
	if res.StatusCode != http.StatusOK || opened != b.Path {
		t.Fatalf("reveal = %d, opened %q, want %q", res.StatusCode, opened, b.Path)
	}
	_ = res.Body.Close()

	// No path comes from the client: an unknown CLI is refused outright.
	opened = ""
	res = postJSON(t, ts, "/api/mcp/reveal", map[string]any{"cli": "../pi", "scope": "user", "path": "/etc/passwd"})
	if res.StatusCode != http.StatusBadRequest || opened != "" {
		t.Fatalf("reveal unknown cli = %d, opened %q", res.StatusCode, opened)
	}
	_ = res.Body.Close()
}

func TestConnectorsHashIsCanonical(t *testing.T) {
	for _, tc := range []struct{ cli, ws, agent, want string }{
		{"", "", "", "#/clis/pi/connectors"},
		{"pi", "W", "", "#/clis/pi/connectors?workspaceId=W"},
		{"omp", "W", "a/1", "#/clis/omp/connectors?agentId=a%2F1&workspaceId=W"},
	} {
		if got := connectorsHash(tc.cli, tc.ws, tc.agent); got != tc.want {
			t.Errorf("connectorsHash(%q,%q,%q) = %q, want %q", tc.cli, tc.ws, tc.agent, got, tc.want)
		}
	}
}
