package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// DeliveryObserver is the one local observer a workspace is bound to: the
// PiCode instance that watches the workspace's repository (ADR-0170 D2). Repo
// is a resolved gitgraph.Key, the repository's absolute common dir, so every
// worktree of one repository resolves to the same binding.
type DeliveryObserver struct {
	WorkspaceID string `json:"workspaceId"`
	Repo        string `json:"repo"`
	Observer    string `json:"observer"`
	UpdatedAt   string `json:"updatedAt"`
}

// The closed set of observers. A workspace with no binding is the absence of a
// row, not a third kind: the delivery read renders that as "unconfigured"
// (O16), and the write path spells it as the removal sentinel.
const (
	observerNone = "none"
	observerSelf = "picode-self"
)

// SetDeliveryObserver upserts the workspace's binding and appends
// delivery.observer.changed in the same transaction as the row, so a
// subscriber can never see one without the other (ADR-0048). observer "none"
// removes the binding; removing one that was never written is not an error and
// announces nothing, because no row changed.
func (s *Store) SetDeliveryObserver(workspaceID, repo, observer string) error {
	if err := validateDeliveryObserver(workspaceID, repo, observer); err != nil {
		return err
	}
	if observer == observerNone {
		return s.clearDeliveryObserver(workspaceID, repo)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	if _, err = tx.Exec(`INSERT INTO delivery_observers (workspace_id, repo, observer, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(workspace_id) DO UPDATE SET repo = excluded.repo, observer = excluded.observer, updated_at = excluded.updated_at`,
		workspaceID, repo, observer, nowUTC()); err != nil {
		return err
	}
	if err = s.AppendEventTx(tx, "delivery.observer.changed", nil, nil, deliveryObserverEvent(workspaceID, repo, observer)); err != nil {
		return err
	}
	return s.commit(tx)
}

// clearDeliveryObserver drops the binding, announcing the unbind only when a
// row went away — the same shape as RevokePeer, where a call that matched
// nothing commits silently.
func (s *Store) clearDeliveryObserver(workspaceID, repo string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`DELETE FROM delivery_observers WHERE workspace_id = ?`, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		if err = s.AppendEventTx(tx, "delivery.observer.changed", nil, nil, deliveryObserverEvent(workspaceID, repo, observerNone)); err != nil {
			return err
		}
	}
	return s.commit(tx)
}

// GetDeliveryObserver returns the workspace's binding; the bool is false when
// the workspace has none, which is not an error — it is the unconfigured
// environment of decision table row 1.
func (s *Store) GetDeliveryObserver(workspaceID string) (DeliveryObserver, bool, error) {
	var o DeliveryObserver
	err := s.db.QueryRow(`SELECT workspace_id, repo, observer, updated_at FROM delivery_observers WHERE workspace_id = ?`, workspaceID).
		Scan(&o.WorkspaceID, &o.Repo, &o.Observer, &o.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return DeliveryObserver{}, false, nil
	}
	if err != nil {
		return DeliveryObserver{}, false, err
	}
	return o, true, nil
}

// ListDeliveryObservers returns every workspace bound to one repository key,
// ordered by workspace so the set is stable between reads. Callers use it to
// tell "another workspace claims this repository" from "nobody does" — a
// binding of a different repository is never substituted (ADR-0170).
func (s *Store) ListDeliveryObservers(repo string) ([]DeliveryObserver, error) {
	rows, err := s.db.Query(`SELECT workspace_id, repo, observer, updated_at FROM delivery_observers WHERE repo = ? ORDER BY workspace_id`, repo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DeliveryObserver{}
	for rows.Next() {
		var o DeliveryObserver
		if err = rows.Scan(&o.WorkspaceID, &o.Repo, &o.Observer, &o.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// validateDeliveryObserver rejects what the store cannot mean. The repository
// key is a value this store only stores: resolving "is this a repository" is
// gitgraph's job, so an unusable path shape is all that is refused here.
func validateDeliveryObserver(workspaceID, repo, observer string) error {
	if strings.TrimSpace(workspaceID) == "" {
		return errors.New("workspaceId is required")
	}
	if observer != observerNone && observer != observerSelf {
		return fmt.Errorf("unknown observer %q", observer)
	}
	if strings.TrimSpace(repo) == "" {
		return errors.New("repository key is required")
	}
	if len(repo) > 4096 {
		return errors.New("repository key must be at most 4096 bytes")
	}
	if strings.ContainsAny(repo, "\x00\r\n") {
		return errors.New("repository key must be one printable line")
	}
	return nil
}

func deliveryObserverEvent(workspaceID, repo, observer string) map[string]string {
	return map[string]string{"workspaceId": workspaceID, "repo": repo, "observer": observer}
}
