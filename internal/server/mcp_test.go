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

// ADR-0150: a request naming a CLI without a driver must fail loudly
// instead of writing Pi's files. claude-code, codex, omp and agy ship
// drivers; every other guest CLI still has none.
func TestMCPRejectsCLIWithoutDriver(t *testing.T) {
	ts := newTestServer(t, "cat")

	get, err := ts.Client().Get(ts.URL + "/api/mcp?cli=grok")
	if err != nil {
		t.Fatal(err)
	}
	if get.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET cli=grok status %d", get.StatusCode)
	}
	_ = get.Body.Close()

	add := postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "grok", "scope": "user", "name": "deepwiki", "url": "https://mcp.deepwiki.com/mcp",
	})
	if add.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST cli=grok status %d", add.StatusCode)
	}
	_ = add.Body.Close()

	del, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/mcp?scope=user&name=deepwiki&cli=grok", nil)
	if err != nil {
		t.Fatal(err)
	}
	rm := do(t, ts.Client(), del)
	if rm.StatusCode != http.StatusBadRequest {
		t.Fatalf("DELETE cli=grok status %d", rm.StatusCode)
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
	_ = pi.Body.Close()
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
// Pi's /api/mcp, guest auth/import refuse with instructions, and guest
// mutations land in the CLI's native config.
func TestMCPGuestDriversDispatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	writeFakeCLI(t, "claude", `{"mcpServers":{"relay":{"type":"http","url":"https://relay.example/mcp"}}}`)
	writeFakeCLI(t, "codex", "")
	ts := newTestServer(t, "cat")

	// GET ?cli=claude-code: Report shape with the guest adapter source.
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
		Found   []any             `json:"found"`
		Imports []any             `json:"imports"`
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
	if len(rep.Presets) == 0 || len(rep.Found) != 0 || len(rep.Packs) != 0 || len(rep.Imports) != 0 {
		t.Fatalf("report shape = presets %d found %v packs %v imports %v", len(rep.Presets), rep.Found, rep.Packs, rep.Imports)
	}

	// Guest add (project scope, no binary needed) writes the native file and
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
		t.Fatalf("guest add = %d body %s", add.StatusCode, body)
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

	// Codex add goes to its TOML file.
	add = postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "codex", "scope": "project", "workspaceId": wk.ID, "name": "docs", "command": "npx", "args": []string{"-y", "x"},
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("codex add = %d", add.StatusCode)
	}
	_ = add.Body.Close()
	tomlRaw, err := os.ReadFile(filepath.Join(wsDir, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tomlRaw), "[mcp_servers.docs]") {
		t.Fatalf("codex project file: %s", tomlRaw)
	}

	// Guest toggle of the codex server flips enabled in place.
	off := mcpPatch(t, ts, map[string]any{"cli": "codex", "scope": "project", "workspaceId": wk.ID, "name": "docs", "disabled": true})
	if off.StatusCode != http.StatusOK {
		t.Fatalf("codex toggle = %d", off.StatusCode)
	}
	_ = off.Body.Close()
	tomlRaw, _ = os.ReadFile(filepath.Join(wsDir, ".codex", "config.toml"))
	if !strings.Contains(string(tomlRaw), "enabled = false") {
		t.Fatalf("codex toggle: %s", tomlRaw)
	}

	// Claude has no per-server switch: the refusal is the observable result.
	off = mcpPatch(t, ts, map[string]any{"cli": "claude-code", "scope": "project", "workspaceId": wk.ID, "name": "docs", "disabled": true})
	if off.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(off), "turn on and off in Claude Code") {
		t.Fatalf("claude toggle = %d %s", off.StatusCode, thisBody(off))
	}

	// Guest remove (project) deletes the table again.
	del, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/mcp?cli=codex&scope=project&workspace="+wk.ID+"&name=docs", nil)
	if err != nil {
		t.Fatal(err)
	}
	rmv := do(t, ts.Client(), del)
	if rmv.StatusCode != http.StatusOK {
		t.Fatalf("codex remove = %d", rmv.StatusCode)
	}
	_ = rmv.Body.Close()
	tomlRaw, _ = os.ReadFile(filepath.Join(wsDir, ".codex", "config.toml"))
	if strings.Contains(string(tomlRaw), "mcp_servers.docs") {
		t.Fatalf("codex remove: %s", tomlRaw)
	}
}

func thisBody(res *http.Response) string {
	raw, _ := io.ReadAll(res.Body)
	return string(raw)
}

// ADR-0150 phase 2: ?cli=omp|agy answer through their guest drivers — the
// same Report shape, native files edited in place, the AGY legacy url-only
// entry reported unowned and refused on write.
func TestMCPGuestDriversPhase2(t *testing.T) {
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

	// AGY add writes serverUrl, never the legacy url key.
	add = postJSON(t, ts, "/api/mcp", map[string]any{
		"cli": "agy", "scope": "project", "workspaceId": wk.ID, "name": "relay", "url": "https://relay.example/mcp",
	})
	if add.StatusCode != http.StatusOK {
		t.Fatalf("agy add = %d %s", add.StatusCode, thisBody(add))
	}
	_ = add.Body.Close()
	agyPath := filepath.Join(wsDir, ".agents", "mcp_config.json")
	agyRaw, err := os.ReadFile(agyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agyRaw), `"serverUrl"`) || strings.Contains(string(agyRaw), `"url"`) {
		t.Fatalf("agy project file: %s", agyRaw)
	}

	// A legacy url-only AGY entry displays unowned; Toggle refuses and the
	// file survives untouched.
	seed := `{"mcpServers":{"old":{"url":"https://old.example/mcp"},"relay":{"serverUrl":"https://relay.example/mcp"}}}`
	if err := os.WriteFile(agyPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	off = mcpPatch(t, ts, map[string]any{"cli": "agy", "scope": "project", "workspaceId": wk.ID, "name": "old", "disabled": true})
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

	// Guest remove of the managed AGY entry works; the legacy row is kept.
	del, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/mcp?cli=agy&scope=project&workspace="+wk.ID+"&name=relay", nil)
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

// Guest auth and import refuse with the vendor instruction; logout points at
// the CLI. The pi-mcp-adapter gate must not apply to guests.
func TestMCPGuestAuthAndImportRefusals(t *testing.T) {
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

	logout := postJSON(t, ts, "/api/mcp/auth/logout", map[string]any{"cli": "codex", "name": "docs"})
	if logout.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(logout), "codex") {
		t.Fatalf("codex logout = %d %s", logout.StatusCode, thisBody(logout))
	}
	_ = logout.Body.Close()

	imp := postJSON(t, ts, "/api/mcp/import", map[string]any{"cli": "claude-code", "picks": []any{map[string]any{"kind": "cursor", "servers": []any{"x"}}}})
	if imp.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(imp), "arrive in a later phase") {
		t.Fatalf("claude import = %d %s", imp.StatusCode, thisBody(imp))
	}
	_ = imp.Body.Close()

	imp = postJSON(t, ts, "/api/mcp/import", map[string]any{"cli": "omp", "picks": []any{map[string]any{"kind": "cursor", "servers": []any{"x"}}}})
	if imp.StatusCode != http.StatusBadRequest || !strings.Contains(thisBody(imp), "arrive in a later phase") {
		t.Fatalf("omp import = %d %s", imp.StatusCode, thisBody(imp))
	}
	_ = imp.Body.Close()
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
