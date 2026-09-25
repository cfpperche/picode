package server

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// Fork agent… for Pi (agent_fork_pi.go): pi's own --fork writes the copy in
// the new agent's private folder before its first start; the agent owns the
// copy (SessionPath), so its launch reopens it with --session and never
// carries --fork; the task waits for the prompt door; lineage is recorded.
func TestForkAgentPiForksIntoTheNewAgentsFolder(t *testing.T) {
	oldWait, oldEvery := forkTaskWait, forkTaskEvery
	forkTaskWait, forkTaskEvery = 2*time.Second, 300*time.Millisecond
	t.Cleanup(func() { forkTaskWait, forkTaskEvery = oldWait, oldEvery })

	ts, deps, _, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, "bin-pi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "pi")
	out := filepath.Join(home, "pi-run")
	script := `#!/bin/sh
for a in "$@"; do if [ "$a" = --fork ]; then
  printf '%s\n' "$PWD" "$@" > "$QA_OUTPUT.fork"
  id= ; sdir=
  while [ $# -gt 0 ]; do
    case "$1" in --session-id) id=$2; shift ;; --session-dir) sdir=$2; shift ;; esac
    shift
  done
  mkdir -p "$sdir"
  printf '{"type":"session","id":"%s"}\n' "$id" > "$sdir/2026-09-25T01-24-00-992Z_$id.jsonl"
  exit 0
fi; done
printf '%s\000' "$@" > "$QA_OUTPUT.args"
exec cat
`
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "PUT", "/api/clis/pi", clilaunch.Config{Executable: bin, Env: map[string]string{"QA_OUTPUT": out}}, 200)
	created := launchFixture(t, ts, "pi", map[string]any{"name": "pi src", "cwd": proj}, 201)
	termID := created["id"].(string)
	t.Cleanup(func() { killTermPane(tmux.ShellSessionName(termID)) })
	waitCLIFile(t, out+".args")
	srcID := created["agentId"].(string)
	src := filepath.Join(session.AgentDir(srcID), "2026-09-24T10-00-00-000Z_src-1.jsonl")
	if err := os.MkdirAll(filepath.Dir(src), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte(`{"type":"session","id":"src-1"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := deps.Store.UpdateAgent(srcID, store.AgentPatch{SessionPath: &src}); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(out + ".args")

	res := cliRequestFull(t, ts, "POST", "/api/agents/"+srcID+"/fork-agent", map[string]any{"name": "pi fork", "prompt": "first line\nsecond line"})
	if res["status"] != "201" {
		t.Fatalf("fork: %v", res)
	}
	body := res["body"].(map[string]any)
	cleanupTerm(t, body)
	if body["task"] != "pending" {
		t.Fatalf("task = %v, want pending", body["task"])
	}
	forkRun := strings.Split(strings.TrimSpace(string(waitCLIFile(t, out+".fork"))), "\n")
	if canonDir(forkRun[0]) != canonDir(proj) || !slices.Contains(forkRun, src) || !slices.Contains(forkRun, "--no-extensions") {
		t.Fatalf("fork run = %q", forkRun)
	}
	forked, err := deps.Store.GetAgent(body["agent"].(map[string]any)["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	if forked.SessionPath == nil || filepath.Dir(*forked.SessionPath) != session.AgentDir(forked.ID) {
		t.Fatalf("fork owns %v, want a file in %s", forked.SessionPath, session.AgentDir(forked.ID))
	}
	if _, err := os.Stat(*forked.SessionPath); err != nil {
		t.Fatal(err)
	}
	args := readArgs(t, out)
	i := slices.Index(args, "--session")
	if i < 0 || args[i+1] != *forked.SessionPath || slices.Contains(args, "--fork") {
		t.Fatalf("launch args = %q", args)
	}
	h := body["handoff"].(map[string]any)
	if h["mode"] != "fork" || h["sourcePath"] != src || h["targetCli"] != "pi" || h["agentId"] != forked.ID || !strings.HasSuffix(*forked.SessionPath, "_"+h["targetId"].(string)+".jsonl") {
		t.Fatalf("lineage = %v", h)
	}
	src2, _ := os.ReadFile(src)
	if !strings.Contains(string(src2), `"src-1"`) {
		t.Fatal("the source changed")
	}
}

// A Pi agent that never had a conversation refuses the fork, and no agent
// is left behind when pi cannot make the copy.
func TestForkAgentPiRefusals(t *testing.T) {
	ts, deps, _, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(home, "pi-fail")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nfor a in \"$@\"; do [ \"$a\" = --fork ] && { echo 'Error: No session found' >&2; exit 1; }; done\nexec cat\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "PUT", "/api/clis/pi", clilaunch.Config{Executable: bin}, 200)
	created := launchFixture(t, ts, "pi", map[string]any{"name": "pi src", "cwd": proj}, 201)
	t.Cleanup(func() { killTermPane(tmux.ShellSessionName(created["id"].(string))) })
	srcID := created["agentId"].(string)
	if res := cliRequestFull(t, ts, "POST", "/api/agents/"+srcID+"/fork-agent", map[string]any{}); res["status"] != "409" {
		t.Fatalf("no conversation: %v", res)
	}
	src := filepath.Join(session.AgentDir(srcID), "2026-09-24T10-00-00-000Z_src-1.jsonl")
	_ = os.MkdirAll(filepath.Dir(src), 0o700)
	_ = os.WriteFile(src, []byte("{}\n"), 0o600)
	_, _ = deps.Store.UpdateAgent(srcID, store.AgentPatch{SessionPath: &src})
	before, _ := deps.Store.ListAllAgents()
	res := cliRequestFull(t, ts, "POST", "/api/agents/"+srcID+"/fork-agent", map[string]any{"name": "doomed"})
	if res["status"] != "502" || !strings.Contains(res["body"].(map[string]any)["error"].(string), "No session found") {
		t.Fatalf("pi failure: %v", res)
	}
	if after, _ := deps.Store.ListAllAgents(); len(after) != len(before) {
		t.Fatalf("an agent was left behind: %d → %d", len(before), len(after))
	}
}
