package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// The mapper needs no muse branch: hook_event_name payloads ride the
// generic path (measured live against R3233.1). This pins that contract —
// SessionStart reads idle the way Claude's does, work brackets working,
// approvals surface as needs-you.
func TestHookMapMuseReport(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not on PATH")
	}
	dir := t.TempDir()
	if _, err := ensureHookScript(dir); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ in, want string }{
		{`{"hook_event_name":"SessionStart","session_id":"s1","cwd":"/w","source":"startup"}`, "idle"},
		{`{"hook_event_name":"UserPromptSubmit","session_id":"s1","turn_id":"t1","prompt":"hi"}`, "working"},
		{`{"hook_event_name":"PreToolUse","session_id":"s1","turn_id":"t1","tool_name":"read","tool_input":{},"tool_use_id":"c1"}`, "working"},
		{`{"hook_event_name":"PostToolUse","session_id":"s1","turn_id":"t1","tool_name":"read","tool_input":{},"tool_response":"ok","tool_use_id":"c1"}`, "working"},
		{`{"hook_event_name":"PermissionRequest","session_id":"s1","turn_id":"t1","tool_name":"write"}`, "needs-you"},
		{`{"hook_event_name":"Stop","session_id":"s1","turn_id":"t1","stop_hook_active":false}`, "idle"},
	}
	for _, c := range cases {
		// Hermetic against ambient terminal vars (see TestHookMapAgyReport).
		cmd := exec.Command("python3", filepath.Join(dir, "picode-hook-map.py"))
		cmd.Stdin = strings.NewReader(c.in)
		cmd.Env = append(os.Environ(), "PICODE_HOOK_CLI=muse", "PICODE_HOOK_REPORT=1", "PICODE_TUI_PID=", "PICODE_TUI_RUN_ID=")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("map %s: %v", c.in, err)
		}
		var got map[string]any
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("report is not JSON for %s: %q", c.in, out)
		}
		if got["state"] != c.want || got["cli"] != "muse" || got["sessionId"] != "s1" {
			t.Fatalf("report for %s = %s, want state %s", c.in, out, c.want)
		}
	}
}

// Decision table: a session report with no native runtime behind the
// terminal (wrapper-less observers: muse hooks, agy title) reports
// plainly instead of 409ing; a terminal WITH a live runtime keeps the
// strict identity fence.
func TestSessionReportWithoutRuntimeFallsBack(t *testing.T) {
	data := t.TempDir()
	s, err := store.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	w, _ := s.AddWorkspace("fixture", data)
	deps := Deps{Store: s, TermRuntimes: NewTermRuntimes(), TermStates: NewTermStates()}
	post := func(termID string, body map[string]any) int {
		t.Helper()
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/terminals/"+termID+"/state", bytes.NewReader(raw))
		req.SetPathValue("id", termID)
		rec := httptest.NewRecorder()
		handleSetTerminalState(deps)(rec, req)
		return rec.Code
	}
	seq := time.Now().UnixNano()
	// Muse Stop-shaped report, no runtime: accepted, idle set.
	plain, _ := s.CreateTerminalIn(w.ID, "Muse", data)
	if code := post(plain.ID, map[string]any{"state": "idle", "cli": "muse", "sessionId": "sx", "sessionSeq": seq}); code != http.StatusOK {
		t.Fatalf("muse fallback = %d, want 200", code)
	}
	if st, _ := deps.TermStates.Get(plain.ID); st.State != TermIdle {
		t.Fatalf("muse state = %q, want idle", st.State)
	}
	// Agy working-shaped report, no runtime: accepted, working set.
	plain2, _ := s.CreateTerminalIn(w.ID, "Agy", data)
	if code := post(plain2.ID, map[string]any{"state": "working", "cli": "agy", "sessionId": "c9", "sessionSeq": seq}); code != http.StatusOK {
		t.Fatalf("agy fallback = %d, want 200", code)
	}
	if st, _ := deps.TermStates.Get(plain2.ID); st.State != TermWorking {
		t.Fatalf("agy state = %q, want working", st.State)
	}
	// Live runtime + foreign session: still a conflict, state untouched.
	native, _ := s.CreateTerminalIn(w.ID, "Grok", data)
	deps.TermRuntimes.Start(native.ID, TermRuntime{CLI: "grok", RunID: "run-1", PID: os.Getpid()})
	if code := post(native.ID, map[string]any{"state": "working", "cli": "grok", "runId": "run-1", "sessionId": "other", "sessionSeq": int64(1)}); code != http.StatusConflict {
		t.Fatalf("stale native report = %d, want 409", code)
	}
	if _, ok := deps.TermStates.Get(native.ID); ok {
		t.Fatal("conflicting report set state")
	}
}
