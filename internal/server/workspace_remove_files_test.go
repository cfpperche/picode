package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func delReq(t *testing.T, ts *httptest.Server, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, ts.URL+url, nil)
	if err != nil {
		t.Fatal(err)
	}
	return do(t, ts.Client(), req)
}

func TestRemoveWorkspaceDeletesFiles(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	wk := addWorkspaceWithAgent(t, ts, "proj", dir)

	// Wrong confirmation: nothing is stopped, removed, or deleted.
	res := delReq(t, ts, "/api/workspaces/"+wk.ID+"?files=1&confirm=wrong")
	if res.StatusCode != 400 {
		t.Fatalf("wrong confirm = %d, want 400", res.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(dir, "f.txt")); err != nil {
		t.Fatalf("folder touched after refused confirm: %v", err)
	}
	if r2 := delReq(t, ts, "/api/workspaces/"+wk.ID+"?files=1"); r2.StatusCode != 400 {
		t.Fatalf("missing confirm = %d, want 400", r2.StatusCode)
	}

	// Right confirmation: record gone and folder gone.
	res = delReq(t, ts, "/api/workspaces/"+wk.ID+"?files=1&confirm=proj")
	if res.StatusCode != 204 {
		t.Fatalf("delete = %d, want 204", res.StatusCode)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("folder still on disk: %v", err)
	}
}

func TestRemoveWorkspaceWithoutFilesKeepsFolder(t *testing.T) {
	ts := newTestServer(t, "cat")
	dir := filepath.Join(t.TempDir(), "keep")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	wk := addWorkspaceWithAgent(t, ts, "keep", dir)
	res := delReq(t, ts, "/api/workspaces/"+wk.ID)
	if res.StatusCode != 204 {
		t.Fatalf("remove = %d, want 204", res.StatusCode)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("plain remove must keep the folder: %v", err)
	}
}

func TestRemoveWorkspaceRefusesHome(t *testing.T) {
	// The guard must refuse the home folder even with a matching
	// confirmation. HOME points at a sacrificial temp dir so a broken
	// guard deletes nothing that matters.
	home := t.TempDir()
	t.Setenv("HOME", home)
	ts := newTestServer(t, "cat")
	wk := addWorkspaceWithAgent(t, ts, "home", home)
	res := delReq(t, ts, "/api/workspaces/"+wk.ID+"?files=1&confirm="+filepath.Base(home))
	if res.StatusCode != 400 {
		t.Fatalf("home delete = %d, want 400", res.StatusCode)
	}
	if _, err := os.Stat(home); err != nil {
		t.Fatalf("home folder gone: %v", err)
	}
}

func TestCheckFolderDeletableRoot(t *testing.T) {
	if err := checkFolderDeletable("/", "/"); err == nil {
		t.Fatal("root must be refused")
	}
}
