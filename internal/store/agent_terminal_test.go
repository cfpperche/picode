package store

import "testing"

func TestEnsureAgentTerminalRollsBackWithEvents(t *testing.T) {
	s := openTest(t)
	a, err := s.AddAgent(FreeWorkspaceID, "atomic", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Fail the last event, after the terminal, launch and binding were staged.
	_, err = s.db.Exec(`CREATE TRIGGER reject_binding BEFORE INSERT ON events WHEN NEW.type='agent.updated' BEGIN SELECT RAISE(ABORT, 'fixture'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.EnsureAgentTerminal(a.ID, t.TempDir()); err == nil {
		t.Fatal("expected transaction failure")
	}
	got, _ := s.GetAgent(a.ID)
	terms, _ := s.ListTerminals()
	if got.TerminalID != nil || len(terms) != 0 {
		t.Fatal("partial binding escaped rollback")
	}
	var launches int
	if err = s.db.QueryRow(`SELECT count(*) FROM terminal_launches`).Scan(&launches); err != nil || launches != 0 {
		t.Fatalf("orphan launches: %d %v", launches, err)
	}
}
