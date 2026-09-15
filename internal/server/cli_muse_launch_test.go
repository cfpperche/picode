package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// Decision table, Fatia 3a: Muse Code's launch is editable but its build
// offers no hook surface, so the plan carries a summary and nothing to
// install or inject; every other integrable CLI carries a mechanism
// (arg branches, files, or environment), Antigravity none.
func TestIntegrationMechanismTable(t *testing.T) {
	for _, c := range clilaunch.Catalog() {
		got := hasIntegrationMechanism(c.ID)
		want := c.ID != "muse" && c.ID != "agy"
		if got != want {
			t.Errorf("%s hasIntegrationMechanism = %v, want %v", c.ID, got, want)
		}
	}
	if p := cliIntegrationPlan("muse", t.TempDir(), "hook"); len(p.Branches) != 0 || len(p.Files) != 0 || len(p.Environment) != 0 || p.Summary == "" {
		t.Errorf("muse plan = %+v, want a summary and nothing to install", p)
	}
}

// Decision table: the Activity toggle saves only where a mechanism exists.
// Muse and Antigravity refuse with their own reason (no hook surface);
// a CLI with hooks saves.
func TestIntegrationToggleGuard(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	// The catalog row carries the toggle contract, so the web never
	// derives it from plan shape (which is empty until enabled).
	rows := cliRequest(t, ts, "GET", "/api/clis", nil, 200)["clis"].([]any)
	seen := map[string]bool{}
	for _, raw := range rows {
		row := raw.(map[string]any)
		id, _ := row["id"].(string)
		if id != "muse" && id != "agy" && id != "pi" {
			continue
		}
		seen[id] = true
		if got, want := row["hasIntegrationMechanism"], id == "pi"; got != want {
			t.Errorf("%s hasIntegrationMechanism = %v, want %v", id, got, want)
		}
	}
	for _, id := range []string{"muse", "agy", "pi"} {
		if !seen[id] {
			t.Errorf("catalog missing %s", id)
		}
	}
	body := func(on bool) map[string]any {
		return map[string]any{"executable": "", "args": []any{}, "env": map[string]any{}, "integration": on}
	}
	bodyErr := func(res map[string]any) string {
		b, _ := res["body"].(map[string]any)
		e, _ := b["error"].(string)
		return e
	}
	res := cliRequestFull(t, ts, "PUT", "/api/clis/muse", body(true))
	if res["status"] != "400" || !strings.Contains(bodyErr(res), "not available for Muse Code") {
		t.Errorf("muse integration on = %v, want 400 naming Muse Code", res)
	}
	cliRequest(t, ts, "PUT", "/api/clis/muse", body(false), 200)
	res = cliRequestFull(t, ts, "PUT", "/api/clis/agy", body(true))
	if res["status"] != "400" || !strings.Contains(bodyErr(res), "not available for Antigravity in this build") {
		t.Errorf("agy integration on = %v, want the mechanism-guard 400", res)
	}
}

// The debt this covers: a launch that fails inside tmux — prepare fine,
// the session creation dead — must persist the attempt the row renders as
// "Last launch failed · review settings". The dead socket fails
// NewSessionEnvSize without creating anything, so no live session leaks.
func TestLaunchFailureInsideTmuxRecordsAttempt(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	// A socket path under a regular file can never bind, so the session
	// creation fails after a good prepare — without starting anything.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	deps := Deps{Store: st, Tmux: tmux.NewWithSocket(filepath.Join(blocker, "sock")), DataDir: dir}
	exe := "/bin/true"
	if _, err := os.Stat(exe); err != nil {
		t.Skip("no /bin/true on this platform")
	}
	term, err := st.CreateTerminal("fail-name", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	v := &store.TerminalLaunch{TerminalID: term.ID, CLI: "muse", Overrides: clilaunch.Overrides{Executable: &exe}}
	if err := st.SetTerminalLaunch(v.TerminalID, v.CLI, v.Overrides); err != nil {
		t.Fatal(err)
	}
	cwd := t.TempDir()
	r := httptest.NewRequest("POST", "/", nil)
	if err := launchCLITerminal(deps, r, "fail-name", cwd, v); err == nil {
		t.Fatal("launch against a dead tmux socket succeeded")
	}
	rec, err := st.TerminalLaunch(v.TerminalID)
	if err != nil || rec.Attempt == nil || !strings.Contains(rec.Attempt.Error, "Launch failed") {
		t.Fatalf("attempt = %+v, err = %v, want the persisted diagnostic", rec, err)
	}
	if strings.Contains(rec.Attempt.Error, "dead-socket") {
		t.Errorf("attempt leaks transport detail: %q", rec.Attempt.Error)
	}
}
