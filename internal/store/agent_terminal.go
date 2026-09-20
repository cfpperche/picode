package store

import "errors"

// EnsureAgentTerminal atomically publishes the interactive runtime and its
// owner. Callers serialize lifecycle operations by agent id.
func (s *Store) EnsureAgentTerminal(id, cwd string) (Agent, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Agent{}, err
	}
	defer s.rollback(tx)
	var a Agent
	if err = scanAgent(tx.QueryRow(`SELECT `+agentCols+` FROM agents WHERE id=?`, id), &a); err != nil {
		return a, err
	}
	if !a.IsPi() {
		return a, errors.New("interactive binding is only for Pi")
	}
	if a.TerminalID != nil {
		return a, nil
	}
	t := Terminal{ID: newID(a.Name, "term"), Name: a.Name, Cwd: cwd, WorkspaceID: a.WorkspaceID, CreatedAt: nowUTC()}
	if _, err = tx.Exec(`INSERT INTO terminals(id,name,cwd,workspace_id,created_at) VALUES(?,?,?,?,?)`, t.ID, t.Name, t.Cwd, t.WorkspaceID, t.CreatedAt); err != nil {
		return a, err
	}
	if _, err = tx.Exec(`INSERT INTO terminal_launches(terminal_id,cli,overrides,updated_at) VALUES(?,?,?,?)`, t.ID, CLIPi, "{}", nowUTC()); err != nil {
		return a, err
	}
	if _, err = tx.Exec(`UPDATE agents SET terminal_id=? WHERE id=?`, t.ID, id); err != nil {
		return a, err
	}
	a.TerminalID = &t.ID
	for _, ev := range []struct {
		kind string
		data any
	}{{"terminal.created", t}, {"terminal.launch", idData(t.ID)}, {"agent.updated", a}} {
		if err = s.AppendEventTx(tx, ev.kind, &a.ID, &a.WorkspaceID, ev.data); err != nil {
			return a, err
		}
	}
	return a, s.commit(tx)
}
