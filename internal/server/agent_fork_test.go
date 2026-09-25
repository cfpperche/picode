package server

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// forkSource launches a CLI agent on a fake executable and pins sessionID
// as its conversation, the state Fork agent… starts from.
func forkSource(t *testing.T, cli, sessionID string) (agentID, proj, out string, deps Deps, request func(body map[string]any) map[string]any) {
	t.Helper()
	ts, d, _, home := handoffServer(t)
	proj = filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	_, out = fakeCLI(t, ts, home, cli)
	created := launchFixture(t, ts, cli, map[string]any{"name": "clis", "cwd": proj}, 201)
	termID := created["id"].(string)
	t.Cleanup(func() { killTermPane(tmux.ShellSessionName(termID)) })
	waitCLIFile(t, out+".args")
	if sessionID != "" {
		if err := d.Store.SetTerminalLastSession(termID, store.TerminalLastSession{CLI: cli, SessionID: sessionID, Cwd: proj, UpdatedAt: "2026-09-23T10:00:00Z"}); err != nil {
			t.Fatal(err)
		}
	}
	// The fork's run writes the same file; start from none so the read
	// below cannot be the source's own launch.
	_ = os.Remove(out + ".args")
	agentID = created["agentId"].(string)
	request = func(body map[string]any) map[string]any {
		return cliRequestFull(t, ts, "POST", "/api/agents/"+agentID+"/fork-agent", body)
	}
	return agentID, proj, out, d, request
}

func readArgs(t *testing.T, out string) []string {
	t.Helper()
	raw := strings.TrimSuffix(string(waitCLIFile(t, out+".args")), "\x00")
	return strings.Split(raw, "\x00")
}

func TestForkAgentClaudeCarriesTaskAndFiles(t *testing.T) {
	srcID, proj, out, deps, fork := forkSource(t, "claude-code", "cc-1")
	res := fork(map[string]any{
		"name":   "fix the race",
		"prompt": "-fix the failure\nyou found",
		"files":  []map[string]any{{"name": "trace.txt", "mime": "text/plain", "data": base64.StdEncoding.EncodeToString([]byte("panic: race"))}},
	})
	if res["status"] != "201" {
		t.Fatalf("fork: %v", res)
	}
	body := res["body"].(map[string]any)
	cleanupTerm(t, body)

	args := readArgs(t, out)
	if len(args) != 6 || !reflect.DeepEqual(args[:4], []string{"--resume", "cc-1", "--fork-session", "--session-id"}) {
		t.Fatalf("claude fork args = %q", args)
	}
	newID := args[4]
	// The task is one line and stays an operand (a leading dash would read
	// as a flag); the staged file follows as a mention.
	if !strings.HasPrefix(args[5], " -fix the failure you found @.picode/drop/") || !strings.HasSuffix(args[5], "-trace.txt") {
		t.Fatalf("prompt = %q", args[5])
	}
	staged := filepath.Join(proj, strings.SplitN(args[5], " @", 2)[1])
	if b, err := os.ReadFile(staged); err != nil || string(b) != "panic: race" {
		t.Fatalf("staged file %s: %q %v", staged, b, err)
	}

	forked := body["agent"].(map[string]any)
	if forked["name"] != "fix the race" || forked["id"] == srcID {
		t.Fatalf("fork agent = %v", forked)
	}
	a, err := deps.Store.GetAgent(forked["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	// Pinned at once: a restart resumes the copy instead of forking again.
	launch, err := deps.Store.TerminalLaunch(*a.TerminalID)
	if err != nil || launch.LastSession == nil || launch.LastSession.SessionID != newID || !reflect.DeepEqual(launch.LastSession.ResumeArgs, []string{"--resume", newID}) {
		t.Fatalf("fork pin = %+v (%v)", launch, err)
	}
	h := body["handoff"].(map[string]any)
	if h["mode"] != "fork" || h["sourceId"] != "cc-1" || h["targetId"] != newID || h["targetCli"] != "claude-code" || h["agentId"] != forked["id"] {
		t.Fatalf("lineage = %v", h)
	}
}

func TestForkAgentCodexInAnotherFolder(t *testing.T) {
	_, _, out, deps, fork := forkSource(t, "codex", "cx-1")
	wt := filepath.Join(t.TempDir(), "wt")
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	res := fork(map[string]any{"prompt": "fix it", "workPath": wt})
	if res["status"] != "201" {
		t.Fatalf("fork: %v", res)
	}
	body := res["body"].(map[string]any)
	cleanupTerm(t, body)
	if args := readArgs(t, out); !reflect.DeepEqual(args, []string{"fork", "cx-1", "fix it"}) {
		t.Fatalf("codex fork args = %q", args)
	}
	forked := body["agent"].(map[string]any)
	if forked["name"] != "clis fork" {
		t.Fatalf("default name = %v", forked["name"])
	}
	a, _ := deps.Store.GetAgent(forked["id"].(string))
	term, _ := deps.Store.GetTerminal(*a.TerminalID)
	if canonDir(term.Cwd) != canonDir(wt) {
		t.Fatalf("fork runs in %q, want %q", term.Cwd, wt)
	}
	// Codex names the copy itself: no pin until its first turn reports.
	if launch, _ := deps.Store.TerminalLaunch(*a.TerminalID); launch.LastSession != nil {
		t.Fatalf("codex fork pinned early: %+v", launch.LastSession)
	}
}

func TestForkAgentRefusals(t *testing.T) {
	t.Run("no conversation yet", func(t *testing.T) {
		_, _, _, _, fork := forkSource(t, "claude-code", "")
		if res := fork(map[string]any{"prompt": "x"}); res["status"] != "409" {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("a CLI without a command-line fork", func(t *testing.T) {
		_, _, out, _, fork := forkSource(t, "hermes", "h-1")
		res := fork(map[string]any{"prompt": "x"})
		if res["status"] != "400" || !strings.Contains(res["body"].(map[string]any)["error"].(string), "can't fork") {
			t.Fatalf("got %v", res)
		}
		time.Sleep(200 * time.Millisecond)
		if _, err := os.Stat(out + ".args"); err == nil {
			t.Fatal("a refused fork launched something")
		}
	})
	t.Run("too many files", func(t *testing.T) {
		_, _, _, _, fork := forkSource(t, "codex", "cx-1")
		res := fork(map[string]any{"paths": []string{"a", "b", "c"}, "files": []map[string]any{{"name": "d", "data": "ZA=="}, {"name": "e", "data": "ZQ=="}}})
		if res["status"] != "400" {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("a task over the launch limit", func(t *testing.T) {
		_, _, _, _, fork := forkSource(t, "codex", "cx-1")
		if res := fork(map[string]any{"prompt": strings.Repeat("x ", 5000)}); res["status"] != "400" {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("a folder that does not exist", func(t *testing.T) {
		_, _, _, _, fork := forkSource(t, "codex", "cx-1")
		if res := fork(map[string]any{"workPath": "/nonexistent/picode-fork"}); res["status"] != "400" {
			t.Fatalf("got %v", res)
		}
	})
}

// The sidebar learns who a fork came from through the agent list: the
// source's live name while it exists, its name at fork time once removed.
func TestForkAgentListsItsOrigin(t *testing.T) {
	srcID, _, _, deps, fork := forkSource(t, "codex", "cx-1")
	res := fork(map[string]any{"name": "side quest", "prompt": "x"})
	if res["status"] != "201" {
		t.Fatalf("fork: %v", res)
	}
	body := res["body"].(map[string]any)
	cleanupTerm(t, body)
	forkID := body["agent"].(map[string]any)["id"].(string)

	origin := func() map[string]any {
		origins := forkOrigins(deps)
		o := origins[forkID]
		if o == nil {
			return nil
		}
		return map[string]any{"agentId": o.AgentID, "name": o.Name, "gone": o.Gone}
	}
	if got := origin(); got == nil || got["agentId"] != srcID || got["name"] != "clis" || got["gone"] != false {
		t.Fatalf("origin = %v", got)
	}
	if _, ok := forkOrigins(deps)[srcID]; ok {
		t.Fatal("the source is not a fork of anything")
	}
	newName := "clis renamed"
	if _, err := deps.Store.UpdateAgent(srcID, store.AgentPatch{Name: &newName}); err != nil {
		t.Fatal(err)
	}
	if got := origin(); got["name"] != "clis renamed" {
		t.Fatalf("renamed source = %v", got)
	}
	if err := deps.Store.DeleteAgent(srcID); err != nil {
		t.Fatal(err)
	}
	// Gone: only the name recorded at fork time is left.
	if got := origin(); got["name"] != "clis" || got["gone"] != true {
		t.Fatalf("removed source = %v", got)
	}
}

// Muse Code forks through `muse serve` (MSP session/fork), opens the copy
// with `muse resume <id>`, and gets its task — line breaks kept — through
// the prompt door once the TUI is up.
func TestForkAgentMuseForksThroughItsProtocol(t *testing.T) {
	oldWait, oldEvery := forkTaskWait, forkTaskEvery
	forkTaskWait, forkTaskEvery = 8*time.Second, 300*time.Millisecond
	t.Cleanup(func() { forkTaskWait, forkTaskEvery = oldWait, oldEvery })

	ts, deps, _, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, "bin-muse")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "muse")
	out := filepath.Join(home, "muse-run")
	script := `#!/bin/sh
if [ "$1" = --version ]; then printf 'muse 1.3.0\n'; exit 0; fi
if [ "$1" = serve ]; then
  pwd > "$QA_OUTPUT.serve-cwd"
  read a; printf '%s\n' '{"id":1,"jsonrpc":"2.0","result":{}}'
  read b; read c; printf '%s\n' "$c" > "$QA_OUTPUT.fork-request"
  printf '%s\n' '{"id":2,"jsonrpc":"2.0","result":{"session":{"sessionId":"mfork-1","forkedFrom":{"sessionId":"ms-1"}}}}'
  read d; exit 0
fi
printf '%s\000' "$@" > "$QA_OUTPUT.args"
exec cat
`
	if err := os.WriteFile(bin, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "PUT", "/api/clis/muse", clilaunch.Config{Executable: bin, Env: map[string]string{"QA_OUTPUT": out}}, 200)
	created := launchFixture(t, ts, "muse", map[string]any{"name": "muse src", "cwd": proj}, 201)
	termID := created["id"].(string)
	t.Cleanup(func() { killTermPane(tmux.ShellSessionName(termID)) })
	waitCLIFile(t, out+".args")
	if err := deps.Store.SetTerminalLastSession(termID, store.TerminalLastSession{CLI: "muse", SessionID: "ms-1", Cwd: proj, UpdatedAt: "2026-09-24T10:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(out + ".args")

	res := cliRequestFull(t, ts, "POST", "/api/agents/"+created["agentId"].(string)+"/fork-agent", map[string]any{"name": "muse fork", "prompt": "first line\nsecond line"})
	if res["status"] != "201" {
		t.Fatalf("fork: %v", res)
	}
	body := res["body"].(map[string]any)
	cleanupTerm(t, body)
	if body["task"] != "pending" {
		t.Fatalf("task = %v, want pending", body["task"])
	}
	if req := string(waitCLIFile(t, out+".fork-request")); !strings.Contains(req, `"session/fork"`) || !strings.Contains(req, `"sessionId":"ms-1"`) || !strings.Contains(req, `"commandId":"`) {
		t.Fatalf("fork request = %s", req)
	}
	if cwd := strings.TrimSpace(string(waitCLIFile(t, out+".serve-cwd"))); canonDir(cwd) != canonDir(proj) {
		t.Fatalf("muse serve ran in %q, want the source's folder", cwd)
	}
	if args := readArgs(t, out); !reflect.DeepEqual(args, []string{"resume", "mfork-1"}) {
		t.Fatalf("launch args = %q", args)
	}
	forked := body["agent"].(map[string]any)
	a, _ := deps.Store.GetAgent(forked["id"].(string))
	launch, _ := deps.Store.TerminalLaunch(*a.TerminalID)
	if launch.LastSession == nil || launch.LastSession.SessionID != "mfork-1" || !reflect.DeepEqual(launch.LastSession.ResumeArgs, []string{"resume", "mfork-1"}) {
		t.Fatalf("pin = %+v", launch.LastSession)
	}
	h := body["handoff"].(map[string]any)
	if h["mode"] != "fork" || h["targetId"] != "mfork-1" {
		t.Fatalf("lineage = %v", h)
	}

	// The task reaches the new TUI (here `cat`, which echoes what it got),
	// or — if the door never takes it — lands in the Inbox with its text.
	pane := tmux.ShellSessionName(*a.TerminalID)
	deadline := time.Now().Add(forkTaskWait + 3*time.Second)
	for {
		screen, _ := tmux.New().CaptureTail(context.Background(), pane, 40)
		if strings.Contains(screen, "second line") {
			return
		}
		items, _ := deps.Store.ListInboxItems(store.InboxFilter{})
		for _, it := range items {
			if it.Reason == "fork task not delivered" {
				if !strings.Contains(it.Body, "first line\nsecond line") {
					t.Fatalf("inbox body lost the task: %q", it.Body)
				}
				t.Logf("task went to the Inbox: %s", it.Body)
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("task neither delivered nor reported; pane:\n%s", screen)
		}
		time.Sleep(300 * time.Millisecond)
	}
}
