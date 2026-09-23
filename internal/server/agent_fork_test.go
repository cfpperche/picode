package server

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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
