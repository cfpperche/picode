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
