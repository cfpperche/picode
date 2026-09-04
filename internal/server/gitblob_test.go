package server

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/gitgraph"
)

// graphBlobRepo is a repository with a committed asset ("v1\n" at HEAD),
// modified in the working tree ("v2\n") — the shape the graph's previews meet.
func graphBlobRepo(t *testing.T) string {
	t.Helper()
	dir := gitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "img.png"), []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit := func(args ...string) { gitRun(t, dir, args...) }
	runGit("add", ".")
	runGit("commit", "-m", "asset")
	if err := os.WriteFile(filepath.Join(dir, "img.png"), []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// One asset, three owners: the preview route is owner-scoped like every git
// route — each owner kind reads it through its own resolved folder.
func TestGitBlobServesAssetThroughEveryOwner(t *testing.T) {
	repo := graphBlobRepo(t)
	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(agent.WorkspaceID, "sh", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	hash := gitgraphRevParse(t, repo)

	for _, tc := range []struct {
		name string
		url  string
	}{
		{"agent", "/api/agents/" + agent.ID + "/git/blob?hash=" + hash + "&path=img.png"},
		{"agent via HEAD", "/api/agents/" + agent.ID + "/git/blob?hash=HEAD&path=img.png"},
		{"terminal", "/api/terminals/" + term.ID + "/git/blob?hash=" + hash + "&path=img.png"},
		{"workspace", "/api/workspaces/" + agent.WorkspaceID + "/git/blob?hash=" + hash + "&path=img.png"},
	} {
		res, err := ts.Client().Get(ts.URL + tc.url)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s: status = %d (%s)", tc.name, res.StatusCode, body)
		}
		if got := res.Header.Get("Content-Type"); got != "image/png" {
			t.Fatalf("%s: content-type = %q", tc.name, got)
		}
		if string(body) != "v1\n" {
			t.Fatalf("%s: body = %q, want the committed version", tc.name, body)
		}
	}
}

// A hand-written URL meets the same table the UI meets: every refusal is a
// status the browser can show as one line.
func TestGitBlobRefusals(t *testing.T) {
	repo := graphBlobRepo(t)
	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	base := "/api/agents/" + agent.ID + "/git/blob"
	hash := gitgraphRevParse(t, repo)

	cases := []struct {
		name string
		url  string
		want int
	}{
		{"missing params", base, http.StatusBadRequest},
		{"bad hash", base + "?hash=nope&path=img.png", http.StatusBadRequest},
		{"path escapes", base + "?hash=" + hash + "&path=../a", http.StatusBadRequest},
		{"unknown file", base + "?hash=" + hash + "&path=missing.png", http.StatusNotFound},
		{"unknown type", base + "?hash=" + hash + "&path=a", http.StatusUnsupportedMediaType},
	}
	for _, c := range cases {
		res, err := ts.Client().Get(ts.URL + c.url)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != c.want {
			t.Fatalf("%s: status = %d, want %d", c.name, res.StatusCode, c.want)
		}
	}
}

// The size cap must hold for committed assets too, not just working-tree ones.
func TestGitBlobCapsCommittedSize(t *testing.T) {
	repo := graphBlobRepo(t)
	st := testStore(t)
	_, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	ts := graphServer(t, st)
	old := gitgraph.MaxBlobBytes
	gitgraph.MaxBlobBytes = 1
	t.Cleanup(func() { gitgraph.MaxBlobBytes = old })

	res, err := ts.Client().Get(ts.URL + "/api/agents/" + agent.ID + "/git/blob?hash=HEAD&path=img.png")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", res.StatusCode)
	}
}

func gitgraphRevParse(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}
