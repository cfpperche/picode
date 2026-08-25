package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Account is one saved login for a provider (no secret material).
type Account struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Type   string `json:"type"`
	Active bool   `json:"active"`
}

type vaultFile map[string]vaultProvider

type vaultProvider struct {
	Active   string         `json:"active"`
	Accounts []vaultAccount `json:"accounts"`
}

type vaultAccount struct {
	ID    string          `json:"id"`
	Label string          `json:"label"`
	Type  string          `json:"type"`
	FP    string          `json:"fp"`
	Cred  json.RawMessage `json:"cred"`
}

func vaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".picode", "accounts.json")
}

func peekCred(provider string) json.RawMessage {
	raw, err := os.ReadFile(AuthPath())
	if err != nil {
		return nil
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return nil
	}
	return obj[provider]
}

func credType(raw json.RawMessage) string {
	var m struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &m) != nil || m.Type == "" {
		return LoginAPIKey
	}
	return m.Type
}

func fingerprint(raw json.RawMessage) string {
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	s := string(raw)
	if id, ok := m["accountId"].(string); ok && id != "" {
		s = "id:" + id
	} else if k, ok := m["key"].(string); ok && k != "" {
		s = "k:" + k
	} else if t, _ := m["type"].(string); t == "oauth" {
		// Refresh tokens rotate. One oauth slot per provider unless accountId exists (Codex).
		s = "oauth"
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func loadVault() vaultFile {
	v := vaultFile{}
	b, err := os.ReadFile(vaultPath())
	if err != nil {
		return v
	}
	_ = json.Unmarshal(b, &v)
	if v == nil {
		v = vaultFile{}
	}
	changed := false
	for id, slot := range v {
		n, c := collapse(slot)
		if c {
			v[id] = n
			changed = true
		}
	}
	if changed {
		_ = saveVault(v)
	}
	return v
}

func collapse(slot vaultProvider) (vaultProvider, bool) {
	byFP := map[string]int{}
	keep := make([]vaultAccount, 0, len(slot.Accounts))
	changed := false
	for _, a := range slot.Accounts {
		fp := fingerprint(a.Cred)
		if fp != a.FP {
			a.FP = fp
			changed = true
		}
		if i, ok := byFP[fp]; ok {
			keep[i] = a // later tokens win
			changed = true
			continue
		}
		byFP[fp] = len(keep)
		keep = append(keep, a)
	}
	slot.Accounts = keep
	if slot.Active != "" {
		found := false
		for _, a := range keep {
			if a.ID == slot.Active {
				found = true
				break
			}
		}
		if !found && len(keep) > 0 {
			slot.Active = keep[len(keep)-1].ID
			changed = true
		}
	}
	return slot, changed
}

func saveVault(v vaultFile) error {
	path := vaultPath()
	if path == "" {
		return fmt.Errorf("no home directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o600)
}

func nextLabel(n int) string {
	if n == 0 {
		return "Default"
	}
	return fmt.Sprintf("Account %d", n+1)
}

func remember(provider string, old, next json.RawMessage) error {
	if len(next) == 0 {
		return nil
	}
	v := loadVault()
	slot := v[provider]
	add := func(raw json.RawMessage) string {
		if len(raw) == 0 {
			return ""
		}
		fp := fingerprint(raw)
		for i, a := range slot.Accounts {
			if a.FP == fp {
				slot.Accounts[i].Cred = raw
				slot.Accounts[i].Type = credType(raw)
				return a.ID
			}
		}
		id := fp[:12]
		slot.Accounts = append(slot.Accounts, vaultAccount{
			ID: id, Label: nextLabel(len(slot.Accounts)), Type: credType(raw), FP: fp, Cred: raw,
		})
		return id
	}
	if len(old) > 0 && fingerprint(old) != fingerprint(next) {
		add(old)
	}
	slot.Active = add(next)
	v[provider] = slot
	return saveVault(v)
}

// AccountsFor returns saved logins for a provider (secrets stripped).
func AccountsFor(provider string) []Account {
	syncFromAuth()
	return accountsOf(provider)
}

func accountsOf(provider string) []Account {
	slot := loadVault()[provider]
	out := make([]Account, 0, len(slot.Accounts))
	for _, a := range slot.Accounts {
		out = append(out, Account{ID: a.ID, Label: a.Label, Type: a.Type, Active: a.ID == slot.Active})
	}
	return out
}

func syncFromAuth() {
	raw, err := os.ReadFile(AuthPath())
	if err != nil {
		return
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return
	}
	v := loadVault()
	changed := false
	for id, cred := range obj {
		if len(cred) == 0 {
			continue
		}
		slot := v[id]
		fp := fingerprint(cred)
		found := ""
		for i, a := range slot.Accounts {
			if a.FP == fp {
				found = a.ID
				if string(a.Cred) != string(cred) {
					slot.Accounts[i].Cred = cred
					changed = true
				}
				break
			}
		}
		if found == "" {
			found = fp[:12]
			slot.Accounts = append(slot.Accounts, vaultAccount{
				ID: found, Label: nextLabel(len(slot.Accounts)), Type: credType(cred), FP: fp, Cred: cred,
			})
			changed = true
		}
		if slot.Active == "" || slot.Active != found {
			// auth.json is source of "what pi uses now"
			slot.Active = found
			changed = true
		}
		v[id] = slot
	}
	if changed {
		_ = saveVault(v)
	}
}

// ActivateAccount writes that saved cred into auth.json (pi's one slot).
func ActivateAccount(provider, accountID string) error {
	provider = strings.TrimSpace(provider)
	slot := loadVault()[provider]
	var cred json.RawMessage
	for _, a := range slot.Accounts {
		if a.ID == accountID {
			cred = a.Cred
			break
		}
	}
	if len(cred) == 0 {
		return fmt.Errorf("unknown account")
	}
	if err := mutateAuth(func(obj map[string]json.RawMessage) error {
		obj[provider] = cred
		return nil
	}); err != nil {
		return err
	}
	slot.Active = accountID
	v := loadVault()
	v[provider] = slot
	return saveVault(v)
}

// RemoveAccount drops one saved login. If it was active, another becomes active.
func RemoveAccount(provider, accountID string) error {
	provider = strings.TrimSpace(provider)
	v := loadVault()
	slot := v[provider]
	keep := slot.Accounts[:0]
	removedActive := slot.Active == accountID
	for _, a := range slot.Accounts {
		if a.ID != accountID {
			keep = append(keep, a)
		}
	}
	if len(keep) == len(slot.Accounts) {
		return fmt.Errorf("unknown account")
	}
	slot.Accounts = keep
	if len(keep) == 0 {
		delete(v, provider)
		if err := saveVault(v); err != nil {
			return err
		}
		return mutateAuth(func(obj map[string]json.RawMessage) error {
			delete(obj, provider)
			return nil
		})
	}
	if removedActive {
		slot.Active = keep[0].ID
		if err := mutateAuth(func(obj map[string]json.RawMessage) error {
			obj[provider] = keep[0].Cred
			return nil
		}); err != nil {
			return err
		}
	}
	v[provider] = slot
	return saveVault(v)
}

// RenameAccount sets the display name. Secrets unchanged.
func RenameAccount(provider, accountID, label string) error {
	provider = strings.TrimSpace(provider)
	label = strings.TrimSpace(label)
	if label == "" {
		return fmt.Errorf("name required")
	}
	v := loadVault()
	slot := v[provider]
	for i, a := range slot.Accounts {
		if a.ID == accountID {
			slot.Accounts[i].Label = label
			v[provider] = slot
			return saveVault(v)
		}
	}
	return fmt.Errorf("unknown account")
}

func clearVaultProvider(provider string) {
	v := loadVault()
	if _, ok := v[provider]; !ok {
		return
	}
	delete(v, provider)
	_ = saveVault(v)
}
