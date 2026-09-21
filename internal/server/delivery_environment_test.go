package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/delivery"
	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// D2 (ADR-0170): the environment read and its only write — the store binding.
func newDeliveryServer(t *testing.T) (*httptest.Server, *store.Store, string) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	data := t.TempDir()
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), AgentCmd: "cat", DataDir: data}).Handler)
	t.Cleanup(ts.Close)
	return ts, st, data
}

func deliveryRead(t *testing.T, ts *httptest.Server, owner, query string) map[string]any {
	t.Helper()
	res, err := http.Get(ts.URL + "/api/" + owner + "/delivery?" + query + "&fresh=1")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("read %s: %d %v", owner, res.StatusCode, out)
	}
	return out
}

func environmentOf(t *testing.T, out map[string]any) map[string]any {
	t.Helper()
	envs, _ := out["environments"].([]any)
	if len(envs) != 1 {
		t.Fatalf("environments = %v, want exactly one", out["environments"])
	}
	return envs[0].(map[string]any)
}

func changeByBranch(t *testing.T, out map[string]any, branch string) map[string]any {
	t.Helper()
	for _, item := range out["changes"].([]any) {
		c := item.(map[string]any)
		if c["branch"] == branch {
			return c
		}
	}
	t.Fatalf("no change for %s in %v", branch, out["changes"])
	return nil
}

// stubIdentity replaces the running artifact for one test.
func stubIdentity(t *testing.T, revision string) {
	t.Helper()
	previous := instanceIdentity
	instanceIdentity = func() delivery.Running {
		return delivery.Running{Display: "0.4.0+test", Revision: revision, Boot: "boot-test", Responding: true}
	}
	t.Cleanup(func() { instanceIdentity = previous })
}

func TestDeliveryObserverRoute(t *testing.T) {
	ts, st, _ := newDeliveryServer(t)
	repo := gitRepo(t)
	ws, err := st.AddWorkspace("d2", repo)
	if err != nil {
		t.Fatal(err)
	}
	a, err := st.AddAgentWithCLI(ws.ID, "pi", "Atlas", "")
	if err != nil {
		t.Fatal(err)
	}
	free, err := st.CreateTerminalIn(store.FreeWorkspaceID, "loose", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	plain := t.TempDir() // a folder that is not a repository
	plainWS, err := st.AddWorkspace("plain", plain)
	if err != nil {
		t.Fatal(err)
	}
	rows := []struct {
		name string
		path string
		body string
		want int
	}{
		{"bind through the workspace", "/api/workspaces/" + ws.ID + "/delivery/observer", `{"observer":"picode-self"}`, 200},
		{"bind through an agent", "/api/agents/" + a.ID + "/delivery/observer", `{"observer":"picode-self"}`, 200},
		{"unknown observer", "/api/workspaces/" + ws.ID + "/delivery/observer", `{"observer":"someone-else"}`, 400},
		{"unknown field", "/api/workspaces/" + ws.ID + "/delivery/observer", `{"observer":"none","path":"/etc"}`, 400},
		{"not a repository", "/api/workspaces/" + plainWS.ID + "/delivery/observer", `{"observer":"picode-self"}`, 404},
		{"no project folder", "/api/terminals/" + free.ID + "/delivery/observer", `{"observer":"picode-self"}`, 404},
		{"missing workspace", "/api/workspaces/absent/delivery/observer", `{"observer":"picode-self"}`, 404},
		{"unbind", "/api/workspaces/" + ws.ID + "/delivery/observer", `{"observer":"none"}`, 200},
	}
	for _, c := range rows {
		t.Run(c.name, func(t *testing.T) {
			res, out := inboxPost(t, ts, c.path, c.body)
			if res.StatusCode != c.want {
				t.Fatalf("status = %d (%v), want %d", res.StatusCode, out, c.want)
			}
			if c.want != 200 {
				return
			}
			bound := c.body == `{"observer":"picode-self"}`
			row, ok, err := st.GetDeliveryObserver(ws.ID)
			if err != nil {
				t.Fatal(err)
			}
			if ok != bound {
				t.Fatalf("binding present = %v, want %v", ok, bound)
			}
			if bound && row.Repo != gitgraph.Key(repo) {
				t.Fatalf("binding repo = %q, want the resolved key %q", row.Repo, gitgraph.Key(repo))
			}
		})
	}
}

// O16/O17/O18/O20 as the read route answers them: the binding decides whether
// anything is observed, and the running revision's membership decides what is
// published. Nothing here is inferred from another workspace.
func TestDeliveryEnvironmentRead(t *testing.T) {
	ts, st, data := newDeliveryServer(t)
	repo := gitRepo(t)
	initial := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	gitRun(t, repo, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repo, "f"), []byte("f"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "feature work")
	feature := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	gitRun(t, repo, "checkout", "main")
	gitRun(t, repo, "merge", "--ff-only", "feature")
	ws, err := st.AddWorkspace("d2", repo)
	if err != nil {
		t.Fatal(err)
	}
	// A second workspace on the *same* repository: a sibling worktree, which
	// is how two projects share one repository key.
	sibling := filepath.Join(t.TempDir(), "wt")
	gitRun(t, repo, "worktree", "add", sibling, "feature")
	other, err := st.AddWorkspace("sibling", sibling)
	if err != nil {
		t.Fatal(err)
	}
	if gitgraph.Key(sibling) != gitgraph.Key(repo) {
		t.Fatalf("fixture: sibling key %q != %q", gitgraph.Key(sibling), gitgraph.Key(repo))
	}
	owner := "workspaces/" + ws.ID
	env := func() map[string]any { return environmentOf(t, deliveryRead(t, ts, owner, "target=main")) }
	status := func() (string, string) { e := env(); return e["status"].(string), e["reasonCode"].(string) }

	if s, r := status(); s != "unconfigured" || r != "no-observer" {
		t.Fatalf("unconfigured read = %s/%s", s, r)
	}
	if c := changeByBranch(t, deliveryRead(t, ts, owner, "target=main"), "feature"); c["publication"] != "unknown" {
		t.Fatalf("publication without an environment = %v", c["publication"])
	}

	if _, out := inboxPost(t, ts, "/api/workspaces/"+ws.ID+"/delivery/observer", `{"observer":"picode-self"}`); out["observer"] != "picode-self" {
		t.Fatalf("bind = %v", out)
	}
	// Another workspace on the same repository is not a substitute for this
	// owner's own binding: it answers unconfigured, not "observed by proxy".
	if s, _ := environmentOf(t, deliveryRead(t, ts, "workspaces/"+other.ID, "target=main"))["status"].(string); s != "unconfigured" {
		t.Fatalf("a sibling workspace must not lend its binding, got %s", s)
	}
	// A binding recorded against another repository, while this one claims the
	// repository, is a conflict rather than an arbitrary pick.
	if err := st.SetDeliveryObserver(ws.ID, filepath.Join(strings.TrimSuffix(repo, "/"), "elsewhere", ".git"), "picode-self"); err != nil {
		t.Fatal(err)
	}
	if err := st.SetDeliveryObserver(other.ID, gitgraph.Key(repo), "picode-self"); err != nil {
		t.Fatal(err)
	}
	if s, r := status(); s != "conflict" || r != "binding-conflict" {
		rows, _ := st.ListDeliveryObservers(gitgraph.Key(repo))
		own, ok, _ := st.GetDeliveryObserver(ws.ID)
		t.Fatalf("contradictory bindings = %s/%s (ws binding %+v ok=%v; repo rows %+v)", s, r, own, ok, rows)
	}
	if err := st.SetDeliveryObserver(other.ID, gitgraph.Key(repo), "none"); err != nil {
		t.Fatal(err)
	}
	if err := st.SetDeliveryObserver(ws.ID, gitgraph.Key(repo), "picode-self"); err != nil {
		t.Fatal(err)
	}

	stubIdentity(t, initial)
	out := deliveryRead(t, ts, owner, "target=main")
	e := environmentOf(t, out)
	if e["status"] != "known" || e["revision"] != initial {
		t.Fatalf("environment = %v (initial=%q)", e, initial)
	}
	if got := e["unpublished"].(map[string]any)["count"].(float64); got != 1 {
		t.Fatalf("unpublished = %v, want 1", e["unpublished"])
	}
	if c := changeByBranch(t, out, "feature"); c["publication"] != "not-published" {
		t.Fatalf("feature publication = %v", c["publication"])
	}

	// The artifact contains the change: it is published, and the set is empty.
	stubIdentity(t, feature)
	out = deliveryRead(t, ts, owner, "target=main")
	if e = environmentOf(t, out); e["unpublished"].(map[string]any)["count"].(float64) != 0 {
		t.Fatalf("published set = %v", e["unpublished"])
	}
	if c := changeByBranch(t, out, "feature"); c["publication"] != "published" {
		t.Fatalf("feature publication = %v", c["publication"])
	}

	// A revision this repository does not have: unknown, never "not published".
	stubIdentity(t, strings.Repeat("d", 40))
	if s, r := status(); s != "unknown" || r != "revision-unmapped" {
		t.Fatalf("unmapped revision = %s/%s", s, r)
	}

	// A passing receipt for a dirty build cannot confirm membership.
	writeDeployment(t, data, deploymentRecord{
		id: "aaaaaaaa-1111-4111-8111-111111111111", repo: gitgraph.Key(repo), outcome: "passed",
		built: feature, observed: feature, clean: false,
	})
	stubIdentity(t, feature)
	if s, r := status(); s != "unknown" || r != "artifact-dirty" {
		t.Fatalf("dirty artifact = %s/%s, want unknown/artifact-dirty", s, r)
	}
}

type deploymentRecord struct {
	id, repo, outcome, built, observed string
	clean                              bool
}

func writeDeployment(t *testing.T, dataDir string, r deploymentRecord) {
	t.Helper()
	dir := filepath.Join(dataDir, "var", "delivery")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"schemaVersion": 1, "id": r.id, "kind": "deploy", "repositoryKey": r.repo,
		"startedAt": "2026-01-01T00:00:00Z", "finishedAt": "2026-01-01T00:00:30Z", "outcome": r.outcome,
		"builtRevision": r.built, "observedRevision": r.observed, "clean": r.clean}
	raw, _ := json.Marshal(body)
	if err := os.WriteFile(filepath.Join(dir, r.id+".json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}
