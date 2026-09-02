package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ActBatchRow is one stored actuation batch (ADR-0053). Actions is the
// raw JSON array; parsing/validation lives in internal/browserhost.
type ActBatchRow struct {
	ID        string `json:"id"`
	AgentID   string `json:"agentId"`
	Origin    string `json:"origin"`
	Actions   string `json:"actions"`
	State     string `json:"state"`
	Round     int    `json:"round"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Act batch states.
const (
	ActPending = "pending"
	ActClaimed = "claimed"
	ActDone    = "done"
	ActExpired = "expired"
)

// actStaleAfter: an unclaimed batch expires — the panel drives execution,
// so a batch nobody came for within ten minutes is dead (ADR-0053).
const actStaleAfter = 10 * time.Minute

// CreateActBatch replaces any pending batch for the agent and inserts a
// new pending one.
func (s *Store) CreateActBatch(agentID, origin, actionsJSON string, round int) (ActBatchRow, error) {
	now := nowUTC()
	b := ActBatchRow{
		ID: newID("act batch", "act"), AgentID: agentID, Origin: origin,
		Actions: actionsJSON, State: ActPending, Round: round,
		CreatedAt: now, UpdatedAt: now,
	}
	tx, err := s.db.Begin()
	if err != nil {
		return ActBatchRow{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE act_batches SET state = 'expired', updated_at = ?
		WHERE agent_id = ? AND state = 'pending'`, now, agentID); err != nil {
		return ActBatchRow{}, err
	}
	if _, err := tx.Exec(`INSERT INTO act_batches
		(id, agent_id, origin, actions, state, round, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		b.ID, b.AgentID, b.Origin, b.Actions, b.State, b.Round, b.CreatedAt, b.UpdatedAt); err != nil {
		return ActBatchRow{}, err
	}
	return b, tx.Commit()
}

// ClaimActBatch expires stale pending rows, then claims the agent's next
// pending batch. ok=false when there is nothing to claim.
func (s *Store) ClaimActBatch(agentID string) (ActBatchRow, bool, error) {
	now := nowUTC()
	cutoff := time.Now().UTC().Add(-actStaleAfter).Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`UPDATE act_batches SET state = 'expired', updated_at = ?
		WHERE state = 'pending' AND created_at < ?`, now, cutoff); err != nil {
		return ActBatchRow{}, false, err
	}
	if _, err := s.db.Exec(`UPDATE act_batches SET state = 'claimed', updated_at = ?
		WHERE id = (SELECT id FROM act_batches
			WHERE agent_id = ? AND state = 'pending'
			ORDER BY created_at DESC LIMIT 1)`, now, agentID); err != nil {
		return ActBatchRow{}, false, err
	}
	return s.pendingOrClaimed(agentID)
}

// PendingActBatch returns the agent's claimed-but-unexecuted batch, if
// the caller already holds one (no second claim).
func (s *Store) PendingActBatch(agentID string) (ActBatchRow, bool, error) {
	return s.pendingOrClaimed(agentID)
}

func (s *Store) pendingOrClaimed(agentID string) (ActBatchRow, bool, error) {
	var b ActBatchRow
	err := s.db.QueryRow(`SELECT id, agent_id, origin, actions, state, round, created_at, updated_at
		FROM act_batches WHERE agent_id = ? AND state IN ('pending','claimed')
		ORDER BY created_at DESC LIMIT 1`, agentID).
		Scan(&b.ID, &b.AgentID, &b.Origin, &b.Actions, &b.State, &b.Round, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return ActBatchRow{}, false, nil
	}
	if err != nil {
		return ActBatchRow{}, false, err
	}
	return b, true, nil
}

// FinishActBatch marks a batch done.
func (s *Store) FinishActBatch(id string) error {
	_, err := s.db.Exec(`UPDATE act_batches SET state = 'done', updated_at = ? WHERE id = ?`,
		nowUTC(), id)
	return err
}

// ExpirePendingActBatches drops an agent's pending batches (a new act
// send supersedes whatever was waiting).
func (s *Store) ExpirePendingActBatches(agentID string) error {
	_, err := s.db.Exec(`UPDATE act_batches SET state = 'expired', updated_at = ?
		WHERE agent_id = ? AND state = 'pending'`, nowUTC(), agentID)
	return err
}

// GetActBatchRow fetches one batch by id.
func (s *Store) GetActBatchRow(id string) (ActBatchRow, bool, error) {
	var b ActBatchRow
	err := s.db.QueryRow(`SELECT id, agent_id, origin, actions, state, round, created_at, updated_at
		FROM act_batches WHERE id = ?`, id).
		Scan(&b.ID, &b.AgentID, &b.Origin, &b.Actions, &b.State, &b.Round, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return ActBatchRow{}, false, nil
	}
	if err != nil {
		return ActBatchRow{}, false, err
	}
	return b, true, nil
}

// DecodeActActions parses the stored actions JSON.
func DecodeActActions(raw string) ([]json.RawMessage, error) {
	var out []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("store: bad act actions: %w", err)
	}
	return out, nil
}
