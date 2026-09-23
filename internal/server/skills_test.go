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
