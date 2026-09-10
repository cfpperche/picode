package store

import (
	"crypto/rand"
	"database/sql"
	"errors"
)

// PeerParticipant is owner consent scoped to the workspace and CLI selected by
// the owner. Moving an owner never transfers this grant to the new workspace.
type PeerParticipant struct {
	Kind              string `json:"kind"`
	OwnerID           string `json:"ownerId"`
	WorkspaceID       string `json:"workspaceId"`
	CLI               string `json:"cli"`
	Enabled           bool   `json:"enabled"`
	Revision          int64  `json:"revision"`
	Phase             string `json:"phase"`
	Problem           string `json:"problem"`
	AppliedSession    string `json:"appliedSession"`
	AppliedConnection string `json:"appliedConnection"`
}
type PeerSelection struct {
	Kind     string `json:"kind"`
	OwnerID  string `json:"ownerId"`
	Enabled  bool   `json:"enabled"`
	Revision int64  `json:"revision"`
}

const participantCols = `CASE WHEN agent_id IS NOT NULL THEN 'agent' ELSE 'terminal' END,COALESCE(agent_id,terminal_id),workspace_id,cli,enabled,revision,phase,problem,applied_session,applied_connection`

func scanParticipant(row peerRow) (p PeerParticipant, err error) {
	err = row.Scan(&p.Kind, &p.OwnerID, &p.WorkspaceID, &p.CLI, &p.Enabled, &p.Revision, &p.Phase, &p.Problem, &p.AppliedSession, &p.AppliedConnection)
	return
}
func (s *Store) PeerParticipant(kind, id string) (PeerParticipant, error) {
	return scanParticipant(s.db.QueryRow(`SELECT `+participantCols+` FROM peer_participants WHERE owner_key=?`, kind+":"+id))
}
func (s *Store) ListPeerParticipants() ([]PeerParticipant, error) {
	rows, err := s.db.Query(`SELECT ` + participantCols + ` FROM peer_participants ORDER BY owner_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PeerParticipant{}
	for rows.Next() {
		p, e := scanParticipant(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetPeerParticipants applies only explicitly supplied rows atomically. Revision
// checks prevent stale tabs from overwriting another owner's recent choice.
func (s *Store) SetPeerParticipants(workspace string, selections []PeerSelection) error {
	if workspace == "" || len(selections) == 0 || len(selections) > 100 {
		return ErrPeerInput
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	seen := map[string]bool{}
	for _, v := range selections {
		key := v.Kind + ":" + v.OwnerID
		if seen[key] || v.Revision < 0 {
			return ErrPeerInput
		}
		seen[key] = true
		o, e := peerOwner(tx, v.Kind, v.OwnerID)
		if e != nil {
			return e
		}
		if o.WorkspaceID != workspace || o.CLI == "" {
			return ErrPeerDenied
		}
		var revision int64
		e = tx.QueryRow(`SELECT revision FROM peer_participants WHERE owner_key=?`, key).Scan(&revision)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		if revision != v.Revision {
			return ErrPeerConflict
		}
	}
	for _, v := range selections {
		o, e := peerOwner(tx, v.Kind, v.OwnerID)
		if e != nil {
			return e
		}
		var agent, terminal any
		if v.Kind == "agent" {
			agent = v.OwnerID
		} else {
			terminal = v.OwnerID
		}
		phase := "disabled"
		if v.Enabled {
			phase = "preparing"
		}
		_, err = tx.Exec(`INSERT INTO peer_participants(owner_key,agent_id,terminal_id,workspace_id,cli,enabled,revision,phase) VALUES(?,?,?,?,?,?,?,?)
   ON CONFLICT(owner_key) DO UPDATE SET workspace_id=excluded.workspace_id,cli=excluded.cli,enabled=excluded.enabled,revision=excluded.revision,phase=excluded.phase,problem='',applied_session='',applied_connection=''`, v.Kind+":"+v.OwnerID, agent, terminal, workspace, o.CLI, v.Enabled, v.Revision+1, phase)
		if err != nil {
			return err
		}
		if !v.Enabled {
			if _, err = tx.Exec(`UPDATE peer_connections SET revoked_at=? WHERE revoked_at IS NULL AND (agent_id=? OR terminal_id=?)`, nowUTC(), agent, terminal); err != nil {
				return err
			}
			if err = s.AppendEventTx(tx, "peer.connection", nil, nil, map[string]any{"ownerId": v.OwnerID}); err != nil {
				return err
			}
		}
		if err = s.AppendEventTx(tx, "peer.participant", nil, nil, map[string]any{"kind": v.Kind, "ownerId": v.OwnerID}); err != nil {
			return err
		}
	}
	return s.commit(tx)
}

// SetPeerPreparation is a compare-and-set against consent AND current identity.
// Stale workers cannot update a participant after revoke, move, or /new.
func (s *Store) SetPeerPreparation(p PeerParticipant, session, phase, problem, connection string) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer s.rollback(tx)
	o, err := peerOwner(tx, p.Kind, p.OwnerID)
	if err != nil {
		return false, err
	}
	if o.WorkspaceID != p.WorkspaceID || o.CLI != p.CLI || o.SessionKey != session {
		return false, ErrPeerDenied
	}
	res, err := tx.Exec(`UPDATE peer_participants SET phase=?,problem=?,applied_session=?,applied_connection=? WHERE owner_key=? AND revision=? AND enabled=1 AND (phase!=? OR problem!=? OR applied_session!=? OR applied_connection!=?)`, phase, problem, session, connection, p.Kind+":"+p.OwnerID, p.Revision, phase, problem, session, connection)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		if err = s.AppendEventTx(tx, "peer.participant", nil, nil, map[string]any{"kind": p.Kind, "ownerId": p.OwnerID}); err != nil {
			return false, err
		}
	}
	return n > 0, s.commit(tx)
}

type PeerCheck struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	SenderID    string `json:"senderId"`
	RecipientID string `json:"recipientId"`
	Phase       string `json:"phase"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func (s *Store) CreatePeerCheck(workspace, from, to string) (PeerCheck, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return PeerCheck{}, err
	}
	defer s.rollback(tx)
	for _, id := range []string{from, to} {
		p, e := scanPeer(tx.QueryRow(`SELECT `+peerCols+` FROM peer_connections WHERE id=?`, id))
		if e != nil || p.WorkspaceID != workspace || !peerCurrent(tx, p) {
			return PeerCheck{}, ErrPeerDenied
		}
	}
	if from == to {
		return PeerCheck{}, ErrPeerInput
	}
	var active int
	if err = tx.QueryRow(`SELECT count(*) FROM peer_checks WHERE workspace_id=? AND phase IN ('pending','attempted','running')`, workspace).Scan(&active); err != nil {
		return PeerCheck{}, err
	}
	if active != 0 {
		return PeerCheck{}, ErrPeerConflict
	}
	c := PeerCheck{ID: "check_" + rand.Text(), WorkspaceID: workspace, SenderID: from, RecipientID: to, Phase: "pending", CreatedAt: nowUTC(), UpdatedAt: nowUTC()}
	_, err = tx.Exec(`INSERT INTO peer_checks(id,workspace_id,sender_id,recipient_id,phase,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, c.ID, workspace, from, to, c.Phase, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return c, err
	}
	if err = s.AppendEventTx(tx, "peer.check", nil, nil, map[string]any{"id": c.ID}); err != nil {
		return c, err
	}
	return c, s.commit(tx)
}
func (s *Store) ListPeerChecks() ([]PeerCheck, error) {
	rows, err := s.db.Query(`SELECT id,workspace_id,sender_id,recipient_id,phase,created_at,updated_at FROM peer_checks WHERE phase IN ('pending','attempted','running') OR id IN (SELECT id FROM peer_checks ORDER BY created_at DESC,id DESC LIMIT 100) ORDER BY created_at DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PeerCheck{}
	for rows.Next() {
		var c PeerCheck
		if err = rows.Scan(&c.ID, &c.WorkspaceID, &c.SenderID, &c.RecipientID, &c.Phase, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (s *Store) SetPeerCheckPhase(id, from, to string) (bool, error) {
	valid := map[string]bool{"pending": true, "attempted": true, "running": true, "passed": true, "uncertain": true, "expired": true, "cancelled": true}
	if !valid[from] || !valid[to] {
		return false, ErrPeerInput
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`UPDATE peer_checks SET phase=?,updated_at=? WHERE id=? AND phase=?`, to, nowUTC(), id, from)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		if err = s.AppendEventTx(tx, "peer.check", nil, nil, map[string]any{"id": id}); err != nil {
			return false, err
		}
	}
	return n > 0, s.commit(tx)
}

// PeerCheckPassed requires native send, reply correlation and both explicit ACKs.
func (s *Store) PeerCheckPassed(c PeerCheck) bool {
	var count int
	err := s.db.QueryRow(`SELECT count(*) FROM peer_messages a JOIN peer_messages b ON b.reply_to=a.id WHERE a.sender_id=? AND a.recipient_id=? AND a.request_id=? AND b.sender_id=a.recipient_id AND b.recipient_id=a.sender_id AND a.acked_at IS NOT NULL AND b.acked_at IS NOT NULL`, c.SenderID, c.RecipientID, c.ID).Scan(&count)
	return err == nil && count > 0
}

// PeerCheckRequest is a bounded, owner-requested diagnostic instruction for
// this authenticated sender. It never grants access to another inbox.
func (s *Store) PeerCheckRequest(token string) (*PeerCheck, error) {
	p, err := s.AuthorizePeer(token)
	if err != nil {
		return nil, err
	}
	var c PeerCheck
	err = s.db.QueryRow(`SELECT id,workspace_id,sender_id,recipient_id,phase,created_at,updated_at FROM peer_checks WHERE sender_id=? AND phase IN ('pending','attempted','running') AND NOT EXISTS (SELECT 1 FROM peer_messages m WHERE m.sender_id=peer_checks.sender_id AND m.request_id=peer_checks.id) ORDER BY created_at LIMIT 1`, p.ID).Scan(&c.ID, &c.WorkspaceID, &c.SenderID, &c.RecipientID, &c.Phase, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) PeerWorkspaceHistory(workspace string, before int64) ([]PeerMessage, error) {
	if before <= 0 {
		before = 1<<63 - 1
	}
	rows, err := s.db.Query(`SELECT `+peerMessageCols+` FROM peer_messages WHERE sender_id IN (SELECT id FROM peer_connections WHERE workspace_id=?) AND seq<? ORDER BY seq DESC LIMIT 100`, workspace, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PeerMessage{}
	for rows.Next() {
		m, e := scanPeerMessage(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
