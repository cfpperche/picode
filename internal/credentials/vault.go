package credentials

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

func trim(s string) string { return strings.TrimSpace(s) }

// File is the vault's plaintext schema (what the envelope carries).
type File struct {
	Version   int             `json:"version"`
	Providers map[string]Slot `json:"providers"`
}

// Slot is one provider's accounts: the rows, and which one Pi's `auth.json`
// holds (ADR-0013 — pi reads exactly one credential per provider).
type Slot struct {
	Active   string `json:"active,omitempty"`
	Accounts []Row  `json:"accounts"`
}

// Row is one saved account. Cred is the provider-shaped credential JSON the
// CLI or pi reads ({"type":"api_key","key":…} or {"type":"oauth",…}); every
// other field is either the person's (Label) or the vendor's (Email, Plan).
type Row struct {
	ID     string          `json:"id"`
	Label  string          `json:"label"`
	Type   string          `json:"type"`
	FP     string          `json:"fp"`
	Cred   json.RawMessage `json:"cred"`
	Email  string          `json:"email,omitempty"`
	Plan   string          `json:"plan,omitempty"`
	Paused bool            `json:"paused,omitempty"`
	// Origin is where the row came from: "vault" (added here or through pi's
	// own login), "imported:<cli>" (read out of a CLI's own store, ADR-0013's
	// import) or "migrated" (absorbed from accounts.json).
	Origin string `json:"origin,omitempty"`
	// Identity is the vendor's own account identity when the CLI's store
	// carries one (an account id, a principal, an email). It is what lets two
	// subscriptions of one provider be two rows instead of one that keeps
	// being replaced.
	Identity string `json:"identity,omitempty"`
	// Hint is a masked echo of the credential, stored at write time so the
	// roster can say *which* key a row holds without ever returning it.
	Hint string `json:"hint,omitempty"`
	// Health is the last Verify answer. Never a guess: unknown until a check
	// or an import says otherwise.
	Health    *Health `json:"health,omitempty"`
	CreatedAt string  `json:"createdAt,omitempty"`
	LastUsed  string  `json:"lastUsedAt,omitempty"`
}

// Health is a verify outcome with its age.
type Health struct {
	State   string `json:"state"` // ok | invalid | expired | no_credit | rate_limited | unknown
	At      string `json:"at,omitempty"`
	Message string `json:"message,omitempty"`
}

// Removal reports what a delete did to the provider's active slot, so the
// caller can write the right credential into pi's auth.json.
type Removal struct {
	Promoted *Row
	Empty    bool
}

func newFile() File {
	return File{Version: Version, Providers: map[string]Slot{}}
}

// ProviderIDs lists the providers this vault holds, sorted.
func (f File) ProviderIDs() []string {
	out := make([]string, 0, len(f.Providers))
	for id := range f.Providers {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Rows returns a provider's rows (copies; Cred included — the caller strips
// it before anything user-facing).
func (f File) Rows(provider string) []Row {
	slot, ok := f.Providers[provider]
	if !ok {
		return nil
	}
	out := make([]Row, len(slot.Accounts))
	copy(out, slot.Accounts)
	return out
}

// Find locates one row.
func (f File) Find(provider, id string) (Row, bool) {
	for _, r := range f.Providers[provider].Accounts {
		if r.ID == id {
			return r, true
		}
	}
	return Row{}, false
}

// Fingerprint identifies a credential for de-duplication: the account id when
// the vendor gives one (Codex), else the key itself, else a constant for
// oauth rows (their refresh tokens rotate, so the stored bytes are not an
// identity — the same trap ADR-0013 documented).
func Fingerprint(raw json.RawMessage) string {
	sum := sha256.Sum256([]byte(tag(raw)))
	return hex.EncodeToString(sum[:])
}

// tag is what a credential is identified by before hashing: the vendor's own
// account id when there is one (Codex), else the key itself, else a constant
// for the shapes whose bytes are not an identity — an oauth entry's tokens
// rotate (ADR-0013), and an env-shaped api_key is the same string for the
// whole provider.
func tag(raw json.RawMessage) string {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	if id, ok := m["accountId"].(string); ok && id != "" {
		return "id:" + id
	}
	if k, ok := m["key"].(string); ok && k != "" {
		return "k:" + k
	}
	switch t, _ := m["type"].(string); t {
	case "oauth":
		return "oauth"
	case "api_key":
		// An api_key entry with no `key` of its own — llama.cpp's env-shaped
		// credential — is not identifiable by its bytes: hashing them minted a
		// new row every time the endpoint changed (the pi roster showed two
		// "Account 2" rows, both active, on 2026-09-20).
		return "api_key"
	}
	return string(raw)
}

// Key is the row identity for a credential: the vendor's own account identity
// when the store carries one, else the credential's fingerprint. Two
// subscriptions of a provider whose file has no account name collapse into one
// row, which is the limit this function exists to remove.
func Key(raw json.RawMessage, identity string) string {
	fp := Fingerprint(raw)
	// Only where the bytes cannot tell two accounts apart: an oauth entry with
	// no accountId and an env-shaped api_key are the same string for the whole
	// provider. Everywhere else the fingerprint wins, so rows saved before
	// identities existed keep their ids.
	if id := trim(identity); id != "" && unkeyed(raw) {
		sum := sha256.Sum256([]byte("who:" + id))
		return hex.EncodeToString(sum[:])
	}
	return fp
}

// unkeyed reports the credentials whose own bytes name no account.
func unkeyed(raw json.RawMessage) bool {
	t := tag(raw)
	return t == "oauth" || t == "api_key"
}

// Exact reports whether Key names one account — that is, whether a caller can
// match a live file to a row by the key alone. False only for a credential
// whose bytes say nothing and which nobody has named: the one-per-provider
// case the pane states on screen.
func Exact(raw json.RawMessage, identity string) bool {
	if unkeyed(raw) {
		return trim(identity) != ""
	}
	return true
}

// Adopt gives the unnamed row that holds this credential the name the person
// gave it, instead of adding a row beside it: "name the login that is live".
// A row that is already named, or a credential with no row, falls through to
// Import — the name is never applied to some other account's row.
func (s *Store) Adopt(provider, identity string, cred json.RawMessage) (Row, error) {
	identity = trim(identity)
	if identity == "" {
		return s.Import(provider, cred, "", "vault", "")
	}
	var out Row
	err := s.Update(func(f *File) error {
		slot, ok := f.Providers[provider]
		if !ok {
			return nil
		}
		unnamed := Fingerprint(cred)
		named := Key(cred, identity)
		if unnamed == named {
			return nil
		}
		for i, r := range slot.Accounts {
			if r.FP != unnamed || r.Identity != "" {
				continue
			}
			slot.Accounts[i].FP = named
			slot.Accounts[i].ID = named[:12]
			slot.Accounts[i].Identity = identity
			slot.Accounts[i].Type = Type(cred)
			slot.Accounts[i].Hint = Hint(cred)
			slot.Accounts[i].Cred = cred
			out = slot.Accounts[i]
			f.Providers[provider] = slot
			return nil
		}
		return nil
	})
	if err != nil || out.ID != "" {
		return out, err
	}
	return s.Import(provider, cred, "", "vault", identity)
}

// Type is the credential's kind: "api_key" or "oauth" (the two shapes pi and
// every CLI here use).
func Type(raw json.RawMessage) string {
	var m struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &m) != nil || m.Type == "" {
		return "api_key"
	}
	return m.Type
}

// Hint masks a credential for display: enough head and tail to recognize
// which key it is, never enough to use it. OAuth rows get no hint — a
// subscription has no key-shaped thing to recognize.
func Hint(raw json.RawMessage) string {
	var m struct {
		Type string `json:"type"`
		Key  string `json:"key"`
	}
	if json.Unmarshal(raw, &m) != nil || m.Type == "oauth" || m.Key == "" {
		return ""
	}
	k := m.Key
	if len(k) <= 10 {
		return "••••"
	}
	head := k
	if len(head) > 7 {
		head = head[:7]
	}
	return head + "…" + k[len(k)-4:]
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

// Accounts returns a provider's rows.
func (s *Store) Accounts(provider string) ([]Row, error) {
	f, err := s.Load()
	if err != nil {
		return nil, err
	}
	return f.Rows(provider), nil
}

// Row returns one saved account.
func (s *Store) Row(provider, id string) (Row, bool, error) {
	f, err := s.Load()
	if err != nil {
		return Row{}, false, err
	}
	r, ok := f.Find(provider, id)
	return r, ok, nil
}

// ActiveID is the row pi's auth.json holds for this provider.
func (s *Store) ActiveID(provider string) (string, error) {
	f, err := s.Load()
	if err != nil {
		return "", err
	}
	return f.Providers[provider].Active, nil
}

// Remember saves a credential, keeping the one it replaces (ADR-0013's
// "Add account appends to the vault and makes it active"): same fingerprint
// updates in place, a different one becomes a new row, and the saved
// credential becomes the active slot. It returns the row's id.
func (s *Store) Remember(provider string, old, next json.RawMessage, origin string) (string, error) {
	if len(next) == 0 {
		return "", nil
	}
	id := ""
	err := s.Update(func(f *File) error {
		slot := f.Providers[provider]
		add := func(raw json.RawMessage, from string) string {
			if len(raw) == 0 {
				return ""
			}
			fp := Fingerprint(raw)
			for i, r := range slot.Accounts {
				if r.FP == fp {
					slot.Accounts[i].Cred = raw
					slot.Accounts[i].Type = Type(raw)
					slot.Accounts[i].Hint = Hint(raw)
					return r.ID
				}
			}
			row := Row{
				ID: fp[:12], Label: nextLabel(len(slot.Accounts)), Type: Type(raw), FP: fp,
				Cred: raw, Origin: from, Hint: Hint(raw), CreatedAt: now(),
			}
			slot.Accounts = append(slot.Accounts, row)
			return row.ID
		}
		if len(old) > 0 && Fingerprint(old) != Fingerprint(next) {
			add(old, origin)
		}
		id = add(next, origin)
		slot.Active = id
		f.Providers[provider] = slot
		return nil
	})
	return id, err
}

// Import saves a credential read from somewhere else (a CLI's own store, or
// the pane's key field) without touching any active slot — importing is not
// using. An identical credential updates the row it already belongs to;
// identity, when the store volunteers one, decides what "identical" means.
func (s *Store) Import(provider string, cred json.RawMessage, label, origin, identity string) (Row, error) {
	var out Row
	if len(cred) == 0 {
		return out, fmt.Errorf("credentials: empty credential")
	}
	err := s.Update(func(f *File) error {
		slot := f.Providers[provider]
		fp := Key(cred, identity)
		for i, r := range slot.Accounts {
			if r.FP == fp {
				slot.Accounts[i].Cred = cred
				slot.Accounts[i].Type = Type(cred)
				slot.Accounts[i].Hint = Hint(cred)
				if identity != "" {
					slot.Accounts[i].Identity = trim(identity)
				}
				if label != "" && label != r.Label {
					slot.Accounts[i].Label = label
				}
				out = slot.Accounts[i]
				f.Providers[provider] = slot
				return nil
			}
		}
		row := Row{
			ID: fp[:12], Label: label, Type: Type(cred), FP: fp,
			Cred: cred, Origin: origin, Hint: Hint(cred), CreatedAt: now(),
			Identity: trim(identity),
			Health:   &Health{State: "unknown", At: now()},
		}
		if row.Label == "" {
			row.Label = nextLabel(len(slot.Accounts))
		}
		slot.Accounts = append(slot.Accounts, row)
		f.Providers[provider] = slot
		out = row
		return nil
	})
	return out, err
}

// SetActive makes a saved row the one pi reads, refusing a paused row.
func (s *Store) SetActive(provider, id string) error {
	return s.Update(func(f *File) error {
		slot := f.Providers[provider]
		for _, r := range slot.Accounts {
			if r.ID != id {
				continue
			}
			if r.Paused {
				return fmt.Errorf("account is paused — resume it first")
			}
			slot.Active = id
			f.Providers[provider] = slot
			return nil
		}
		return fmt.Errorf("unknown account")
	})
}

// Rename sets the display name. Secrets unchanged.
func (s *Store) Rename(provider, id, label string) error {
	label = trim(label)
	if label == "" {
		return fmt.Errorf("name required")
	}
	return s.Update(func(f *File) error {
		slot := f.Providers[provider]
		for i, r := range slot.Accounts {
			if r.ID == id {
				slot.Accounts[i].Label = label
				f.Providers[provider] = slot
				return nil
			}
		}
		return fmt.Errorf("unknown account")
	})
}

// Pause keeps the credential and takes the row out of play. Pausing the
// active row promotes another live one (returned so the caller can write it
// to pi's auth.json); pausing the only live row is refused, because that is
// Sign out with extra steps.
func (s *Store) Pause(provider, id string, paused bool) (*Row, error) {
	var promoted *Row
	err := s.Update(func(f *File) error {
		slot, ok := f.Providers[provider]
		if !ok {
			return fmt.Errorf("unknown provider")
		}
		idx := -1
		live := 0
		for i, r := range slot.Accounts {
			if r.ID == id {
				idx = i
			}
			if !r.Paused {
				live++
			}
		}
		if idx < 0 {
			return fmt.Errorf("unknown account")
		}
		if slot.Accounts[idx].Paused == paused {
			return nil
		}
		if paused && live <= 1 {
			return fmt.Errorf("this is the only live account — sign out instead")
		}
		slot.Accounts[idx].Paused = paused
		if paused && slot.Active == id {
			for _, r := range slot.Accounts {
				if r.ID == id || r.Paused {
					continue
				}
				copy := r
				promoted = &copy
				slot.Active = r.ID
				break
			}
		}
		f.Providers[provider] = slot
		return nil
	})
	return promoted, err
}

// Remove drops one saved account and reports what happened to the slot.
func (s *Store) Remove(provider, id string) (Removal, error) {
	out := Removal{}
	err := s.Update(func(f *File) error {
		slot, ok := f.Providers[provider]
		if !ok {
			return fmt.Errorf("unknown account")
		}
		keep := make([]Row, 0, len(slot.Accounts))
		removedActive := slot.Active == id
		found := false
		for _, r := range slot.Accounts {
			if r.ID == id {
				found = true
				continue
			}
			keep = append(keep, r)
		}
		if !found {
			return fmt.Errorf("unknown account")
		}
		if len(keep) == 0 {
			delete(f.Providers, provider)
			out.Empty = true
			return nil
		}
		slot.Accounts = keep
		if removedActive {
			slot.Active = keep[0].ID
			copy := keep[0]
			out.Promoted = &copy
		}
		f.Providers[provider] = slot
		return nil
	})
	return out, err
}

// DeleteProvider drops every row of one provider (Sign out of a custom
// provider definition removes its credential too).
func (s *Store) DeleteProvider(provider string) error {
	return s.Update(func(f *File) error {
		delete(f.Providers, provider)
		return nil
	})
}

// SetIdentity records what the vendor said this login is. A blank value
// leaves the stored one alone, so an adapter that cannot answer does not
// erase what another already learned.
func (s *Store) SetIdentity(provider, id, email, plan string) error {
	email, plan = trim(email), trim(plan)
	if email == "" && plan == "" {
		return nil
	}
	return s.Update(func(f *File) error {
		slot, ok := f.Providers[provider]
		if !ok {
			return nil
		}
		for i, r := range slot.Accounts {
			if r.ID != id {
				continue
			}
			if email != "" {
				slot.Accounts[i].Email = email
			}
			if plan != "" {
				slot.Accounts[i].Plan = plan
			}
			break
		}
		f.Providers[provider] = slot
		return nil
	})
}

// SetHealth stores a verify outcome on one row.
func (s *Store) SetHealth(provider, id, state, message string) error {
	state = trim(state)
	if state == "" {
		return nil
	}
	return s.Update(func(f *File) error {
		slot, ok := f.Providers[provider]
		if !ok {
			return nil
		}
		for i, r := range slot.Accounts {
			if r.ID != id {
				continue
			}
			slot.Accounts[i].Health = &Health{State: state, At: now(), Message: trim(message)}
			break
		}
		f.Providers[provider] = slot
		return nil
	})
}

// UpdateTokens writes new OAuth tokens onto one row, keeping every field the
// caller does not set (accountId, scopes). It reports whether the row is the
// active slot, so the caller can refresh pi's auth.json in step.
func (s *Store) UpdateTokens(provider, id, access, refresh string, expires int64) (bool, error) {
	active := false
	err := s.Update(func(f *File) error {
		slot, ok := f.Providers[provider]
		if !ok {
			return nil
		}
		for i, r := range slot.Accounts {
			if r.ID != id {
				continue
			}
			merged, err := mergeOAuth(r.Cred, access, refresh, expires)
			if err != nil {
				return err
			}
			slot.Accounts[i].Cred = merged
			slot.Accounts[i].Type = Type(merged)
			slot.Accounts[i].Hint = Hint(merged)
			active = slot.Active == id
			break
		}
		f.Providers[provider] = slot
		return nil
	})
	return active, err
}

func mergeOAuth(raw json.RawMessage, access, refresh string, expires int64) (json.RawMessage, error) {
	obj := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil, err
		}
	}
	obj["type"] = "oauth"
	if access != "" {
		obj["access"] = access
	}
	if refresh != "" {
		obj["refresh"] = refresh
	}
	if expires != 0 {
		obj["expires"] = expires
	}
	return json.Marshal(obj)
}

// SyncEntries absorbs a provider→credential map (pi's auth.json) into the
// vault: each entry becomes or updates a row and becomes the active slot,
// because that file is the truth about what pi is using right now.
func (s *Store) SyncEntries(entries map[string]json.RawMessage, origin string) error {
	if len(entries) == 0 {
		return nil
	}
	return s.Update(func(f *File) error {
		for provider, cred := range entries {
			if len(cred) == 0 {
				continue
			}
			slot := f.Providers[provider]
			fp := Fingerprint(cred)
			found := ""
			for i, r := range slot.Accounts {
				if r.FP == fp {
					found = r.ID
					if string(r.Cred) != string(cred) {
						slot.Accounts[i].Cred = cred
						slot.Accounts[i].Type = Type(cred)
						slot.Accounts[i].Hint = Hint(cred)
					}
					break
				}
			}
			if found == "" {
				row := Row{
					ID: fp[:12], Label: nextLabel(len(slot.Accounts)), Type: Type(cred),
					FP: fp, Cred: cred, Origin: origin, Hint: Hint(cred), CreatedAt: now(),
				}
				slot.Accounts = append(slot.Accounts, row)
				found = row.ID
			}
			slot.Active = found
			f.Providers[provider] = slot
		}
		return nil
	})
}

func nextLabel(n int) string {
	if n == 0 {
		return "Default"
	}
	return fmt.Sprintf("Account %d", n+1)
}
