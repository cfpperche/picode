package server

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// fakeNpmCLI creates an executable whose realpath classifies as npm
// (/node_modules/) plus a fake npm binary on the config's PATH.
func fakeNpmCLI(t *testing.T, home, versionOutput string) (exe, npmDir string) {
	t.Helper()
	pkgDir := filepath.Join(home, "fake", "node_modules", "@earendil-works", "pi-coding-agent", "bin")
	npmDir = filepath.Join(home, "fake", "bin")
	for _, dir := range []string{pkgDir, npmDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	exe = filepath.Join(pkgDir, "pi")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo '" + versionOutput + "'; fi\nif [ \"$1\" = \"update\" ]; then echo 'pi update ran'; fi\n"
	if err := os.WriteFile(exe, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	npm := filepath.Join(npmDir, "npm")
	if err := os.WriteFile(npm, []byte("#!/bin/sh\necho npm \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe, npmDir
}

func configureCLI(t *testing.T, ts *httptest.Server, id, exe, pathDir string) {
	t.Helper()
	config := map[string]any{"executable": exe, "args": []any{}, "env": map[string]any{}, "integration": false}
	if pathDir != "" {
		config["path"] = []any{pathDir}
	}
	cliRequest(t, ts, "PUT", "/api/clis/"+id, config, 200)
}

func waitForJobState(t *testing.T, ts *httptest.Server, id, state string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, row := range cliRequest(t, ts, "GET", "/api/cli-jobs", nil, 200)["jobs"].([]any) {
			j := row.(map[string]any)
			if j["id"] == id && j["state"] == state {
				return j
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("job %s never reached %s", id, state)
	return nil
}

func TestCLIInstallMissingDecisionTable(t *testing.T) {
	ts, _, home := cleanupServer(t)
	// codex is configured to a not-yet-existing path inside the fake
	// node_modules tree; the fake npm "install" creates it.
	exe := filepath.Join(home, "fake", "node_modules", "@openai", "codex", "bin", "codex")
	npmDir := filepath.Join(home, "fake", "bin")
	if err := os.MkdirAll(npmDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(npmDir, "npm"), []byte("#!/bin/sh\nmkdir -p \""+filepath.Dir(exe)+"\"\ntouch \""+exe+"\" && chmod +x \""+exe+"\"\necho npm installed $3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	configureCLI(t, ts, "codex", exe, npmDir)

	// Missing CLI: the catalog offers install, not update.
	lifecycle := catalogCLI(t, ts, "codex")["lifecycle"].(map[string]any)
	if lifecycle["canInstall"] != true || lifecycle["canUpdate"] != false {
		t.Fatalf("missing codex lifecycle: %v", lifecycle)
	}
	// Install runs the fake npm and flips the CLI to Installed with an npm
	// method — the cycle closes without a restart.
	j := cliRequest(t, ts, "POST", "/api/clis/codex/lifecycle", map[string]any{"action": "install", "requestKey": "i1"}, 202)
	done := waitForJobState(t, ts, j["id"].(string), "succeeded")
	if !strings.Contains(done["output"].(string), "npm installed @openai/codex@latest") {
		t.Fatalf("install argv: %v", done["output"])
	}
	v := catalogCLI(t, ts, "codex")
	if v["installed"] != true {
		t.Fatalf("codex still not installed: %v", v["installed"])
	}
	lifecycle = v["lifecycle"].(map[string]any)
	if lifecycle["canInstall"] != false || lifecycle["canUpdate"] != true || lifecycle["method"] != "npm" {
		t.Fatalf("installed codex lifecycle: %v", lifecycle)
	}
	// Installing an installed CLI refuses — reinstall is that action.
	r := cliRequest(t, ts, "POST", "/api/clis/codex/lifecycle", map[string]any{"action": "install", "requestKey": "i2"}, 400)
	if !strings.Contains(r["error"].(string), "already installed") {
		t.Fatalf("install on installed: %v", r)
	}
	// Grok has no managed install: guided docs, refused action.
	grok := filepath.Join(home, ".grok", "bin", "grok2")
	if err := os.MkdirAll(filepath.Dir(grok), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(grok, []byte("#!/bin/sh\necho grok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	configureCLI(t, ts, "grok", grok, "")
	grokView := catalogCLI(t, ts, "grok")["lifecycle"].(map[string]any)
	if grokView["canInstall"] != false || grokView["docs"] == "" {
		t.Fatalf("grok install guidance: %v", grokView)
	}
	cliRequest(t, ts, "POST", "/api/clis/grok/lifecycle", map[string]any{"action": "install", "requestKey": "i3"}, 400)
}

func TestCLIUpdateCheckDecisionTable(t *testing.T) {
	ts, _, home := cleanupServer(t)
	orig := cliNpmLatest
	t.Cleanup(func() { cliNpmLatest = orig })
	exe, npmDir := fakeNpmCLI(t, home, "pi 1.0.0")
	configureCLI(t, ts, "pi", exe, npmDir)

	// No stored version yet: the update check refuses instead of guessing.
	d := cliRequest(t, ts, "POST", "/api/clis/pi/update-check", nil, 200)
	if !strings.Contains(d["updateError"].(string), "Check setup") {
		t.Fatalf("expected check-setup-first error, got %v", d["updateError"])
	}
	// Run the setup check so the installed version is known.
	cliRequest(t, ts, "POST", "/api/clis/pi/check", nil, 200)

	// Registry newer → update available, source npm.
	cliNpmLatest = func(ctx context.Context, name string) (string, error) { return "9.9.9", nil }
	d = cliRequest(t, ts, "POST", "/api/clis/pi/update-check", nil, 200)
	if d["updateAvailable"] != true || d["latest"] != "9.9.9" || d["updateSource"] != "npm" {
		t.Fatalf("available: %+v", d)
	}
	if d["installMethod"] != "npm" {
		t.Fatalf("install method: %+v", d)
	}

	// Registry down → visible error, never invented state.
	cliNpmLatest = func(ctx context.Context, name string) (string, error) { return "", errors.New("down") }
	d = cliRequest(t, ts, "POST", "/api/clis/pi/update-check", nil, 200)
	if v, present := d["updateAvailable"]; present && v != false {
		t.Fatalf("registry down must not claim availability: %+v", d)
	}
	if !strings.Contains(d["updateError"].(string), "npm registry") {
		t.Fatalf("registry down: %+v", d)
	}

	// Equal version → no update, no error.
	cliNpmLatest = func(ctx context.Context, name string) (string, error) { return "1.0.0", nil }
	d = cliRequest(t, ts, "POST", "/api/clis/pi/update-check", nil, 200)
	if v, present := d["updateError"]; v != nil && present && v != "" {
		t.Fatalf("equal version must not error: %+v", d)
	}
	if d["updateAvailable"] != false {
		t.Fatalf("equal version: %+v", d)
	}
	// The catalog view carries the lifecycle facts.
	v := catalogCLI(t, ts, "pi")
	lifecycle := v["lifecycle"].(map[string]any)
	if lifecycle["method"] != "npm" || lifecycle["canUpdate"] != true || lifecycle["uninstall"] != "npm" {
		t.Fatalf("lifecycle view: %v", lifecycle)
	}
}

func TestCLIUpdateCheckVendorKinds(t *testing.T) {
	ts, _, home := cleanupServer(t)
	// Grok classifies vendor only under ~/.grok.
	binDir := filepath.Join(home, ".grok", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	grok := filepath.Join(binDir, "grok")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'grok 1.0.13'; fi\nif [ \"$1\" = \"update\" ]; then echo '{\"currentVersion\":\"1.0.13\",\"latestVersion\":\"1.0.14\",\"updateAvailable\":true,\"installer\":\"internal\",\"channel\":\"stable\",\"autoUpdate\":true,\"error\":null}'; fi\n"
	if err := os.WriteFile(grok, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	configureCLI(t, ts, "grok", grok, "")
	cliRequest(t, ts, "POST", "/api/clis/grok/check", nil, 200)
	d := cliRequest(t, ts, "POST", "/api/clis/grok/update-check", nil, 200)
	if d["updateAvailable"] != true || d["latest"] != "1.0.14" || d["updateSource"] != "vendor" {
		t.Fatalf("grok vendor check: %+v", d)
	}
	// Grok offers update/reinstall but only a guided uninstall.
	lifecycle := catalogCLI(t, ts, "grok")["lifecycle"].(map[string]any)
	if lifecycle["canUpdate"] != true || lifecycle["uninstall"] != "guided" {
		t.Fatalf("grok lifecycle view: %v", lifecycle)
	}
}

func TestCLILifecycleDecisionTable(t *testing.T) {
	ts, _, home := cleanupServer(t)
	exe, npmDir := fakeNpmCLI(t, home, "pi 1.0.0")
	configureCLI(t, ts, "pi", exe, npmDir)
	cliRequest(t, ts, "POST", "/api/clis/pi/check", nil, 200)

	// Uninstall without typed confirmation refuses.
	r := cliRequest(t, ts, "POST", "/api/clis/pi/lifecycle", map[string]any{"action": "uninstall", "requestKey": "u0", "confirmName": ""}, 400)
	if !strings.Contains(r["error"].(string), "Pi") {
		t.Fatalf("uninstall confirm message: %v", r)
	}
	// Unknown action refuses before reserving a job.
	cliRequest(t, ts, "POST", "/api/clis/pi/lifecycle", map[string]any{"action": "explode", "requestKey": "x0"}, 400)
	// A valid update job runs the vendor command (pi update) and reaches
	// succeeded.
	j := cliRequest(t, ts, "POST", "/api/clis/pi/lifecycle", map[string]any{"action": "update", "requestKey": "u1"}, 202)
	done := waitForJobState(t, ts, j["id"].(string), "succeeded")
	if !strings.Contains(done["output"].(string), "pi update ran") {
		t.Fatalf("vendor argv: %v", done["output"])
	}
	// Reinstall is npm-backed under the npm method and prints the npm argv.
	rj := cliRequest(t, ts, "POST", "/api/clis/pi/lifecycle", map[string]any{"action": "reinstall", "requestKey": "u2"}, 202)
	done = waitForJobState(t, ts, rj["id"].(string), "succeeded")
	if !strings.Contains(done["output"].(string), "install -g @earendil-works/pi-coding-agent@latest") {
		t.Fatalf("npm argv: %v", done["output"])
	}
	// Same request key returns the same job.
	again := cliRequest(t, ts, "POST", "/api/clis/pi/lifecycle", map[string]any{"action": "update", "requestKey": "u1"}, 202)
	if again["id"] != j["id"] {
		t.Fatalf("idempotent: %v vs %v", again["id"], j["id"])
	}
	// An unknown install method has no lifecycle: refuse.
	plain := filepath.Join(home, "plain-cli")
	if err := os.WriteFile(plain, []byte("#!/bin/sh\necho x\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	configureCLI(t, ts, "codex", plain, "")
	cliRequest(t, ts, "POST", "/api/clis/codex/check", nil, 200)
	r = cliRequest(t, ts, "POST", "/api/clis/codex/lifecycle", map[string]any{"action": "update", "requestKey": "c1"}, 400)
	if !strings.Contains(r["error"].(string), "no managed lifecycle") {
		t.Fatalf("unknown method: %v", r)
	}
	lifecycle := catalogCLI(t, ts, "codex")["lifecycle"].(map[string]any)
	if lifecycle["canUpdate"] != false || lifecycle["method"] != "unknown" {
		t.Fatalf("codex lifecycle view: %v", lifecycle)
	}
}

func TestCLILifecycleSingleLaneAndPersistence(t *testing.T) {
	ts, dataDir, home := cleanupServer(t)
	exe, npmDir := fakeNpmCLI(t, home, "pi 1.0.0")
	configureCLI(t, ts, "pi", exe, npmDir)
	cliRequest(t, ts, "POST", "/api/clis/pi/check", nil, 200)

	// Make the npm reinstall slow so the lane stays deterministically busy.
	slow := filepath.Join(home, "fake", "bin", "npm")
	if err := os.WriteFile(slow, []byte("#!/bin/sh\nsleep 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	j := cliRequest(t, ts, "POST", "/api/clis/pi/lifecycle", map[string]any{"action": "reinstall", "requestKey": "slow-1"}, 202)
	// Wait until it is actually running, then the lane must refuse another job.
	waitForJobState(t, ts, j["id"].(string), "running")
	cliRequest(t, ts, "POST", "/api/clis/pi/lifecycle", map[string]any{"action": "reinstall", "requestKey": "slow-2"}, 409)
	done := waitForJobState(t, ts, j["id"].(string), "succeeded")
	_ = done
	// The lane frees: a new job starts.
	cliRequest(t, ts, "POST", "/api/clis/pi/lifecycle", map[string]any{"action": "reinstall", "requestKey": "slow-3"}, 202)

	// Jobs survive a server restart (durable records, never replayed).
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	jobs, err := st.CLIJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) < 2 {
		t.Fatalf("persisted jobs: %d", len(jobs))
	}
	st.Close()
}

func TestDetectOnlyCLIsAndMuseChannelCheck(t *testing.T) {
	ts, _, home := cleanupServer(t)
	bin := filepath.Join(home, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	muse := filepath.Join(bin, "muse")
	script := "#!/bin/sh\n# muse-code/launcher\n[ -n \"$MUSE_NO_AUTO_UPDATE\" ] || { echo auto-update; exit 1; }\nif [ \"$1\" = \"--version\" ]; then echo 'Muse Code 1.2.1 (1.2.1-R2847.1)'; fi\n"
	if err := os.WriteFile(muse, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "agy"), []byte("#!/bin/sh\necho 1.2.2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Prepend the fake bin: replacing PATH would hide tmux and the shell the
	// launch itself needs.
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	museRow := catalogCLI(t, ts, "muse")
	if museRow["surface"] != "terminal" || museRow["installed"] != true || museRow["launchable"] != true || museRow["integrationCapable"] != false {
		t.Fatalf("muse = %+v", museRow)
	}
	agyRow := catalogCLI(t, ts, "agy")
	if agyRow["surface"] != "terminal" || agyRow["installed"] != true || agyRow["launchable"] != true || agyRow["integrationCapable"] != false {
		t.Fatalf("agy = %+v", agyRow)
	}
	// Launch settings stay refused: the surface opens a terminal with the
	// CLI's own defaults and edits nothing.
	cliRequest(t, ts, "PUT", "/api/clis/muse", map[string]any{"executable": muse}, 400)
	// Lifecycle jobs stay refused: no managed update for this install.
	cliRequest(t, ts, "POST", "/api/clis/muse/lifecycle", map[string]any{"action": "update", "requestKey": "x"}, 400)

	// New terminal is the one new door: the CLI runs in a PiCode terminal
	// with no integration files behind it.
	if !tmux.New().Available() {
		t.Log("tmux unavailable: terminal creation not exercised")
	} else {
		created := cliRequest(t, ts, "POST", "/api/clis/muse/terminals", map[string]any{"cwd": home, "name": "muse qa"}, 201)
		id, _ := created["id"].(string)
		if id == "" {
			t.Fatalf("terminal = %+v", created)
		}
		rows := cliRequest(t, ts, "GET", "/api/terminals", nil, 200)["terminals"].([]any)
		found := false
		for _, raw := range rows {
			row := raw.(map[string]any)
			if row["id"] == id {
				found = true
				if row["launchCli"] != "muse" {
					t.Fatalf("launched terminal = %+v", row)
				}
				if applied, _ := row["launchApplied"].(map[string]any); applied != nil && applied["injection"] != nil {
					t.Fatalf("muse terminal carries an integration plan: %+v", applied)
				}
			}
		}
		if !found {
			t.Fatalf("created terminal %s is not listed", id)
		}
		cliRequest(t, ts, "DELETE", "/api/terminals/"+id, nil, 204)
	}

	check := cliRequest(t, ts, "POST", "/api/clis/muse/check", map[string]any{}, 200)
	if check["error"] != nil && check["error"] != "" {
		t.Fatalf("check error: %v", check["error"])
	}
	if check["version"] != "Muse Code 1.2.1 (1.2.1-R2847.1)" {
		t.Fatalf("version = %v", check["version"])
	}

	old := museChannelGet
	t.Cleanup(func() { museChannelGet = old })
	museChannelGet = func(ctx context.Context, url string) ([]byte, error) {
		if url != "https://api.meta.ai/muse-code/channels/muse-stable" {
			t.Errorf("channel url %s", url)
		}
		return []byte(`{"version":"1.2.2-R3000.0"}`), nil
	}
	upd := cliRequest(t, ts, "POST", "/api/clis/muse/update-check", map[string]any{}, 200)
	if upd["latest"] != "1.2.2-R3000.0" || upd["updateAvailable"] != true || upd["updateSource"] != "channel" {
		t.Fatalf("update-check = %+v", upd)
	}
	lifecycle := catalogCLI(t, ts, "muse")["lifecycle"].(map[string]any)
	if lifecycle["canCheckUpdate"] != true || lifecycle["canUpdate"] != false {
		t.Fatalf("lifecycle = %+v", lifecycle)
	}
	agyLife := catalogCLI(t, ts, "agy")["lifecycle"].(map[string]any)
	if agyLife["canCheckUpdate"] == true {
		t.Fatalf("agy must not offer update-check yet: %+v", agyLife)
	}
}
