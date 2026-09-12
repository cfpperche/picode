package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	if kind["npm:pi-web-search"] != "web-search" {
		t.Fatalf("pi-web-search configKind = %q, want the descriptor id \"web-search\"", kind["npm:pi-web-search"])
	}
	// Honest absence survives: a package with no descriptor and no manifest
	// still carries no configKind, and the card shows no Configure button.
	if kind["npm:pi-nowhere"] != "" {
		t.Fatalf("pi-nowhere configKind = %q, want empty", kind["npm:pi-nowhere"])
	}
}

// --- descriptor-driven configs (docs/plans/package-config-manifest.md) ---
//
// One test per row of the plan's decision table: honest refusal for unknown
// packages, create-on-save, per-field validation, unknown keys preserved,
// a file the parser refuses never silently overwritten, and the save
// announces itself on the feed.

func withAgentDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return dir }
	t.Cleanup(func() { pipkg.UserDir = old })
	return dir
}

func descriptorServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	dir := withAgentDir(t)
	st := testStore(t)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)
	return ts, dir
}

func putDescriptor(t *testing.T, ts *httptest.Server, body map[string]any) *http.Response {
	return putConfig(t, ts, body)
}

func TestDescriptorUnknownPackageIsRefusedHonestly(t *testing.T) {
	ts, _ := descriptorServer(t)
	res, err := http.Get(ts.URL + "/api/packages/config?package=pi-nowhere")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
	var body struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&body)
	if !strings.Contains(body.Error, "no configuration is available for pi-nowhere") {
		t.Fatalf("error = %q, want the honest refusal", body.Error)
	}
}

func TestDescriptorCreatesTheFileOnSave(t *testing.T) {
	ts, agentDir := descriptorServer(t)
	values := map[string]any{"provider": "google-generative-ai", "model": "gemini-3.5-flash"}
	res := putDescriptor(t, ts, map[string]any{"package": "web-search", "config": values})
	if res.StatusCode != 200 {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	b, err := os.ReadFile(filepath.Join(agentDir, "web-search.json"))
	if err != nil {
		t.Fatalf("save did not create the file: %v", err)
	}
	var saved map[string]any
	if err := json.Unmarshal(b, &saved); err != nil {
		t.Fatalf("saved file is not JSON: %v", err)
	}
	if saved["provider"] != "google-generative-ai" || saved["model"] != "gemini-3.5-flash" {
		t.Fatalf("saved = %v", saved)
	}
}

func TestDescriptorRejectsInvalidValues(t *testing.T) {
	ts, agentDir := descriptorServer(t)
	for _, body := range []map[string]any{
		{"package": "web-search", "config": map[string]any{"provider": "google-generative-ai"}},
		{"package": "web-search", "config": map[string]any{"provider": "not-a-kind", "model": "m"}},
	} {
		res := putDescriptor(t, ts, body)
		if res.StatusCode != 400 {
			t.Fatalf("status %d, want 400 for %v", res.StatusCode, body["config"])
		}
		res.Body.Close()
	}
	if _, err := os.Stat(filepath.Join(agentDir, "web-search.json")); !os.IsNotExist(err) {
		t.Fatal("invalid values must not create the file")
	}
}

func TestDescriptorNeverOverwritesAnUnparseableFile(t *testing.T) {
	ts, agentDir := descriptorServer(t)
	p := filepath.Join(agentDir, "web-search.json")
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := putDescriptor(t, ts, map[string]any{
		"package": "web-search",
		"config":  map[string]any{"provider": "xai", "model": "grok-4.6"},
	})
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, want 409 without force", res.StatusCode)
	}
	res.Body.Close()
	if b, _ := os.ReadFile(p); string(b) != "{not json" {
		t.Fatalf("the 409 must leave the file untouched, got %q", b)
	}
	res = putDescriptor(t, ts, map[string]any{
		"package": "web-search", "force": true,
		"config": map[string]any{"provider": "xai", "model": "grok-4.6"},
	})
	if res.StatusCode != 200 {
		t.Fatalf("status %d, want 200 with force", res.StatusCode)
	}
	res.Body.Close()
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), "\"provider\": \"xai\"") {
		t.Fatalf("forced replace wrote %q", b)
	}
}

func TestDescriptorPreservesUnknownKeys(t *testing.T) {
	ts, agentDir := descriptorServer(t)
	p := filepath.Join(agentDir, "web-search.json")
	if err := os.WriteFile(p, []byte(`{"provider":"xai","model":"grok-4.6","customNote":"keep"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	res := putDescriptor(t, ts, map[string]any{
		"package": "web-search",
		"config":  map[string]any{"provider": "xai", "model": "grok-4.6-fast"},
	})
	if res.StatusCode != 200 {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	res.Body.Close()
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), "customNote") || !strings.Contains(string(b), "keep") {
		t.Fatalf("unknown keys must survive the write: %q", b)
	}
}

func TestDescriptorDeleteRemovesTheFileIdempotently(t *testing.T) {
	ts, agentDir := descriptorServer(t)
	p := filepath.Join(agentDir, "web-search.json")
	if err := os.WriteFile(p, []byte(`{"provider":"xai","model":"grok-4.6"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/packages/config?package=web-search", nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatalf("delete %d: status %d, want 200 (idempotent)", i+1, res.StatusCode)
		}
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("delete must remove the file")
	}
}

func TestDescriptorSaveAnnouncesOnTheFeed(t *testing.T) {
	withAgentDir(t)
	ts, _, _ := newFeedServer(t)
	body, closeStream := openStream(t, ts, "")
	defer closeStream()
	readFrames(t, body, 1, 3*time.Second)
	res := putDescriptor(t, ts, map[string]any{
		"package": "web-search",
		"config":  map[string]any{"provider": "xai", "model": "grok-4.6"},
	})
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	fr := readFrames(t, body, 1, 3*time.Second)[0]
	if !strings.Contains(fr.Data, `"type":"packages.config"`) || !strings.Contains(fr.Data, "web-search") {
		t.Fatalf("feed frame = %+v, want packages.config for web-search", fr)
	}
}

// --- workspace-scope descriptors (pi-compact / .pi/compact.json) ---

func TestCompactDescriptorResolvesThroughWorkspace(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "CompactKind", dir)

	res, err := http.Get(ts.URL + "/api/packages/config?package=compact&workspace=" + wk.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	var view struct {
		Kind  string `json:"kind"`
		Scope string `json:"scope"`
		Path  string `json:"path"`
		Layer struct {
			Exists bool `json:"exists"`
		} `json:"layer"`
	}
	if err := json.NewDecoder(res.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.Kind != "compact" || view.Scope != "workspace" || !strings.HasSuffix(view.Path, ".pi/compact.json") {
		t.Fatalf("view = %+v", view)
	}
	if view.Layer.Exists {
		t.Fatal("no compact.json was written yet")
	}
}

func TestCompactWithoutWorkspaceIsRefused(t *testing.T) {
	ts := newTestServer(t, "cat")
	res, err := http.Get(ts.URL + "/api/packages/config?package=compact")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Fatalf("status %d, want 400", res.StatusCode)
	}
	var body struct {
		Error string `json:"error"`
	}
	_ = json.NewDecoder(res.Body).Decode(&body)
	if !strings.Contains(body.Error, "workspace folder") {
		t.Fatalf("error = %q", body.Error)
	}
}

func TestCompactSaveWritesIntoTheWorkspaceAndOmitsUnset(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := t.TempDir()
	wk := addWorkspaceWithAgent(t, ts, "CompactSave", dir)

	// enabled omitted (checkbox untouched), atPercent out of range → 400.
	res := putDescriptor(t, ts, map[string]any{"package": "compact", "workspaceId": wk.ID,
		"config": map[string]any{"enabled": true, "atPercent": 5}})
	if res.StatusCode != 400 {
		t.Fatalf("status %d, want 400 for atPercent 5", res.StatusCode)
	}
	res.Body.Close()

	res = putDescriptor(t, ts, map[string]any{"package": "compact", "workspaceId": wk.ID,
		"config": map[string]any{"enabled": true, "atPercent": 0.5, "model": "zai/glm-5.3"}})
	if res.StatusCode != 200 {
		t.Fatalf("status %d, want 200", res.StatusCode)
	}
	res.Body.Close()

	b, err := os.ReadFile(filepath.Join(dir, ".pi", "compact.json"))
	if err != nil {
		t.Fatalf("save did not create the workspace file: %v", err)
	}
	var saved map[string]any
	if err := json.Unmarshal(b, &saved); err != nil {
		t.Fatal(err)
	}
	if saved["enabled"] != true || saved["atPercent"] != 0.5 || saved["model"] != "zai/glm-5.3" {
		t.Fatalf("saved = %v", saved)
	}
	if _, present := saved["atTokens"]; present {
		t.Fatalf("unset fields must be omitted, got %v", saved)
	}
}
