package server

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

// ADR-0184: an agent may be created with launch overrides (a profile, a
// resumed session, a handoff) — the door every user-facing CLI launch
// goes through. Each row of the plan's table is one case here.
func TestAddAgentWithLaunchOverrides(t *testing.T) {
	ts := newTestServer(t, "cat")
	bin := filepath.Join(t.TempDir(), "codex")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexec cat\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "PUT", "/api/clis/codex", clilaunch.Config{Executable: bin}, 200)
	proj := t.TempDir()
	wk := cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "proj", "path": proj}, 201)
	sub := filepath.Join(proj, "sub")

	// Workspace agent: overrides reach its terminal's launch, tools are
	// filled in, and the CLI starts in the agent's own folder.
	ag := cliRequest(t, ts, "POST", "/api/workspaces/"+wk["id"].(string)+"/agents", map[string]any{
		"cli": "codex", "name": "Resume", "workPath": sub, "overrides": map[string]any{"args": []string{"resume", "abc"}},
	}, 201)
	tid, _ := ag["terminalId"].(string)
	if tid == "" {
		t.Fatalf("agent without a terminal: %v", ag)
	}
	launch := cliRequest(t, ts, "GET", "/api/terminals/"+tid+"/launch", nil, 200)
	ov := launch["overrides"].(map[string]any)
	if !reflect.DeepEqual(ov["args"], []any{"resume", "abc"}) || ov["tools"] == nil {
		t.Fatalf("launch overrides = %v", ov)
	}
	terms := cliRequest(t, ts, "GET", "/api/terminals", nil, 200)
	for _, raw := range terms["terminals"].([]any) {
		if tm := raw.(map[string]any); tm["id"] == tid && tm["cwd"] != sub {
			t.Fatalf("terminal cwd = %v, want %s", tm["cwd"], sub)
		}
	}

	// A launch that cannot run is refused before any row exists.
	before := len(cliRequest(t, ts, "GET", "/api/terminals", nil, 200)["terminals"].([]any))
	cliRequest(t, ts, "PUT", "/api/clis/grok", clilaunch.Config{Executable: filepath.Join(t.TempDir(), "missing")}, 200)
	cliRequest(t, ts, "POST", "/api/agents", map[string]any{"cli": "grok", "name": "G", "overrides": map[string]any{}}, 400)
	if after := len(cliRequest(t, ts, "GET", "/api/terminals", nil, 200)["terminals"].([]any)); after != before {
		t.Fatalf("refused launch left %d terminals behind", after-before)
	}

	// Pi owns its session: a reserved flag in the overrides is refused;
	// other args make a Pi agent bound to a launch terminal.
	cliRequest(t, ts, "POST", "/api/agents", map[string]any{"name": "P", "overrides": map[string]any{"args": []string{"--session", "x"}}}, 400)
	pi := cliRequest(t, ts, "POST", "/api/agents", map[string]any{"name": "P", "path": t.TempDir(), "overrides": map[string]any{"args": []string{"--verbose"}}}, 201)
	if pi["cli"] != "pi" || pi["terminalId"] == nil {
		t.Fatalf("Pi agent with overrides = %v", pi)
	}
}

func TestTakeFlagValue(t *testing.T) {
	for _, c := range []struct {
		in   []string
		rest []string
		val  string
	}{
		{[]string{"--session", "/a", "-v"}, []string{"-v"}, "/a"},
		{[]string{"--session=/b"}, []string{}, "/b"},
		{[]string{"-v"}, []string{"-v"}, ""},
		{[]string{"--session"}, []string{"--session"}, ""},
	} {
		rest, val := takeFlagValue(c.in, "--session")
		if !reflect.DeepEqual(rest, c.rest) || val != c.val {
			t.Fatalf("takeFlagValue(%q) = %q, %q", c.in, rest, val)
		}
	}
}

func TestOmpArgsPickSession(t *testing.T) {
	for in, want := range map[string]bool{"--resume sid": true, "--session-dir=/x": true, "--verbose": false, "": false} {
		if got := ompArgsPickSession(strings.Fields(in)); got != want {
			t.Fatalf("ompArgsPickSession(%q) = %v", in, got)
		}
	}
}
