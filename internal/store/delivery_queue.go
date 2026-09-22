package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// QueueEntry is one integration intent (ADR-0182): a durable request that the
// reviewed revision be integrated into its target. What the row records is what
// the queue itself did; eligibility — the branch still pointing at the reviewed
// revision, the target not having moved, the evidence still covering it — is
// derived from Git and receipts when the entry is read or run, never stored here.
type QueueEntry struct {
	ID           string `json:"id"`
	Sequence     int64  `json:"sequence"`
	DeliveryID   string `json:"deliveryId"`
	Principal    string `json:"principal"`
	Revision     string `json:"revision"`
	Target       string `json:"target"`
	State        string `json:"state"`
	OrderKey     int64  `json:"orderKey"`
	Version      int    `json:"version"`
	Note         string `json:"note,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	AuthorizedAt string `json:"authorizedAt,omitempty"`
}

// Queue states. waiting and authorized are the owner's to move; running and the
// three terminal states are the executor's (which acts as the owner, because the
// authorization already happened).
const (
	QueueWaiting    = "waiting"
	QueueAuthorized = "authorized"
	QueueRunning    = "running"
	QueueDone       = "done"
	QueueFailed     = "failed"
	QueueWithdrawn  = "withdrawn"
)

// OwnerActor is the actor string the daemon's owner surfaces use. An agent may
// enqueue and withdraw its own delivery; ordering and authorizing are the
// owner's alone (ADR-0182), and the store refuses them for any other actor so a
// door that forgets to check cannot grant them by accident.
const OwnerActor = "owner"

// QueueMutation is one queue write. Payloads compare byte-for-byte on retry, so
// a retry must repeat the exact same request (the requestId rule of ADR-0171).
type QueueMutation struct {
	Action          string `json:"action"`
	RequestID       string `json:"requestId"`
	ID              string `json:"id,omitempty"`
	ExpectedVersion int    `json:"expectedVersion,omitempty"`
	DeliveryID      string `json:"deliveryId,omitempty"`
	Revision        string `json:"revision,omitempty"`
	Target          string `json:"target,omitempty"`
	OrderKey        *int64 `json:"orderKey,omitempty"`
	Note            string `json:"note,omitempty"`
}

var (
	ErrQueueConflict = errors.New("queue entry changed or request ID was reused with different content")
	ErrQueueCapacity = errors.New("queue entry capacity reached")
)

// queueActions are the writes the queue serves.
var queueActions = []string{"enqueue", "withdraw", "order", "authorize", "start", "finish", "fail"}

func validateQueueNote(m QueueMutation) error {
	if strings.TrimSpace(m.Note) == "" || len(m.Note) > 300 || strings.ContainsAny(m.Note, "\r\n") {
		return errors.New("a finish or fail needs a one-line note, at most 300 bytes")
	}
	return nil
}

// ValidateQueueMutation checks a mutation's shape — required fields, bounds, and
// the fields an action must not carry. It says nothing about state: that lives
// in ApplyQueueMutation, where the current row is known.
func ValidateQueueMutation(m QueueMutation) error {
	if strings.TrimSpace(m.RequestID) == "" || len(m.RequestID) > 128 {
		return errors.New("requestId is required (at most 128 bytes)")
	}
	known := false
	for _, a := range queueActions {
		if m.Action == a {
			known = true
			break
		}
	}
	if !known {
		return fmt.Errorf("unsupported queue action %q", m.Action)
	}
	if m.Action == "enqueue" {
		if m.ID != "" || m.ExpectedVersion != 0 || m.OrderKey != nil || m.Note != "" {
			return errors.New("enqueue takes deliveryId, revision and target only")
		}
		if m.DeliveryID == "" || len(m.DeliveryID) > 128 || !deliveryOID.MatchString(m.Revision) || m.Target == "" || len(m.Target) > 512 {
			return errors.New("deliveryId, a full lowercase revision and target are required")
		}
		return nil
	}
	if m.ID == "" || len(m.ID) > 128 || m.ExpectedVersion < 1 {
		return errors.New("id and expectedVersion are required")
	}
	switch m.Action {
	case "order":
		if m.OrderKey == nil {
			return errors.New("order takes an orderKey")
		}
		if m.DeliveryID != "" || m.Revision != "" || m.Target != "" || m.Note != "" {
			return errors.New("order takes only id, expectedVersion, orderKey and requestId")
		}
	case "finish", "fail":
		if m.DeliveryID != "" || m.Revision != "" || m.Target != "" || m.OrderKey != nil {
			return errors.New("this action takes only id, expectedVersion, note and requestId")
		}
		return validateQueueNote(m)
	default: // withdraw, authorize, start
		if m.DeliveryID != "" || m.Revision != "" || m.Target != "" || m.OrderKey != nil || m.Note != "" {
			return errors.New("this action takes only id, expectedVersion and requestId")
		}
	}
	return nil
}

// ReplayQueueMutation returns the original receipt; a retry never rewrites
// current state.
func (s *Store) ReplayQueueMutation(repo, actor string, m QueueMutation) (QueueEntry, bool, error) {
	raw, _ := json.Marshal(m)
	var payload, result string
	err := s.db.QueryRow(`SELECT payload,result FROM delivery_queue_requests WHERE repo=? AND actor=? AND request_id=?`, repo, actor, m.RequestID).Scan(&payload, &result)
	if errors.Is(err, sql.ErrNoRows) {
		return QueueEntry{}, false, nil
	}
	if err != nil {
		return QueueEntry{}, false, err
	}
	if payload != string(raw) {
		return QueueEntry{}, false, ErrQueueConflict
	}
	var e QueueEntry
	err = json.Unmarshal([]byte(result), &e)
	return e, true, err
}

// ApplyQueueMutation serializes concurrent writers; the row, the retry receipt
// and the `delivery.changed` event commit together. It shares the delivery
// mutex because a queue write and a declaration write touch the same view.
func (s *Store) ApplyQueueMutation(repo, actor string, m QueueMutation) (QueueEntry, error) {
	if repo == "" || actor == "" {
		return QueueEntry{}, errors.New("repository and actor are required")
	}
	if err := ValidateQueueMutation(m); err != nil {
		return QueueEntry{}, err
	}
	s.deliveryMu.Lock()
	defer s.deliveryMu.Unlock()
	if e, ok, err := s.ReplayQueueMutation(repo, actor, m); ok || err != nil {
		return e, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return QueueEntry{}, err
	}
	defer s.rollback(tx)
	var n int
	if err = tx.QueryRow(`SELECT count(*) FROM delivery_queue_requests WHERE repo=? AND actor=?`, repo, actor).Scan(&n); err != nil {
		return QueueEntry{}, err
	}
	if n >= 10000 {
		return QueueEntry{}, ErrQueueCapacity
	}
	e := QueueEntry{}
	if m.Action == "enqueue" {
		var body string
		if err = tx.QueryRow(`SELECT body FROM delivery_intents WHERE id=? AND repo=?`, m.DeliveryID, repo).Scan(&body); errors.Is(err, sql.ErrNoRows) {
			return e, ErrNotFound
		} else if err != nil {
			return e, err
		}
		// The entry belongs to whoever *declared* the delivery, not to whoever
		// wrote it down: the owner may enqueue on an agent's behalf, and the
		// agent keeps the right to withdraw its own.
		var declared Delivery
		if err = json.Unmarshal([]byte(body), &declared); err != nil {
			return e, err
		}
		var active int
		if err = tx.QueryRow(`SELECT count(*) FROM delivery_queue WHERE repo=? AND delivery_id=? AND state IN (?,?,?)`,
			repo, m.DeliveryID, QueueWaiting, QueueAuthorized, QueueRunning).Scan(&active); err != nil {
			return e, err
		}
		if active > 0 {
			return e, fmt.Errorf("%w: this delivery already has an entry in the queue", ErrQueueConflict)
		}
		if err = tx.QueryRow(`SELECT count(*) FROM delivery_queue WHERE repo=?`, repo).Scan(&n); err != nil {
			return e, err
		}
		if n >= 1000 {
			return e, ErrQueueCapacity
		}
		e = QueueEntry{ID: "queue_" + rand.Text(), DeliveryID: m.DeliveryID, Principal: declared.Principal,
			Revision: m.Revision, Target: m.Target, State: QueueWaiting, Version: 1, CreatedAt: nowUTC()}
	} else {
		var raw string
		err = tx.QueryRow(`SELECT body FROM delivery_queue WHERE id=? AND repo=?`, m.ID, repo).Scan(&raw)
		if errors.Is(err, sql.ErrNoRows) {
			return e, ErrNotFound
		}
		if err != nil {
			return e, err
		}
		if err = json.Unmarshal([]byte(raw), &e); err != nil {
			return e, err
		}
		if e.Version != m.ExpectedVersion {
			return e, ErrQueueConflict
		}
		if err := s.transitionQueueEntry(&e, actor, m); err != nil {
			return e, err
		}
		e.Version++
	}
	e.UpdatedAt = nowUTC()
	raw, _ := json.Marshal(e)
	if e.Sequence == 0 {
		res, err := tx.Exec(`INSERT INTO delivery_queue(id,repo,delivery_id,actor,state,body) VALUES(?,?,?,?,?,?)`,
			e.ID, repo, e.DeliveryID, e.Principal, e.State, "{}")
		if err != nil {
			return e, err
		}
		e.Sequence, _ = res.LastInsertId()
		// The default order is the order things arrived; `order` moves an entry
		// by writing a smaller or larger key, and Sequence still breaks ties.
		e.OrderKey = e.Sequence
		raw, _ = json.Marshal(e)
	}
	if _, err = tx.Exec(`UPDATE delivery_queue SET state=?, body=? WHERE id=?`, e.State, string(raw), e.ID); err != nil {
		return e, err
	}
	payload, _ := json.Marshal(m)
	if _, err = tx.Exec(`INSERT INTO delivery_queue_requests(repo,actor,request_id,payload,result) VALUES(?,?,?,?,?)`,
		repo, actor, m.RequestID, string(payload), string(raw)); err != nil {
		return e, err
	}
	if err = s.AppendEventTx(tx, "delivery.changed", nil, nil, map[string]any{
		"id": e.DeliveryID, "queueId": e.ID, "state": e.State, "version": e.Version}); err != nil {
		return e, err
	}
	return e, s.commit(tx)
}

// transitionQueueEntry is the decision table: from the entry's current state,
// for this actor, this action either moves the entry or is refused with the
// reason. Nothing here reads Git — state is the queue's own memory.
func (s *Store) transitionQueueEntry(e *QueueEntry, actor string, m QueueMutation) error {
	owner := actor == OwnerActor
	switch m.Action {
	case "withdraw":
		if e.State != QueueWaiting && e.State != QueueAuthorized {
			return fmt.Errorf("a %s entry cannot be withdrawn", e.State)
		}
		if !owner && actor != e.Principal {
			return errors.New("only the agent that declared this delivery, or the owner, may withdraw it")
		}
		e.State = QueueWithdrawn
		e.Note = "withdrawn"
	case "order":
		if !owner {
			return errors.New("only the owner may order the queue")
		}
		if e.State != QueueWaiting {
			return fmt.Errorf("a %s entry cannot be ordered", e.State)
		}
		e.OrderKey = *m.OrderKey
	case "authorize":
		if !owner {
			return errors.New("only the owner may authorize an integration")
		}
		if e.State != QueueWaiting {
			return fmt.Errorf("a %s entry cannot be authorized", e.State)
		}
		e.State = QueueAuthorized
		e.AuthorizedAt = nowUTC()
	case "start":
		if !owner {
			return errors.New("only the owner's operation may start an entry")
		}
		if e.State != QueueAuthorized {
			return fmt.Errorf("a %s entry cannot start; it must be authorized first", e.State)
		}
		e.State = QueueRunning
	case "finish":
		if !owner {
			return errors.New("only the owner's operation may finish an entry")
		}
		if e.State != QueueRunning {
			return fmt.Errorf("a %s entry cannot finish", e.State)
		}
		e.State = QueueDone
		e.Note = m.Note
	case "fail":
		if !owner {
			return errors.New("only the owner's operation may fail an entry")
		}
		if e.State != QueueRunning {
			return fmt.Errorf("a %s entry cannot fail", e.State)
		}
		e.State = QueueFailed
		e.Note = m.Note
	}
	return nil
}

// GetQueueEntry reads one entry of this repository.
func (s *Store) GetQueueEntry(repo, id string) (QueueEntry, error) {
	var raw string
	err := s.db.QueryRow(`SELECT body FROM delivery_queue WHERE repo=? AND id=?`, repo, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return QueueEntry{}, ErrNotFound
	}
	if err != nil {
		return QueueEntry{}, err
	}
	var e QueueEntry
	err = json.Unmarshal([]byte(raw), &e)
	return e, err
}

// ListQueueEntries reads the queue in execution order: active entries first
// (waiting, authorized, running) by OrderKey and then Sequence, the terminal
// ones after them by Sequence. The order is applied over the JSON body, which
// is where an entry's truth lives — no column duplicates it. before is a
// sequence cursor: zero reads the head, otherwise entries older than it.
func (s *Store) ListQueueEntries(repo string, before int64) ([]QueueEntry, error) {
	rows, err := s.db.Query(`SELECT body FROM delivery_queue WHERE repo=? AND (?=0 OR seq<?) ORDER BY seq LIMIT 101`,
		repo, before, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []QueueEntry{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		var e QueueEntry
		if err = json.Unmarshal([]byte(raw), &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai, aj := queueActive(out[i].State), queueActive(out[j].State)
		if ai != aj {
			return ai
		}
		if out[i].OrderKey != out[j].OrderKey {
			return out[i].OrderKey < out[j].OrderKey
		}
		return out[i].Sequence < out[j].Sequence
	})
	if len(out) > 100 {
		out = out[:100]
	}
	return out, nil
}

// queueActive says whether an entry is still the queue's business.
func queueActive(state string) bool {
	return state == QueueWaiting || state == QueueAuthorized || state == QueueRunning
}
