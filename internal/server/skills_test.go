package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/skills"
)

// The Skills report answers every declared CLI, reads the workspace the
// request names, and refuses an unknown CLI by naming the declared ones.
func TestSkillsReportRoute(t *testing.T) {
	st := testStore(t)
	proj := t.TempDir()
	dir := filepath.Join(proj, ".agents", "skills", "route-probe")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: route-probe\ndescription: probe\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := st.AddWorkspace("Proj", proj)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)

	for _, cli := range skills.CLIs() {
		res, err := http.Get(ts.URL + "/api/skills/report?cli=" + cli + "&workspace=" + w.ID)
		if err != nil {
			t.Fatal(err)
		}
		var rep skills.Report
		err = json.NewDecoder(res.Body).Decode(&rep)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || err != nil {
			t.Fatalf("%s: status %d err %v", cli, res.StatusCode, err)
		}
		found := false
		for _, r := range rep.Rows {
			found = found || r.Name == "route-probe"
		}
		// Claude Code reads only .claude/skills (measured, ADR-0196).
		if found == (cli == "claude-code") {
			t.Fatalf("%s: route-probe found=%v", cli, found)
		}
	}

	res, err := http.Get(ts.URL + "/api/skills/report?cli=nope")
	if err != nil {
		t.Fatal(err)
	}
	var e struct{ Error string }
	_ = json.NewDecoder(res.Body).Decode(&e)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(e.Error, "claude-code") {
		t.Fatalf("unknown cli: %d %q", res.StatusCode, e.Error)
	}
}

func skillsCall(t *testing.T, method, url string, body any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(method, url, strings.NewReader(string(b)))
	req.Header.Set("content-type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// Preview, install into a workspace, the question a second folder raises,
// and removal — through the routes (ADR-0196 slice 2).
func TestSkillsInstallRoutes(t *testing.T) {
	st := testStore(t)
	proj := t.TempDir()
	w, err := st.AddWorkspace("Proj", proj)
	if err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: route-skill\ndescription: from a folder\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat", DataDir: t.TempDir()}).Handler)
	t.Cleanup(ts.Close)

	code, p := skillsCall(t, "POST", ts.URL+"/api/skills/preview", map[string]string{"source": src})
	if code != 200 || p["id"] == nil {
		t.Fatalf("preview: %d %v", code, p)
	}
	install := map[string]any{"preview": p["id"], "path": "", "scope": "workspace", "workspace": w.ID}
	code, res := skillsCall(t, "POST", ts.URL+"/api/skills", install)
	if code != 200 || res["status"] != "installed" {
		t.Fatalf("install: %d %v", code, res)
	}
	if _, err := os.Stat(filepath.Join(proj, ".agents/skills/route-skill/SKILL.md")); err != nil {
		t.Fatal("not installed")
	}
	// Different content, recorded in the lock: the pane is asked to update.
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: route-skill\ndescription: changed\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, p = skillsCall(t, "POST", ts.URL+"/api/skills/preview", map[string]string{"source": src})
	install["preview"] = p["id"]
	code, res = skillsCall(t, "POST", ts.URL+"/api/skills", install)
	if code != http.StatusConflict || res["code"] != "update" {
		t.Fatalf("second install: %d %v", code, res)
	}
	code, res = skillsCall(t, "DELETE", ts.URL+"/api/skills", map[string]any{"name": "route-skill", "scope": "workspace", "workspace": w.ID})
	if code != 200 || res["status"] != "removed" {
		t.Fatalf("remove: %d %v", code, res)
	}
	code, _ = skillsCall(t, "POST", ts.URL+"/api/skills/preview", map[string]string{"source": "not a source"})
	if code != http.StatusBadRequest {
		t.Fatalf("bad source: %d", code)
	}
}

// The agent scope (ADR-0196 slice 4): an install caches the skill and adds
// it to the agent's list, which the report shows under the agent; the same
// content again is "already"; remove drops it from the list and leaves the
// cached copy; an agent whose CLI takes no skills at launch is refused.
func TestSkillsAgentRoutes(t *testing.T) {
	st := testStore(t)
	proj := t.TempDir()
	w, err := st.AddWorkspace("Proj", proj)
	if err != nil {
		t.Fatal(err)
	}
	a, err := st.AddAgentWithCLI(w.ID, "claude-code", "Atlas", "")
	if err != nil {
		t.Fatal(err)
	}
	codex, err := st.AddAgentWithCLI(w.ID, "codex", "Vega", "")
	if err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: agent-skill\ndescription: for one agent\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := t.TempDir()
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat", DataDir: data}).Handler)
	t.Cleanup(ts.Close)

	preview := func() any {
		_, p := skillsCall(t, "POST", ts.URL+"/api/skills/preview", map[string]string{"source": src})
		return p["id"]
	}
	code, res := skillsCall(t, "POST", ts.URL+"/api/skills", map[string]any{"preview": preview(), "path": "", "scope": "agent", "agent": a.ID})
	if code != 200 || res["status"] != "installed" {
		t.Fatalf("install: %d %v", code, res)
	}
	got, _ := st.GetAgent(a.ID)
	if len(got.Skills) != 1 || !strings.HasPrefix(got.Skills[0].Dir, filepath.Join(data, "skills", "cache")) || got.Skills[0].Source != src {
		t.Fatalf("agent skills %+v", got.Skills)
	}
	for _, p := range []string{filepath.Join(proj, ".agents"), filepath.Join(proj, "skills-lock.json")} {
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("an agent install wrote %s", p)
		}
	}
	code, res = skillsCall(t, "POST", ts.URL+"/api/skills", map[string]any{"preview": preview(), "path": "", "scope": "agent", "agent": a.ID})
	if code != 200 || res["status"] != "already" {
		t.Fatalf("again: %d %v", code, res)
	}

	r, err := http.Get(ts.URL + "/api/skills/report?cli=claude-code&workspace=" + w.ID + "&agent=" + a.ID)
	if err != nil {
		t.Fatal(err)
	}
	var rep skills.Report
	_ = json.NewDecoder(r.Body).Decode(&rep)
	r.Body.Close()
	found := false
	for _, row := range rep.Rows {
		found = found || (row.Scope == skills.Agent && row.Name == "agent-skill" && row.Status == skills.StatusLoaded)
	}
	if rep.Agent == nil || !found {
		t.Fatalf("report %+v", rep)
	}

	code, res = skillsCall(t, "POST", ts.URL+"/api/skills", map[string]any{"preview": preview(), "path": "", "scope": "agent", "agent": codex.ID})
	if code != http.StatusBadRequest {
		t.Fatalf("codex agent: %d %v", code, res)
	}

	code, res = skillsCall(t, "DELETE", ts.URL+"/api/skills", map[string]any{"name": "agent-skill", "scope": "agent", "agent": a.ID})
	if code != 200 || res["status"] != "removed" {
		t.Fatalf("remove: %d %v", code, res)
	}
	if got, _ := st.GetAgent(a.ID); len(got.Skills) != 0 {
		t.Fatalf("still listed %+v", got.Skills)
	}
	code, _ = skillsCall(t, "DELETE", ts.URL+"/api/skills", map[string]any{"name": "agent-skill", "scope": "agent", "agent": a.ID})
	if code != http.StatusNotFound {
		t.Fatalf("second remove: %d", code)
	}
}

// The toggle route writes the CLI's own key for a row the reader found, and
// refuses a path the report never listed (it cannot be made to write one).
func TestSkillsToggleRoute(t *testing.T) {
	st := testStore(t) // sets its own HOME; ours comes after
	home := t.TempDir()
	t.Setenv("HOME", home)
	proj := t.TempDir()
	dir := filepath.Join(proj, ".agents", "skills", "probe")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: probe\ndescription: probe\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := st.AddWorkspace("Proj", proj)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)
	post := func(body string) (int, map[string]any) {
		res, err := http.Post(ts.URL+"/api/skills/toggle", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		out := map[string]any{}
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}

	code, out := post(`{"cli":"codex","scope":"workspace","workspace":"` + w.ID + `","dir":"` + dir + `","enabled":false}`)
	if code != http.StatusOK || out["enabled"] != false {
		t.Fatalf("toggle: %d %v", code, out)
	}
	cfg, _ := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if !strings.Contains(string(cfg), filepath.Join(dir, "SKILL.md")) {
		t.Fatalf("codex config: %s (result %v)", cfg, out)
	}
	if code, _ := post(`{"cli":"codex","scope":"machine","dir":"/etc/passwd","enabled":false}`); code != http.StatusConflict {
		t.Fatalf("a path the report never listed: %d, want 409", code)
	}
	if code, _ := post(`{"cli":"pi","scope":"workspace","workspace":"` + w.ID + `","dir":"` + dir + `","enabled":false}`); code != http.StatusBadRequest {
		t.Fatalf("pi has no switch: %d, want 400", code)
	}
}

// An agent's own skills are checked against their source and updated in
// place: current, then behind once the source changes, updated to the new
// copy (a new digest, its own cached folder), current again; unreachable
// once the source is gone.
func TestSkillsAgentCheckAndUpdate(t *testing.T) {
	st := testStore(t)
	w, err := st.AddWorkspace("Proj", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, err := st.AddAgentWithCLI(w.ID, "claude-code", "Atlas", "")
	if err != nil {
		t.Fatal(err)
	}
	src := t.TempDir()
	write := func(desc string) {
		if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: agent-skill\ndescription: "+desc+"\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("first")
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat", DataDir: t.TempDir()}).Handler)
	t.Cleanup(ts.Close)
	_, p := skillsCall(t, "POST", ts.URL+"/api/skills/preview", map[string]string{"source": src})
	if code, res := skillsCall(t, "POST", ts.URL+"/api/skills", map[string]any{"preview": p["id"], "path": "", "scope": "agent", "agent": a.ID}); code != 200 {
		t.Fatalf("install: %d %v", code, res)
	}
	status := func() string {
		t.Helper()
		_, res := skillsCall(t, "GET", ts.URL+"/api/skills/updates?workspace="+w.ID+"&agent="+a.ID, nil)
		for _, r := range res["rows"].([]any) {
			row := r.(map[string]any)
			if row["scope"] == "agent" && row["name"] == "agent-skill" {
				return row["status"].(string)
			}
		}
		return "absent"
	}
	if got := status(); got != "current" {
		t.Fatalf("after install: %s", got)
	}
	before, _ := st.GetAgent(a.ID)
	write("second")
	if got := status(); got != "behind" {
		t.Fatalf("after a source change: %s", got)
	}
	code, res := skillsCall(t, "POST", ts.URL+"/api/skills/update", map[string]any{"name": "agent-skill", "scope": "agent", "agent": a.ID})
	if code != 200 || res["status"] != "updated" {
		t.Fatalf("update: %d %v", code, res)
	}
	after, _ := st.GetAgent(a.ID)
	if len(after.Skills) != 1 || after.Skills[0].Digest == before.Skills[0].Digest || after.Skills[0].Dir == before.Skills[0].Dir || after.Skills[0].Source != src {
		t.Fatalf("list after update %+v (was %+v)", after.Skills, before.Skills)
	}
	if b, err := os.ReadFile(filepath.Join(after.Skills[0].Dir, "SKILL.md")); err != nil || !strings.Contains(string(b), "second") {
		t.Fatalf("cached copy %q %v", b, err)
	}
	if got := status(); got != "current" {
		t.Fatalf("after update: %s", got)
	}
	if err := os.RemoveAll(src); err != nil {
		t.Fatal(err)
	}
	if got := status(); got != "unreachable" {
		t.Fatalf("source gone: %s", got)
	}
}
