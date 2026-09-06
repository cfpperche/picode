package server

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGh puts a scripted `gh` first on PATH. FAKE_GH_AUTH=no fails `auth
// status`; FAKE_GH_PR selects the `pr view` answer: none, noremote, or the
// JSON file named by FAKE_GH_JSON. git stays reachable through the real PATH.
func fakeGh(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	script := `#!/bin/sh
case "$1 $2" in
  "auth status")
    if [ "$FAKE_GH_AUTH" = "no" ]; then echo "You are not logged into any GitHub hosts. Run gh auth login to authenticate." >&2; exit 1; fi
    exit 0 ;;
  "pr view")
    case "$FAKE_GH_PR" in
      none) echo "no pull requests found for branch \"main\"" >&2; exit 1 ;;
      noremote) echo "could not determine base repo: no git remotes found" >&2; exit 1 ;;
      *) cat "$FAKE_GH_JSON"; exit 0 ;;
    esac ;;
esac
echo "fake gh: unexpected $*" >&2
exit 2
`
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GH_AUTH", "yes")
	t.Setenv("FAKE_GH_PR", "ok")
	resetPRCache()
	t.Cleanup(resetPRCache)
	return dir
}

const samplePR = `{"number":3981,"title":"Rebuild the homepage around live workflows","url":"https://github.com/acme/site/pull/3981",
"state":"OPEN","isDraft":false,"reviewDecision":"APPROVED","headRefName":"feat/homepage","baseRefName":"main",
"additions":1934,"deletions":684,"changedFiles":21,"author":{"login":"octo"},"updatedAt":"2026-09-05T12:00:00Z",
"statusCheckRollup":[
 {"__typename":"CheckRun","name":"Classify changes","status":"COMPLETED","conclusion":"SUCCESS","detailsUrl":"https://ci/1"},
 {"__typename":"CheckRun","name":"Frontend","status":"COMPLETED","conclusion":"FAILURE","detailsUrl":"https://ci/2"},
 {"__typename":"CheckRun","name":"Public docs","status":"IN_PROGRESS","conclusion":"","detailsUrl":"https://ci/3"},
 {"__typename":"CheckRun","name":"Lint","status":"COMPLETED","conclusion":"SKIPPED","detailsUrl":"https://ci/4"},
 {"__typename":"StatusContext","context":"vercel","state":"SUCCESS","targetUrl":"https://vercel/1"}
]}`

func TestPRPageStatesThroughFakeGh(t *testing.T) {
	repo := gitRepo(t)
	dir := fakeGh(t)
	jsonPath := filepath.Join(dir, "pr.json")
	if err := os.WriteFile(jsonPath, []byte(samplePR), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_GH_JSON", jsonPath)

	t.Setenv("FAKE_GH_AUTH", "no")
	resetPRCache()
	if page := prPage(repo, false); page["status"] != "blocked" || page["reason"] != "gh-unauth" {
		t.Fatalf("not logged in = %v", page)
	}

	t.Setenv("FAKE_GH_AUTH", "yes")
	t.Setenv("FAKE_GH_PR", "none")
	resetPRCache()
	if page := prPage(repo, false); page["status"] != "none" || page["branch"] != "main" {
		t.Fatalf("no PR = %v", page)
	}

	t.Setenv("FAKE_GH_PR", "noremote")
	resetPRCache()
	if page := prPage(repo, false); page["status"] != "blocked" || page["reason"] != "no-remote" {
		t.Fatalf("no remote = %v", page)
	}

	t.Setenv("FAKE_GH_PR", "ok")
	resetPRCache()
	page := prPage(repo, false)
	if page["status"] != "ok" {
		t.Fatalf("ok = %v", page)
	}
	pr := page["pr"].(map[string]any)
	if pr["number"] != 3981 || pr["state"] != "open" || pr["draft"] != false || pr["head"] != "feat/homepage" || pr["base"] != "main" || pr["author"] != "octo" {
		t.Fatalf("pr = %v", pr)
	}
	checks := pr["checks"].(map[string]any)
	if checks["total"] != 5 || checks["passed"] != 2 || checks["failed"] != 1 || checks["pending"] != 1 || checks["skipped"] != 1 {
		t.Fatalf("checks = %v", checks)
	}
	failing := checks["failing"].([]map[string]string)
	if len(failing) != 1 || failing[0]["name"] != "Frontend" || failing[0]["url"] != "https://ci/2" {
		t.Fatalf("failing = %v", failing)
	}

	// The minute cache serves the last answer until Refresh asks again.
	t.Setenv("FAKE_GH_PR", "none")
	if page := prPage(repo, false); page["status"] != "ok" {
		t.Fatalf("cached read = %v, want the earlier ok", page)
	}
	if page := prPage(repo, true); page["status"] != "none" {
		t.Fatalf("refreshed read = %v, want none", page)
	}
}

func TestPRPageWithoutGhOrGit(t *testing.T) {
	repo := gitRepo(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git missing")
	}
	only := t.TempDir()
	if err := os.Symlink(gitPath, filepath.Join(only, "git")); err != nil {
		t.Skip("cannot symlink git:", err)
	}
	t.Setenv("PATH", only)
	resetPRCache()
	t.Cleanup(resetPRCache)
	if page := prPage(repo, false); page["status"] != "blocked" || page["reason"] != "gh-missing" {
		t.Fatalf("without gh = %v", page)
	}
	if page := prPage(t.TempDir(), false); page["status"] != "blocked" || page["reason"] != "no-git" {
		t.Fatalf("plain folder = %v", page)
	}
}

// The route reads through the owner and keeps the root precondition.
func TestPRRouteThroughOwnersWithRoot(t *testing.T) {
	repo := gitRepo(t)
	fakeGh(t)
	t.Setenv("FAKE_GH_PR", "none")
	st := testStore(t)
	ts := graphServer(t, st)
	ws, agent, err := storeWorkspaceWithAgent(st, "App", repo)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/workspaces/" + ws.ID + "/pr", "/api/agents/" + agent.ID + "/pr"} {
		res := do(t, ts.Client(), mustGet(t, ts.URL+path))
		var page map[string]any
		_ = json.NewDecoder(res.Body).Decode(&page)
		if res.StatusCode != http.StatusOK || page["status"] != "none" || page["branch"] != "main" {
			t.Fatalf("%s = %d %v", path, res.StatusCode, page)
		}
	}
	res := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/"+agent.ID+"/pr?root=/somewhere/else"))
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("mismatched root = %d, want 409", res.StatusCode)
	}
}

func TestTypeTextProblem(t *testing.T) {
	for text, want := range map[string]string{
		"gh pr create --fill":              "",
		"gh auth login":                    "",
		"":                                 "nothing to type",
		"   ":                              "nothing to type",
		"-rf /":                            "a command cannot start with a dash",
		"echo hi\n":                        "control characters and newlines are not typed",
		"echo \x1b[2J":                     "control characters and newlines are not typed",
		strings.Repeat("x", typeTextMax+1): "that command is too long to type",
	} {
		if got := typeTextProblem(text); got != want {
			t.Errorf("typeTextProblem(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestTerminalTypeRouteRefusesBadInput(t *testing.T) {
	st := testStore(t)
	ts := graphServer(t, st)
	term, err := st.CreateTerminalIn("", "QA", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	post := func(id, body string) *http.Response {
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/terminals/"+id+"/type", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		return do(t, ts.Client(), req)
	}
	if res := post(term.ID, `{"text":"echo hi\n"}`); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("newline = %d, want 400", res.StatusCode)
	}
	if res := post(term.ID, `{"text":""}`); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty = %d, want 400", res.StatusCode)
	}
	if res := post("nope", `{"text":"echo hi"}`); res.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown terminal = %d, want 404", res.StatusCode)
	}
}
