package store

// PendingPeerAttention is owner-side delivery work, never an agent inbox read.
func (s *Store) PendingPeerAttention() ([]PeerMessage, error) {
	rows, err := s.db.Query(`SELECT ` + peerMessageCols + ` FROM peer_messages WHERE seq IN (
 SELECT MIN(m.seq) FROM peer_messages m JOIN peer_connections p ON p.id=m.recipient_id
 JOIN (` + peerOwnersSQL + `) o ON o.id=COALESCE(p.agent_id,p.terminal_id)
 AND o.kind=CASE WHEN p.agent_id IS NOT NULL THEN 'agent' ELSE 'terminal' END
 WHERE m.attention_status='pending' AND m.acked_at IS NULL AND p.revoked_at IS NULL
 AND o.session_key=p.session_key AND o.cli=p.cli AND o.workspace_id=p.workspace_id
 GROUP BY m.recipient_id) ORDER BY seq`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPeerMessages(rows)
}

// SetPeerAttention advances one attempt. A durable claim must precede any write;
// an attempted row after a crash is intentionally not automatically retried.
func (s *Store) SetPeerAttention(id, recipient, expected, next string) (bool, error) {
	if !((expected == "pending" && next == "attempted") || (expected == "attempted" && (next == "notified" || next == "uncertain"))) {
		return false, ErrPeerInput
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer s.rollback(tx)
	p, err := scanPeer(tx.QueryRow(`SELECT `+peerCols+` FROM peer_connections WHERE id=?`, recipient))
	if err != nil || !peerCurrent(tx, p) {
		return false, ErrPeerDenied
	}
	result, err := tx.Exec(`UPDATE peer_messages SET attention_status=? WHERE id=? AND recipient_id=? AND attention_status=? AND acked_at IS NULL`, next, id, recipient, expected)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return false, nil
	}
	if err = s.AppendEventTx(tx, "peer.attention", nil, nil, map[string]any{"id": id, "recipientId": recipient}); err != nil {
		return false, err
	}
	return true, s.commit(tx)
}

func (s *Store) PeerConnection(id string) (PeerConnection, error) {
	p, err := scanPeer(s.db.QueryRow(`SELECT `+peerCols+` FROM peer_connections WHERE id=?`, id))
	if err != nil {
		return p, err
	}
	if !peerCurrent(s.db, p) {
		return p, ErrPeerDenied
	}
	p.Active = true
	return p, nil
}
