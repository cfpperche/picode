package store

import (
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

// ADR-0184 slice 2 (migration 067): each row of the plan's table.
func TestMigrateCLITerminalsOntoAgents(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()
	w, err := s.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	launch := func(ws, name, cwd, cli string) Terminal {
		t.Helper()
		tm, err := s.CreateTerminalIn(ws, name, cwd)
		if err != nil {
			t.Fatal(err)
		}
		if cli != "" {
			if err := s.SetTerminalLaunch(tm.ID, cli, clilaunch.Overrides{}); err != nil {
				t.Fatal(err)
			}
		}
		return tm
	}
	inWS := launch(w.ID, "Omp mirror", proj, "omp")
	sub := t.TempDir()
	inSub := launch(w.ID, "Codex sub", sub, "codex")
	free := launch(FreeWorkspaceID, "Claude free", sub, "claude-code")
	signin := launch(FreeWorkspaceID, "Claude Code sign-in", sub, "claude-code")
	shell := launch(w.ID, "zsh", proj, "")
	bound := launch(w.ID, "Bound", proj, "grok")
	owner, err := s.AddAgentWithCLI(w.ID, "grok", "Bound", "")
	if err != nil {
		t.Fatal(err)
	}
	btid := bound.ID
	if _, err := s.UpdateAgent(owner.ID, AgentPatch{TerminalID: &btid}); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"browser.policy.term:" + inWS.ID:     `{"tier":"act"}`,
		"computer.policy.term:" + free.ID:    `{"enabled":true}`,
		"computer.policy.term:" + shell.ID:   `{"enabled":true}`,
		"computer.policy.term:gone-terminal": `{"enabled":true}`,
		"browser.policy.term:" + bound.ID:    `{"tier":"full"}`,
		"browser.policy." + owner.ID:         `{"tier":"read"}`,
	} {
		if err := s.SetSetting(k, v); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 67`); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(); err != nil {
		t.Fatalf("migrate 067: %v", err)
	}

	check := func(tm Terminal, cli, ws string, work *string) Agent {
		t.Helper()
		a, err := s.AgentByTerminal(tm.ID)
		if err != nil {
			t.Fatalf("%s not migrated: %v", tm.Name, err)
		}
		if a.CLI != cli || a.WorkspaceID != ws || a.Name != tm.Name {
			t.Fatalf("%s → %+v", tm.Name, a)
		}
		if (work == nil) != (a.WorkPath == nil) || (work != nil && *work != *a.WorkPath) {
			t.Fatalf("%s work path = %v, want %v", tm.Name, a.WorkPath, work)
		}
		return a
	}
	a := check(inWS, "omp", w.ID, nil)
	check(inSub, "codex", w.ID, &sub)
	f := check(free, "claude-code", FreeWorkspaceID, &sub)
	for _, tm := range []Terminal{signin, shell} {
		if _, err := s.AgentByTerminal(tm.ID); err == nil {
			t.Fatalf("%s must stay a terminal", tm.Name)
		}
	}
	if got, _ := s.AgentByTerminal(bound.ID); got.ID != owner.ID {
		t.Fatalf("bound terminal re-migrated: %+v", got)
	}

	want := map[string]string{
		"browser.policy." + a.ID:     `{"tier":"act"}`,
		"computer.policy." + f.ID:    `{"enabled":true}`,
		"browser.policy." + owner.ID: `{"tier":"read"}`, // the agent's own grant wins
	}
	for k, v := range want {
		if got, ok, _ := s.GetSetting(k); !ok || got != v {
			t.Fatalf("%s = %q (%v), want %q", k, got, ok, v)
		}
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM settings WHERE key LIKE '%.policy.term:%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("%d terminal grants survived", n)
	}

	// Idempotent: a second run changes nothing.
	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 67`); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(); err != nil {
		t.Fatalf("migrate 067 again: %v", err)
	}
	all, err := s.ListAllAgents()
	if err != nil || len(all) != 4 {
		t.Fatalf("agents after rerun = %d %v", len(all), err)
	}
}

// ADR-0184 slice 3: a sign-in terminal keeps its kind; others have none.
func TestSigninTerminalKind(t *testing.T) {
	s := openTest(t)
	in, err := s.CreateSigninTerminal("Codex sign-in", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	plain, err := s.CreateTerminal("zsh", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetTerminal(in.ID)
	if err != nil || got.Kind != TerminalKindSignin || got.WorkspaceID != FreeWorkspaceID {
		t.Fatalf("sign-in = %+v %v", got, err)
	}
	if got, _ := s.GetTerminal(plain.ID); got.Kind != "" {
		t.Fatalf("shell kind = %q", got.Kind)
	}
}

// The adversarial rows of migration 067: Pi's session, a terminal whose
// workspace is gone, a person's "Sign-In" terminal, and deliveries a
// terminal owned.
func TestMigrateCLITerminalsEdgeRows(t *testing.T) {
	s := openTest(t)
	w, err := s.AddWorkspace("App", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mk := func(ws, name, cli string, ov clilaunch.Overrides) Terminal {
		t.Helper()
		tm, err := s.CreateTerminalIn(ws, name, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if err := s.SetTerminalLaunch(tm.ID, cli, ov); err != nil {
			t.Fatal(err)
		}
		return tm
	}
	args := []string{"--session", "/s/handoff.jsonl"}
	piArgs := mk(w.ID, "pi handoff", "pi", clilaunch.Overrides{Args: &args})
	piPinned := mk(w.ID, "pi pinned", "pi", clilaunch.Overrides{})
	if _, err := s.db.Exec(`UPDATE terminal_launches SET last_session = ? WHERE terminal_id = ?`, `{"cli":"pi","path":"/s/pinned.jsonl"}`, piPinned.ID); err != nil {
		t.Fatal(err)
	}
	person := mk(w.ID, "Claude Sign-In", "claude-code", clilaunch.Overrides{})
	orphan := mk(w.ID, "orphan", "codex", clilaunch.Overrides{})
	if _, err := s.db.Exec(`UPDATE terminals SET workspace_id = 'gone' WHERE id = ?`, orphan.ID); err != nil {
		t.Fatal(err)
	}
	owner := mk(w.ID, "deliverer", "codex", clilaunch.Overrides{})
	key := "term:" + owner.ID
	if _, err := s.db.Exec(`INSERT INTO delivery_intents(id,repo,principal,body) VALUES('d1','r',?,?)`, key, `{"id":"d1","principal":"`+key+`"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO delivery_requests(repo,principal,request_id,payload,result) VALUES('r',?,'q1','{}','{}')`, key); err != nil {
		t.Fatal(err)
	}

	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 67`); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(); err != nil {
		t.Fatalf("migrate 067 with an orphan terminal: %v", err)
	}

	a, err := s.AgentByTerminal(piArgs.ID)
	if err != nil || a.SessionPath == nil || *a.SessionPath != "/s/handoff.jsonl" {
		t.Fatalf("pi handoff agent = %+v %v", a, err)
	}
	if l, _ := s.TerminalLaunch(piArgs.ID); l == nil || l.Overrides.Args != nil {
		t.Fatalf("reserved --session kept on the launch: %+v", l)
	}
	if p, err := s.AgentByTerminal(piPinned.ID); err != nil || p.SessionPath == nil || *p.SessionPath != "/s/pinned.jsonl" {
		t.Fatalf("pi pinned agent = %+v %v", p, err)
	}
	if _, err := s.AgentByTerminal(person.ID); err != nil {
		t.Fatalf("a person's Sign-In terminal was taken for a sign-in: %v", err)
	}
	if _, err := s.AgentByTerminal(orphan.ID); err == nil {
		t.Fatal("an orphan terminal became an agent")
	}
	d, err := s.AgentByTerminal(owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	var principal, body string
	if err := s.db.QueryRow(`SELECT principal, body FROM delivery_intents WHERE id='d1'`).Scan(&principal, &body); err != nil || principal != d.ID || !strings.Contains(body, `"principal":"`+d.ID+`"`) {
		t.Fatalf("delivery = %q %q %v", principal, body, err)
	}
	var n int
	if err := s.db.QueryRow(`SELECT count(*) FROM delivery_requests WHERE principal = ?`, d.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("delivery request moved = %d %v", n, err)
	}
}

// Migration 068 marks the credential flow's own "<CLI> sign-in" terminals,
// and nothing a person named.
func TestMigrateSigninKind(t *testing.T) {
	s := openTest(t)
	mk := func(name string) Terminal {
		t.Helper()
		tm, err := s.CreateTerminal(name, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if err := s.SetTerminalLaunch(tm.ID, "codex", clilaunch.Overrides{}); err != nil {
			t.Fatal(err)
		}
		return tm
	}
	flow, person := mk("Codex sign-in"), mk("Codex Sign-In")
	if _, err := s.db.Exec(`ALTER TABLE terminals DROP COLUMN kind`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`DELETE FROM schema_migrations WHERE version = 68`); err != nil {
		t.Fatal(err)
	}
	if err := s.migrate(); err != nil {
		t.Fatalf("migrate 068: %v", err)
	}
	if got, _ := s.GetTerminal(flow.ID); got.Kind != TerminalKindSignin {
		t.Fatalf("flow sign-in kind = %q", got.Kind)
	}
	if got, _ := s.GetTerminal(person.ID); got.Kind != "" {
		t.Fatalf("a person's terminal became a sign-in")
	}
}
