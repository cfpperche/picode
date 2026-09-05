package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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
	for _, id := range []string{"pi", "claude-code", "codex", "grok"} {
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
		v := cliRequest(t, ts, "POST", "/api/clis/pi/terminals", args, 201)
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
	v := cliRequest(t, ts, "POST", "/api/clis/pi/terminals", map[string]any{"cwd": home}, 201)
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
if [ "$1" = --help ]; then printf '%s\n' --dangerously-bypass-hook-trust; exit; fi
printf '%s\000' "$@" > "$QA_FILE"
exec cat
`
	if err := os.WriteFile(tool, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"pi", "claude-code", "codex", "grok"} {
		out := filepath.Join(home, id+".args")
		c := clilaunch.Config{Executable: tool, Integration: true, Env: map[string]string{"QA_FILE": out}, Args: []string{"two words", "$literal", ""}}
		cliRequest(t, ts, "PUT", "/api/clis/"+id, c, 200)
		v := cliRequest(t, ts, "POST", "/api/clis/"+id+"/terminals", map[string]any{"cwd": home}, 201)
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
	v := cliRequest(t, ts, "POST", "/api/clis/pi/terminals", map[string]any{"workspaceId": ws}, 201)
	id := v["id"].(string)
	session := tmux.ShellSessionName(id)
	t.Cleanup(func() { _ = tmux.New().KillSession(context.Background(), session) })
	pid, _ := tmux.New().PanePID(context.Background(), session)
	root := filepath.Join(data, "cli-launch", id)
	if err := os.Rename(root, root+"-held"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte("blocks mkdir"), 0600); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "POST", "/api/terminals/"+id+"/launch/restart", map[string]any{"confirm": true}, 400)
	if got, _ := tmux.New().PanePID(context.Background(), session); got != pid {
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
