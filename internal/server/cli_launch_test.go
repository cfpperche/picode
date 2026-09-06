package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/tmux"
)

func cliRequest(t *testing.T, ts *httptest.Server, method, path string, body any, want int) map[string]any {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	res := do(t, ts.Client(), req)
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		t.Fatalf("%s %s: %d, want %d: %s", method, path, res.StatusCode, want, b)
	}
	var out map[string]any
	if len(b) > 0 {
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func waitCLIFile(t *testing.T, path string) []byte {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		if raw, err := os.ReadFile(path); err == nil {
			return raw
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("launch did not produce %s", path)
	return nil
}

func TestCLITerminalLifecycleDecisionTable(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	ts, dataDir, home := cleanupServer(t)
	t.Setenv("SHELL", "/bin/bash")
	toolDir := filepath.Join(home, "tools with spaces")
	if err := os.MkdirAll(toolDir, 0o700); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(toolDir, "fake-cli")
	// The fixture records exact values, then remains attached to its terminal.
	// --version is deliberately side-effect free and never uses launch arguments.
	script := `#!/bin/sh
if [ "$1" = --version ]; then printf 'fixture-cli 1.0\n'; exit 0; fi
printf '%s\000' "$@" > "$QA_OUTPUT.args"
printf '%s\000' "$QA_VALUE" "$PATH" "$PWD" > "$QA_OUTPUT.env"
exec cat
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	args := []string{"two words", "'quoted'", "$(touch NEVER_RUN)", "", "--api-key", "private-arg"}
	base := clilaunch.Config{Executable: binary, Args: args, Env: map[string]string{"QA_OUTPUT": filepath.Join(home, "first"), "QA_VALUE": "literal '$value'"}, Path: []string{toolDir}}
	cliRequest(t, ts, "PUT", "/api/clis/pi", base, 200)
	diag := cliRequest(t, ts, "POST", "/api/clis/pi/check", map[string]any{}, 200)
	if diag["version"] != "fixture-cli 1.0" || diag["error"] != nil {
		t.Fatalf("diagnostic: %v", diag)
	}
	if _, err := os.Stat(filepath.Join(home, "first.args")); !os.IsNotExist(err) {
		t.Fatal("check started a conversation")
	}
	created := cliRequest(t, ts, "POST", "/api/clis/pi/terminals", map[string]any{"name": "Launch fixture", "cwd": home}, 201)
	id := created["id"].(string)
	name := tmux.ShellSessionName(id)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), name) })
	endpoint := "/api/terminals/" + id + "/launch"
	raw := waitCLIFile(t, filepath.Join(home, "first.args"))
	if got := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00"); !reflect.DeepEqual(got, args) {
		t.Fatalf("argv=%q", got)
	}
	env := strings.Split(string(waitCLIFile(t, filepath.Join(home, "first.env"))), "\x00")
	if env[0] != base.Env["QA_VALUE"] || !strings.HasPrefix(env[1], toolDir+":") || env[2] != home {
		t.Fatalf("environment=%q", env)
	}
	if _, err := os.Stat(filepath.Join(home, "NEVER_RUN")); !os.IsNotExist(err) {
		t.Fatal("evaluated shell argument")
	}
	if created["cli"] != nil || created["state"] != nil {
		t.Fatal("configuration invented activity")
	}
	applied := created["launchApplied"].(map[string]any)
	if applied["cli"] != "pi" || created["launchPending"] != false {
		t.Fatalf("snapshot=%v", created)
	}
	b, _ := json.Marshal(created)
	if strings.Contains(string(b), "private-arg") || strings.Contains(string(b), base.Env["QA_VALUE"]) {
		t.Fatal("diagnostics leaked a value")
	}

	// Live Start is idempotent, including concurrent requests and browser reopen.
	pid, _ := tmux.New().PanePID(context.Background(), name)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); cliRequest(t, ts, "POST", endpoint+"/start", map[string]any{}, 200) }()
	}
	wg.Wait()
	cliRequest(t, ts, "POST", "/api/terminals/"+id+"/open", map[string]any{}, 200)
	if next, _ := tmux.New().PanePID(context.Background(), name); next != pid {
		t.Fatal("live Start replaced process")
	}

	base.Env["QA_OUTPUT"] = filepath.Join(home, "second")
	base.Env["QA_VALUE"] = "next launch"
	cliRequest(t, ts, "PUT", "/api/clis/pi", base, 200)
	current := cliRequest(t, ts, "POST", endpoint+"/start", map[string]any{}, 200)
	if current["launchPending"] != true {
		t.Fatal("missing pending changes")
	}
	if _, err := os.Stat(filepath.Join(home, "second.args")); !os.IsNotExist(err) {
		t.Fatal("saving settings restarted work")
	}
	for _, action := range []string{"stop", "restart", "remove"} {
		cliRequest(t, ts, "POST", endpoint+"/"+action, map[string]any{}, 409)
	}
	if next, _ := tmux.New().PanePID(context.Background(), name); next != pid {
		t.Fatal("unconfirmed action interrupted work")
	}

	// Invalid next launch must not destroy the existing one.
	missing := filepath.Join(home, "missing")
	cliRequest(t, ts, "PUT", endpoint, map[string]any{"cli": "pi", "overrides": map[string]any{"executable": missing}}, 200)
	cliRequest(t, ts, "POST", endpoint+"/restart", map[string]any{"confirm": true}, 400)
	if next, _ := tmux.New().PanePID(context.Background(), name); next != pid {
		t.Fatal("invalid restart killed live process")
	}
	cliRequest(t, ts, "PUT", endpoint, map[string]any{"cli": "pi", "overrides": map[string]any{"args": []string{"override"}}}, 200)
	restarted := cliRequest(t, ts, "POST", endpoint+"/restart", map[string]any{"confirm": true}, 200)
	if restarted["launchPending"] != false {
		t.Fatal("restart did not apply settings")
	}
	if got := string(waitCLIFile(t, filepath.Join(home, "second.args"))); got != "override\x00" {
		t.Fatalf("override=%q", got)
	}
	entries, _ := os.ReadDir(filepath.Join(dataDir, "cli-launch", id))
	if len(entries) != 1 {
		t.Fatalf("kept old launch artifacts: %v", entries)
	}

	stopped := cliRequest(t, ts, "POST", endpoint+"/stop", map[string]any{"confirm": true}, 200)
	if stopped["running"] != false {
		t.Fatal("stop not reflected")
	}
	saved := cliRequest(t, ts, "GET", endpoint, nil, 200)
	if saved["cli"] != "pi" {
		t.Fatal("stop removed launch settings")
	}
	restored := cliRequest(t, ts, "POST", "/api/terminals/"+id+"/open", map[string]any{}, 200)
	if restored["running"] != false {
		t.Fatal("browser restoration restarted stopped work")
	}
	cliRequest(t, ts, "POST", endpoint+"/start", map[string]any{}, 200)
	// Another terminal is outside the action's scope.
	other := cliRequest(t, ts, "POST", "/api/terminals", map[string]any{"name": "Unrelated", "cwd": home}, 201)
	otherName := tmux.ShellSessionName(other["id"].(string))
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), otherName) })
	cliRequest(t, ts, "POST", endpoint+"/remove", map[string]any{"confirm": true}, 204)
	cliRequest(t, ts, "GET", endpoint, nil, 404)
	if live, _ := tmux.New().HasSession(context.Background(), otherName); !live {
		t.Fatal("removed another terminal")
	}
	if _, err := os.Stat(filepath.Join(dataDir, "cli-launch", id)); !os.IsNotExist(err) {
		t.Fatal("launch artifacts not removed")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatal("removed user-owned CLI")
	}
}

func TestCLITerminalRejectsInvalidRequests(t *testing.T) {
	ts, _, home := cleanupServer(t)
	for _, config := range []any{
		map[string]any{"env": map[string]string{"PICODE_TERM_ID": "someone-else"}},
		map[string]any{"args": []string{"new\nline"}},
		map[string]any{"path": []string{"relative"}},
		map[string]any{"unknown": true},
	} {
		cliRequest(t, ts, "PUT", "/api/clis/pi", config, 400)
	}
	cliRequest(t, ts, "PUT", "/api/clis/unknown", map[string]any{}, 404)
	if !tmux.New().Available() {
		return
	}
	cliRequest(t, ts, "PUT", "/api/clis/pi", clilaunch.Config{Executable: "/bin/cat"}, 200)
	cliRequest(t, ts, "POST", "/api/clis/pi/terminals", map[string]any{"cwd": filepath.Join(home, "missing")}, 400)
	cliRequest(t, ts, "POST", "/api/clis/pi/terminals", map[string]any{"cwd": home, "overrides": map[string]any{"executable": "/not-installed"}}, 400)
	listed := cliRequest(t, ts, "GET", "/api/terminals", nil, 200)
	if len(listed["terminals"].([]any)) != 0 {
		t.Fatal("invalid request created a terminal")
	}
}

func TestCLIExecutableSkipsWrapperAndRejectsRelativePath(t *testing.T) {
	dir := t.TempDir()
	cli, _ := clilaunch.Find("pi")
	if err := os.WriteFile(filepath.Join(dir, "pi"), []byte("#!/bin/sh\n# PiCode intercept\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if _, err := resolveCLIExecutable(cli, clilaunch.Config{}); err == nil {
		t.Fatal("resolved own wrapper")
	}
	if _, err := resolveCLIExecutable(cli, clilaunch.Config{Executable: "relative/pi"}); err == nil {
		t.Fatal("resolved relative executable")
	}
	if err := cleanCLILaunches(dir, "..", ""); err == nil {
		t.Fatal("unbounded cleanup allowed")
	}
}

func TestCLITerminalWithoutTmuxKeepsConfigurationAvailable(t *testing.T) {
	ts, _, home := cleanupServer(t)
	t.Setenv("PATH", t.TempDir())
	listed := cliRequest(t, ts, "GET", "/api/clis", nil, 200)
	if listed["terminalAvailable"] != false {
		t.Fatal("missing tmux reported as available")
	}
	cliRequest(t, ts, "PUT", "/api/clis/pi", clilaunch.Config{Executable: "/bin/cat"}, 200)
	cliRequest(t, ts, "POST", "/api/clis/pi/terminals", map[string]any{"cwd": home}, 503)
	terms := cliRequest(t, ts, "GET", "/api/terminals", nil, 200)
	if len(terms["terminals"].([]any)) != 0 {
		t.Fatal("blocked launch created a terminal")
	}
}

// ADR-0084 decision table: start-with-resume recovers the pinned native
// conversation of a stopped CLI terminal; every other condition refuses
// cleanly, and a plain start keeps the CLI defaults.
func TestCLITerminalResumeDecisionTable(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	ts, _, home := cleanupServer(t)
	t.Setenv("SHELL", "/bin/bash")
	toolDir := filepath.Join(home, "tools")
	if err := os.MkdirAll(toolDir, 0o700); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(home, "out")
	binary := filepath.Join(toolDir, "fake-cli")
	script := `#!/bin/sh
if [ "$1" = --version ]; then printf 'fixture-cli 1.0\n'; exit 0; fi
printf '%s\000' "$@" > "$QA_OUTPUT.args"
exec cat
`
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	base := clilaunch.Config{Executable: binary, Args: []string{"--default"}, Env: map[string]string{"QA_OUTPUT": output}, Path: []string{toolDir}}
	cliRequest(t, ts, "PUT", "/api/clis/claude-code", base, 200)

	created := cliRequest(t, ts, "POST", "/api/clis/claude-code/terminals", map[string]any{"name": "Resume fixture", "cwd": home}, 201)
	id := created["id"].(string)
	name := tmux.ShellSessionName(id)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), name) })
	endpoint := "/api/terminals/" + id + "/launch"
	waitCLIFile(t, filepath.Join(output+".args")) // creation launches the CLI

	// Stop it: the terminal is stopped, no conversation pinned yet.
	cliRequest(t, ts, "POST", endpoint+"/stop", map[string]any{"confirm": true}, 200)
	resume := func() map[string]any {
		return cliRequestFull(t, ts, "POST", endpoint+"/start", map[string]any{"resume": true})
	}
	if res := resume(); res["status"] != "400" {
		t.Fatalf("resume without pin: %v", res)
	}

	// The wrapper reports a run; its end pins the newest native session
	// written during that run.
	runtime := "/api/terminals/" + id + "/runtime"
	cliRequest(t, ts, "POST", runtime, map[string]any{"action": "start", "cli": "claude-code", "runId": "run-1", "pid": os.Getpid()}, 200)
	seed := filepath.Join(home, ".claude", "projects", "fixture", "sess-1.jsonl")
	if err := os.MkdirAll(filepath.Dir(seed), 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"user","message":{"role":"user","content":[{"text":"incident work","type":"text"}]},"timestamp":"` + time.Now().Add(2*time.Second).UTC().Format(time.RFC3339) + `","cwd":"` + home + `","session_id":"sess-1"}`
	if err := os.WriteFile(seed, []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "POST", runtime, map[string]any{"action": "end", "runId": "run-1"}, 200)

	terms := cliRequest(t, ts, "GET", "/api/terminals", map[string]any{}, 200)
	list := terms["terminals"].([]any)
	var view map[string]any
	for _, raw := range list {
		if m := raw.(map[string]any); m["id"] == id {
			view = m
		}
	}
	if view == nil || view["lastSession"] == nil {
		t.Fatalf("no pinned lastSession after runtime end: %v", view)
	}

	// Resume restarts the CLI with the session's verified resume args,
	// replacing the defaults for this one launch.
	res := resume()
	if res["status"] != "200" {
		t.Fatalf("resume with pin: %v", res)
	}
	if got := strings.Split(strings.TrimSuffix(string(waitCLIFile(t, filepath.Join(output+".args"))), "\x00"), "\x00"); !reflect.DeepEqual(got, []string{"--resume", "sess-1"}) {
		t.Fatalf("resume argv=%q", got)
	}
	if live, _ := tmux.New().HasSession(context.Background(), name); !live {
		t.Fatal("resumed terminal is not live")
	}

	// Already live: refuse instead of double-launching.
	if res := resume(); res["status"] != "409" {
		t.Fatalf("resume while live: %v", res)
	}

	// Plain start stays opt-in: stop, then start without resume — defaults.
	cliRequest(t, ts, "POST", endpoint+"/stop", map[string]any{"confirm": true}, 200)
	_ = os.Remove(output + ".args")
	cliRequest(t, ts, "POST", endpoint+"/start", map[string]any{}, 200)
	if got := strings.Split(strings.TrimSuffix(string(waitCLIFile(t, filepath.Join(output+".args"))), "\x00"), "\x00"); !reflect.DeepEqual(got, []string{"--default"}) {
		t.Fatalf("plain start argv=%q", got)
	}

	// A terminal without a CLI launch has nothing to resume.
	plain := cliRequest(t, ts, "POST", "/api/terminals", map[string]any{"name": "plain", "cwd": home}, 201)
	plainID := plain["id"].(string)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), tmux.ShellSessionName(plainID)) })
	res = cliRequestFull(t, ts, "POST", "/api/terminals/"+plainID+"/launch/start", map[string]any{"resume": true})
	if res["status"] != "400" {
		t.Fatalf("resume without launch: %v", res)
	}
}

// cliRequestFull is cliRequest without the fatal status check: resume has a
// decision table, so non-2xx statuses carry the expected branch.
func cliRequestFull(t *testing.T, ts *httptest.Server, method, path string, body any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	res := do(t, ts.Client(), req)
	b, _ := io.ReadAll(res.Body)
	out := map[string]any{"status": strconv.Itoa(res.StatusCode)}
	if len(b) > 0 {
		var parsed map[string]any
		if json.Unmarshal(b, &parsed) == nil {
			out["body"] = parsed
		}
	}
	return out
}

// The generated launch script is the pane root (ADR-0085): it must ignore
// SIGHUP from its first executable line, right after the shebang.
func TestLaunchScriptIgnoresHUP(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	ts, dataDir, home := cleanupServer(t)
	t.Setenv("SHELL", "/bin/bash")
	binary := filepath.Join(home, "tool")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexec cat\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "PUT", "/api/clis/pi", clilaunch.Config{Executable: binary}, 200)
	created := cliRequest(t, ts, "POST", "/api/clis/pi/terminals", map[string]any{"name": "HUP fixture", "cwd": home}, 201)
	id := created["id"].(string)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), tmux.ShellSessionName(id)) })

	// The kept run generation holds the script the pane is running.
	matches, err := filepath.Glob(filepath.Join(dataDir, "cli-launch", id, "run-*", "launch.sh"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("launch.sh runs = %v, %v", matches, err)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "#!") || !strings.Contains(lines[2], "trap '' HUP") {
		t.Fatalf("launch.sh head = %q", lines[:3])
	}
}
