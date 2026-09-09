package store

// Direct conversation inboxes. ADR-0104 deliberately does not use tasks.
import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type peerRow interface{ Scan(...any) error }

const PeerTokenPrefix = "pcc_"

var ErrPeerDenied = errors.New("connection is disabled or its recorded conversation changed")
var ErrPeerInput = errors.New("invalid communication request")
var ErrPeerConflict = errors.New("request ID was already used with different content")
var ErrPeerCapacity = errors.New("message capacity reached")

// PeerOwner is a recorded native conversation, not a live-process claim.
type PeerOwner struct {
	Kind        string `json:"kind"`
	OwnerID     string `json:"ownerId"`
	WorkspaceID string `json:"workspaceId"`
	Label       string `json:"label"`
	CLI         string `json:"cli"`
	SessionKey  string `json:"sessionKey"`
}
type PeerConnection struct {
	ID string `json:"id"`
	PeerOwner
	CreatedAt string  `json:"createdAt"`
	RevokedAt *string `json:"revokedAt"`
	Active    bool    `json:"active"`
}
type PeerMessage struct {
	Attention   string  `json:"attention"`
	Seq         int64   `json:"seq"`
	ID          string  `json:"id"`
	SenderID    string  `json:"senderId"`
	RecipientID string  `json:"recipientId"`
	RequestID   string  `json:"requestId"`
	Body        string  `json:"body"`
	ReplyTo     string  `json:"replyTo,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	AckedAt     *string `json:"ackedAt"`
}

const peerOwnersSQL = `SELECT 'agent' kind,id,workspace_id,name label,'pi' cli,COALESCE(session_path,'') session_key FROM agents
 UNION ALL SELECT 'terminal',t.id,COALESCE(t.workspace_id,''),t.name,COALESCE(l.cli,''),
 CASE WHEN json_valid(l.last_session) THEN CASE WHEN json_extract(l.last_session,'$.cli')=l.cli THEN COALESCE(json_extract(l.last_session,'$.sessionId'),'') ELSE '' END ELSE '' END
 FROM terminals t LEFT JOIN terminal_launches l ON l.terminal_id=t.id`

func scanPeerOwner(row peerRow) (p PeerOwner, err error) {
	err = row.Scan(&p.Kind, &p.OwnerID, &p.WorkspaceID, &p.Label, &p.CLI, &p.SessionKey)
	return
}
func peerOwner(q txRunner, kind, id string) (PeerOwner, error) {
	return scanPeerOwner(q.QueryRow(`SELECT * FROM (`+peerOwnersSQL+`) WHERE kind=? AND id=?`, kind, id))
}
func (s *Store) ListPeerOwners() ([]PeerOwner, error) {
	rows, err := s.db.Query(peerOwnersSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PeerOwner{}
	for rows.Next() {
		p, e := scanPeerOwner(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

const peerCols = `id,CASE WHEN agent_id IS NOT NULL THEN 'agent' ELSE 'terminal' END,COALESCE(agent_id,terminal_id),workspace_id,label,cli,session_key,created_at,revoked_at`

func scanPeer(row peerRow) (p PeerConnection, err error) {
	err = row.Scan(&p.ID, &p.Kind, &p.PeerOwner.OwnerID, &p.WorkspaceID, &p.Label, &p.CLI, &p.SessionKey, &p.CreatedAt, &p.RevokedAt)
	return
}
func peerCurrent(q txRunner, p PeerConnection) bool {
	if p.RevokedAt != nil {
		return false
	}
	owner, err := peerOwner(q, p.Kind, p.PeerOwner.OwnerID)
	return err == nil && owner.SessionKey != "" && owner.SessionKey == p.SessionKey && owner.CLI == p.CLI && owner.WorkspaceID == p.WorkspaceID
}
func (s *Store) ListPeerConnections() ([]PeerConnection, error) {
	// Close rows before querying current owners: the store has one connection.
	rows, err := s.db.Query(`SELECT ` + peerCols + ` FROM peer_connections ORDER BY created_at DESC,id`)
	if err != nil {
		return nil, err
	}
	out := []PeerConnection{}
	for rows.Next() {
		p, e := scanPeer(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		out = append(out, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Active = peerCurrent(s.db, out[i])
	}
	return out, nil
}
func peerHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
func peerAuth(q txRunner, token string) (PeerConnection, error) {
	if !strings.HasPrefix(token, PeerTokenPrefix) || len(token) != len(PeerTokenPrefix)+64 {
		return PeerConnection{}, ErrPeerDenied
	}
	p, err := scanPeer(q.QueryRow(`SELECT `+peerCols+` FROM peer_connections WHERE token_hash=?`, peerHash(token)))
	if err != nil || !peerCurrent(q, p) {
		return PeerConnection{}, ErrPeerDenied
	}
	p.Active = true
	return p, nil
}

// AuthorizePeer is also called before protocol negotiation; tool operations
// repeat the check in their own transaction to close revocation races.
func (s *Store) AuthorizePeer(token string) (PeerConnection, error) { return peerAuth(s.db, token) }

// EnablePeer returns a one-time secret. Expected session prevents stale UI opt-in.
func (s *Store) EnablePeer(kind, id, expectedSession string) (PeerConnection, string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return PeerConnection{}, "", err
	}
	defer s.rollback(tx)
	owner, err := peerOwner(tx, kind, id)
	if errors.Is(err, sql.ErrNoRows) {
		return PeerConnection{}, "", ErrNotFound
	}
	if err != nil {
		return PeerConnection{}, "", err
	}
	if expectedSession == "" || owner.SessionKey != expectedSession || owner.WorkspaceID == "" || owner.CLI == "" {
		return PeerConnection{}, "", ErrPeerDenied
	}
	var duplicate int
	err = tx.QueryRow(`SELECT count(*) FROM peer_connections p
 WHERE p.revoked_at IS NULL AND p.cli=? AND p.session_key=?
 AND NOT (COALESCE(p.agent_id,p.terminal_id)=? AND CASE WHEN p.agent_id IS NOT NULL THEN 'agent' ELSE 'terminal' END=?)`, owner.CLI, owner.SessionKey, id, kind).Scan(&duplicate)
	if err != nil {
		return PeerConnection{}, "", err
	}
	if duplicate != 0 {
		return PeerConnection{}, "", fmt.Errorf("%w: this conversation is already connected through another owner", ErrPeerInput)
	}
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return PeerConnection{}, "", err
	}
	token := PeerTokenPrefix + hex.EncodeToString(b)
	p := PeerConnection{ID: "peer_" + rand.Text(), PeerOwner: owner, CreatedAt: nowUTC(), Active: true}
	var agent, terminal any
	if kind == "agent" {
		agent = id
	} else {
		terminal = id
	}
	if _, err = tx.Exec(`UPDATE peer_connections SET revoked_at=? WHERE revoked_at IS NULL AND (agent_id=? OR terminal_id=?)`, p.CreatedAt, agent, terminal); err != nil {
		return p, "", err
	}
	_, err = tx.Exec(`INSERT INTO peer_connections(id,agent_id,terminal_id,workspace_id,session_key,cli,label,token_hash,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, p.ID, agent, terminal, p.WorkspaceID, p.SessionKey, p.CLI, p.Label, peerHash(token), p.CreatedAt)
	if err != nil {
		return p, "", err
	}
	if err = s.AppendEventTx(tx, "peer.connection", nil, nil, map[string]any{"id": p.ID}); err != nil {
		return p, "", err
	}
	return p, token, s.commit(tx)
}
func (s *Store) RevokePeer(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`UPDATE peer_connections SET revoked_at=? WHERE id=? AND revoked_at IS NULL`, nowUTC(), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		if err = s.AppendEventTx(tx, "peer.connection", nil, nil, map[string]any{"id": id}); err != nil {
			return err
		}
	}
	return s.commit(tx)
}
func (s *Store) PeerContacts(token string) ([]PeerConnection, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer s.rollback(tx)
	me, err := peerAuth(tx, token)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(`SELECT `+peerCols+` FROM peer_connections WHERE workspace_id=? AND id!=? AND revoked_at IS NULL ORDER BY label,id`, me.WorkspaceID, me.ID)
	if err != nil {
		return nil, err
	}
	found := []PeerConnection{}
	for rows.Next() {
		p, e := scanPeer(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		found = append(found, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := []PeerConnection{}
	for _, p := range found {
		if peerCurrent(tx, p) {
			p.Active = true
			out = append(out, p)
		}
	}
	return out, nil
}

const peerMessageCols = `seq,id,sender_id,recipient_id,request_id,body,COALESCE(reply_to,''),created_at,acked_at,attention_status`

func scanPeerMessage(row peerRow) (m PeerMessage, err error) {
	err = row.Scan(&m.Seq, &m.ID, &m.SenderID, &m.RecipientID, &m.RequestID, &m.Body, &m.ReplyTo, &m.CreatedAt, &m.AckedAt, &m.Attention)
	return
}
func (s *Store) SendPeerMessage(token, to, requestID, body, replyTo string) (PeerMessage, error) {
	if len(requestID) < 1 || len(requestID) > 128 || len(body) > 16384 || !utf8.ValidString(body) || strings.TrimSpace(body) == "" || len(to) > 200 || len(replyTo) > 200 {
		return PeerMessage{}, ErrPeerInput
	}
	tx, err := s.db.Begin()
	if err != nil {
		return PeerMessage{}, err
	}
	defer s.rollback(tx)
	me, err := peerAuth(tx, token)
	if err != nil {
		return PeerMessage{}, err
	}
	old, err := scanPeerMessage(tx.QueryRow(`SELECT `+peerMessageCols+` FROM peer_messages WHERE sender_id=? AND request_id=?`, me.ID, requestID))
	if err == nil {
		if old.RecipientID != to || old.Body != body || old.ReplyTo != replyTo {
			return PeerMessage{}, ErrPeerConflict
		}
		return old, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return PeerMessage{}, err
	}
	recipient, err := scanPeer(tx.QueryRow(`SELECT `+peerCols+` FROM peer_connections WHERE id=?`, to))
	if err != nil || to == me.ID || recipient.WorkspaceID != me.WorkspaceID || !peerCurrent(tx, recipient) {
		return PeerMessage{}, ErrPeerDenied
	}
	if replyTo != "" {
		var n int
		err = tx.QueryRow(`SELECT count(*) FROM peer_messages WHERE id=? AND ((sender_id=? AND recipient_id=?) OR (sender_id=? AND recipient_id=?))`, replyTo, me.ID, to, to, me.ID).Scan(&n)
		if err != nil {
			return PeerMessage{}, err
		}
		if n != 1 {
			return PeerMessage{}, ErrPeerInput
		}
	}
	var pending, total int
	if err = tx.QueryRow(`SELECT count(*) FROM peer_messages WHERE recipient_id=? AND acked_at IS NULL`, to).Scan(&pending); err != nil {
		return PeerMessage{}, err
	}
	if err = tx.QueryRow(`SELECT count(*) FROM peer_messages WHERE sender_id=? AND recipient_id=?`, me.ID, to).Scan(&total); err != nil {
		return PeerMessage{}, err
	}
	if pending >= 1000 || total >= 10000 {
		return PeerMessage{}, ErrPeerCapacity
	}
	m := PeerMessage{Attention: "pending", ID: "msg_" + rand.Text(), SenderID: me.ID, RecipientID: to, RequestID: requestID, Body: body, ReplyTo: replyTo, CreatedAt: nowUTC()}
	result, err := tx.Exec(`INSERT INTO peer_messages(id,sender_id,recipient_id,request_id,body,reply_to,created_at) VALUES(?,?,?,?,?,?,?)`, m.ID, me.ID, to, requestID, body, replyTo, m.CreatedAt)
	if err != nil {
		return m, err
	}
	m.Seq, _ = result.LastInsertId()
	if err = s.AppendEventTx(tx, "peer.message", nil, nil, map[string]any{"id": m.ID, "senderId": me.ID, "recipientId": to}); err != nil {
		return m, err
	}
	return m, s.commit(tx)
}

// ReadPeerMessages only reads the caller's inbox; it never acknowledges.
func (s *Store) ReadPeerMessages(token string, after int64, limit int, pendingOnly bool) ([]PeerMessage, error) {
	if after < 0 || limit < 1 || limit > 100 {
		return nil, ErrPeerInput
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer s.rollback(tx)
	me, err := peerAuth(tx, token)
	if err != nil {
		return nil, err
	}
	query := `SELECT ` + peerMessageCols + ` FROM peer_messages WHERE recipient_id=? AND seq>?`
	if pendingOnly {
		query += ` AND acked_at IS NULL`
	}
	query += ` ORDER BY seq LIMIT ?`
	rows, err := tx.Query(query, me.ID, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPeerMessages(rows)
}
func collectPeerMessages(rows *sql.Rows) ([]PeerMessage, error) {
	out := []PeerMessage{}
	for rows.Next() {
		m, err := scanPeerMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// PeerHistory is an owner-only read, including sent and acknowledged messages.
func (s *Store) PeerHistory(id string, before int64) ([]PeerMessage, error) {
	if before < 0 {
		return nil, ErrPeerInput
	}
	if before == 0 {
		before = 1<<63 - 1
	}
	rows, err := s.db.Query(`SELECT `+peerMessageCols+` FROM peer_messages WHERE (sender_id=? OR recipient_id=?) AND seq<? ORDER BY seq DESC LIMIT 100`, id, id, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPeerMessages(rows)
}
func (s *Store) AckPeerMessages(token string, ids []string) error {
	if len(ids) < 1 || len(ids) > 100 {
		return ErrPeerInput
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	me, err := peerAuth(tx, token)
	if err != nil {
		return err
	}
	for _, id := range ids {
		var recipient string
		if err = tx.QueryRow(`SELECT recipient_id FROM peer_messages WHERE id=?`, id).Scan(&recipient); err != nil || recipient != me.ID {
			return ErrPeerInput
		}
	}
	changed := false
	for _, id := range ids {
		res, e := tx.Exec(`UPDATE peer_messages SET acked_at=? WHERE id=? AND acked_at IS NULL`, nowUTC(), id)
		if e != nil {
			return e
		}
		n, _ := res.RowsAffected()
		changed = changed || n > 0
	}
	if changed {
		if err = s.AppendEventTx(tx, "peer.ack", nil, nil, map[string]any{"recipientId": me.ID}); err != nil {
			return fmt.Errorf("ack event: %w", err)
		}
	}
	return s.commit(tx)
}
