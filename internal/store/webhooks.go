package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

// Webhook is one outbound subscription (ADR-0071): the deliverer POSTs
// every durable event whose type matches one of the Types prefixes to
// URL, signed with Secret (HMAC-SHA256). Secret never marshals — it is
// returned once by AddWebhook so the owner can store it, and is used by
// the deliverer to sign.
type Webhook struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	Types         []string `json:"types"`
	Enabled       bool     `json:"enabled"`
	Cursor        int64    `json:"cursor"`
	LastStatus    string   `json:"lastStatus,omitempty"`
	LastError     string   `json:"lastError,omitempty"`
	LastAttemptAt string   `json:"lastAttemptAt,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	Secret        string   `json:"-"`
}

const webhookCols = `id, url, secret, types, enabled, cursor, last_status, last_error, last_attempt_at, created_at`

func scanWebhook(row interface{ Scan(...any) error }) (Webhook, error) {
	var w Webhook
	var types string
	if err := row.Scan(&w.ID, &w.URL, &w.Secret, &types, &w.Enabled, &w.Cursor, &w.LastStatus, &w.LastError, &w.LastAttemptAt, &w.CreatedAt); err != nil {
		return Webhook{}, err
	}
	_ = json.Unmarshal([]byte(types), &w.Types)
	return w, nil
}

// normalizeTypes trims, lowercases, drops empties and duplicates, and
// sorts — so an event prefix list compares equal regardless of input
// order and the fingerprint/JSON stays stable.
func normalizeTypes(types []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(types))
	for _, t := range types {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// AddWebhook registers a subscription and returns it with the generated
// secret — the only time the secret is available to show the owner. The
// cursor starts at the newest event: a subscription receives what happens
// after it was created, not the retained history.
func (s *Store) AddWebhook(url string, types []string) (Webhook, error) {
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return Webhook{}, errors.New("webhook: url must be an http(s) URL")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Webhook{}, err
	}
	secret := base64.RawURLEncoding.EncodeToString(raw)
	types = normalizeTypes(types)
	tj, _ := json.Marshal(types)
	now := nowUTC()
	id := newID(url, "wh")
	cursor, _ := s.LatestEventID()
	if _, err := s.db.Exec(`INSERT INTO webhook_subscriptions (`+webhookCols+`) VALUES (?,?,?,?,?,?, '', '', '', ?)`,
		id, url, secret, string(tj), 1, cursor, now); err != nil {
		return Webhook{}, err
	}
	w, err := s.GetWebhook(id)
	if err != nil {
		return Webhook{}, err
	}
	w.Secret = secret
	s.note("webhook.created", nil, nil, map[string]any{"id": id, "url": url, "types": w.Types})
	return w, nil
}

// UpdateWebhook changes url, types and enabled on an existing
// subscription. Secret and cursor are delivery state — not editable here.
func (s *Store) UpdateWebhook(w Webhook) (Webhook, error) {
	if w.ID == "" {
		return Webhook{}, errors.New("webhook: id required")
	}
	types := normalizeTypes(w.Types)
	tj, _ := json.Marshal(types)
	res, err := s.db.Exec(`UPDATE webhook_subscriptions SET url = ?, types = ?, enabled = ? WHERE id = ?`,
		strings.TrimSpace(w.URL), string(tj), boolToInt(w.Enabled), w.ID)
	if err != nil {
		return Webhook{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Webhook{}, ErrNotFound
	}
	out, err := s.GetWebhook(w.ID)
	if err != nil {
		return Webhook{}, err
	}
	s.note("webhook.updated", nil, nil, map[string]any{"id": out.ID, "url": out.URL, "types": out.Types, "enabled": out.Enabled})
	return out, nil
}

// DeleteWebhook removes the subscription. Deliveries already queued are
// dropped; receivers keep whatever they got (at-least-once per event).
func (s *Store) DeleteWebhook(id string) error {
	res, err := s.db.Exec(`DELETE FROM webhook_subscriptions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	s.note("webhook.deleted", nil, nil, idData(id))
	return nil
}

// SetWebhookDelivery records the deliverer's bookkeeping: the advanced
// cursor after a successful POST, or the failure reason. Deliberately no
// event — one row per delivery would flood the log the feature reads.
func (s *Store) SetWebhookDelivery(id string, cursor int64, status, lastErr string) error {
	res, err := s.db.Exec(`UPDATE webhook_subscriptions SET cursor = ?, last_status = ?, last_error = ?, last_attempt_at = ? WHERE id = ?`,
		cursor, status, lastErr, nowUTC(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) GetWebhook(id string) (Webhook, error) {
	row := s.db.QueryRow(`SELECT `+webhookCols+` FROM webhook_subscriptions WHERE id = ?`, id)
	w, err := scanWebhook(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Webhook{}, ErrNotFound
	}
	return w, err
}

func (s *Store) ListWebhooks() ([]Webhook, error) {
	rows, err := s.db.Query(`SELECT ` + webhookCols + ` FROM webhook_subscriptions ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Webhook
	for rows.Next() {
		w, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
