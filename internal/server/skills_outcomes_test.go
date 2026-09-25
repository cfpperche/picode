package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
)

// ADR-0196 slice 6: a launch records the skills it loaded — the agent's own
// and the folders' — and an isolated agent records only its own.
func TestLaunchRecordsItsSkills(t *testing.T) {
	for _, isolated := range []bool{false, true} {
		deps, cwd, launch := ompLaunchFixture(t, clilaunch.Config{}, nil, isolated)
		// A folder skill in the workspace Omp reads (.agents/skills).
		dir := filepath.Join(cwd, ".agents", "skills", "shared")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: shared\ndescription: d\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		a, _ := deps.Store.AgentByTerminal(launch.TerminalID)
		agentSkillFixture(t, deps.Store, a.ID, filepath.Join(deps.DataDir, "skills", "cache"), "own")
		prepared, err := prepareCLITerminal(deps, cwd, launch)
		if err != nil {
			t.Fatal(err)
		}
		prepared.discard()
		names := map[string]string{}
		for _, sk := range prepared.snapshot.Skills {
			names[sk.Name] = sk.Scope
		}
		if names["own"] != "agent" {
			t.Fatalf("isolated=%v: %+v", isolated, prepared.snapshot.Skills)
		}
		if _, ok := names["shared"]; ok == isolated {
			t.Fatalf("isolated=%v: folder skill recorded=%v: %+v", isolated, ok, prepared.snapshot.Skills)
		}
	}
}

func TestAgentExitSkillsRoute(t *testing.T) {
	st := testStore(t)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)
	a, _ := st.AddAgent(store.FreeWorkspaceID, "a", "")
	if _, err := st.RemoveAgentWithExit(a.ID, store.ExitInput{Label: store.ExitLabel{Outcome: store.ExitResolved}}); err != nil {
		t.Fatal(err)
	}
	res, err := http.Get(ts.URL + "/api/agent-exits/skills?range=all")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var got store.ExitSkillStats
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil || res.StatusCode != 200 {
		t.Fatalf("%d %v", res.StatusCode, err)
	}
	if got.Unrecorded != 1 || got.Recorded != 0 || got.Rows == nil {
		t.Fatalf("%+v", got)
	}
}

// Promote: the agent's skill lands in its workspace's .agents/skills and
// leaves the agent's own list; a free agent has no project to promote into.
func TestSkillsPromoteRoute(t *testing.T) {
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
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("---\nname: tried\ndescription: works\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat", DataDir: t.TempDir()}).Handler)
	t.Cleanup(ts.Close)
	_, p := skillsCall(t, "POST", ts.URL+"/api/skills/preview", map[string]string{"source": src})
	if code, res := skillsCall(t, "POST", ts.URL+"/api/skills", map[string]any{"preview": p["id"], "path": "", "scope": "agent", "agent": a.ID}); code != 200 {
		t.Fatalf("add: %d %v", code, res)
	}
	code, res := skillsCall(t, "POST", ts.URL+"/api/skills/promote", map[string]any{"agent": a.ID, "name": "tried"})
	if code != 200 || res["status"] != "installed" {
		t.Fatalf("promote: %d %v", code, res)
	}
	if _, err := os.Stat(filepath.Join(proj, ".agents/skills/tried/SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if got, _ := st.GetAgent(a.ID); len(got.Skills) != 0 {
		t.Fatalf("still the agent's own: %+v", got.Skills)
	}
	if code, _ := skillsCall(t, "POST", ts.URL+"/api/skills/promote", map[string]any{"agent": a.ID, "name": "tried"}); code != http.StatusNotFound {
		t.Fatalf("second promote: %d", code)
	}
}
