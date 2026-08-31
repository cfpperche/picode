package server

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestRelUnderCwd(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	abs, rel, err := relUnderCwd(root, "src")
	if err != nil || rel != "src" {
		t.Fatalf("src = %s %s %v", abs, rel, err)
	}
	if _, _, err := relUnderCwd(root, "../secret"); err == nil {
		t.Fatal("escaped")
	}
	if _, _, err := relUnderCwd(root, "src/../../etc"); err == nil {
		t.Fatal("nested escape")
	}
	abs2, rel2, err := relUnderCwd(root, filepath.Join(root, "src"))
	if err != nil || rel2 != "src" || abs2 != abs {
		t.Fatalf("abs in cwd %s %s %v", abs2, rel2, err)
	}
}

func TestRelUnderCwdHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cwd := filepath.Join(home, "proj")
	if err := os.MkdirAll(filepath.Join(cwd, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, rel, err := relUnderCwd(cwd, "~/proj/web")
	if err != nil || rel != "web" {
		t.Fatalf("tilde = %s %v", rel, err)
	}
	if _, _, err := relUnderCwd(cwd, "~/other"); err == nil {
		t.Fatal("tilde outside")
	}
}

func TestBrowseAndReadImage(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "readme.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(root, "src", "dot.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	page, err := browseAgentDir(root, "")
	if err != nil || !page["cwdOk"].(bool) {
		t.Fatalf("browse: %v %+v", err, page)
	}
	dirs := page["dirs"].([]browseHit)
	files := page["files"].([]browseHit)
	if len(dirs) != 1 || dirs[0].Name != "src" {
		t.Fatalf("dirs=%+v", dirs)
	}
	if len(files) != 1 || files[0].Name != "readme.md" {
		t.Fatalf("files=%+v", files)
	}
	if _, err := readAgentImage(root, "readme.md"); err == nil {
		t.Fatal("non-image")
	}
	got, err := readAgentImage(root, "src/dot.png")
	if err != nil || got["mime"] != "image/png" {
		t.Fatalf("png: %v %+v", err, got)
	}
	if _, err := readAgentImage(root, "../dot.png"); err == nil {
		t.Fatal("escaped read")
	}
}

func TestBrowseHTTP(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "a.go"), []byte("pkg"), 0o644); err != nil {
		t.Fatal(err)
	}
	wk := addWorkspaceWithAgent(t, ts, "App", proj)
	id := wk.Agent.ID
	got := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+id+"/browse"))
	if got.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", got.StatusCode)
	}
	bad := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+id+"/browse?dir=../x"))
	if bad.StatusCode != http.StatusBadRequest {
		t.Fatalf("escape=%d", bad.StatusCode)
	}
}
