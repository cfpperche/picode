package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Delivery records an agent declaration. Review requested is not review approved.
// Git, checks and publication evidence are deliberately outside this row (ADR-0171).
type Delivery struct {
	ID                string `json:"id"`
	Sequence          int64  `json:"sequence"`
	Principal         string `json:"principal"`
	Title             string `json:"title"`
	Branch            string `json:"branch"`
	Revision          string `json:"revision"`
	Target            string `json:"target"`
	Version           int    `json:"version"`
	Review            string `json:"review"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
	ReviewRequestedAt string `json:"reviewRequestedAt,omitempty"`
}

type DeliveryMutation struct {
	Action          string `json:"action"`
	RequestID       string `json:"requestId"`
	ID              string `json:"id,omitempty"`
	ExpectedVersion int    `json:"expectedVersion,omitempty"`
	Title           string `json:"title,omitempty"`
	Branch          string `json:"branch,omitempty"`
	Revision        string `json:"revision,omitempty"`
	Target          string `json:"target,omitempty"`
}

var ErrDeliveryConflict = errors.New("delivery changed or request ID was reused with different content")
var ErrDeliveryCapacity = errors.New("delivery record capacity reached")
var deliveryOID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

func ValidateDeliveryMutation(m DeliveryMutation) error {
	if strings.TrimSpace(m.RequestID) == "" || len(m.RequestID) > 128 {
		return errors.New("requestId is required (at most 128 bytes)")
	}
	switch m.Action {
	case "register", "update":
		if strings.TrimSpace(m.Title) == "" || len(m.Title) > 300 || strings.ContainsAny(m.Title, "\r\n") {
			return errors.New("title must be one line, at most 300 bytes")
		}
		if m.Branch == "" || len(m.Branch) > 512 || m.Target == "" || len(m.Target) > 512 || !deliveryOID.MatchString(m.Revision) {
			return errors.New("branch, target and full lowercase revision are required")
		}
	case "request-review", "withdraw-review":
		if m.Title != "" || m.Branch != "" || m.Target != "" || m.Revision != "" {
			return errors.New("review actions take only id, expectedVersion and requestId")
		}
	default:
		return errors.New("unsupported delivery action")
	}
	if m.Action == "register" {
		if m.ID != "" || m.ExpectedVersion != 0 {
			return errors.New("register does not take id or expectedVersion")
		}
	} else if m.ID == "" || len(m.ID) > 128 || m.ExpectedVersion < 1 {
		return errors.New("id and expectedVersion are required")
	}
	return nil
}

// ReplayDelivery returns the original receipt; a retry never rewrites current state.
func (s *Store) ReplayDelivery(repo, principal string, m DeliveryMutation) (Delivery, bool, error) {
	raw, _ := json.Marshal(m)
	var payload, result string
	err := s.db.QueryRow(`SELECT payload,result FROM delivery_requests WHERE repo=? AND principal=? AND request_id=?`, repo, principal, m.RequestID).Scan(&payload, &result)
	if errors.Is(err, sql.ErrNoRows) {
		return Delivery{}, false, nil
	}
	if err != nil {
		return Delivery{}, false, err
	}
	if payload != string(raw) {
		return Delivery{}, false, ErrDeliveryConflict
	}
	var d Delivery
	err = json.Unmarshal([]byte(result), &d)
	return d, true, err
}

// ApplyDelivery serializes duplicate/concurrent writers; row, retry receipt and
// event commit together. Reads never infer native session ownership from principal.
func (s *Store) ApplyDelivery(repo, principal string, m DeliveryMutation) (Delivery, error) {
	if repo == "" || principal == "" {
		return Delivery{}, errors.New("repository and principal are required")
	}
	if err := ValidateDeliveryMutation(m); err != nil {
		return Delivery{}, err
	}
	s.deliveryMu.Lock()
	defer s.deliveryMu.Unlock()
	if d, ok, err := s.ReplayDelivery(repo, principal, m); ok || err != nil {
		return d, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Delivery{}, err
	}
	defer s.rollback(tx)
	var n int
	if err = tx.QueryRow(`SELECT count(*) FROM delivery_requests WHERE repo=? AND principal=?`, repo, principal).Scan(&n); err != nil {
		return Delivery{}, err
	}
	if n >= 10000 {
		return Delivery{}, ErrDeliveryCapacity
	}
	d := Delivery{}
	if m.Action == "register" {
		if err = tx.QueryRow(`SELECT count(*) FROM delivery_intents WHERE repo=?`, repo).Scan(&n); err != nil {
			return d, err
		}
		if n >= 1000 {
			return d, ErrDeliveryCapacity
		}
		d = Delivery{ID: "delivery_" + rand.Text(), Principal: principal, Version: 1, Review: "not-requested", CreatedAt: nowUTC()}
	} else {
		var raw string
		err = tx.QueryRow(`SELECT body FROM delivery_intents WHERE id=? AND repo=? AND principal=?`, m.ID, repo, principal).Scan(&raw)
		if errors.Is(err, sql.ErrNoRows) {
			return d, ErrNotFound
		}
		if err != nil {
			return d, err
		}
		if err = json.Unmarshal([]byte(raw), &d); err != nil {
			return d, err
		}
		if d.Version != m.ExpectedVersion {
			return d, ErrDeliveryConflict
		}
		d.Version++
	}
	switch m.Action {
	case "register", "update":
		d.Title = m.Title
		d.Branch = m.Branch
		d.Revision = m.Revision
		d.Target = m.Target
		d.Review = "not-requested"
		d.ReviewRequestedAt = ""
	case "request-review":
		d.Review = "requested"
		d.ReviewRequestedAt = nowUTC()
	case "withdraw-review":
		d.Review = "not-requested"
		d.ReviewRequestedAt = ""
	}
	d.UpdatedAt = nowUTC()
	if m.Action == "register" {
		res, e := tx.Exec(`INSERT INTO delivery_intents(id,repo,principal,body) VALUES(?,?,?,?)`, d.ID, repo, principal, "{}")
		if e != nil {
			return d, e
		}
		d.Sequence, _ = res.LastInsertId()
	}
	raw, _ := json.Marshal(d)
	if _, err = tx.Exec(`UPDATE delivery_intents SET body=? WHERE id=?`, string(raw), d.ID); err != nil {
		return d, err
	}
	payload, _ := json.Marshal(m)
	if _, err = tx.Exec(`INSERT INTO delivery_requests(repo,principal,request_id,payload,result) VALUES(?,?,?,?,?)`, repo, principal, m.RequestID, string(payload), string(raw)); err != nil {
		return d, err
	}
	if err = s.AppendEventTx(tx, "delivery.changed", nil, nil, map[string]any{"id": d.ID, "version": d.Version, "principal": d.Principal}); err != nil {
		return d, err
	}
	return d, s.commit(tx)
}

func (s *Store) GetDelivery(repo, id string) (Delivery, error) {
	var raw string
	err := s.db.QueryRow(`SELECT body FROM delivery_intents WHERE repo=? AND id=?`, repo, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return Delivery{}, ErrNotFound
	}
	if err != nil {
		return Delivery{}, err
	}
	var d Delivery
	err = json.Unmarshal([]byte(raw), &d)
	return d, err
}

// ListDeliveries pages by stable creation sequence, not mutable update time.
func (s *Store) ListDeliveries(repo string, before int64) ([]Delivery, error) {
	if before < 0 {
		return nil, fmt.Errorf("before must not be negative")
	}
	rows, err := s.db.Query(`SELECT body FROM delivery_intents WHERE repo=? AND (?=0 OR seq<?) ORDER BY seq DESC LIMIT 101`, repo, before, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Delivery{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		var d Delivery
		if err = json.Unmarshal([]byte(raw), &d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
