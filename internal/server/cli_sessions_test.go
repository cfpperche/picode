package server

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"

	"github.com/cfpperche/picode/internal/store"
	"strings"
	"testing"
)

func TestCLISessionsEndpointDecisionTable(t *testing.T) {
	ts, _, home := cleanupServer(t)

	// The session's folder must exist to be registrable as a workspace.
	proj := filepath.Join(t.TempDir(), "proj")
	if err := os.MkdirAll(proj, 0o755); err != nil {
		t.Fatal(err)
	}

	// Claude Code: one real-shaped transcript inside that workspace folder.
	claudeDir := filepath.Join(home, ".claude", "projects", "-tmp-proj")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	claudeBody := strings.Join([]string{
		`{"type":"user","message":{"role":"user","content":[{"text":"fix the race","type":"text"}]},"timestamp":"2026-09-01T10:00:01.000Z","cwd":"` + proj + `","sessionId":"cc-1","version":"2.1.241"}`,
		`{"type":"assistant","message":{"role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"on it"}]},"timestamp":"2026-09-01T10:00:03.000Z","session_id":"cc-1","cwd":"` + proj + `"}`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(claudeDir, "cc-1.jsonl"), []byte(claudeBody), 0o644); err != nil {
		t.Fatal(err)
	}
	// A malformed line must be skipped, not fail the listing.
	if err := os.WriteFile(filepath.Join(claudeDir, "broken.jsonl"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Register the folder as a PiCode workspace so tagging applies.
	cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "proj", "path": proj}, 201)

	// Codex rollout in a folder that is not a workspace: untagged.
	codexDir := filepath.Join(home, ".codex", "sessions", "2026", "09", "01")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	codexBody := `{"timestamp":"2026-09-01T11:00:00.000Z","type":"session_meta","payload":{"id":"cx-1","cwd":"/home/goat/solo","cli_version":"0.65.0"}}`
	if err := os.WriteFile(filepath.Join(codexDir, "rollout-2026-09-01T08-00-00-cx-1.jsonl"), []byte(codexBody), 0o644); err != nil {
		t.Fatal(err)
	}

	// 404 row: unknown catalog id.
	cliRequest(t, ts, "GET", "/api/clis/nope/sessions", nil, 404)

	// Empty machine rows: a CLI with no data answers 200 with an empty list.
	empty := cliRequest(t, ts, "GET", "/api/clis/grok/sessions", nil, 200)
	if n := len(empty["sessions"].([]any)); n != 0 {
		t.Errorf("grok on an empty machine returned %d rows, want 0", n)
	}
	empty = cliRequest(t, ts, "GET", "/api/clis/hermes/sessions", nil, 200)
	if n := len(empty["sessions"].([]any)); n != 0 {
		t.Errorf("hermes on an empty machine returned %d rows, want 0", n)
	}
	empty = cliRequest(t, ts, "GET", "/api/clis/opencode/sessions", nil, 200)
	if n := len(empty["sessions"].([]any)); n != 0 {
		t.Errorf("opencode on an empty machine returned %d rows, want 0", n)
	}

	// Claude listing: parsed, workspace-tagged, broken file skipped.
	got := cliRequest(t, ts, "GET", "/api/clis/claude-code/sessions", nil, 200)
	rows := got["sessions"].([]any)
	if len(rows) != 1 {
		t.Fatalf("claude rows = %d, want 1 (broken file skipped)", len(rows))
	}
	row := rows[0].(map[string]any)
	if row["id"] != "cc-1" || row["cli"] != "claude-code" || row["preview"] != "fix the race" {
		t.Errorf("row wrong: %+v", row)
	}
	if row["workspaceId"] == "" || row["workspace"] != "proj" {
		t.Errorf("workspace tag missing: %+v", row)
	}
	args := row["resumeArgs"].([]any)
	if len(args) != 2 || args[0] != "--resume" || args[1] != "cc-1" {
		t.Errorf("resumeArgs = %v", args)
	}

	// Scoped row: cwd filter only answers that folder.
	scoped := cliRequest(t, ts, "GET", "/api/clis/claude-code/sessions?cwd="+url.QueryEscape(proj), nil, 200)
	if n := len(scoped["sessions"].([]any)); n != 1 {
		t.Errorf("scoped claude rows = %d, want 1", n)
	}
	scoped = cliRequest(t, ts, "GET", "/api/clis/claude-code/sessions?cwd=/elsewhere", nil, 200)
	if n := len(scoped["sessions"].([]any)); n != 0 {
		t.Errorf("other-folder rows = %d, want 0", n)
	}

	// Codex row in a non-workspace folder carries no workspace tag.
	got = cliRequest(t, ts, "GET", "/api/clis/codex/sessions", nil, 200)
	rows = got["sessions"].([]any)
	if len(rows) != 1 {
		t.Fatalf("codex rows = %d, want 1", len(rows))
	}
	row = rows[0].(map[string]any)
	if row["id"] != "cx-1" || row["workspaceId"] != nil {
		t.Errorf("codex row wrong: %+v", row)
	}
	if args := row["resumeArgs"].([]any); len(args) != 2 || args[0] != "resume" {
		t.Errorf("codex resumeArgs = %v", args)
	}

	// totalBytes counts listed files only.
	if tb, ok := got["totalBytes"].(float64); !ok || tb != float64(len(codexBody)) {
		t.Errorf("totalBytes = %v, want %d", got["totalBytes"], len(codexBody))
	}
}

// Pi-only guards: the delete/cleanup actions refuse every other CLI —
// other CLIs' session files are never written or deleted by PiCode.
func TestCLISessionsPiOnlyGuards(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	for _, row := range []struct{ method, path string }{
		{http.MethodPost, "/api/clis/claude-code/sessions/delete"},
		{http.MethodPut, "/api/clis/claude-code/sessions/cleanup"},
		{http.MethodGet, "/api/clis/claude-code/sessions/cleanup"},
		{http.MethodPost, "/api/clis/hermes/sessions/delete"},
		{http.MethodPost, "/api/clis/opencode/sessions/delete"},
	} {
		res := postJSONMethod(t, ts, row.method, row.path, map[string]any{"path": "x", "days": 1})
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s %s = %d, want 400 (pi-only)", row.method, row.path, res.StatusCode)
		}
	}
}

// TestCLISessionsOmpWorkspaceUnionsAgentDirs: a workspace Omp agent writes
// into PiCode's per-agent directory (--session-dir), not omp's cwd bucket.
// The workspace scope adds each of its Omp agents' directories, as pi's
// does (ADR-0040); a plain ?cwd= keeps reading omp's own tree only.
//
//	?workspace=W, W's Omp agent session     → listed, resumes by its path
//	?workspace=W, omp's cwd bucket session  → listed, resumes by its id
//	?workspace=W, another workspace's agent → not listed
//	?cwd=W.path                             → omp's own tree only
func TestCLISessionsOmpWorkspaceUnionsAgentDirs(t *testing.T) {
	ts, deps, dataDir, home := handoffServer(t)
	proj := filepath.Join(t.TempDir(), "proj")
	other := filepath.Join(t.TempDir(), "other")
	for _, d := range []string{proj, other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	ws := cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "proj", "path": proj}, 201)
	ows := cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "other", "path": other}, 201)
	created := cliRequest(t, ts, "POST", "/api/workspaces/"+ws["id"].(string)+"/agents", map[string]any{"name": "delivery", "cli": "omp"}, 201)
	mine, err := deps.Store.GetAgent(created["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	theirs, err := deps.Store.AddAgentWithCLI(ows["id"].(string), "omp", "elsewhere", "")
	if err != nil {
		t.Fatal(err)
	}
	write := func(dir, id, cwd string) string {
		t.Helper()
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, "2026-09-21T20-56-12-695Z_"+id+".jsonl")
		body := `{"type":"session","version":3,"id":"` + id + `","timestamp":"2026-09-21T20:56:12.695Z","cwd":"` + cwd + `"}` + "\n" +
			`{"type":"message","id":"m1","parentId":null,"timestamp":"2026-09-21T20:56:13.000Z","message":{"role":"user","content":[{"type":"text","text":"hi"}]}}` + "\n"
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	agentFile := write(ompAgentSessionDir(dataDir, mine.ID), "omp-agent", proj)
	// The artifacts folder omp keeps beside each session is not a session.
	write(filepath.Join(ompAgentSessionDir(dataDir, mine.ID), "2026-09-21T20-56-12-695Z_omp-agent"), "omp-nested", proj)
	write(ompAgentSessionDir(dataDir, theirs.ID), "omp-theirs", other)
	write(filepath.Join(home, ".omp", "agent", "sessions", strings.ReplaceAll(proj, "/", "-")), "omp-shared", proj)

	rows := func(q string) map[string][]any {
		out := map[string][]any{}
		for _, r := range cliRequest(t, ts, "GET", "/api/clis/omp/sessions"+q, nil, 200)["sessions"].([]any) {
			m := r.(map[string]any)
			out[m["id"].(string)] = m["resumeArgs"].([]any)
		}
		return out
	}
	use := func(q string) map[string]any {
		out := map[string]any{}
		for _, r := range cliRequest(t, ts, "GET", "/api/clis/omp/sessions"+q, nil, 200)["sessions"].([]any) {
			m := r.(map[string]any)
			out[m["id"].(string)] = m["inUseBy"]
		}
		return out
	}
	got := rows("?workspace=" + ws["id"].(string))
	if len(got) != 2 || got["omp-shared"] == nil || got["omp-agent"] == nil {
		t.Fatalf("workspace rows = %v, want omp-shared and omp-agent", got)
	}
	if !reflect.DeepEqual(got["omp-agent"], []any{"--resume", agentFile}) || !reflect.DeepEqual(got["omp-shared"], []any{"--resume", "omp-shared"}) {
		t.Fatalf("resume args = %v", got)
	}
	if got := rows("?cwd=" + proj); len(got) != 1 || got["omp-shared"] == nil {
		t.Fatalf("cwd rows = %v, want only omp-shared", got)
	}
	// The agent's terminal is pinned to its file: that row is in use, so the
	// view opens the agent instead of resuming the file a second time.
	if mine.TerminalID == nil {
		t.Fatal("omp agent has no terminal")
	}
	if u := use("?workspace=" + ws["id"].(string)); u["omp-agent"] != nil || u["omp-shared"] != nil {
		t.Fatalf("in use before any pin = %v", u)
	}
	if err := deps.Store.SetTerminalLastSession(*mine.TerminalID, store.TerminalLastSession{CLI: "omp", SessionID: "omp-agent", Path: agentFile}); err != nil {
		t.Fatal(err)
	}
	u := use("?workspace=" + ws["id"].(string))
	if in, _ := u["omp-agent"].(map[string]any); in == nil || in["agentId"] != mine.ID || in["agentName"] != "delivery" || u["omp-shared"] != nil {
		t.Fatalf("in use after the pin = %v", u)
	}
}
