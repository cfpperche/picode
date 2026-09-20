package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cfpperche/picode/internal/credentials"
)

// Account is one saved login for a provider (no secret material).
type Account struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Type      string `json:"type"`
	Active    bool   `json:"active"`
	QuotaKind string `json:"quotaKind,omitempty"` // oauth | api_key when this row can fetch Usage
	Email     string `json:"email,omitempty"`     // learned from the vendor, never typed
	Plan      string `json:"plan,omitempty"`      // learned from the vendor, never typed
	Paused    bool   `json:"paused,omitempty"`    // kept, but not offered to agents
	// Origin is where the row came from: "vault" (added here or through pi's
	// own login), "imported:<cli>" (read out of a CLI's own store) or
	// "migrated" (absorbed from ADR-0013's accounts.json).
	Origin string `json:"origin,omitempty"`
	// Hint is the masked echo of the stored key (empty for a subscription),
	// so the roster can say which key a row holds without returning it.
	Hint string `json:"hint,omitempty"`
	// Health is the last Verify answer with its age; nil until one runs.
	Health *credentials.Health `json:"health,omitempty"`
}

// vault is the one credential store (ADR-0165): every provider account for pi
// and for the guest CLIs lives in it.
func vault() *credentials.Store { return credentials.Default() }

// AccountsFor returns saved logins for a provider (secrets stripped).
func AccountsFor(provider string) []Account {
	syncFromAuth()
	return accountsOf(provider)
}

func accountsOf(provider string) []Account {
	f, err := vault().Load()
	if err != nil {
		return nil
	}
	active := f.Providers[provider].Active
	rows := f.Rows(provider)
	out := make([]Account, 0, len(rows))
	for _, r := range rows {
		out = append(out, Account{
			ID: r.ID, Label: r.Label, Type: r.Type, Active: r.ID == active,
			QuotaKind: QuotaKind(provider, r.Type),
			Email:     r.Email, Plan: r.Plan, Paused: r.Paused,
			Origin: r.Origin, Hint: r.Hint, Health: r.Health,
		})
	}
	return out
}

// rememberAuth records a credential pi's auth.json now holds: the one it
// replaced stays as a row (ADR-0013), and the new one becomes the active slot.
func rememberAuth(provider string, old, next json.RawMessage) error {
	_, err := vault().Remember(strings.TrimSpace(provider), old, next, "vault")
	return err
}

// syncFromAuth absorbs a login made in pi's own TUI: whatever auth.json holds
// is a vault row, and it is what pi is using right now (ADR-0013).
func syncFromAuth() {
	raw, err := os.ReadFile(AuthPath())
	if err != nil {
		return
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return
	}
	_ = vault().SyncEntries(obj, "vault")
}

// ActivateAccount writes that saved cred into auth.json (pi's one slot).
func ActivateAccount(provider, accountID string) error {
	provider = strings.TrimSpace(provider)
	row, ok, err := vault().Row(provider, accountID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("unknown account")
	}
	if row.Paused {
		return fmt.Errorf("account is paused — resume it first")
	}
	if err := mutateAuth(func(obj map[string]json.RawMessage) error {
		obj[provider] = row.Cred
		return nil
	}); err != nil {
		return err
	}
	return vault().SetActive(provider, accountID)
}

// RemoveAccount drops one saved login. If it was active, another becomes active.
func RemoveAccount(provider, accountID string) error {
	provider = strings.TrimSpace(provider)
	rm, err := vault().Remove(provider, accountID)
	if err != nil {
		return err
	}
	if rm.Empty {
		return mutateAuth(func(obj map[string]json.RawMessage) error {
			delete(obj, provider)
			return nil
		})
	}
	if rm.Promoted != nil {
		cred := rm.Promoted.Cred
		return mutateAuth(func(obj map[string]json.RawMessage) error {
			obj[provider] = cred
			return nil
		})
	}
	return nil
}

// RenameAccount sets the display name. Secrets unchanged.
func RenameAccount(provider, accountID, label string) error {
	return vault().Rename(strings.TrimSpace(provider), accountID, label)
}

func clearVaultProvider(provider string) {
	_ = vault().DeleteProvider(strings.TrimSpace(provider))
}

// SetAccountIdentity records what the vendor said this login is (email,
// plan). Both are read from the provider's own API by internal/usage and
// are never typed by a person — the label stays the user's alias. A blank
// value leaves the stored one alone, so one adapter that cannot answer
// does not erase what another already learned.
func SetAccountIdentity(provider, accountID, email, plan string) error {
	return vault().SetIdentity(strings.TrimSpace(provider), strings.TrimSpace(accountID), email, plan)
}

// PauseAccount keeps the credential but takes the row out of play: it stops
// being offered to agents. Pausing the active row promotes another live one,
// the way RemoveAccount does; pausing the only row is refused, because that
// is Sign out with extra steps.
func PauseAccount(provider, accountID string, paused bool) error {
	provider = strings.TrimSpace(provider)
	promoted, err := vault().Pause(provider, accountID, paused)
	if err != nil || promoted == nil {
		return err
	}
	cred := promoted.Cred
	return mutateAuth(func(obj map[string]json.RawMessage) error {
		obj[provider] = cred
		return nil
	})
}
