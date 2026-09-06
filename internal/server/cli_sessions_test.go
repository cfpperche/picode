package server

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

// Pi-only guards: the delete/adopt/cleanup actions refuse every other CLI —
// other CLIs' session files are never written, deleted or adopted by PiCode.
func TestCLISessionsPiOnlyGuards(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	for _, row := range []struct{ method, path string }{
		{http.MethodPost, "/api/clis/claude-code/sessions/delete"},
		{http.MethodPost, "/api/clis/claude-code/sessions/adopt"},
		{http.MethodPut, "/api/clis/claude-code/sessions/cleanup"},
		{http.MethodGet, "/api/clis/claude-code/sessions/cleanup"},
	} {
		res := postJSONMethod(t, ts, row.method, row.path, map[string]any{"path": "x", "days": 1})
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s %s = %d, want 400 (pi-only)", row.method, row.path, res.StatusCode)
		}
	}
}
