package server

import (
	"encoding/json"
	"github.com/cfpperche/picode/internal/rpc"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// ADR-0205's restore table, row by row: a Pi agent (session file), a CLI
// agent (native session resumed by id), and every refusal.

// seedPiSession writes a Pi session file into dir and returns its path.
func seedPiSession(t *testing.T, dir, id, cwd string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "2026-09-23_"+id+".jsonl")
	body := `{"type":"session","id":"` + id + `","cwd":"` + cwd + `","timestamp":"2026-09-23T10:00:00Z"}` + "\n" +
		`{"type":"message","message":{"role":"user","content":[{"type":"text","text":"fix the race"}]}}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAgentHistoryPiRestoreAndRefusals(t *testing.T) {
	ts, deps, _, _ := handoffServer(t)
	folder := t.TempDir()
	ag := cliRequest(t, ts, "POST", "/api/agents", map[string]any{"name": "fixer", "path": folder}, 201)
	id := ag["id"].(string)
	// Lazy binding: the agent never named its session, the exit still
	// points at the newest file in its private folder.
	path := seedPiSession(t, session.AgentDir(id), "s-1", folder)
	cliRequest(t, ts, "PATCH", "/api/agents/"+id, map[string]any{"model": "sonnet", "provider": "anthropic", "checklist": "always"}, 200)
	// The agent's own skills (ADR-0196 slice 4) come back with it.
	own := store.AgentSkill{Name: "review", Digest: "d", Dir: filepath.Join(t.TempDir(), "d", "review")}
	if _, err := deps.Store.SetAgentSkills(id, []store.AgentSkill{own}); err != nil {
		t.Fatal(err)
	}
	if res := sendJSON(t, ts, "DELETE", "/api/agents/"+id, nil); res.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", res.StatusCode)
	}

	list := cliRequest(t, ts, "GET", "/api/agent-history", nil, 200)["entries"].([]any)
	if len(list) != 1 {
		t.Fatalf("history = %v", list)
	}
	e := list[0].(map[string]any)
	exit := e["exit"].(map[string]any)
	exitID := exit["id"].(string)
	if e["session"].(map[string]any)["path"] != path || e["folder"] != folder || e["folderExists"] != true || e["workspaceExists"] != true || e["canDeleteFile"] != true {
		t.Fatalf("entry = %v", e)
	}

	res := cliRequest(t, ts, "POST", "/api/agent-history/"+exitID+"/restore", map[string]any{}, 201)
	back := res["agent"].(map[string]any)
	// It comes back as itself: same id, so its private session folder
	// (ADR-0040) is its own again and nothing moves.
	if back["id"] != id || back["name"] != "fixer" || back["sessionPath"] != path || back["model"] != "sonnet" || back["checklist"] != "always" || res["resume"] != false {
		t.Fatalf("restored = %v", res)
	}
	if got, _ := deps.Store.GetAgent(id); len(got.Skills) != 1 || got.Skills[0] != own {
		t.Fatalf("restored skills = %+v", got.Skills)
	}
	if got, _ := deps.Store.GetAgentExit(exitID); got.UndoneAt == nil || got.RestoredAgentID != back["id"] {
		t.Fatalf("exit not linked to the restored agent: %+v", got)
	}
	if list := cliRequest(t, ts, "GET", "/api/agent-history", nil, 200)["entries"].([]any); len(list) != 0 {
		t.Fatalf("a restored agent leaves the history: %v", list)
	}
	cliRequest(t, ts, "POST", "/api/agent-history/"+exitID+"/restore", map[string]any{}, 409)

	// Removed again, then its transcript disappears: out of the history,
	// and a restore says why.
	if res := sendJSON(t, ts, "DELETE", "/api/agents/"+back["id"].(string), nil); res.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	second := cliRequest(t, ts, "GET", "/api/agent-history", nil, 200)["entries"].([]any)
	if len(second) != 1 {
		t.Fatalf("history after the second removal = %v", second)
	}
	secondID := second[0].(map[string]any)["exit"].(map[string]any)["id"].(string)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if list := cliRequest(t, ts, "GET", "/api/agent-history", nil, 200)["entries"].([]any); len(list) != 0 {
		t.Fatalf("no transcript, no history entry: %v", list)
	}
	cliRequest(t, ts, "POST", "/api/agent-history/"+secondID+"/restore", map[string]any{}, 410)
	// The catalog keeps both exits either way.
	if _, err := deps.Store.GetAgentExit(secondID); err != nil {
		t.Fatalf("exit left the catalog: %v", err)
	}
}

func TestAgentHistoryWorkspaceAndFolderGone(t *testing.T) {
	ts, _, _, _ := handoffServer(t)
	proj := t.TempDir()
	ws := cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "proj", "path": proj}, 201)
	wsID := ws["id"].(string)
	ag := cliRequest(t, ts, "POST", "/api/workspaces/"+wsID+"/agents", map[string]any{"name": "a"}, 201)
	path := seedPiSession(t, session.AgentDir(ag["id"].(string)), "s-2", proj)
	cliRequest(t, ts, "PATCH", "/api/agents/"+ag["id"].(string), map[string]any{"sessionPath": path}, 200)
	if res := sendJSON(t, ts, "DELETE", "/api/workspaces/"+wsID, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("workspace delete = %d", res.StatusCode)
	}
	list := cliRequest(t, ts, "GET", "/api/agent-history", nil, 200)["entries"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["workspaceExists"] != false {
		t.Fatalf("history = %v", list)
	}
	exitID := list[0].(map[string]any)["exit"].(map[string]any)["id"].(string)
	res := cliRequest(t, ts, "POST", "/api/agent-history/"+exitID+"/restore", map[string]any{}, 409)
	if res["code"] != "workspace_gone" {
		t.Fatalf("refusal = %v", res)
	}
	// Choosing another place brings it back there, in the folder its
	// conversation ran in.
	other := cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "other", "path": t.TempDir()}, 201)
	res = cliRequest(t, ts, "POST", "/api/agent-history/"+exitID+"/restore", map[string]any{"workspaceId": other["id"]}, 201)
	back := res["agent"].(map[string]any)
	if back["workspaceId"] != other["id"] || back["workPath"] != proj || back["sessionPath"] != path || back["id"] != ag["id"] {
		t.Fatalf("restored elsewhere = %v", back)
	}

	// A folder that is gone is refused, not recreated.
	gone := filepath.Join(t.TempDir(), "gone")
	if err := os.MkdirAll(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	ag2 := cliRequest(t, ts, "POST", "/api/agents", map[string]any{"name": "b", "path": gone}, 201)
	p2 := seedPiSession(t, session.AgentDir(ag2["id"].(string)), "s-3", gone)
	cliRequest(t, ts, "PATCH", "/api/agents/"+ag2["id"].(string), map[string]any{"sessionPath": p2}, 200)
	if res := sendJSON(t, ts, "DELETE", "/api/agents/"+ag2["id"].(string), nil); res.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	var goneID string
	for _, e := range cliRequest(t, ts, "GET", "/api/agent-history", nil, 200)["entries"].([]any) {
		if ex := e.(map[string]any)["exit"].(map[string]any); ex["agentName"] == "b" {
			goneID = ex["id"].(string)
			if e.(map[string]any)["folderExists"] != false {
				t.Fatalf("entry = %v", e)
			}
		}
	}
	if res := cliRequest(t, ts, "POST", "/api/agent-history/"+goneID+"/restore", map[string]any{}, 409); res["code"] != "folder_gone" {
		t.Fatalf("refusal = %v", res)
	}
}

func TestAgentHistoryForget(t *testing.T) {
	ts, deps, _, _ := handoffServer(t)
	folder := t.TempDir()
	ag := cliRequest(t, ts, "POST", "/api/agents", map[string]any{"name": "c", "path": folder}, 201)
	path := seedPiSession(t, session.AgentDir(ag["id"].(string)), "s-4", folder)
	cliRequest(t, ts, "PATCH", "/api/agents/"+ag["id"].(string), map[string]any{"sessionPath": path}, 200)
	if res := sendJSON(t, ts, "DELETE", "/api/agents/"+ag["id"].(string), nil); res.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	exitID := cliRequest(t, ts, "GET", "/api/agent-history", nil, 200)["entries"].([]any)[0].(map[string]any)["exit"].(map[string]any)["id"].(string)

	// A living agent bound to the file keeps it.
	holder := cliRequest(t, ts, "POST", "/api/agents", map[string]any{"name": "holder", "path": folder}, 201)
	cliRequest(t, ts, "PATCH", "/api/agents/"+holder["id"].(string), map[string]any{"sessionPath": path}, 200)
	cliRequest(t, ts, "POST", "/api/agent-history/"+exitID+"/forget", map[string]any{"deleteFile": true}, 409)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("a bound file was deleted: %v", err)
	}
	if res := sendJSON(t, ts, "DELETE", "/api/agents/"+holder["id"].(string), nil); res.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", res.StatusCode)
	}

	cliRequest(t, ts, "POST", "/api/agent-history/"+exitID+"/forget", map[string]any{"deleteFile": true}, 200)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file survived: %v", err)
	}
	ex, err := deps.Store.GetAgentExit(exitID)
	if err != nil || ex.ForgottenAt == nil {
		t.Fatalf("forgotten exit stays in the catalog: %+v %v", ex, err)
	}
}

func TestAgentHistoryCLIRestoreResumesTheSession(t *testing.T) {
	ts, deps, _, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}
	seedHandoffClaude(t, home, proj)
	_, out := fakeCLI(t, ts, home, "claude-code")
	ag := cliRequest(t, ts, "POST", "/api/agents", map[string]any{"cli": "claude-code", "name": "cc", "path": proj, "overrides": map[string]any{"env": map[string]any{"API_TOKEN": "secret"}}}, 201)
	tid := ag["terminalId"].(string)
	t.Cleanup(func() { killTermPane(tmux.ShellSessionName(tid)) })
	if err := deps.Store.SetTerminalLastSession(tid, store.TerminalLastSession{CLI: "claude-code", SessionID: "cc-1", UpdatedAt: "2026-09-01T10:00:05Z"}); err != nil {
		t.Fatal(err)
	}
	if res := sendJSON(t, ts, "DELETE", "/api/agents/"+ag["id"].(string), nil); res.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	list := cliRequest(t, ts, "GET", "/api/agent-history", nil, 200)["entries"].([]any)
	if len(list) != 1 {
		t.Fatalf("history = %v", list)
	}
	e := list[0].(map[string]any)
	if e["canDeleteFile"] != false || e["folder"] != proj {
		t.Fatalf("entry = %v", e)
	}
	exitID := e["exit"].(map[string]any)["id"].(string)
	// Another CLI's files are never PiCode's to delete.
	cliRequest(t, ts, "POST", "/api/agent-history/"+exitID+"/forget", map[string]any{"deleteFile": true}, 400)

	res := cliRequest(t, ts, "POST", "/api/agent-history/"+exitID+"/restore", map[string]any{}, 201)
	newTid, _ := res["terminalId"].(string)
	if newTid == "" || res["resume"] != true || !reflect.DeepEqual(res["envKeys"], []any{"API_TOKEN"}) {
		t.Fatalf("restore = %v", res)
	}
	t.Cleanup(func() { killTermPane(tmux.ShellSessionName(newTid)) })
	cliRequest(t, ts, "POST", "/api/terminals/"+newTid+"/launch/start", map[string]any{"confirm": false, "resume": true}, 200)
	argv := strings.Split(strings.TrimSuffix(string(waitCLIFile(t, out+".args")), "\x00"), "\x00")
	if len(argv) < 2 || !reflect.DeepEqual(argv[:2], []string{"--resume", "cc-1"}) {
		t.Fatalf("claude launched with %q", argv)
	}
}

// The removal toast's Undo goes through restore: an agent that never had a
// conversation comes back as itself, and its purged private folder is made
// again. Without undo, the same exit is refused — nothing to resume.
func TestAgentHistoryUndoWithoutTranscript(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	// Production's layout: a free agent's private folder is under the
	// data dir's work/, which is what a removal may purge.
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat", DataDir: filepath.Join(home, ".picode")}).Handler)
	t.Cleanup(ts.Close)
	ag := cliRequest(t, ts, "POST", "/api/agents", map[string]any{"name": "fresh"}, 201)
	id := ag["id"].(string)
	folder := ag["workPath"].(string)
	res := sendJSON(t, ts, "DELETE", "/api/agents/"+id+"?work=1", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	if dirExists(folder) {
		t.Fatalf("the removal did not purge %s; the test proves nothing", folder)
	}
	var body struct {
		Exit store.AgentExit `json:"exit"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	cliRequest(t, ts, "POST", "/api/agent-history/"+body.Exit.ID+"/restore", map[string]any{}, 410)
	back := cliRequest(t, ts, "POST", "/api/agent-history/"+body.Exit.ID+"/restore", map[string]any{"undo": true}, 201)["agent"].(map[string]any)
	if back["id"] != id || back["name"] != "fresh" {
		t.Fatalf("undo = %v", back)
	}
	if !dirExists(folder) {
		t.Fatalf("purged folder not made again: %s", folder)
	}
	cliRequest(t, ts, "POST", "/api/agent-history/"+body.Exit.ID+"/restore", map[string]any{"undo": true}, 409)
}
