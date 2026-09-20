package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/store"
)

func strptr(s string) *string { return &s }

func TestGitDirs(t *testing.T) {
	ws := []store.Workspace{
		{ID: "ws-main", Path: "/repo/main"},
		{ID: "ws_free", Path: store.FreeWorkspacePath}, // the reserved unbound workspace is not a folder to watch
		{ID: "ws-empty", Path: ""},                     // no path, nothing to inspect
	}
	agents := []store.Agent{
		{ID: "a-ws", WorkspaceID: "ws-main"},                                                   // rides the workspace path
		{ID: "a-own", WorkspaceID: "ws-main", WorkPath: strptr("/repo/own")},                   // workPath wins — and must NOT carry the workspace id
		{ID: "a-free-own", WorkspaceID: store.FreeWorkspaceID, WorkPath: strptr("/repo/free")}, // free agents are watched on workPath only
		{ID: "a-free-none", WorkspaceID: store.FreeWorkspaceID},                                // no workPath, no pill, not watched
		{ID: "a-lost", WorkspaceID: "ws-gone"},                                                 // unknown workspace, skipped
	}
	got := gitDirs(ws, agents)
	sort.Slice(got, func(i, j int) bool {
		if got[i].path != got[j].path {
			return got[i].path < got[j].path
		}
		return got[i].agentID < got[j].agentID
	})
	want := []gitDir{
		{path: "/repo/free", agentID: "a-free-own"},
		{path: "/repo/main", workspaceID: "ws-main"},
		{path: "/repo/main", workspaceID: "ws-main", agentID: "a-ws"},
		{path: "/repo/own", agentID: "a-own"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %+v want %+v", got, want)
		}
	}
}

func TestDiffGit(t *testing.T) {
	on := map[string]string{"/a": "main\x00\x000", "/b": "dev\x00\x002"}
	flipped := map[string]string{"/a": "main\x00\x001", "/b": "dev\x00\x002"}

	tests := []struct {
		name string
		prev map[string]string
		cur  map[string]string
		want []string
	}{
		{"no change", on, map[string]string{"/a": "main\x00\x000", "/b": "dev\x00\x002"}, nil},
		{"dirty flip", on, flipped, []string{"/a"}},
		{"branch flip", on, map[string]string{"/a": "feat\x00\x000", "/b": "dev\x00\x002"}, []string{"/a"}},
		{"path appears", nil, map[string]string{"/a": "main\x00\x000"}, []string{"/a"}},
		{"path disappears", on, map[string]string{"/a": "main\x00\x000"}, []string{"/b"}},
		{"dir left the repo", on, map[string]string{"/a": "main\x00\x000", "/b": ""}, []string{"/b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffGit(tt.prev, tt.cur)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v want %v", got, tt.want)
				}
			}
		})
	}
}

// TestGitWatchTickSurvivesARemovedWorkspace is the regression test for the
// 2026-09-13 crash: the pass that notices a workspace is gone has no group for
// its path, and reading one anyway killed the whole daemon on every "Remove
// workspace" in the UI. The removal is the durable event's business (ADR-0048),
// so the pass says nothing about a folder nobody reads.
func TestGitWatchTickSurvivesARemovedWorkspace(t *testing.T) {
	st := testStore(t)
	dir := gitRepo(t)
	f := &feed.Feed{Store: st}
	var got []store.Event
	f.Listen(func(ev store.Event) { got = append(got, ev) })
	deps := Deps{Store: st, Feed: f}

	ws, err := st.AddWorkspace("watched", dir)
	if err != nil {
		t.Fatalf("add workspace: %v", err)
	}

	// First pass: the new folder's git state is news.
	prev := gitWatchTick(deps, nil)
	if len(got) != 1 {
		t.Fatalf("first pass published %d event(s), want 1: %+v", len(got), got)
	}
	if got[0].Type != "git.updated" {
		t.Fatalf("first pass published %q, want git.updated", got[0].Type)
	}

	// Second pass on an unchanged store: nothing moved, nothing said.
	got = nil
	prev = gitWatchTick(deps, prev)
	if len(got) != 0 {
		t.Fatalf("an unchanged pass published %+v, want nothing", got)
	}

	// The workspace is removed. This pass used to panic here.
	removed, err := st.RemoveWorkspace(ws.ID)
	if err != nil || !removed {
		t.Fatalf("remove workspace: removed=%v err=%v", removed, err)
	}
	got = nil
	prev = gitWatchTick(deps, prev)
	if len(got) != 0 {
		t.Fatalf("a removed workspace published %+v, want nothing (the removed event carries it)", got)
	}
	if len(prev) != 0 {
		t.Fatalf("watched paths after removal = %v, want none", prev)
	}

	// And the next pass is a clean slate: the path is not remembered as a
	// change to announce.
	got = nil
	gitWatchTick(deps, prev)
	if len(got) != 0 {
		t.Fatalf("the pass after removal published %+v, want nothing", got)
	}
}

// Linked worktrees of watched repositories are watched too: the Inspector
// follows dirty siblings, so a sibling's dirty flip must arrive as
// git.updated naming the sibling — speaking with its anchor's group.
func TestGitWatchTickPublishesLinkedWorktreeChanges(t *testing.T) {
	st := testStore(t)
	repo := gitRepo(t)
	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	ws, err := st.AddWorkspace("watched", repo)
	if err != nil {
		t.Fatalf("add workspace: %v", err)
	}
	f := &feed.Feed{Store: st}
	var got []store.Event
	f.Listen(func(ev store.Event) { got = append(got, ev) })
	deps := Deps{Store: st, Feed: f}

	prev := gitWatchTick(deps, nil)
	if paths := gitUpdatedPaths(t, got); !(paths[canonDir(repo)] && paths[canonDir(side)]) {
		t.Fatalf("first pass published %v, want the anchor and its sibling", paths)
	}
	// The sibling speaks with the anchor's group.
	for _, ev := range got {
		var data struct {
			Path         string   `json:"path"`
			WorkspaceIDs []string `json:"workspaceIds"`
		}
		_ = json.Unmarshal(ev.Data, &data)
		if sameDir(data.Path, side) && (len(data.WorkspaceIDs) != 1 || data.WorkspaceIDs[0] != ws.ID) {
			t.Fatalf("sibling event carries %+v, want the anchor workspace", data.WorkspaceIDs)
		}
	}

	got = nil
	prev = gitWatchTick(deps, prev)
	if len(got) != 0 {
		t.Fatalf("an unchanged pass published %+v, want nothing", got)
	}

	if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = nil
	gitWatchTick(deps, prev)
	paths := gitUpdatedPaths(t, got)
	if len(paths) != 1 || !paths[canonDir(side)] {
		t.Fatalf("sibling dirty flip published %v, want only the sibling", paths)
	}
}

func gitUpdatedPaths(t *testing.T, evs []store.Event) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, ev := range evs {
		if ev.Type != "git.updated" {
			t.Fatalf("published %q, want git.updated", ev.Type)
		}
		var data struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			t.Fatal(err)
		}
		out[canonDir(data.Path)] = true
	}
	return out
}
