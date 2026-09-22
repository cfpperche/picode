package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func TestClipCLIVersion(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"fixture-cli 1.0\n", "fixture-cli 1.0"},
		{"\n\nHermes Agent v0.18.2 (2026.7.7.2) · upstream 2a25d53e\nInstall directory: /home/goat/.hermes/hermes-agent\nPython: 3.11.15\n", "Hermes Agent v0.18.2 (2026.7.7.2) · upstream 2a25d53e"},
		{"", ""},
		{strings.Repeat("x", 200), strings.Repeat("x", 160)},
	} {
		if got := clipCLIVersion(tc.in); got != tc.want {
			t.Errorf("clipCLIVersion(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLaunchWithPinnedSession(t *testing.T) {
	if launchWithPinnedSession(nil) != nil {
		t.Fatal("nil launch")
	}
	plain := &store.TerminalLaunch{CLI: "claude-code"}
	if launchWithPinnedSession(plain) != plain {
		t.Fatal("no pin should return the same pointer")
	}
	noID := &store.TerminalLaunch{LastSession: &store.TerminalLastSession{CLI: "claude-code"}}
	if launchWithPinnedSession(noID) != noID {
		t.Fatal("empty session id should return the same pointer")
	}
	noArgs := &store.TerminalLaunch{LastSession: &store.TerminalLastSession{CLI: "claude-code", SessionID: "s1"}}
	if launchWithPinnedSession(noArgs) != noArgs {
		t.Fatal("empty recipe should return the same pointer")
	}
	pi := &store.TerminalLaunch{CLI: "pi", LastSession: &store.TerminalLastSession{CLI: "pi", SessionID: "s1", Path: "/p/session.jsonl"}}
	got := launchWithPinnedSession(pi)
	if got == pi {
		t.Fatal("pi resume should copy")
	}
	if got.Overrides.Args == nil || !reflect.DeepEqual(*got.Overrides.Args, []string{"--session", "/p/session.jsonl"}) {
		t.Fatalf("pi args=%v", got.Overrides.Args)
	}
	if pi.Overrides.Args != nil {
		t.Fatal("mutated original pi launch")
	}
	defaults := []string{"--default"}
	cc := &store.TerminalLaunch{
		CLI:       "claude-code",
		Overrides: clilaunch.Overrides{Args: &defaults},
		LastSession: &store.TerminalLastSession{
			CLI: "claude-code", SessionID: "sess-1", ResumeArgs: []string{"--resume", "sess-1"},
		},
	}
	got = launchWithPinnedSession(cc)
	if got == cc {
		t.Fatal("claude resume should copy")
	}
	if !reflect.DeepEqual(*got.Overrides.Args, []string{"--resume", "sess-1"}) {
		t.Fatalf("claude args=%v", *got.Overrides.Args)
	}
	if !reflect.DeepEqual(*cc.Overrides.Args, []string{"--default"}) {
		t.Fatal("mutated original claude args")
	}
}

func TestCLICheckUsesFirstVersionLine(t *testing.T) {
	ts, _, home := cleanupServer(t)
	binary := filepath.Join(home, "verbose-cli")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then printf 'Hermes Agent v0.18.2\\nInstall directory: /tmp\\n'; exit 0; fi\n"
	if err := os.WriteFile(binary, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "PUT", "/api/clis/hermes", clilaunch.Config{Executable: binary}, 200)
	diag := cliRequest(t, ts, "POST", "/api/clis/hermes/check", map[string]any{}, 200)
	if diag["version"] != "Hermes Agent v0.18.2" || diag["error"] != nil {
		t.Fatalf("diagnostic: %v", diag)
	}
}

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
	// PATH leads with the intercept bin dir (ADR-0180: CLI panes resolve
	// openers by PATH), then the CLI's own configured dirs.
	if env[0] != base.Env["QA_VALUE"] || !strings.HasPrefix(env[1], interceptBinDir(dataDir)+":"+toolDir+":") || env[2] != home {
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

func TestResolveInstalledCLIRetriesThroughAnUpdateWindow(t *testing.T) {
	dir := t.TempDir()
	cli, _ := clilaunch.Find("pi")
	target := filepath.Join(dir, "pi")
	old := resolveRetryDelay
	resolveRetryDelay = 30 * time.Millisecond
	t.Cleanup(func() { resolveRetryDelay = old })
	t.Setenv("PATH", dir)
	// Absent on the first attempt, materialized mid-flight: a vendor
	// self-update swaps its launcher while a request is in flight.
	go func() {
		time.Sleep(10 * time.Millisecond)
		if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o700); err != nil {
			t.Error(err)
		}
	}()
	if _, err := resolveInstalledCLI(cli, clilaunch.Config{}); err != nil {
		t.Fatalf("the retry did not ride out an update window: %v", err)
	}
	// A genuinely absent CLI still fails both attempts — the retry must not
	// paper over a real uninstall.
	if _, err := resolveInstalledCLI(cli, clilaunch.Config{Executable: "/nonexistent/pi"}); err == nil {
		t.Fatal("a genuinely missing CLI resolved")
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
	// replacing the defaults for this one launch. The fixture file is removed
	// first: waitCLIFile returns as soon as the path exists, so the bytes of
	// the creation launch would otherwise be read back as a stale "--default"
	// (sharded runs failed 2/2 at load ~5 while a single process passed). The
	// plain start below uses the same idiom.
	_ = os.Remove(output + ".args")
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

// ADR-0158: Restart on a live Agent CLI terminal reopens the pinned
// conversation (same recipe as start?resume=true). Without a pin it still
// applies current settings to a fresh conversation.
func TestCLITerminalRestartResumesPinnedSession(t *testing.T) {
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

	created := cliRequest(t, ts, "POST", "/api/clis/claude-code/terminals", map[string]any{"name": "Restart fixture", "cwd": home}, 201)
	id := created["id"].(string)
	name := tmux.ShellSessionName(id)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), name) })
	endpoint := "/api/terminals/" + id + "/launch"
	waitCLIFile(t, filepath.Join(output+".args"))

	pid, err := tmux.New().PanePID(context.Background(), name)
	if err != nil || pid <= 0 {
		t.Fatalf("pane pid: %d %v", pid, err)
	}

	// No pin yet: restart applies current settings, fresh conversation.
	_ = os.Remove(output + ".args")
	cliRequest(t, ts, "POST", endpoint+"/restart", map[string]any{"confirm": true}, 200)
	if got := strings.Split(strings.TrimSuffix(string(waitCLIFile(t, filepath.Join(output+".args"))), "\x00"), "\x00"); !reflect.DeepEqual(got, []string{"--default"}) {
		t.Fatalf("restart without pin argv=%q", got)
	}
	next, err := tmux.New().PanePID(context.Background(), name)
	if err != nil || next <= 0 || next == pid {
		t.Fatalf("restart without pin did not replace process: before=%d after=%d err=%v", pid, next, err)
	}
	pid = next

	runtime := "/api/terminals/" + id + "/runtime"
	cliRequest(t, ts, "POST", runtime, map[string]any{"action": "start", "cli": "claude-code", "runId": "run-restart", "pid": os.Getpid()}, 200)
	seed := filepath.Join(home, ".claude", "projects", "fixture", "sess-1.jsonl")
	if err := os.MkdirAll(filepath.Dir(seed), 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"user","message":{"role":"user","content":[{"text":"incident work","type":"text"}]},"timestamp":"` + time.Now().Add(2*time.Second).UTC().Format(time.RFC3339) + `","cwd":"` + home + `","session_id":"sess-1"}`
	if err := os.WriteFile(seed, []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "POST", runtime, map[string]any{"action": "end", "runId": "run-restart"}, 200)

	terms := cliRequest(t, ts, "GET", "/api/terminals", map[string]any{}, 200)
	var view map[string]any
	for _, raw := range terms["terminals"].([]any) {
		if m := raw.(map[string]any); m["id"] == id {
			view = m
		}
	}
	if view == nil || view["lastSession"] == nil {
		t.Fatalf("no pinned lastSession before restart: %v", view)
	}

	_ = os.Remove(output + ".args")
	restarted := cliRequest(t, ts, "POST", endpoint+"/restart", map[string]any{"confirm": true}, 200)
	if restarted["running"] != true {
		t.Fatalf("restart with pin not live: %v", restarted)
	}
	if got := strings.Split(strings.TrimSuffix(string(waitCLIFile(t, filepath.Join(output+".args"))), "\x00"), "\x00"); !reflect.DeepEqual(got, []string{"--resume", "sess-1"}) {
		t.Fatalf("restart with pin argv=%q", got)
	}
	after, err := tmux.New().PanePID(context.Background(), name)
	if err != nil || after <= 0 || after == pid {
		t.Fatalf("restart with pin did not replace process: before=%d after=%d err=%v", pid, after, err)
	}
	if restarted["lastSession"] == nil {
		t.Fatal("restart dropped the pin")
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
	if !strings.Contains(string(raw), "Starting Pi...") {
		t.Fatalf("launch.sh missing start banner:\n%s", raw)
	}
}

func TestNewWiresOpenCodeActivityByDefault(t *testing.T) {
	ts, dataDir, _ := cleanupServer(t)
	v := catalogCLI(t, ts, "opencode")
	cfg, _ := v["config"].(map[string]any)
	if cfg["integration"] != true {
		t.Fatalf("opencode integration default = %#v", cfg["integration"])
	}
	if v["integrationApplied"] != true {
		t.Fatalf("opencode integrationApplied = %#v", v["integrationApplied"])
	}
	if _, err := os.Stat(wrapperPath(dataDir, "opencode")); err != nil {
		t.Fatalf("opencode wrapper missing: %v", err)
	}
}

// Regression for the ADR-0180 follow-up: CLI panes are /bin/sh launch
// scripts, not interactive shells, so the rcfile PATH prepend never runs
// for them. omp resolves its opener by PATH lookup (wslview, then
// xdg-open) and /login opened WSL chromium even with BROWSER set. Both the
// launch script's export and the pane's tmux environment must put the
// intercept bin dir first.
func TestCLILaunchPutsTheInterceptBinFirstOnPATH(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	ts, dataDir, home := cleanupServer(t)
	t.Setenv("SHELL", "/bin/bash")
	binary := filepath.Join(home, "tool")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexec cat\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wrapperPath(dataDir, "xdg-open")); err != nil {
		ensureOpenURLWrappers(dataDir)
	}
	cliRequest(t, ts, "PUT", "/api/clis/pi", clilaunch.Config{Executable: binary}, 200)
	created := cliRequest(t, ts, "POST", "/api/clis/pi/terminals", map[string]any{"name": "PATH fixture", "cwd": home}, 201)
	id := created["id"].(string)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), tmux.ShellSessionName(id)) })

	matches, err := filepath.Glob(filepath.Join(dataDir, "cli-launch", id, "run-*", "launch.sh"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("launch.sh runs = %v, %v", matches, err)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	var pathLine string
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "export PATH=") {
			pathLine = line
			break
		}
	}
	if pathLine == "" {
		t.Fatalf("launch.sh has no PATH export:\n%s", raw)
	}
	if bin := interceptBinDir(dataDir); !strings.Contains(pathLine, "'"+bin+":") && !strings.HasPrefix(strings.TrimPrefix(pathLine, "export PATH="), bin+":") {
		t.Fatalf("launch.sh PATH does not lead with the intercept bin dir (%q): %s", bin, pathLine)
	}

	// The pane's own environment agrees (tmux -e PATH=…). The probe must
	// see the same server the Manager used, so it inherits the process env
	// ($TMUX included); the target name is this fixture's unique session
	// and show-environment is read-only.
	out, err := exec.Command("tmux", "show-environment", "-t", tmux.ShellSessionName(id), "PATH").CombinedOutput()
	if err != nil {
		t.Fatalf("show-environment: %v: %s", err, out)
	}
	envPath := strings.TrimSpace(strings.TrimPrefix(string(out), "PATH="))
	if !strings.HasPrefix(envPath, interceptBinDir(dataDir)+string(os.PathListSeparator)) {
		t.Fatalf("session PATH = %q, want the intercept bin dir first", envPath)
	}
}
