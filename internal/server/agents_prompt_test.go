package server

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

// A workspace agent created with a first prompt (Draft with an agent):
// one row per outcome of startPromptLaunch.
func TestAddWorkspaceAgentWithPrompt(t *testing.T) {
	ts := newTestServer(t, "cat")
	fake := func(name string) string {
		bin := filepath.Join(t.TempDir(), name)
		if err := os.WriteFile(bin, []byte("#!/bin/sh\nexec cat\n"), 0o700); err != nil {
			t.Fatal(err)
		}
		return bin
	}
	cliRequest(t, ts, "PUT", "/api/clis/claude-code", clilaunch.Config{Executable: fake("claude"), Args: []string{"--verbose"}}, 200)
	cliRequest(t, ts, "PUT", "/api/clis/hermes", clilaunch.Config{Executable: fake("hermes")}, 200)
	cliRequest(t, ts, "PUT", "/api/clis/pi", clilaunch.Config{Executable: fake("pi")}, 200)
	wk := cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "proj", "path": t.TempDir()}, 201)
	url := "/api/workspaces/" + wk["id"].(string) + "/agents"
	const prompt = "Write an AGENTS.md for this repository."

	// A terminal CLI starts with the prompt after its own configured args.
	ag := cliRequest(t, ts, "POST", url, map[string]any{"cli": "claude-code", "prompt": prompt}, 201)
	launch := cliRequest(t, ts, "GET", "/api/terminals/"+ag["terminalId"].(string)+"/launch", nil, 200)
	if got := launch["overrides"].(map[string]any)["args"]; !reflect.DeepEqual(got, []any{"--verbose", prompt}) {
		t.Fatalf("claude args = %v", got)
	}

	// A CLI with no prompt form is refused before any row exists.
	before := len(cliRequest(t, ts, "GET", "/api/terminals", nil, 200)["terminals"].([]any))
	cliRequest(t, ts, "POST", url, map[string]any{"cli": "hermes", "prompt": prompt}, 409)
	if after := len(cliRequest(t, ts, "GET", "/api/terminals", nil, 200)["terminals"].([]any)); after != before {
		t.Fatalf("refused prompt left %d terminals", after-before)
	}

	// A prompt and explicit launch arguments do not combine; nor does an
	// oversized prompt.
	cliRequest(t, ts, "POST", url, map[string]any{"cli": "claude-code", "prompt": prompt, "overrides": map[string]any{"args": []string{"-c"}}}, 400)
	cliRequest(t, ts, "POST", url, map[string]any{"cli": "claude-code", "prompt": strings.Repeat("x", maxStartPrompt+1)}, 400)

	// Managed Pi takes the prompt as its first queued task.
	pi := cliRequest(t, ts, "POST", url, map[string]any{"cli": "pi", "prompt": prompt}, 201)
	if pi["terminalId"] != nil {
		t.Fatalf("managed Pi got a terminal: %v", pi)
	}
	var tasks []map[string]any
	getJSON(t, ts, "/api/agents/"+pi["id"].(string)+"/tasks", &tasks)
	if len(tasks) != 1 || tasks[0]["payload"] != prompt || tasks[0]["kind"] != "prompt" {
		t.Fatalf("Pi tasks = %v", tasks)
	}
}
