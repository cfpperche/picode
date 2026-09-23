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
