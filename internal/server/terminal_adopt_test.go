package server

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

// ADR-0184 slice 4: Make agent, one case per row of the plan's table.
func TestAdoptTerminalMakesAnAgent(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	rts := NewTermRuntimes()
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, TermRuntimes: rts}).Handler)
	t.Cleanup(ts.Close)
	proj := t.TempDir()
	ws, err := st.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	running := func(tm store.Terminal, cli string) {
		rts.Start(tm.ID, TermRuntime{CLI: cli, Source: "wrapper", RunID: "r-" + tm.ID, StartedAt: time.Now()})
	}

	// A shell with nothing running: nothing to adopt.
	idle, _ := st.CreateTerminalIn(ws.ID, "idle", proj)
	cliRequest(t, ts, "POST", "/api/terminals/"+idle.ID+"/adopt", map[string]any{}, 409)

	// A workspace shell running Claude Code: an agent in that workspace,
	// bound to the terminal, launch set for its next start.
	shell, _ := st.CreateTerminalIn(ws.ID, "claude here", proj)
	running(shell, "claude-code")
	a := cliRequest(t, ts, "POST", "/api/terminals/"+shell.ID+"/adopt", map[string]any{}, 201)
	if a["cli"] != "claude-code" || a["workspaceId"] != ws.ID || a["terminalId"] != shell.ID || a["name"] != "claude here" || a["workPath"] != nil {
		t.Fatalf("adopted = %v", a)
	}
	if l, err := st.TerminalLaunch(shell.ID); err != nil || l == nil || l.CLI != "claude-code" {
		t.Fatalf("launch = %+v %v", l, err)
	}
	// Twice is refused: it is an agent now.
	cliRequest(t, ts, "POST", "/api/terminals/"+shell.ID+"/adopt", map[string]any{}, 409)

	// A free shell: a free agent working where the shell is.
	dir := t.TempDir()
	free, _ := st.CreateTerminal("codex here", dir)
	running(free, "codex")
	f := cliRequest(t, ts, "POST", "/api/terminals/"+free.ID+"/adopt", map[string]any{}, 201)
	if f["workspaceId"] != store.FreeWorkspaceID || f["workPath"] != dir {
		t.Fatalf("free adopted = %v", f)
	}

	// A sign-in terminal is never adopted.
	in, _ := st.CreateSigninTerminal("Grok sign-in", dir)
	running(in, "grok")
	cliRequest(t, ts, "POST", "/api/terminals/"+in.ID+"/adopt", map[string]any{}, 409)

	// Unknown terminal.
	cliRequest(t, ts, "POST", "/api/terminals/missing/adopt", map[string]any{}, 404)
}
