package server

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clilifecycle"
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
	// Full surface serializes omitempty: the key is absent, not "terminal".
	if s, ok := museRow["surface"]; (ok && s != "") || museRow["installed"] != true || museRow["launchable"] != true || museRow["integrationCapable"] != true {
		t.Fatalf("muse = %+v", museRow)
	}
	agyRow := catalogCLI(t, ts, "agy")
	// Full surface serializes omitempty: the key is absent, not "terminal".
	if s, ok := agyRow["surface"]; (ok && s != "") || agyRow["installed"] != true || agyRow["launchable"] != true || agyRow["integrationCapable"] != true {
		t.Fatalf("agy = %+v", agyRow)
	}
	// Launch settings save: the surface opens a terminal with editable
	// defaults (Fatia 3a), but Activity reporting stays refused — 1.3.0
	// offers no hook surface.
	cliRequest(t, ts, "PUT", "/api/clis/muse", map[string]any{"executable": muse}, 200)
	// Wrapper slice: the PATH wrapper is a mechanism, so Activity saves.
	cliRequest(t, ts, "PUT", "/api/clis/muse", map[string]any{"executable": muse, "integration": true}, 200)
	// Same for Antigravity (Fatia 3b): editable defaults, no activity.
	cliRequest(t, ts, "PUT", "/api/clis/agy", map[string]any{"executable": filepath.Join(bin, "agy")}, 200)
	// Fatia 5: the title reporter is a mechanism, so Activity saves (files
	// land on repair, which is a separate notice, not a refusal).
	cliRequest(t, ts, "PUT", "/api/clis/agy", map[string]any{"executable": filepath.Join(bin, "agy"), "integration": true}, 200)
	// Lifecycle jobs are managed now: the launcher entrypoint updates (the
	// fake wrapper exits 0 on a bare run, exactly like INSTALL=1 does).
	cliRequest(t, ts, "POST", "/api/clis/muse/lifecycle", map[string]any{"action": "update", "requestKey": "x"}, 202)

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
				// The wrapper slice: integration is on, so the terminal
				// launches through the PATH wrapper with the presence
				// lease plan behind it.
				applied, _ := row["launchApplied"].(map[string]any)
				if applied == nil || applied["injection"] == nil {
					t.Fatalf("muse terminal carries no integration plan: %+v", row)
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

	old := channelGet
	t.Cleanup(func() { channelGet = old })
	seen := map[string]bool{}
	channelGet = func(ctx context.Context, url string) ([]byte, error) {
		seen[url] = true
		switch {
		case url == "https://api.meta.ai/muse-code/channels/muse-stable":
			return []byte(`{"version":"1.2.2-R3000.0"}`), nil
		case strings.Contains(url, "/manifests/") && strings.HasSuffix(url, ".json"):
			return []byte(`{"version":"9.9.9","url":"https://storage.googleapis.com/x"}`), nil
		default:
			t.Errorf("unexpected channel url %s", url)
			return nil, fmt.Errorf("unexpected url")
		}
	}
	upd := cliRequest(t, ts, "POST", "/api/clis/muse/update-check", map[string]any{}, 200)
	if upd["latest"] != "1.2.2-R3000.0" || upd["updateAvailable"] != true || upd["updateSource"] != "channel" {
		t.Fatalf("muse update-check = %+v", upd)
	}
	lifecycle := catalogCLI(t, ts, "muse")["lifecycle"].(map[string]any)
	if lifecycle["canCheckUpdate"] != true || lifecycle["canUpdate"] != true || lifecycle["canReinstall"] != true || lifecycle["uninstall"] != "guided" {
		t.Fatalf("muse lifecycle = %+v", lifecycle)
	}

	// Antigravity reads the vendor's release manifest: same shape, same menu.
	agyUpdLife := catalogCLI(t, ts, "agy")["lifecycle"].(map[string]any)
	if agyUpdLife["canCheckUpdate"] != true || agyUpdLife["canUpdate"] != true || agyUpdLife["canReinstall"] != true || agyUpdLife["uninstall"] != "guided" {
		t.Fatalf("agy lifecycle = %+v", agyUpdLife)
	}
	// The update check needs the installed version first, exactly as the
	// surface's Check setup provides it.
	check = cliRequest(t, ts, "POST", "/api/clis/agy/check", map[string]any{}, 200)
	if check["error"] != nil && check["error"] != "" {
		t.Fatalf("agy check error: %v", check["error"])
	}
	agyUpd := cliRequest(t, ts, "POST", "/api/clis/agy/update-check", map[string]any{}, 200)
	if agyUpd["latest"] != "9.9.9" || agyUpd["updateAvailable"] != true || agyUpd["updateSource"] != "channel" {
		t.Fatalf("agy update-check = %+v", agyUpd)
	}
	if !seen["https://api.meta.ai/muse-code/channels/muse-stable"] {
		t.Fatal("muse channel was never read")
	}
	if !seen["https://antigravity-cli-auto-updater-974169037036.us-central1.run.app/manifests/"+clilifecycle.AntigravityPlatform(runtime.GOOS, runtime.GOARCH, false)+".json"] {
		t.Fatalf("agy manifest was never read: %v", seen)
	}
}

func TestResolveLifecycleExecParity(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	deps := Deps{Store: st, DataDir: dir}
	bin := filepath.Join(dir, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	muse := filepath.Join(bin, "muse")
	if err := os.WriteFile(muse, []byte("#!/bin/sh\n# muse-code/launcher\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	agy := filepath.Join(bin, "agy")
	if err := os.WriteFile(agy, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := st.SetCLIConfig("muse", clilaunch.Config{Executable: muse}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetCLIConfig("agy", clilaunch.Config{Executable: agy}); err != nil {
		t.Fatal(err)
	}
	museCLI := clilaunch.CLI{ID: "muse", Command: "muse", Name: "Muse Code"}
	agyCLI := clilaunch.CLI{ID: "agy", Command: "agy", Name: "Antigravity"}

	mu, err := resolveLifecycleExec(deps, museCLI, clilifecycle.ActionUpdate)
	if err != nil {
		t.Fatal(err)
	}
	if len(mu.Args) != 0 || mu.Exe != muse {
		t.Fatalf("muse update exec = %+v, want bare launcher run", mu)
	}
	if n := countEnv(mu.Env, "MUSE_LAUNCHER_INSTALL=1"); n != 1 {
		t.Fatalf("muse update env carries %d launcher flags, want exactly 1", n)
	}

	au, err := resolveLifecycleExec(deps, agyCLI, clilifecycle.ActionUpdate)
	if err != nil {
		t.Fatal(err)
	}
	if len(au.Args) != 1 || au.Args[0] != "update" || au.Exe != agy {
		t.Fatalf("agy update exec = %+v, want `agy update`", au)
	}
	for _, kv := range au.Env {
		if k, _, _ := strings.Cut(kv, "="); k == "MUSE_LAUNCHER_INSTALL" {
			t.Fatalf("agy update env leaks the muse entrypoint: %q", kv)
		}
	}

	mr, err := resolveLifecycleExec(deps, museCLI, clilifecycle.ActionReinstall)
	if err != nil {
		t.Fatal(err)
	}
	if n := countEnv(mr.Env, "MUSE_LAUNCHER_INSTALL=1"); n != 1 {
		t.Fatalf("muse reinstall env carries %d launcher flags, want exactly 1", n)
	}
}

func countEnv(env []string, want string) int {
	n := 0
	for _, kv := range env {
		if kv == want {
			n++
		}
	}
	return n
}
