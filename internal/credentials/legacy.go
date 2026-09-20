package credentials

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// legacyName is ADR-0013's vault: a plaintext file PiCode wrote until this
// package existed. It is read once, absorbed, and never written or deleted —
// a person's file is not ours to remove, and a backup restored from an older
// release carries one, which is exactly how a restore keeps working.
const legacyName = "accounts.json"

type legacyFile map[string]legacySlot

type legacySlot struct {
	Active   string          `json:"active"`
	Accounts []legacyAccount `json:"accounts"`
}

type legacyAccount struct {
	ID     string          `json:"id"`
	Label  string          `json:"label"`
	Type   string          `json:"type"`
	FP     string          `json:"fp"`
	Email  string          `json:"email,omitempty"`
	Plan   string          `json:"plan,omitempty"`
	Paused bool            `json:"paused,omitempty"`
	Cred   json.RawMessage `json:"cred"`
}

// readLegacy converts the plaintext vault when the encrypted one does not
// exist yet. A file that does not parse is ignored (the old release's writer
// always produced valid JSON; anything else was hand-edited) and stays on
// disk untouched.
func (s *Store) readLegacy() (File, bool) {
	raw, err := os.ReadFile(filepath.Join(s.dir, legacyName))
	if err != nil {
		return File{}, false
	}
	var old legacyFile
	if json.Unmarshal(raw, &old) != nil || len(old) == 0 {
		return File{}, false
	}
	f := newFile()
	for provider, slot := range old {
		out := Slot{Active: slot.Active}
		for _, a := range slot.Accounts {
			if len(a.Cred) == 0 {
				continue
			}
			fp := a.FP
			if fp == "" {
				fp = Fingerprint(a.Cred)
			}
			id := a.ID
			if id == "" {
				id = fp[:12]
			}
			out.Accounts = append(out.Accounts, Row{
				ID: id, Label: a.Label, Type: a.Type, FP: fp, Cred: a.Cred,
				Email: a.Email, Plan: a.Plan, Paused: a.Paused,
				Origin: "migrated", Hint: Hint(a.Cred),
			})
		}
		if len(out.Accounts) == 0 {
			continue
		}
		if out.Active == "" {
			out.Active = out.Accounts[0].ID
		}
		f.Providers[provider] = out
	}
	if len(f.Providers) == 0 {
		return File{}, false
	}
	return f, true
}
