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

	"strings"

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

// TestPackageVendorRoutesPinTheRefusals: a CLI's own surface on the packages
// family refuses what PiCode refuses — an unknown CLI naming the drivers that
// exist, Pi itself (its mutations are PiCode's own calls on the same paths), a
// scope the CLI does not have, a project scope without a workspace folder, and
// the agent layer, which is PiCode's own list on the agent row rather than a
// vendor command (ADR-0176).
func TestPackageVendorRoutesPinTheRefusals(t *testing.T) {
	stubVendor(t, "omp", `printf '%s' '{"npm":[{"name":"ext","version":"1.2.3","enabled":true}],"marketplace":[]}'`)
	ts := newTestServer(t, "cat")

	cases := []struct {
		name string
		path string
		want int
	}{
		{"unknown CLI", "/api/packages/marketplaces?cli=nope&scope=user", 400},
		{"pi keeps its own calls", "/api/packages/marketplaces?cli=pi&scope=user", 400},
		{"no per-agent scope", "/api/packages/marketplaces?cli=omp&scope=agent", 400},
		{"project scope needs a workspace", "/api/packages/marketplaces?cli=omp&scope=project", 400},
		{"codex has no project scope", "/api/packages/marketplaces?cli=codex&scope=project&workspace=ws_x", 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cliRequest(t, ts, "GET", tc.path, nil, tc.want)
		})
	}
}

// TestPackageReportReadsTheVendor is the pane's happy path for a CLI whose own
// commands manage its plugins: the rows, the capability set and the notes all
// come from the driver's declaration, on the one read every CLI answers.
func TestPackageReportReadsTheVendor(t *testing.T) {
	stubVendor(t, "omp", `printf '%s' '{"npm":[{"name":"ext","version":"1.2.3","enabled":true},{"name":"off","enabled":false}],"marketplace":[]}'`)
	ts := newTestServer(t, "cat")

	v := cliRequest(t, ts, "GET", "/api/packages/report?cli=omp&vendor=user&refresh=1", nil, 200)
	if v["cli"] != "omp" {
		t.Fatalf("cli = %v", v["cli"])
	}
	scopes, _ := v["scopes"].([]any)
	// No workspace bound: the workspace layer is not offered at all — a
	// project mutation without a folder refuses, and the radio must not
	// promise an error.
	if len(scopes) != 1 {
		t.Fatalf("scopes = %v (an unbound read offers the machine scope alone)", scopes)
	}
	first, _ := scopes[0].(map[string]any)
	if first["id"] != "machine" || first["vendor"] != "user" || first["label"] != "Global" {
		t.Fatalf("scope = %v, want the class PiCode draws and the CLI's own word", first)
	}

	// Bound to a workspace and an agent, the radio names them — the same
	// honour pi's pane pays.
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "PiCode", "path": t.TempDir()})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("workspace = %d", res.StatusCode)
	}
	var wsv workspaceView
	if err := json.NewDecoder(res.Body).Decode(&wsv); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	ares := postJSON(t, ts, "/api/workspaces/"+wsv.ID+"/agents", map[string]string{"name": "browser", "cli": "omp"})
	if ares.StatusCode != http.StatusCreated {
		t.Fatalf("agent = %d", ares.StatusCode)
	}
	var agv struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(ares.Body).Decode(&agv); err != nil {
		t.Fatal(err)
	}
	ares.Body.Close()
	v = cliRequest(t, ts, "GET", "/api/packages/report?cli=omp&vendor=user&refresh=1&workspace="+wsv.ID+"&agent="+agv.ID, nil, 200)
	scopes, _ = v["scopes"].([]any)
	if len(scopes) != 3 {
		t.Fatalf("scopes = %v (a bound read names the workspace and the agent)", scopes)
	}
	wsScope, _ := scopes[1].(map[string]any)
	if wsScope["id"] != "workspace" || wsScope["label"] != "PiCode" {
		t.Fatalf("workspace scope = %v, want the workspace's own name", wsScope)
	}
	agScope, _ := scopes[2].(map[string]any)
	if agScope["id"] != "agent" || agScope["label"] != "This agent" || agScope["note"] != "Only browser, every session" {
		t.Fatalf("agent scope = %v, want the agent named in the note", agScope)
	}
	caps, _ := v["caps"].(map[string]any)
	// Omp has a catalog (`omp plugin discover`) and no per-row Install: the
	// list never says which marketplace provides a plugin, so the pane offers
	// the rows as information and states the install form in the note.
	if caps["install"] != true || caps["toggle"] != true || caps["available"] != true || caps["catalogInstall"] != false {
		t.Fatalf("caps = %v", caps)
	}
	rows, _ := v["rows"].([]any)
	if len(rows) != 2 {
		t.Fatalf("rows = %v", rows)
	}
	row0, _ := rows[0].(map[string]any)
	if row0["name"] != "ext" || row0["version"] != "1.2.3" || row0["installed"] != true {
		t.Fatalf("row = %v", row0)
	}
}

// TestPackageReportUnparsableIsNotAnEmptyList: a vendor whose output PiCode does
// not recognize is a 502 with the vendor's own words, never "nothing
// installed".
func TestPackageReportUnparsableIsNotAnEmptyList(t *testing.T) {
	stubVendor(t, "grok", `printf '%s\n' 'plugin list: unrecognized penguin'`)
	ts := newTestServer(t, "cat")

	res := cliRequestRaw(t, ts, "GET", "/api/packages/report?cli=grok&vendor=user&refresh=1", nil)
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

// TestPackageReportMissingVendorFailsLoudly: an uninstalled CLI is not an empty
// list either.
func TestPackageReportMissingVendorFailsLoudly(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	ts := newTestServer(t, "cat")
	clipkgs.Invalidate("agy")
	cliRequest(t, ts, "GET", "/api/packages/report?cli=agy&vendor=user&refresh=1", nil, 400)
}

// TestPackageToggleAnswersTheFreshReport: the pane never guesses a row's next
// state, and the answer is the same unified report the pane just read — not a
// second shape for the same list.
func TestPackageToggleAnswersTheFreshReport(t *testing.T) {
	stubVendor(t, "omp", `printf '%s' '{"npm":[{"name":"ext","enabled":false}],"marketplace":[]}'`)
	ts := newTestServer(t, "cat")

	v := cliRequest(t, ts, "POST", "/api/packages/toggle", map[string]any{
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
	if v["cli"] != "omp" || v["caps"] == nil || v["scopes"] == nil {
		t.Fatalf("the answer is not the unified report: %v", v)
	}
}

// TestPackageInstallIsAnIdempotentJob covers the mutation path end to end: a
// 202 job, the same job for a repeated request, and the vendor command the
// job actually runs — all on the path Pi's own install uses, told apart by the
// `cli` the request names.
func TestPackageInstallIsAnIdempotentJob(t *testing.T) {
	stubVendor(t, "omp", `printf '%s' '{"npm":[],"marketplace":[]}'`)
	ts := newTestServer(t, "cat")

	body := map[string]any{"cli": "omp", "scope": "user", "source": "my-ext", "requestKey": "req-1"}
	first := cliRequest(t, ts, "POST", "/api/packages", body, 202)
	again := cliRequest(t, ts, "POST", "/api/packages", body, 202)
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

// TestPackageMarketplaceRemoveLeavesNoJob: a removal is a local change, and the
// answer is the remaining sources in the unified row shape.
func TestPackageMarketplaceRemoveLeavesNoJob(t *testing.T) {
	stubVendor(t, "omp", `printf '%s\n' 'No marketplaces configured' 'Add one with: omp plugin marketplace add <source>'`)
	ts := newTestServer(t, "cat")

	v := cliRequest(t, ts, "POST", "/api/packages/marketplace", map[string]any{
		"cli": "omp", "scope": "user", "action": "remove", "name": "mp",
	}, 200)
	rows, ok := v["marketplaces"].([]any)
	if !ok || len(rows) != 0 {
		t.Fatalf("marketplaces = %v", v["marketplaces"])
	}
	if v["cli"] != "omp" {
		t.Fatalf("cli = %v, want the CLI the answer is about", v["cli"])
	}
	if _, err := resolvePackageJob("omp", "pkg-marketplace-add", `{}`); err == nil {
		t.Fatal("a marketplace add without a source must be refused")
	}
}

// TestPackageSyncMutationPublishesTheEvent: a toggle edits no PiCode store, so
// the feed event is the only thing that tells another open pane the plugin list
// moved (ADR-0048).
func TestPackageSyncMutationPublishesTheEvent(t *testing.T) {
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

	cliRequest(t, ts, "POST", "/api/packages/toggle", map[string]any{
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

// TestPackageMarketplacesReadOnLoad: the pane must learn the CLI's own sources
// without having to remove one first.
func TestPackageMarketplacesReadOnLoad(t *testing.T) {
	stubVendor(t, "grok", `printf '%s' '{"marketplaces":[{"name":"team","source":{"url":"https://example.test/mp"}}]}'`)
	ts := newTestServer(t, "cat")
	v := cliRequest(t, ts, "GET", "/api/packages/marketplaces?cli=grok&scope=user", nil, 200)
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
	cliRequest(t, ts, "GET", "/api/packages/marketplaces?cli=hermes&scope=user", nil, 400)
}

// TestPackageInspectIsVendorText: the pane shows the vendor's own words.
func TestPackageInspectIsVendorText(t *testing.T) {
	stubVendor(t, "muse", `printf '%s' '{"capabilities":["read"]}'`)
	ts := newTestServer(t, "cat")
	v := cliRequest(t, ts, "POST", "/api/packages/inspect", map[string]any{
		"cli": "muse", "scope": "user", "name": "pl",
	}, 200)
	if v["output"] != `{"capabilities":["read"]}` {
		t.Fatalf("output = %v", v["output"])
	}
	// A CLI with no inspect verb is refused rather than answered empty.
	cliRequest(t, ts, "POST", "/api/packages/inspect", map[string]any{
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

// TestPackageRefusalCarriesTheCommand: when the CLI refuses and only a person
// in a terminal can answer (Grok's --trust), the answer carries the exact
// command — rendered by the builder the request itself used — so the pane
// offers the line to run instead of a dead end.
func TestPackageRefusalCarriesTheCommand(t *testing.T) {
	stubVendor(t, "grok", `printf '%s\n' 'refusing to disable without confirmation'; exit 1`)
	ts := newTestServer(t, "cat")

	res := cliRequestRaw(t, ts, "POST", "/api/packages/toggle", map[string]any{
		"cli": "grok", "scope": "user", "name": "probe", "on": false,
	})
	defer res.Body.Close()
	if res.StatusCode != 502 && res.StatusCode != 400 {
		t.Fatalf("status = %d, want the vendor failure reported", res.StatusCode)
	}
	var body struct {
		Error   string `json:"error"`
		Command string `json:"command"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.Error, "refusing to disable") {
		t.Fatalf("error = %q, want the vendor's own words", body.Error)
	}
	if body.Command != "grok plugin disable probe" {
		t.Fatalf("command = %q, want the exact line the pane can run", body.Command)
	}
}

// TestPackageJobCarriesTheCommand: the asynchronous half needs the same
// affordance, so a failed package job answers with the command it ran.
func TestPackageJobCarriesTheCommand(t *testing.T) {
	stubVendor(t, "omp", `printf '%s' '{"npm":[],"marketplace":[]}'`)
	ts := newTestServer(t, "cat")

	started := cliRequest(t, ts, "POST", "/api/packages", map[string]any{
		"cli": "omp", "scope": "user", "source": "my-ext", "requestKey": "cmd-1",
	}, 202)
	if started["command"] != "omp plugin install my-ext --json" {
		t.Fatalf("202 command = %v, want the line the job runs", started["command"])
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		jobs, err := jobsNow(t, ts)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range jobs {
			if row["id"] == started["id"] {
				if row["state"] == "succeeded" {
					if row["command"] != "omp plugin install my-ext --json" {
						t.Fatalf("job command = %v", row["command"])
					}
					return
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the install job did not settle")
}

// TestPackageOpenCodeRemoveIsPiCodeOwnWrite: OpenCode keeps its plugins in its
// own config array and exposes no removal command, so the removal must not
// reserve a job that would run nothing — the answer is the CLI's own fresh
// report (200). The plugin is out of the config file that named it with every
// other byte intact, and a module the CLI's own configs do not name is still
// refused, by name.
func TestPackageOpenCodeRemoveIsPiCodeOwnWrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "") // resolve OpenCode's user file from HOME
	// The report reads the CLI's own files, but it only answers for a CLI the
	// driver can locate (ADR-0167) — a stub keeps this hermetic on a machine
	// where opencode is not installed (CI).
	stubVendor(t, "opencode", "")
	cfg := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(cfg, "opencode.json")
	body := "{\n  // the module list\n  \"plugin\": [\"opencode-wakatime\", \"keep-me\"],\n  \"theme\": \"dark\"\n}\n"
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	ts := newTestServer(t, "cat")
	clipkgs.Invalidate("opencode")

	// The declaration the pane reads says the removal is not the lane's: the
	// report carries it (`Caps.Lane`), and the removal that follows is the
	// write that declaration promised.
	rep := cliRequest(t, ts, "GET", "/api/packages/report?cli=opencode&vendor=user&refresh=1", nil, 200)
	caps, _ := rep["caps"].(map[string]any)
	lane, _ := caps["lane"].(map[string]any)
	if lane["install"] != true || lane["remove"] != false {
		t.Fatalf("caps.lane = %v, want the install on the lane and the removal a write", lane)
	}
	v := cliRequest(t, ts, "GET", "/api/packages/report?cli=opencode&vendor=user&refresh=1", nil, 200)
	rowBySource(t, v, "opencode-wakatime") // it must be there before the removal

	v = cliRequest(t, ts, "DELETE", "/api/packages", map[string]any{
		"cli": "opencode", "scope": "user", "name": "opencode-wakatime", "source": "opencode-wakatime",
	}, 200)
	rows, _ := v["rows"].([]any)
	for _, raw := range rows {
		if m, _ := raw.(map[string]any); m["source"] == "opencode-wakatime" {
			t.Fatalf("the answer still lists the removed module: %v", m)
		}
	}
	if len(rows) != 1 {
		t.Fatalf("the answer is not the CLI's fresh list: %v", rows)
	}
	rowBySource(t, v, "keep-me")
	at := strings.Index(body, `"opencode-wakatime", `)
	if at < 0 {
		t.Fatal("the fixture does not name the removed module")
	}
	want := body[:at] + body[at+len(`"opencode-wakatime", `):]
	got, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("the config after the removal = %q, want %q", got, want)
	}

	// The module the configs do not name: refused by name, and nothing written.
	res := cliRequestRaw(t, ts, "DELETE", "/api/packages", map[string]any{
		"cli": "opencode", "scope": "user", "name": "ghost", "source": "ghost",
	})
	if res.StatusCode != 409 {
		t.Fatalf("status = %d, want 409 for a module no config names", res.StatusCode)
	}
	var refusal struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&refusal); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(refusal.Error, "ghost") {
		t.Fatalf("the refusal does not name the module: %q", refusal.Error)
	}
	if after, err := os.ReadFile(file); err != nil || string(after) != string(got) {
		t.Fatalf("a refused removal wrote the config: %q (%v)", after, err)
	}
}

// TestPackageUpdatesMarksWhatIsBehind: the badge comes from the CLI's own
// catalog compared with its own roster, on the read every CLI answers, and a
// row is marked only when the catalog offers something newer.
func TestPackageUpdatesMarksWhatIsBehind(t *testing.T) {
	stubVendor(t, "grok", `case "$*" in
  *--available*) printf '%s' '[{"status":"available","name":"probe","version":"0.4.0","marketplace":"mp"},{"status":"available","name":"steady","version":"1.0.0","marketplace":"mp"}]' ;;
  *) printf '%s' '[{"status":"installed","name":"probe","version":"0.2.0","path":"/p"},{"status":"installed","name":"steady","version":"1.0.0","path":"/s"}]' ;;
esac`)
	ts := newTestServer(t, "cat")

	v := cliRequest(t, ts, "GET", "/api/packages/updates?cli=grok&scope=user&refresh=1", nil, 200)
	updates, _ := v["updates"].([]any)
	if len(updates) != 1 {
		t.Fatalf("updates = %v, want only the row the catalog moved ahead of", updates)
	}
	behind, _ := updates[0].(map[string]any)
	if behind["source"] != "probe" || behind["current"] != "0.2.0" || behind["latest"] != "0.4.0" {
		t.Fatalf("behind row = %v, want the version pair the pane badges with", behind)
	}

	// A CLI without an update verb refuses the check rather than inventing one.
	cliRequest(t, ts, "GET", "/api/packages/updates?cli=agy&scope=user", nil, 400)
}
