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
	prev, wtKnown := gitWatchTick(deps, nil, nil, true)
	if len(got) != 1 {
		t.Fatalf("first pass published %d event(s), want 1: %+v", len(got), got)
	}
	if got[0].Type != "git.updated" {
		t.Fatalf("first pass published %q, want git.updated", got[0].Type)
	}

	// Second pass on an unchanged store: nothing moved, nothing said.
	got = nil
	prev, wtKnown = gitWatchTick(deps, prev, wtKnown, true)
	if len(got) != 0 {
		t.Fatalf("an unchanged pass published %+v, want nothing", got)
	}

	// The workspace is removed. This pass used to panic here.
	removed, err := st.RemoveWorkspace(ws.ID)
	if err != nil || !removed {
		t.Fatalf("remove workspace: removed=%v err=%v", removed, err)
	}
	got = nil
	prev, wtKnown = gitWatchTick(deps, prev, wtKnown, true)
	if len(got) != 0 {
		t.Fatalf("a removed workspace published %+v, want nothing (the removed event carries it)", got)
	}
	if len(prev) != 0 {
		t.Fatalf("watched paths after removal = %v, want none", prev)
	}

	// And the next pass is a clean slate: the path is not remembered as a
	// change to announce.
	got = nil
	gitWatchTick(deps, prev, wtKnown, true)
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
	if _, err := st.AddWorkspace("watched", repo); err != nil {
		t.Fatalf("add workspace: %v", err)
	}
	f := &feed.Feed{Store: st}
	var got []store.Event
	f.Listen(func(ev store.Event) { got = append(got, ev) })
	deps := Deps{Store: st, Feed: f}

	prev, wtKnown := gitWatchTick(deps, nil, nil, true)
	if paths := gitUpdatedPaths(t, got); !(paths[canonDir(repo)] && paths[canonDir(side)]) {
		t.Fatalf("first pass published %v, want the anchor and its sibling", paths)
	}
	// The sibling speaks for no pill: carrying the anchor's ids would let
	// the fleet reducer overwrite the workspace's and agents' pills with
	// the sibling's branch. Path-matching readers (the Inspector's
	// followed groups) still match on the path.
	sawSibling := false
	for _, ev := range got {
		var data struct {
			Path         string   `json:"path"`
			WorkspaceIDs []string `json:"workspaceIds"`
			AgentIDs     []string `json:"agentIds"`
		}
		_ = json.Unmarshal(ev.Data, &data)
		if sameDir(data.Path, side) {
			sawSibling = true
			if len(data.WorkspaceIDs) != 0 || len(data.AgentIDs) != 0 {
				t.Fatalf("sibling event carries ids %+v/%+v, want none", data.WorkspaceIDs, data.AgentIDs)
			}
		}
	}
	if !sawSibling {
		t.Fatalf("no sibling event among %+v", got)
	}

	got = nil
	prev, wtKnown = gitWatchTick(deps, prev, wtKnown, true)
	if len(got) != 0 {
		t.Fatalf("an unchanged pass published %+v, want nothing", got)
	}

	if err := os.WriteFile(filepath.Join(side, "messy.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = nil
	gitWatchTick(deps, prev, wtKnown, true)
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

// A checkout watched under two spellings (an agent bound inside it through
// a symlink) still publishes under the name git gives it: path-matching
// readers know the worktree under the gitstatus spelling, not the
// registered one.
func TestGitWatchTickWorktreeAliasUnderSecondSpelling(t *testing.T) {
	st := testStore(t)
	repo := gitRepo(t)
	// The real checkout is built under a resolved base. This test is about
	// one deliberate alias — `link` — and on a host where TMPDIR is itself
	// a symlink (macOS: /var/folders/… is /private/var/folders/…) there is
	// a second, accidental one: git records the worktree under the resolved
	// name while the test holds the spelling it wrote, and the assertion
	// below fails on two spellings of the same directory. Resolving the
	// base removes the alias nobody asked for and leaves the one under test.
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	side := filepath.Join(base, "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(side, link); err != nil {
		t.Fatal(err)
	}
	ws, err := st.AddWorkspace("watched", repo)
	if err != nil {
		t.Fatalf("add workspace: %v", err)
	}
	if _, err := st.AddAgent(ws.ID, "Side", link); err != nil {
		t.Fatalf("add agent: %v", err)
	}
	f := &feed.Feed{Store: st}
	var got []store.Event
	f.Listen(func(ev store.Event) { got = append(got, ev) })
	deps := Deps{Store: st, Feed: f}

	_, _ = gitWatchTick(deps, nil, nil, true)
	// Raw spellings, not canonical dirs: both names must be published,
	// and canonDir would collapse them into one key.
	raw := map[string]bool{}
	for _, ev := range got {
		var data struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			t.Fatal(err)
		}
		raw[data.Path] = true
	}
	if !raw[side] {
		t.Fatalf("first pass published %v, want the git spelling too", raw)
	}
	if !raw[link] {
		t.Fatalf("first pass published %v, want the anchor spelling too", raw)
	}
}

// The worktree set is listed on listing passes only; an anchor with no
// known worktrees still lists every tick, so a fresh `git worktree add`
// is watched from its next tick.
func TestGitWatchTickCachesWorktreeList(t *testing.T) {
	st := testStore(t)
	repo := gitRepo(t)
	if _, err := st.AddWorkspace("watched", repo); err != nil {
		t.Fatalf("add workspace: %v", err)
	}
	f := &feed.Feed{Store: st}
	var got []store.Event
	f.Listen(func(ev store.Event) { got = append(got, ev) })
	deps := Deps{Store: st, Feed: f}

	prev, wtKnown := gitWatchTick(deps, nil, nil, false)
	if len(wtKnown[canonDir(repo)]) != 0 && len(wtKnown[repo]) != 0 {
		t.Fatalf("empty anchor cache = %+v, want no worktrees", wtKnown)
	}

	side := filepath.Join(t.TempDir(), "side")
	gitRun(t, repo, "worktree", "add", "-b", "side", side)
	got = nil
	prev, wtKnown = gitWatchTick(deps, prev, wtKnown, false)
	if paths := gitUpdatedPaths(t, got); !paths[canonDir(side)] {
		t.Fatalf("fresh checkout published %v, want the sibling at once", paths)
	}

	docs := filepath.Join(t.TempDir(), "docs")
	gitRun(t, repo, "worktree", "add", "-b", "docs", docs)
	got = nil
	prev, wtKnown = gitWatchTick(deps, prev, wtKnown, false)
	if paths := gitUpdatedPaths(t, got); paths[canonDir(docs)] {
		t.Fatalf("cached pass published %v, want no listing without relist", paths)
	}
	got = nil
	_, _ = gitWatchTick(deps, prev, wtKnown, true)
	if paths := gitUpdatedPaths(t, got); !paths[canonDir(docs)] {
		t.Fatalf("listing pass published %v, want the new sibling", paths)
	}
}
