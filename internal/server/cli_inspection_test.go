package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/tmux"
)

func catalogCLI(t *testing.T, ts *httptest.Server, id string) map[string]any {
	t.Helper()
	for _, row := range cliRequest(t, ts, "GET", "/api/clis", nil, 200)["clis"].([]any) {
		v := row.(map[string]any)
		if v["id"] == id {
			return v
		}
	}
	t.Fatal("missing CLI")
	return nil
}

func TestCLIPreviewDecisionTable(t *testing.T) {
	ts, data, home := cleanupServer(t)
	tool := filepath.Join(home, "preview-cli")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\ntouch '"+filepath.Join(home, "EXECUTED")+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"pi", "claude-code", "codex", "grok", "hermes", "opencode"} {
		for _, enabled := range []bool{false, true} {
			c := clilaunch.Config{Executable: tool, Args: []string{"two words", "--api-key", "hidden-argument"}, Env: map[string]string{"SECRET": "hidden-environment"}, Integration: enabled}
			v := cliRequest(t, ts, "POST", "/api/clis/"+id+"/preview", map[string]any{"config": c}, 200)
			raw, _ := json.Marshal(v)
			if strings.Contains(string(raw), "hidden-argument") || strings.Contains(string(raw), "hidden-environment") {
				t.Fatal("preview leaked a value")
			}
			plan := v["plan"].(map[string]any)
			if plan["executable"] != tool {
				t.Fatal(plan)
			}
			injection := plan["injection"].(map[string]any)
			if enabled && len(injection["files"].([]any)) < 2 {
				t.Fatal("missing generated paths")
			}
			if enabled && id == "codex" {
				env := injection["environment"].(map[string]any)
				if env["PICODE_MESSAGES_CLI"] != "codex" || env["PICODE_MESSAGES_DIR"] == "" || env["PICODE_MESSAGES_BIN"] == "" {
					t.Fatal("Codex preview omitted native message discovery", env)
				}
			}
			if !enabled && injection["branches"] != nil {
				t.Fatal("injected with reporting off")
			}
		}
	}
	if _, err := os.Stat(filepath.Join(home, "EXECUTED")); !os.IsNotExist(err) {
		t.Fatal("preview ran a CLI")
	}
	if _, err := os.Stat(filepath.Join(data, "cli-launch")); !os.IsNotExist(err) {
		t.Fatal("preview wrote launch files")
	}
	pi, _ := clilaunch.Find("pi")
	t.Setenv("PATH", home)
	if err := os.Rename(tool, filepath.Join(home, "pi")); err != nil {
		t.Fatal(err)
	}
	p, c := launchPlan(Deps{DataDir: data}, pi, clilaunch.Config{}, clilaunch.Overrides{}, "preview")
	if c.Executable != "" || p.Executable != filepath.Join(home, "pi") || p.Origins["executable"] != "Automatic detection" {
		t.Fatal("automatic path was pinned")
	}
	p, _ = launchPlan(Deps{DataDir: data}, pi, clilaunch.Config{Integration: true, Args: []string{"install"}}, clilaunch.Overrides{}, "preview")
	if len(p.Injection.Branches) != 0 {
		t.Fatal("maintenance args were instrumented")
	}
	p, _ = launchPlan(Deps{DataDir: data}, pi, clilaunch.Config{Executable: "/missing"}, clilaunch.Overrides{}, "preview")
	if p.Problem == "" {
		t.Fatal("missing executable not reported")
	}
	cliRequest(t, ts, "POST", "/api/clis/pi/preview", map[string]any{"overrides": map[string]any{"env": map[string]string{"HOME": "/other"}}}, 400)
	cliRequest(t, ts, "POST", "/api/clis/nope/preview", map[string]any{}, 404)
}

func TestCLISetupCheckAndRepairAreSeparate(t *testing.T) {
	ts, data, home := cleanupServer(t)
	tool := filepath.Join(home, "version-cli")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf 'fixture 2.0\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	c := clilaunch.Config{Executable: tool, Integration: true}
	cliRequest(t, ts, "PUT", "/api/clis/pi", c, 200)
	if err := os.Remove(piTerminalStateExtensionFile(data)); err != nil {
		t.Fatal(err)
	}
	d := cliRequest(t, ts, "POST", "/api/clis/pi/check", map[string]any{}, 200)
	if d["version"] != "fixture 2.0" || d["error"] != nil {
		t.Fatal(d)
	}
	v := catalogCLI(t, ts, "pi")
	if v["integrationApplied"] != false || v["diagnostic"].(map[string]any)["stale"] != false {
		t.Fatal(v)
	}
	cliRequest(t, ts, "POST", "/api/clis/pi/repair", map[string]any{}, 200)
	if catalogCLI(t, ts, "pi")["integrationApplied"] != true {
		t.Fatal("repair did not prepare files")
	}
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf 'fixture 2.0 replaced\\n'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if catalogCLI(t, ts, "pi")["diagnostic"].(map[string]any)["stale"] != true {
		t.Fatal("binary replacement kept a fresh check")
	}
	cliRequest(t, ts, "POST", "/api/clis/pi/check", map[string]any{}, 200)
	c.Args = []string{"--new"}
	cliRequest(t, ts, "PUT", "/api/clis/pi", c, 200)
	if catalogCLI(t, ts, "pi")["diagnostic"].(map[string]any)["stale"] != true {
		t.Fatal("config replacement kept a fresh check")
	}
}

func TestCLIProfileRoutesAndAffectedLaunches(t *testing.T) {
	ts, _, home := cleanupServer(t)
	c := clilaunch.Config{Executable: "/bin/cat", Args: []string{}}
	cliRequest(t, ts, "PUT", "/api/clis/profiles/review", map[string]any{"cli": "pi", "name": "Review", "config": c}, 200)
	profiles := cliRequest(t, ts, "GET", "/api/clis/profiles", nil, 200)["profiles"].([]any)
	if len(profiles) != 1 || profiles[0].(map[string]any)["name"] != "Review" {
		t.Fatal(profiles)
	}
	cliRequest(t, ts, "PUT", "/api/clis/profiles/bad", map[string]any{"cli": "pi", "name": ""}, 400)
	cliRequest(t, ts, "DELETE", "/api/clis/profiles/review", nil, 204)
	cliRequest(t, ts, "DELETE", "/api/clis/profiles/review", nil, 404)
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	cliRequest(t, ts, "PUT", "/api/clis/pi", c, 200)
	for _, explicit := range []bool{false, true} {
		args := map[string]any{"cwd": home, "name": "Inherited"}
		if explicit {
			args["name"] = "Pinned"
			args["overrides"] = map[string]any{"args": []string{}}
		}
		v := launchFixture(t, ts, "pi", args, 201)
		id := v["id"].(string)
		t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), tmux.ShellSessionName(id)) })
	}
	c.Args = []string{"--new"}
	v := cliRequest(t, ts, "POST", "/api/clis/pi/preview", map[string]any{"config": c}, 200)
	affected := v["affected"].([]any)
	if len(affected) != 1 || affected[0].(map[string]any)["name"] != "Inherited" {
		t.Fatal(affected)
	}
	cliRequest(t, ts, "PUT", "/api/clis/pi", c, 200)
	for _, row := range cliRequest(t, ts, "GET", "/api/terminals", nil, 200)["terminals"].([]any) {
		v := row.(map[string]any)
		if v["launchPending"] != (v["name"] == "Inherited") {
			t.Fatalf("incorrect pending state: %v", v)
		}
	}
}

func TestCLICreateFailureRetainsLaunchForRetry(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	ts, data, home := cleanupServer(t)
	cliRequest(t, ts, "PUT", "/api/clis/pi", clilaunch.Config{Executable: "/bin/cat"}, 200)
	root := filepath.Join(data, "cli-launch")
	if err := os.WriteFile(root, []byte("blocks preparation"), 0600); err != nil {
		t.Fatal(err)
	}
	v := launchFixture(t, ts, "pi", map[string]any{"cwd": home}, 201)
	if v["launchError"] == nil {
		t.Fatal("expected saved launch failure")
	}
	id := v["id"].(string)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), tmux.ShellSessionName(id)) })
	v = cliRequest(t, ts, "GET", "/api/terminals/"+id+"/launch", nil, 200)
	if v["attempt"].(map[string]any)["error"] == nil {
		t.Fatal("lost failure")
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	v = cliRequest(t, ts, "POST", "/api/terminals/"+id+"/launch/start", map[string]any{}, 200)
	if v["launchAttempt"] != nil || v["running"] != true {
		t.Fatal("retry did not clear the failure", v)
	}
}

func TestCLIAdapterPreviewMatchesExecution(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	ts, _, home := cleanupServer(t)
	tool := filepath.Join(home, "fake-cli")
	script := `#!/bin/sh
if [ "$1" = config ] && [ "$2" = path ]; then printf '%s\n' "$HOME/.hermes/config.yaml"; exit; fi
if [ "$1" = plugins ]; then exit 0; fi
if [ "$1" = --help ]; then printf '%s\n' --dangerously-bypass-hook-trust; exit; fi
printf '%s\000' "$@" > "$QA_FILE"
exec cat
`
	if err := os.WriteFile(tool, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"pi", "claude-code", "codex", "grok", "hermes", "opencode"} {
		out := filepath.Join(home, id+".args")
		c := clilaunch.Config{Executable: tool, Integration: true, Env: map[string]string{"QA_FILE": out}, Args: []string{"two words", "$literal", ""}}
		cliRequest(t, ts, "PUT", "/api/clis/"+id, c, 200)
		v := launchFixture(t, ts, id, map[string]any{"cwd": home}, 201)
		terminal := v["id"].(string)
		t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), tmux.ShellSessionName(terminal)) })
		var applied clilaunch.Snapshot
		raw, _ := json.Marshal(v["launchApplied"])
		if err := json.Unmarshal(raw, &applied); err != nil {
			t.Fatal(err)
		}
		if applied.Injection == nil {
			t.Fatal(v)
		}
		want := []string{}
		if len(applied.Injection.Branches) > 0 {
			want = append(want, applied.Injection.Branches[0].Args...)
		}
		want = append(want, c.Args...)
		got := strings.Split(strings.TrimSuffix(string(waitCLIFile(t, out)), "\x00"), "\x00")
		// A Pi launch is a Pi agent (ADR-0184): its own flags follow.
		if id == "pi" && len(got) > len(want) {
			got = got[:len(want)]
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s argv=%q want=%q", id, got, want)
		}
		cliRequest(t, ts, "POST", "/api/terminals/"+terminal+"/launch/remove", map[string]any{"confirm": true}, 204)
	}
}

func TestCLIRestartPreparationFailureAndWorkspaceCleanup(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	ts, data, home := cleanupServer(t)
	project := filepath.Join(home, "project")
	_ = os.MkdirAll(project, 0700)
	ws := cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "QA", "path": project}, 201)["id"].(string)
	c := clilaunch.Config{Executable: "/bin/cat"}
	cliRequest(t, ts, "PUT", "/api/clis/pi", c, 200)
	v := launchFixture(t, ts, "pi", map[string]any{"workspaceId": ws}, 201)
	id := v["id"].(string)
	session := tmux.ShellSessionName(id)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), session) })
	pid := panePIDSoon(t, session)
	// A pane that dies keeps its session (and so the suite's private server)
	// with remain-on-exit, so the flake below reports which of its two
	// causes it was: our process exiting (dead pane, its status and last
	// lines) or something ending the session or server (privateTmuxReport).
	if out, err := exec.Command("tmux", "set-option", "-t", session+":", "remain-on-exit", "on").CombinedOutput(); err != nil {
		t.Fatalf("remain-on-exit: %v %s", err, out)
	}
	// The pane existing is not the script running: tmux reports the pane's
	// pid as soon as it forks /bin/sh, and a loaded machine can take longer
	// than that to open launch.sh. Renaming its folder first made sh fail
	// to open the script, the pane died, and — the only session on the
	// suite's private server — took the server with it: the "no server
	// running" flake (2026-09-23..25). The launch banner is the script's
	// first line, so it proves sh is past the open.
	scriptRunning(t, session, "Starting Pi...")
	root := filepath.Join(data, "cli-launch", id)
	if err := os.Rename(root, root+"-held"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte("blocks mkdir"), 0600); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "POST", "/api/terminals/"+id+"/launch/restart", map[string]any{"confirm": true}, 400)
	paneAlive(t, session)
	if got := panePIDSoon(t, session); got != pid {
		t.Fatal("preparation failure killed the old process")
	}
	launch := cliRequest(t, ts, "GET", "/api/terminals/"+id+"/launch", nil, 200)
	if launch["attempt"].(map[string]any)["error"] == nil {
		t.Fatal("failure not retained")
	}
	_ = os.Remove(root)
	_ = os.Rename(root+"-held", root)
	unrelated := filepath.Join(data, "cli-launch", "unrelated", "keep")
	_ = os.MkdirAll(filepath.Dir(unrelated), 0700)
	_ = os.WriteFile(unrelated, []byte("keep"), 0600)
	cliRequest(t, ts, "DELETE", "/api/workspaces/"+ws, nil, 204)
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("workspace leaked private launch files")
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatal("removed unrelated launch")
	}
	if _, err := os.Stat(project); err != nil {
		t.Fatal("removed native project data")
	}
}

// panePIDSoon reads a live pane's pid, retrying briefly. Both reads in the
// restart test used to drop the error: under a loaded 4-shard gate a tmux
// call that failed read as pid 0, and "0 != pid" was reported as a killed
// process (2026-09-23). An answer that never comes is a failure of its own.
func panePIDSoon(t *testing.T, session string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		pid, err := tmux.New().PanePID(context.Background(), session)
		if err == nil && pid > 0 {
			return pid
		}
		if time.Now().After(deadline) {
			t.Fatalf("pane pid of %s: %v (pid %d)\n%s", session, err, pid, privateTmuxReport())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// scriptRunning waits until the pane shows want, the proof its launch script
// is executing rather than merely forked.
func scriptRunning(t *testing.T, session, want string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		out, _ := exec.Command("tmux", "capture-pane", "-p", "-t", session+":").CombinedOutput()
		if strings.Contains(string(out), want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s never showed %q\n--- pane ---\n%s\n%s", session, want, out, privateTmuxReport())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// paneAlive fails with the pane's exit status and last lines when its
// process has died (the session survives it under remain-on-exit).
func paneAlive(t *testing.T, session string) {
	t.Helper()
	out, err := exec.Command("tmux", "display-message", "-p", "-t", session+":", "#{pane_dead} #{pane_dead_status} #{pane_dead_signal}").CombinedOutput()
	if err != nil {
		t.Fatalf("pane state of %s: %v %s\n%s", session, err, out, privateTmuxReport())
	}
	if f := strings.Fields(string(out)); len(f) > 0 && f[0] == "1" {
		lines, _ := exec.Command("tmux", "capture-pane", "-p", "-S", "-40", "-t", session+":").CombinedOutput()
		t.Fatalf("the pane's process died (status/signal %v) — the restart must not touch it\n--- pane ---\n%s", f[1:], lines)
	}
}

// privateTmuxReport says what the suite's private tmux server looked like when a
// pane could not be read, so the next occurrence of the "no server running"
// flake names its cause (2026-09-25: not reproduced in 5 rounds of four
// parallel shards nor 42 isolated runs; the restart handler returns before it
// kills anything). It reports the socket, whether it still exists, and the
// sessions the server lists — a server that is gone points at another test
// ending it; one that lists no session of ours points at our pane dying.
func privateTmuxReport() string {
	dir := os.Getenv(tmux.SocketDirEnv)
	sock := filepath.Join(dir, "tmux-"+strconv.Itoa(os.Getuid()), "default")
	_, statErr := os.Stat(sock)
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name} #{session_created}").CombinedOutput()
	return fmt.Sprintf("tmux state: socket %s (stat: %v); list-sessions: %v\n%s", sock, statErr, err, out)
}
