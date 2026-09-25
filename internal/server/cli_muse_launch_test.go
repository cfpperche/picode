package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/pimission"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// Decision table: every integrable CLI carries a mechanism (arg branches,
// files, or environment). Muse Code and Antigravity launch through the PATH
// wrapper like the rest since the wrapper slice; their reporting differs
// (agy title reporter, muse presence lease only). Omp reports through the
// pi-shaped terminal-state extension since the omp-adapter slice.
func TestIntegrationMechanismTable(t *testing.T) {
	for _, c := range clilaunch.Catalog() {
		want := c.Integrable()
		if got := hasIntegrationMechanism(c.ID); got != want {
			t.Errorf("%s hasIntegrationMechanism = %v, want %v", c.ID, got, want)
		}
	}
	if p := cliIntegrationPlan("muse", t.TempDir(), "hook"); len(p.Files) != 1 || p.Summary == "" {
		t.Errorf("muse plan = %+v, want a summary and the wrapper file", p)
	}
}

func TestPiTerminalLaunchIncludesOneNativeMissionExtension(t *testing.T) {
	dir := t.TempDir()
	p := cliIntegrationPlan("pi", dir, "hook")
	if len(p.Branches) != 1 {
		t.Fatalf("Pi integration branches = %+v", p.Branches)
	}
	args := p.Branches[0].Args
	count := 0
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && strings.HasSuffix(args[i+1], "pi-mission.ts") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("Pi CLI extension args = %v", args)
	}
	path, err := pimission.Ensure(dir)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(contents), "@earendil-works/pi-coding-agent") || strings.Contains(string(contents), "typebox") {
		t.Fatal("embedded extension depends on unresolved package imports")
	}
}

// Decision table: the Activity toggle saves where a mechanism exists.
// Muse and Antigravity launch through the wrapper; a CLI with hooks saves.
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
		if got := row["hasIntegrationMechanism"]; got != true {
			t.Errorf("%s hasIntegrationMechanism = %v, want true", id, got)
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
	cliRequest(t, ts, "PUT", "/api/clis/muse", body(true), 200)
	row := catalogCLI(t, ts, "muse")
	if row["integrationApplied"] != true {
		t.Fatalf("muse applied = %v after toggle on", row["integrationApplied"])
	}
	cliRequest(t, ts, "PUT", "/api/clis/muse", body(false), 200)
	row = catalogCLI(t, ts, "muse")
	if row["integrationApplied"] != false {
		t.Fatalf("muse applied = %v after toggle off", row["integrationApplied"])
	}
	cliRequest(t, ts, "PUT", "/api/clis/agy", body(true), 200)
	// On means installed: the row reports applied and the user's settings
	// carry our title block; off removes the block again.
	row = catalogCLI(t, ts, "agy")
	if row["integrationApplied"] != true {
		t.Fatalf("agy applied = %v after toggle on", row["integrationApplied"])
	}
	cliRequest(t, ts, "PUT", "/api/clis/agy", body(false), 200)
	row = catalogCLI(t, ts, "agy")
	if row["integrationApplied"] != false {
		t.Fatalf("agy applied = %v after toggle off", row["integrationApplied"])
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
	term, err := st.CreateTerminalIn(store.FreeWorkspaceID, "fail-name", t.TempDir())
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
