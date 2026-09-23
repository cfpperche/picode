package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The Instructions route reads a workspace's instruction files for every
// CLI, confined to the registered folder, and never returns a file's text.
func TestWorkspaceInstructions(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	for rel, body := range map[string]string{"AGENTS.md": "secret-agents-text\n", "CLAUDE.md": "claude\n", "pkg/AGENTS.md": "pkg\n"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(proj, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(proj, rel), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("add workspace = %d", res.StatusCode)
	}
	var wk workspaceView
	_ = json.NewDecoder(res.Body).Decode(&wk)

	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/workspaces/"+wk.ID+"/instructions"))
	if got.StatusCode != http.StatusOK {
		t.Fatalf("instructions = %d", got.StatusCode)
	}
	var raw json.RawMessage
	_ = json.NewDecoder(got.Body).Decode(&raw)
	var rep struct {
		Files []struct {
			Path  string                       `json:"path"`
			Cells map[string]map[string]string `json:"cells"`
		} `json:"files"`
		CLIs []struct {
			ID string `json:"id"`
		} `json:"clis"`
		Folders []string `json:"folders"`
	}
	if err := json.Unmarshal(raw, &rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.CLIs) != 9 {
		t.Fatalf("clis = %d, want the nine", len(rep.CLIs))
	}
	cells := map[string]map[string]map[string]string{}
	for _, f := range rep.Files {
		cells[f.Path] = f.Cells
	}
	if cells["AGENTS.md"]["claude-code"]["status"] != "shadowed" || cells["AGENTS.md"]["codex"]["status"] != "reads" {
		t.Fatalf("AGENTS.md cells = %+v", cells["AGENTS.md"])
	}
	if cells["pkg/AGENTS.md"] == nil || len(rep.Folders) != 2 {
		t.Fatalf("nested file or folders missing: %+v %v", rep.Files, rep.Folders)
	}
	if strings.Contains(string(raw), "secret-agents-text") {
		t.Fatal("the report carried a file's text")
	}

	for path, want := range map[string]int{
		"/api/workspaces/" + wk.ID + "/instructions?start=pkg":   http.StatusOK,
		"/api/workspaces/" + wk.ID + "/instructions?start=../":   http.StatusBadRequest,
		"/api/workspaces/" + wk.ID + "/instructions?start=/etc":  http.StatusBadRequest,
		"/api/workspaces/" + wk.ID + "/instructions?start=gone":  http.StatusNotFound,
		"/api/workspaces/nope/instructions":                      http.StatusNotFound,
		"/api/workspaces/ws_free/instructions":                   http.StatusNotFound,
		"/api/workspaces/" + wk.ID + "/instructions?root=/other": http.StatusConflict,
	} {
		if got := do(t, ts.Client(), mustGet(t, ts.URL+path)); got.StatusCode != want {
			t.Errorf("%s = %d, want %d", path, got.StatusCode, want)
		}
	}
}
