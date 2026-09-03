package server

import (
	"sort"
	"testing"

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
