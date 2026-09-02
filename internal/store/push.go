package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
)

// PushPrefs is what one device wants to be woken for (ADR-0047).
type PushPrefs struct {
	Actions  bool `json:"actions"`  // an agent or inbox item needs a decision
	Finished bool `json:"finished"` // a run finished unobserved
}

// Wants maps the notifier's kind ("actions" | "finished") to the prefs.
func (p PushPrefs) Wants(kind string) bool {
	switch kind {
	case "actions":
		return p.Actions
	case "finished":
		return p.Finished
	}
	return false
}

// PushSubscription is one browser's Web Push subscription.
type PushSubscription struct {
	ID        string    `json:"id"`
	Endpoint  string    `json:"endpoint"`
	P256dh    string    `json:"-"`
	Auth      string    `json:"-"`
	DeviceID  string    `json:"deviceId,omitempty"`
	UserAgent string    `json:"userAgent,omitempty"`
	Prefs     PushPrefs `json:"prefs"`
	CreatedAt string    `json:"createdAt"`
	LastOKAt  *string   `json:"lastOkAt,omitempty"`
	Failures  int       `json:"failures"`
}

const pushCols = `id, endpoint, p256dh, auth, device_id, user_agent, prefs, created_at, last_ok_at, failures`

func scanPush(row interface{ Scan(...any) error }) (PushSubscription, error) {
	var s PushSubscription
	var prefs string
	var lastOK sql.NullString
	if err := row.Scan(&s.ID, &s.Endpoint, &s.P256dh, &s.Auth, &s.DeviceID, &s.UserAgent, &prefs, &s.CreatedAt, &lastOK, &s.Failures); err != nil {
		return PushSubscription{}, err
	}
	s.Prefs = PushPrefs{Actions: true, Finished: true}
	_ = json.Unmarshal([]byte(prefs), &s.Prefs)
	if lastOK.Valid {
		v := lastOK.String
		s.LastOKAt = &v
	}
	return s, nil
}

// UpsertPushSubscription records a browser's subscription; re-subscribing
// the same endpoint refreshes keys and prefs and resets the failure count.
func (s *Store) UpsertPushSubscription(endpoint, p256dh, auth, deviceID, userAgent string, prefs PushPrefs) (PushSubscription, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || !strings.HasPrefix(endpoint, "https://") {
		return PushSubscription{}, errors.New("push: endpoint must be an https URL")
	}
	if strings.TrimSpace(p256dh) == "" || strings.TrimSpace(auth) == "" {
		return PushSubscription{}, errors.New("push: keys required")
	}
	pj, _ := json.Marshal(prefs)
	now := nowUTC()
	_, err := s.db.Exec(`INSERT INTO push_subscriptions (`+pushCols+`) VALUES (?,?,?,?,?,?,?,?,NULL,0)
		ON CONFLICT(endpoint) DO UPDATE SET p256dh=excluded.p256dh, auth=excluded.auth, device_id=excluded.device_id,
		user_agent=excluded.user_agent, prefs=excluded.prefs, failures=0`,
		newID("push", "ps"), endpoint, p256dh, auth, deviceID, userAgent, string(pj), now)
	if err != nil {
		return PushSubscription{}, err
	}
	s.note("push.updated", nil, nil, nil)
	return s.GetPushSubscription(endpoint)
}

func (s *Store) GetPushSubscription(endpoint string) (PushSubscription, error) {
	row := s.db.QueryRow(`SELECT `+pushCols+` FROM push_subscriptions WHERE endpoint = ?`, endpoint)
	sub, err := scanPush(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PushSubscription{}, ErrNotFound
	}
	return sub, err
}

func (s *Store) ListPushSubscriptions() ([]PushSubscription, error) {
	rows, err := s.db.Query(`SELECT ` + pushCols + ` FROM push_subscriptions ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PushSubscription{}
	for rows.Next() {
		sub, err := scanPush(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

func (s *Store) SetPushPrefs(endpoint string, prefs PushPrefs) (PushSubscription, error) {
	pj, _ := json.Marshal(prefs)
	res, err := s.db.Exec(`UPDATE push_subscriptions SET prefs = ? WHERE endpoint = ?`, string(pj), endpoint)
	if err != nil {
		return PushSubscription{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return PushSubscription{}, ErrNotFound
	}
	s.note("push.updated", nil, nil, nil) // endpoints are secrets: type only
	return s.GetPushSubscription(endpoint)
}

func (s *Store) DeletePushSubscription(endpoint string) error {
	if _, err := s.db.Exec(`DELETE FROM push_subscriptions WHERE endpoint = ?`, endpoint); err != nil {
		return err
	}
	s.note("push.updated", nil, nil, nil)
	return nil
}

func (s *Store) MarkPushOK(endpoint string) error {
	_, err := s.db.Exec(`UPDATE push_subscriptions SET last_ok_at = ?, failures = 0 WHERE endpoint = ?`, nowUTC(), endpoint)
	return err
}

func (s *Store) MarkPushFailure(endpoint string) error {
	_, err := s.db.Exec(`UPDATE push_subscriptions SET failures = failures + 1 WHERE endpoint = ?`, endpoint)
	return err
}
