package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clipkgs"
	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// stubVendor puts a fake vendor binary first on PATH. The drivers locate their
// binary by name (ADR-0167), so this is the seam every handler test uses.
func stubVendor(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	clipkgs.Invalidate(name)
}

// cliRequestRaw sends a request and returns the live response, for the cases
// where the status alone is not enough (an error body must name the vendor).
func cliRequestRaw(t *testing.T, ts *httptest.Server, method, path string, body any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return do(t, ts.Client(), req)
}

// TestCLIPackagesGuards pins the refusals: no driver for Pi, no per-agent
// scope for anyone, no project scope without a workspace, and no project scope
// at all for a machine-only CLI.
func TestCLIPackagesGuests(t *testing.T) {
	stubVendor(t, "omp", `printf '%s' '{"npm":[{"name":"ext","version":"1.2.3","enabled":true}],"marketplace":[]}'`)
	ts := newTestServer(t, "cat")

	cases := []struct {
		name string
		path string
		want int
	}{
		{"pi keeps its own pane", "/api/cli-packages?cli=pi&scope=user", 400},
		{"unknown CLI", "/api/cli-packages?cli=nope&scope=user", 400},
		{"no per-agent scope", "/api/cli-packages?cli=omp&scope=agent", 400},
		{"project scope needs a workspace", "/api/cli-packages?cli=omp&scope=project", 400},
		{"codex has no project scope", "/api/cli-packages?cli=codex&scope=project&workspace=ws_x", 400},
		{"claude has no agent scope", "/api/cli-packages?cli=claude-code&scope=agent", 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cliRequest(t, ts, "GET", tc.path, nil, tc.want)
		})
	}
}

// TestCLIPackagesListReadsTheVendor is the pane's happy path: the rows, the
// capability set and the notes all come from the driver's declaration.
func TestCLIPackagesListReadsTheVendor(t *testing.T) {
	stubVendor(t, "omp", `printf '%s' '{"npm":[{"name":"ext","version":"1.2.3","enabled":true},{"name":"off","enabled":false}],"marketplace":[]}'`)
	ts := newTestServer(t, "cat")

	v := cliRequest(t, ts, "GET", "/api/cli-packages?cli=omp&scope=user&refresh=1", nil, 200)
	if v["cli"] != "omp" {
		t.Fatalf("cli = %v", v["cli"])
	}
	scopes, _ := v["scopes"].([]any)
	if len(scopes) != 2 {
		t.Fatalf("scopes = %v (the declaration says machine + workspace)", scopes)
	}
	caps, _ := v["caps"].(map[string]any)
	if caps["install"] != true || caps["toggle"] != true || caps["available"] != false {
		t.Fatalf("caps = %v", caps)
	}
	rows, _ := v["rows"].([]any)
	if len(rows) != 2 {
		t.Fatalf("rows = %v", rows)
	}
	first, _ := rows[0].(map[string]any)
	if first["name"] != "ext" || first["version"] != "1.2.3" || first["installed"] != true {
		t.Fatalf("row = %v", first)
	}
}

// TestCLIPackagesUnparsableIsNotAnEmptyList: a vendor whose output PiCode does
// not recognize is a 502 with the vendor's own words, never "nothing
// installed".
func TestCLIPackagesUnparsableIsNotAnEmptyList(t *testing.T) {
	stubVendor(t, "grok", `printf '%s\n' 'plugin list: unrecognized penguin'`)
	ts := newTestServer(t, "cat")

	res := cliRequestRaw(t, ts, "GET", "/api/cli-packages?cli=grok&scope=user&refresh=1", nil)
	if res.StatusCode != 502 {
		t.Fatalf("status = %d, want 502", res.StatusCode)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error == "" {
		t.Fatal("a refused roster must carry the vendor's words")
	}
}

// TestCLIPackagesMissingVendorFailsLoudly: an uninstalled CLI is not an empty
// list either.
func TestCLIPackagesMissingVendorFailsLoudly(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	ts := newTestServer(t, "cat")
	clipkgs.Invalidate("agy")
	cliRequest(t, ts, "GET", "/api/cli-packages?cli=agy&scope=user&refresh=1", nil, 400)
}

// TestCLIPackagesToggleAnswersTheFreshList: the pane never guesses a row's
// next state.
func TestCLIPackagesToggleAnswersTheFreshList(t *testing.T) {
	stubVendor(t, "omp", `printf '%s' '{"npm":[{"name":"ext","enabled":false}],"marketplace":[]}'`)
	ts := newTestServer(t, "cat")

	v := cliRequest(t, ts, "POST", "/api/cli-packages/toggle", map[string]any{
		"cli": "omp", "scope": "user", "name": "ext", "on": false,
	}, 200)
	rows, _ := v["rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("rows = %v", rows)
	}
	row, _ := rows[0].(map[string]any)
	if row["enabled"] != false {
		t.Fatalf("row = %v", row)
	}
}

// TestCLIPackagesInstallIsAnIdempotentJob covers the mutation path end to end:
// a 202 job, the same job for a repeated request, and the vendor command the
// job actually runs.
func TestCLIPackagesInstallIsAnIdempotentJob(t *testing.T) {
	stubVendor(t, "omp", `printf '%s' '{"npm":[],"marketplace":[]}'`)
	ts := newTestServer(t, "cat")

	body := map[string]any{"cli": "omp", "scope": "user", "source": "my-ext", "requestKey": "req-1"}
	first := cliRequest(t, ts, "POST", "/api/cli-packages/install", body, 202)
	again := cliRequest(t, ts, "POST", "/api/cli-packages/install", body, 202)
	if first["id"] != again["id"] {
		t.Fatalf("the same request key made two jobs: %v vs %v", first["id"], again["id"])
	}
	if first["action"] != "pkg-install" {
		t.Fatalf("action = %v", first["action"])
	}

	// The job resolves its own argv from the payload it stored.
	exe, err := resolvePackageJob("omp", "pkg-install", `{"source":"my-ext","scope":"user"}`)
	if err != nil {
		t.Fatal(err)
	}
	if exe.Exe != "omp" || len(exe.Args) != 4 || exe.Args[0] != "plugin" || exe.Args[1] != "install" {
		t.Fatalf("job argv = %v", exe.Args)
	}

	// Wait for the job to settle so the test does not leak a worker.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		jobs, _ := jobsNow(t, ts)
		for _, j := range jobs {
			if j["id"] == first["id"] {
				if state, _ := j["state"].(string); state == "succeeded" {
					return
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the install job did not settle")
}

// TestResolvePackageJobKeepsTheProjectDirectory proves the workspace folder
// travels with the job: a project-scope install must not run in the daemon's
// own directory.
func TestResolvePackageJobKeepsTheProjectDirectory(t *testing.T) {
	exe, err := resolvePackageJob("omp", "pkg-install", `{"source":"my-ext","scope":"project","cwd":"/ws/one"}`)
	if err != nil {
		t.Fatal(err)
	}
	if exe.Dir != "/ws/one" {
		t.Fatalf("dir = %q", exe.Dir)
	}
	market, err := resolvePackageJob("muse", "pkg-marketplace-add", `{"marketplace":"add","source":"owner/repo","name":"mp","ref":"project","cwd":"/ws/two"}`)
	if err != nil {
		t.Fatal(err)
	}
	if market.Dir != "/ws/two" || market.Args[0] != "plugins" {
		t.Fatalf("marketplace job = %+v", market)
	}
	if _, err := resolvePackageJob("pi", "pkg-install", `{}`); err == nil {
		t.Fatal("Pi has no package jobs here")
	}
}

// TestCLIPackagesMarketplaceRemoveLeavesNoJob: a removal is a local change,
// and the answer is the remaining sources.
func TestCLIPackagesMarketplaceRemoveLeavesNoJob(t *testing.T) {
	stubVendor(t, "omp", `printf '%s\n' 'No marketplaces configured' 'Add one with: omp plugin marketplace add <source>'`)
	ts := newTestServer(t, "cat")

	v := cliRequest(t, ts, "POST", "/api/cli-packages/marketplace", map[string]any{
		"cli": "omp", "scope": "user", "action": "remove", "name": "mp",
	}, 200)
	rows, ok := v["marketplaces"].([]any)
	if !ok || len(rows) != 0 {
		t.Fatalf("marketplaces = %v", v["marketplaces"])
	}
	if _, err := resolvePackageJob("omp", "pkg-marketplace-add", `{}`); err == nil {
		t.Fatal("a marketplace add without a source must be refused")
	}
}

// TestCLIPackagesSyncMutationPublishesTheEvent: a toggle edits no PiCode
// store, so the feed event is the only thing that tells another open pane the
// plugin list moved (ADR-0048).
func TestCLIPackagesSyncMutationPublishesTheEvent(t *testing.T) {
	stubVendor(t, "omp", `printf '%s' '{"npm":[{"name":"ext","enabled":false}],"marketplace":[]}'`)
	data := t.TempDir()
	st, err := store.Open(filepath.Join(data, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	f := &feed.Feed{Store: st}
	seen := make(chan string, 8)
	f.Listen(func(ev store.Event) {
		select {
		case seen <- ev.Type:
		default:
		}
	})
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat", DataDir: data, Feed: f}).Handler)
	t.Cleanup(ts.Close)

	cliRequest(t, ts, "POST", "/api/cli-packages/toggle", map[string]any{
		"cli": "omp", "scope": "user", "name": "ext", "on": false,
	}, 200)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case ev := <-seen:
			if ev == "cli.packages" {
				return
			}
		case <-time.After(50 * time.Millisecond):
		}
	}
	t.Fatal("a toggle never announced itself on the feed")
}

// TestCLIPackagesMarketplacesReadOnLoad: the pane must learn the CLI's own
// sources without having to remove one first.
func TestCLIPackagesMarketplacesReadOnLoad(t *testing.T) {
	stubVendor(t, "grok", `printf '%s' '{"marketplaces":[{"name":"team","source":{"url":"https://example.test/mp"}}]}'`)
	ts := newTestServer(t, "cat")
	v := cliRequest(t, ts, "GET", "/api/cli-packages/marketplaces?cli=grok&scope=user", nil, 200)
	rows, ok := v["marketplaces"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("marketplaces = %v", v["marketplaces"])
	}
	row, _ := rows[0].(map[string]any)
	if row["name"] != "team" || row["source"] != "https://example.test/mp" {
		t.Fatalf("row = %v", row)
	}
	// A CLI with no source-management verb answers 400 rather than an empty
	// list that would read as "no sources configured".
	clipkgs.Invalidate("hermes")
	cliRequest(t, ts, "GET", "/api/cli-packages/marketplaces?cli=hermes&scope=user", nil, 400)
}

// TestCLIPackagesInspectIsVendorText: the pane shows the vendor's own words.
func TestCLIPackagesInspectIsVendorText(t *testing.T) {
	stubVendor(t, "muse", `printf '%s' '{"capabilities":["read"]}'`)
	ts := newTestServer(t, "cat")
	v := cliRequest(t, ts, "POST", "/api/cli-packages/inspect", map[string]any{
		"cli": "muse", "scope": "user", "name": "pl",
	}, 200)
	if v["output"] != `{"capabilities":["read"]}` {
		t.Fatalf("output = %v", v["output"])
	}
	// A CLI with no inspect verb is refused rather than answered empty.
	cliRequest(t, ts, "POST", "/api/cli-packages/inspect", map[string]any{
		"cli": "opencode", "scope": "user", "name": "x",
	}, 400)
}

func jobsNow(t *testing.T, ts *httptest.Server) ([]map[string]any, error) {
	t.Helper()
	res := cliRequestRaw(t, ts, "GET", "/api/cli-jobs", nil)
	defer res.Body.Close()
	var body struct {
		Jobs []map[string]any `json:"jobs"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Jobs, nil
}
