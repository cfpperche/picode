package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/hookurl"
)

var ErrWebhookConflict = errors.New("This webhook changed. Refresh and try again.")

// Webhook is an outbound subscription (ADR-0075). Secrets are available to
// the in-process deliverer, never to normal JSON responses or event payloads.
type Webhook struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	Types         []string `json:"types"`
	Enabled       bool     `json:"enabled"`
	Cursor        int64    `json:"cursor"`
	Revision      int64    `json:"revision"`
	Failures      int      `json:"failures"`
	NextAttemptAt string   `json:"nextAttemptAt,omitempty"`
	LastStatus    string   `json:"lastStatus,omitempty"`
	LastError     string   `json:"lastError,omitempty"`
	LastAttemptAt string   `json:"lastAttemptAt,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	Secret        string   `json:"-"`
}

const webhookCols = `id,url,secret,types,enabled,cursor,revision,failures,next_attempt_at,last_status,last_error,last_attempt_at,created_at`

func scanWebhook(row interface{ Scan(...any) error }) (Webhook, error) {
	var w Webhook
	var types string
	if err := row.Scan(&w.ID, &w.URL, &w.Secret, &types, &w.Enabled, &w.Cursor, &w.Revision, &w.Failures, &w.NextAttemptAt, &w.LastStatus, &w.LastError, &w.LastAttemptAt, &w.CreatedAt); err != nil {
		return w, err
	}
	err := json.Unmarshal([]byte(types), &w.Types)
	return w, err
}

var webhookType = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,79}$`)

func normalizeTypes(types []string) []string {
	seen := map[string]bool{}
	out := []string{}
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

func validateWebhook(w *Webhook) error {
	w.URL = strings.TrimSpace(w.URL)
	if err := hookurl.Validate(w.URL); err != nil {
		return err
	}
	w.Types = normalizeTypes(w.Types)
	if len(w.Types) == 0 || len(w.Types) > 32 {
		return errors.New("Choose between 1 and 32 event prefixes.")
	}
	for _, t := range w.Types {
		if !webhookType.MatchString(t) || strings.HasPrefix(t, "webhook") {
			return errors.New("Use event prefixes such as agent. or inbox.; webhook events are internal.")
		}
	}
	return nil
}

func webhookSecret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s *Store) AddWebhook(url string, types []string) (Webhook, error) {
	w := Webhook{URL: url, Types: types, Enabled: true, Revision: 1, CreatedAt: nowUTC()}
	if err := validateWebhook(&w); err != nil {
		return Webhook{}, err
	}
	var err error
	w.Secret, err = webhookSecret()
	if err != nil {
		return Webhook{}, err
	}
	// IDs never derive from URLs, whose paths/queries may contain credentials.
	w.ID = newID("subscription", "wh")
	tx, err := s.db.Begin()
	if err != nil {
		return Webhook{}, err
	}
	defer s.rollback(tx)
	var count int
	if err := tx.QueryRow(`SELECT count(*) FROM webhook_subscriptions`).Scan(&count); err != nil {
		return Webhook{}, err
	}
	if count >= 32 {
		return Webhook{}, errors.New("Remove a webhook before adding another (limit 32).")
	}
	if err := tx.QueryRow(`SELECT COALESCE(MAX(id),0) FROM events`).Scan(&w.Cursor); err != nil {
		return Webhook{}, err
	}
	tj, _ := json.Marshal(w.Types)
	if _, err := tx.Exec(`INSERT INTO webhook_subscriptions (`+webhookCols+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, w.ID, w.URL, w.Secret, string(tj), w.Enabled, w.Cursor, w.Revision, 0, "", "", "", "", w.CreatedAt); err != nil {
		return Webhook{}, err
	}
	if err := s.AppendEventTx(tx, "webhook.created", nil, nil, idData(w.ID)); err != nil {
		return Webhook{}, err
	}
	return w, s.commit(tx)
}

func (s *Store) UpdateWebhook(w Webhook) (Webhook, error) {
	if err := validateWebhook(&w); err != nil {
		return Webhook{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Webhook{}, err
	}
	defer s.rollback(tx)
	tj, _ := json.Marshal(w.Types)
	res, err := tx.Exec(`UPDATE webhook_subscriptions SET url=?,types=?,enabled=?,revision=revision+1,failures=0,next_attempt_at='',last_status='',last_error='',last_attempt_at='' WHERE id=? AND revision=?`, w.URL, string(tj), w.Enabled, w.ID, w.Revision)
	if err := webhookChanged(res, err); err != nil {
		return Webhook{}, err
	}
	if err := s.AppendEventTx(tx, "webhook.updated", nil, nil, idData(w.ID)); err != nil {
		return Webhook{}, err
	}
	if err := s.commit(tx); err != nil {
		return Webhook{}, err
	}
	return s.GetWebhook(w.ID)
}

func webhookChanged(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrWebhookConflict
	}
	return nil
}

func (s *Store) DeleteWebhook(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`DELETE FROM webhook_subscriptions WHERE id=?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err := s.AppendEventTx(tx, "webhook.deleted", nil, nil, idData(id)); err != nil {
		return err
	}
	return s.commit(tx)
}

func (s *Store) RotateWebhookSecret(id string, revision int64) (Webhook, error) {
	secret, err := webhookSecret()
	if err != nil {
		return Webhook{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Webhook{}, err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`UPDATE webhook_subscriptions SET secret=?,revision=revision+1,failures=0,next_attempt_at='',last_status='',last_error='',last_attempt_at='' WHERE id=? AND revision=?`, secret, id, revision)
	if err := webhookChanged(res, err); err != nil {
		return Webhook{}, err
	}
	if err := s.AppendEventTx(tx, "webhook.updated", nil, nil, idData(id)); err != nil {
		return Webhook{}, err
	}
	if err := s.commit(tx); err != nil {
		return Webhook{}, err
	}
	return s.GetWebhook(id)
}

// SaveWebhookProgress acknowledges the exact configuration and starting cursor
// the worker read. Edits, pauses, rotation or deletion invalidate an old POST's
// acknowledgement. An attempt/status change emits once; scan-only advancement
// deliberately does not, otherwise skipping webhook.* would sustain itself.
func (s *Store) SaveWebhookProgress(before, after Webhook) error {
	if after.Cursor < before.Cursor {
		return errors.New("Webhook cursor cannot move backwards.")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer s.rollback(tx)
	res, err := tx.Exec(`UPDATE webhook_subscriptions SET cursor=?,failures=?,next_attempt_at=?,last_status=?,last_error=?,last_attempt_at=? WHERE id=? AND revision=? AND cursor=?`, after.Cursor, after.Failures, after.NextAttemptAt, after.LastStatus, after.LastError, after.LastAttemptAt, before.ID, before.Revision, before.Cursor)
	if err := webhookChanged(res, err); err != nil {
		return err
	}
	if before.LastAttemptAt != after.LastAttemptAt || before.LastStatus != after.LastStatus || before.LastError != after.LastError {
		if err := s.AppendEventTx(tx, "webhook.delivery", nil, nil, map[string]any{"id": before.ID, "status": after.LastStatus, "error": after.LastError, "cursor": after.Cursor}); err != nil {
			return err
		}
	}
	return s.commit(tx)
}

func (s *Store) GetWebhook(id string) (Webhook, error) {
	w, err := scanWebhook(s.db.QueryRow(`SELECT `+webhookCols+` FROM webhook_subscriptions WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Webhook{}, ErrNotFound
	}
	return w, err
}

func (s *Store) ListWebhooks() ([]Webhook, error) {
	rows, err := s.db.Query(`SELECT ` + webhookCols + ` FROM webhook_subscriptions ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Webhook{}
	for rows.Next() {
		w, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
