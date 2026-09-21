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

// A bound terminal borrows its agent's name at bind time; renaming the
// agent keeps that loan current — the tab strip labels a t:<id> tab with
// term.name, so a rename that skipped the terminal left the strip stale.
func TestUpdateAgentRenamePropagatesToBoundTerminal(t *testing.T) {
	for _, tc := range []struct {
		name        string
		bind        bool
		renameTo    string
		wantRenamed bool
	}{
		{"bound terminal follows the rename", true, "renamed", true},
		{"unbound terminal is untouched", false, "renamed", false},
		{"same-name patch is a no-op", true, "borrowed", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := openTest(t)
			a, err := s.AddAgent(FreeWorkspaceID, "borrowed", t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			var bound string
			if tc.bind {
				term, err := s.CreateTerminalIn(a.WorkspaceID, a.Name, *a.WorkPath)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := s.UpdateAgent(a.ID, AgentPatch{TerminalID: &term.ID}); err != nil {
					t.Fatal(err)
				}
				bound = term.ID
			}
			var events []string
			s.OnEvent = func(ev Event) { events = append(events, ev.Type) }

			if _, err := s.UpdateAgent(a.ID, AgentPatch{Name: &tc.renameTo}); err != nil {
				t.Fatal(err)
			}

			sawTerminalUpdated := false
			for i, kind := range events {
				if kind != "terminal.updated" {
					continue
				}
				sawTerminalUpdated = true
				// The terminal event lands inside the same mutation, before
				// the agent row announces itself.
				if i+1 >= len(events) || events[i+1] != "agent.updated" {
					t.Fatalf("terminal.updated not followed by agent.updated: %v", events)
				}
			}
			if sawTerminalUpdated != tc.wantRenamed {
				t.Fatalf("terminal.updated = %v, want %v (events %v)", sawTerminalUpdated, tc.wantRenamed, events)
			}
			if !tc.wantRenamed {
				return
			}
			term, err := s.GetTerminal(bound)
			if err != nil {
				t.Fatal(err)
			}
			if term.Name != tc.renameTo {
				t.Fatalf("terminal name = %q, want %q", term.Name, tc.renameTo)
			}
		})
	}
}
