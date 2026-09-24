package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func TestWorkspaceForCwd(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "project")
	child := filepath.Join(parent, "packages", "app")
	outside := filepath.Join(root, "project-other")
	for _, dir := range []string{child, outside} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "project-link")
	if err := os.Symlink(parent, link); err != nil {
		t.Fatal(err)
	}
	rows := []store.Workspace{{ID: "parent", Path: parent}, {ID: "child", Path: child}}
	for _, tc := range []struct{ name, path, want string }{
		{"root", parent, "parent"},
		{"nested workspace", child, "child"},
		{"subdirectory", filepath.Join(parent, "src"), "parent"},
		{"symlink", link, "parent"},
		{"sibling prefix", outside, ""},
		{"unknown", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := workspaceForCwd(tc.path, rows); got != tc.want {
				t.Fatalf("owner = %q, want %q", got, tc.want)
			}
		})
	}
	duplicate := append(rows, store.Workspace{ID: "other", Path: link})
	if got := workspaceForCwd(parent, duplicate); got != "" {
		t.Fatalf("ambiguous owner = %q", got)
	}
}

func TestWorkspaceStatsScopesWholeWindow(t *testing.T) {
	root := withTestSessionRoot(t)
	project := t.TempDir()
	other := t.TempDir()
	dir := filepath.Join(root, "--test--")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct{ name, cwd, cost string }{
		{"in.jsonl", project, "0.5"}, {"out.jsonl", other, "9.0"},
	} {
		body := `{"type":"session","id":"s","cwd":"` + row.cwd + `"}` + "\n" +
			`{"type":"message","message":{"role":"assistant","provider":"xai","model":"grok","usage":{"cost":{"total":` + row.cost + `}}}}` + "\n"
		if err := os.WriteFile(filepath.Join(dir, row.name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ws, err := st.AddWorkspace("Project", project)
	if err != nil {
		t.Fatal(err)
	}
	request := func(id string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/api/workspaces/"+id+"/stats?range=all", nil)
		r.SetPathValue("id", id)
		w := httptest.NewRecorder()
		handleWorkspaceStats(Deps{Store: st})(w, r)
		return w
	}
	w := request(ws.ID)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var got workspaceStatsView
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Range != "all" || got.Current.Cost != 0.5 || got.Current.Messages != 1 || got.Current.Sessions != 1 || len(got.Series) == 0 {
		t.Fatalf("scope = %+v", got)
	}
	if w := request("missing"); w.Code != http.StatusNotFound {
		t.Fatalf("missing workspace status = %d", w.Code)
	}
}
