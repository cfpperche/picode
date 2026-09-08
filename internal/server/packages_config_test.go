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
)

// Decision table (one row per test): the GUI must be able to read, edit and
// clear pi-roles files with the same semantics the extension uses —
// workspace file + agent overlay + effective merge — while a file it cannot
// parse is reported, never silently overwritten.
func putConfig(t *testing.T, ts *httptest.Server, body map[string]any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/packages/config", strings.NewReader(string(b)))
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

func decode[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	var out T
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func rolesDoc(t *testing.T, dir, rel, doc string) {
	t.Helper()
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readRolesFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestPackageConfigWorkspaceLayer(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "RolesWs", dir)

	// GET before any file: dormant, not an error.
	res, err := http.Get(ts.URL + "/api/packages/config?package=pi-roles&workspace=" + wk.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	view := decode[packageConfigView](t, res)
	if view.Workspace.Exists || view.Workspace.Invalid != "" {
		t.Fatalf("missing file must be dormant, got %+v", view.Workspace)
	}
	if view.Agent != nil {
		t.Fatal("no agent queried — agent layer must be absent")
	}

	// PUT creates the workspace file atomically and echoes fresh layers.
	res2 := putConfig(t, ts, map[string]any{
		"package": "pi-roles", "scope": "workspace", "workspaceId": wk.ID,
		"config": map[string]any{
			"builtin": map[string]any{"default": map[string]any{"model": "zai/glm-5.3", "thinking": "medium"}},
			"custom":  []any{map[string]any{"name": "redteam", "model": "kimi-coding/k3"}},
		},
	})
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("put status %d", res2.StatusCode)
	}
	saved := decode[packageConfigView](t, res2)
	if !saved.Workspace.Exists || saved.Workspace.Config.Builtin["default"] == nil {
		t.Fatalf("saved view = %+v", saved.Workspace)
	}
	onDisk := readRolesFile(t, filepath.Join(dir, ".pi", "roles.json"))
	if !strings.Contains(onDisk, `"keep"`) && !strings.Contains(onDisk, "zai/glm-5.3") {
		t.Fatalf("file content wrong: %s", onDisk)
	}
}

func TestPackageConfigAgentOverlayAndEffective(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "RolesOverlay", dir)
	agentID := wk.Agents[0].ID

	rolesDoc(t, dir, ".pi/roles.json", `{"builtin":{"default":{"model":"zai/glm-5.3"},"vision":{"model":"xai/grok-4.6"}},"custom":[{"name":"shared","model":"p/a"}]}`)

	res, err := http.Get(ts.URL + "/api/packages/config?package=pi-roles&workspace=" + wk.ID + "&agent=" + agentID)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	view := decode[packageConfigView](t, res)
	if view.Agent == nil || view.Agent.Exists {
		t.Fatalf("no overlay yet, got %+v", view.Agent)
	}
	if view.Effective.Builtin["default"] == nil || view.Effective.Builtin["default"].Model != "zai/glm-5.3" {
		t.Fatalf("effective must inherit the workspace file, got %+v", view.Effective.Builtin)
	}

	// The overlay wins its slot and shadows one custom; the rest inherits.
	res2 := putConfig(t, ts, map[string]any{
		"package": "pi-roles", "scope": "agent", "workspaceId": wk.ID, "agentId": agentID,
		"config": map[string]any{
			"builtin": map[string]any{"default": map[string]any{"model": "anthropic/claude-haiku-4-5"}},
			"custom":  []any{map[string]any{"name": "shared", "model": "p/c"}},
		},
	})
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("overlay put status %d", res2.StatusCode)
	}
	saved := decode[packageConfigView](t, res2)
	if saved.Effective.Builtin["default"].Model != "anthropic/claude-haiku-4-5" {
		t.Fatalf("overlay slot must win, got %+v", saved.Effective.Builtin["default"])
	}
	if saved.Effective.Builtin["vision"] == nil || saved.Effective.Builtin["vision"].Model != "xai/grok-4.6" {
		t.Fatal("unset overlay slot must inherit vision")
	}
	names := map[string]string{}
	for _, c := range saved.Effective.Custom {
		names[c.Name] = c.Model
	}
	if names["shared"] != "p/c" {
		t.Fatalf("overlay custom must shadow, got %v", names)
	}

	// The overlay file lands in the agent's cwd (here: the workspace dir).
	if _, err := os.Stat(filepath.Join(dir, ".pi", "roles", agentID+".json")); err != nil {
		t.Fatalf("overlay file: %v", err)
	}
}

func TestPackageConfigInvalidFileIsConflictNotOverwrite(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "RolesInvalid", dir)
	rolesDoc(t, dir, ".pi/roles.json", "{broken")

	res := putConfig(t, ts, map[string]any{
		"package": "pi-roles", "scope": "workspace", "workspaceId": wk.ID,
		"config": map[string]any{"builtin": map[string]any{}, "custom": []any{}},
	})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("invalid file must 409, got %d", res.StatusCode)
	}
	if readRolesFile(t, filepath.Join(dir, ".pi", "roles.json")) != "{broken" {
		t.Fatal("a conflict must not touch the file")
	}

	res2 := putConfig(t, ts, map[string]any{
		"package": "pi-roles", "scope": "workspace", "workspaceId": wk.ID, "force": true,
		"config": map[string]any{"builtin": map[string]any{}, "custom": []any{}},
	})
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("force put status %d", res2.StatusCode)
	}
}

func TestPackageConfigValidationAndScopes(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "RolesGuard", dir)

	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"unknown package", map[string]any{"package": "pi-compact", "scope": "workspace", "workspaceId": wk.ID, "config": map[string]any{}}, http.StatusBadRequest},
		{"bad scope (no machine layer)", map[string]any{"package": "pi-roles", "scope": "user", "workspaceId": wk.ID, "config": map[string]any{}}, http.StatusBadRequest},
		{"invalid role name", map[string]any{"package": "pi-roles", "scope": "workspace", "workspaceId": wk.ID,
			"config": map[string]any{"custom": []any{map[string]any{"name": "1bad", "model": "p/m"}}}}, http.StatusBadRequest},
		{"reserved name", map[string]any{"package": "pi-roles", "scope": "workspace", "workspaceId": wk.ID,
			"config": map[string]any{"custom": []any{map[string]any{"name": "roles", "model": "p/m"}}}}, http.StatusBadRequest},
		{"bad model", map[string]any{"package": "pi-roles", "scope": "workspace", "workspaceId": wk.ID,
			"config": map[string]any{"builtin": map[string]any{"plan": map[string]any{"model": "nope"}}}}, http.StatusBadRequest},
		{"missing workspace", map[string]any{"package": "pi-roles", "scope": "workspace", "config": map[string]any{}}, http.StatusBadRequest},
		{"agent scope without agent", map[string]any{"package": "pi-roles", "scope": "agent", "workspaceId": wk.ID, "config": map[string]any{}}, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := putConfig(t, ts, c.body)
			if res.StatusCode != c.want {
				t.Fatalf("status = %d, want %d", res.StatusCode, c.want)
			}
		})
	}

	// GET without a workspace names the missing context instead of 500ing.
	res, err := http.Get(ts.URL + "/api/packages/config?package=pi-roles")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET without workspace = %d", res.StatusCode)
	}
}

func TestPackageConfigDelete(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "RolesClear", dir)
	agentID := wk.Agents[0].ID
	rolesDoc(t, dir, ".pi/roles.json", `{"custom":[]}`)
	overlay := filepath.Join(dir, ".pi", "roles", agentID+".json")
	rolesDoc(t, dir, ".pi/roles/"+agentID+".json", `{"custom":[{"name":"x","model":"p/m"}]}`)

	del := func(q string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/packages/config?"+q, nil)
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

	if res := del("package=pi-roles&scope=agent&workspace=" + wk.ID + "&agent=" + agentID); res.StatusCode != http.StatusOK {
		t.Fatalf("delete overlay = %d", res.StatusCode)
	}
	if _, err := os.Stat(overlay); !os.IsNotExist(err) {
		t.Fatalf("overlay must be gone, err %v", err)
	}
	// Deleting again is idempotent.
	if res := del("package=pi-roles&scope=agent&workspace=" + wk.ID + "&agent=" + agentID); res.StatusCode != http.StatusOK {
		t.Fatalf("idempotent delete = %d", res.StatusCode)
	}
	// The workspace file is untouched by the agent-scope clear.
	if readRolesFile(t, filepath.Join(dir, ".pi", "roles.json")) == "" {
		t.Fatal("workspace file must survive an agent clear")
	}
	if res := del("package=pi-roles&scope=workspace&workspace=" + wk.ID); res.StatusCode != http.StatusOK {
		t.Fatalf("delete workspace = %d", res.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(dir, ".pi", "roles.json")); !os.IsNotExist(err) {
		t.Fatalf("workspace file must be gone, err %v", err)
	}
}

func TestPackageListCarriesConfigKind(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "RolesKind", dir)

	userDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(userDir, "settings.json"),
		[]byte(`{"packages":["/opt/picode/packages/pi-roles","npm:pi-web-search"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	oldUserDir := pipkg.UserDir
	pipkg.UserDir = func() string { return userDir }
	t.Cleanup(func() { pipkg.UserDir = oldUserDir })

	res, err := http.Get(ts.URL + "/api/packages?workspace=" + wk.ID + "&agent=" + wk.Agents[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	rep := decode[pipkg.Report](t, res)
	kind := map[string]string{}
	for _, p := range rep.Packages {
		kind[p.Source] = p.ConfigKind
	}
	if kind["/opt/picode/packages/pi-roles"] != "roles" {
		t.Fatalf("pi-roles configKind = %q", kind["/opt/picode/packages/pi-roles"])
	}
	if kind["npm:pi-web-search"] != "" {
		t.Fatalf("pi-web-search must have no configKind, got %q", kind["npm:pi-web-search"])
	}
}
