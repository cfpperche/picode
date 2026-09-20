package credentials

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func keyCred(key string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{"type": "api_key", "key": key})
	return raw
}

func oauthCred(access, refresh string, expires int64, accountID string) json.RawMessage {
	m := map[string]any{"type": "oauth", "access": access, "refresh": refresh, "expires": expires}
	if accountID != "" {
		m["accountId"] = accountID
	}
	raw, _ := json.Marshal(m)
	return raw
}

func TestVaultIsEncryptedAndHoldsRows(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	const secret = "sk-ant-supersecret-0123456789"
	if _, err := s.Remember("anthropic", nil, keyCred(secret), "vault"); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatalf("vault file: %v", err)
	}
	if strings.Contains(string(raw), secret) || strings.Contains(string(raw), "sk-ant") {
		t.Fatal("the vault file carries the key in the clear")
	}
	if err := json.Unmarshal(raw, &envelope{}); err != nil {
		t.Fatalf("vault file is not an envelope: %v", err)
	}
	fi, err := os.Stat(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Fatalf("vault mode = %o, want 600", perm)
	}
	key, err := os.Stat(filepath.Join(dir, keyName))
	if err != nil {
		t.Fatalf("key file: %v", err)
	}
	if perm := key.Mode().Perm(); perm != 0o600 {
		t.Fatalf("key mode = %o, want 600", perm)
	}
	if key.Size() != 32 {
		t.Fatalf("key size = %d, want 32", key.Size())
	}

	rows, err := s.Accounts("anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Label != "Default" || rows[0].Type != "api_key" {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].Hint != "sk-ant-…6789" {
		t.Fatalf("hint = %q, want the masked key", rows[0].Hint)
	}
	if !strings.Contains(string(rows[0].Cred), secret) {
		t.Fatal("the row lost its credential")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".credentials-") {
			t.Fatalf("temp file left behind: %s", e.Name())
		}
	}
}

func TestVaultRefusesWhatItCannotRead(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if _, err := s.Remember("openai", nil, keyCred("sk-live-abcd"), "vault"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}

	// A flipped byte inside the ciphertext is a corrupt vault, not a new one.
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	data := []byte(env.Data)
	data[len(data)/2] ^= 0x01
	env.Data = string(data)
	tampered, _ := json.Marshal(env)
	if err := os.WriteFile(s.Path(), tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Load after tamper = %v, want ErrCorrupt", err)
	}
	if _, err := s.Remember("openai", nil, keyCred("sk-new"), "vault"); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("a write over a corrupt vault = %v, want the refusal to keep its bytes", err)
	}

	// A missing key beside an existing vault is a state, not an invitation to
	// mint a new one (that would orphan the rows).
	if err := os.WriteFile(s.Path(), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, keyName)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); !errors.Is(err, ErrLocked) {
		t.Fatalf("Load without a key = %v, want ErrLocked", err)
	}
}

func TestLegacyVaultIsAbsorbedOnce(t *testing.T) {
	dir := t.TempDir()
	legacy := map[string]any{
		"anthropic": map[string]any{
			"active": "row1",
			"accounts": []map[string]any{
				{"id": "row1", "label": "Work", "type": "oauth", "cred": map[string]any{"type": "oauth", "access": "a", "refresh": "r", "expires": 11}},
				{"id": "row2", "label": "Home", "type": "api_key", "cred": map[string]any{"type": "api_key", "key": "sk-home-1234"}},
			},
		},
	}
	raw, _ := json.MarshalIndent(legacy, "", "  ")
	path := filepath.Join(dir, legacyName)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	s := New(dir)
	rows, err := s.Accounts("anthropic")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want the legacy file absorbed", len(rows))
	}
	if rows[0].Origin != "migrated" || rows[0].Label != "Work" {
		t.Fatalf("row = %+v, want the migrated label and origin", rows[0])
	}
	active, err := s.ActiveID("anthropic")
	if err != nil || active != "row1" {
		t.Fatalf("active = %q (%v), want the legacy active row", active, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the legacy file was removed: %v", err)
	}

	// A second store sees the encrypted file; the legacy one is not re-read.
	other := New(dir)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	rows, err = other.Accounts("anthropic")
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows after removing the legacy file = %d (%v), want the vault's own copy", len(rows), err)
	}
}

func TestRememberKeepsTheCredentialItReplaces(t *testing.T) {
	s := New(t.TempDir())
	first := keyCred("sk-one")
	second := keyCred("sk-two")
	if _, err := s.Remember("xai", nil, first, "vault"); err != nil {
		t.Fatal(err)
	}
	id, err := s.Remember("xai", first, second, "vault")
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := s.Accounts("xai")
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want the replaced credential kept", len(rows))
	}
	active, _ := s.ActiveID("xai")
	if active != id {
		t.Fatalf("active = %q, want the new row %q", active, id)
	}
}

func TestPauseAndRemoveRules(t *testing.T) {
	s := New(t.TempDir())
	a, _ := s.Remember("anthropic", nil, keyCred("sk-a"), "vault")

	// Pausing the only live row is sign-out with extra steps.
	if _, err := s.Pause("anthropic", a, true); err == nil {
		t.Fatal("paused the only live row")
	}
	b, _ := s.Remember("anthropic", keyCred("sk-a"), keyCred("sk-b"), "vault")
	// Pausing the active row promotes the other one.
	promoted, err := s.Pause("anthropic", b, true)
	if err != nil {
		t.Fatal(err)
	}
	if promoted == nil || promoted.ID != a {
		t.Fatalf("promoted = %+v, want row %s", promoted, a)
	}
	active, _ := s.ActiveID("anthropic")
	if active != a {
		t.Fatalf("active = %q, want the promoted row", active)
	}
	// A paused row cannot be activated.
	if err := s.SetActive("anthropic", b); err == nil {
		t.Fatal("activated a paused row")
	}
	// Removing the active row promotes the remaining one; removing the last
	// row empties the provider.
	rm, err := s.Remove("anthropic", a)
	if err != nil {
		t.Fatal(err)
	}
	if rm.Empty || rm.Promoted == nil || rm.Promoted.ID != b {
		t.Fatalf("removal = %+v, want the paused row promoted", rm)
	}
	rm, err = s.Remove("anthropic", b)
	if err != nil {
		t.Fatal(err)
	}
	if !rm.Empty || rm.Promoted != nil {
		t.Fatalf("removal = %+v, want the provider emptied", rm)
	}
	if rows, _ := s.Accounts("anthropic"); len(rows) != 0 {
		t.Fatalf("rows = %d after removing everything", len(rows))
	}
}

func TestUpdateTokensKeepsExtraFieldsAndReportsActive(t *testing.T) {
	s := New(t.TempDir())
	id, err := s.Remember("openai-codex", nil, oauthCred("a1", "r1", 10, "acct-9"), "vault")
	if err != nil {
		t.Fatal(err)
	}
	active, err := s.UpdateTokens("openai-codex", id, "a2", "r2", 20)
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("the row reported inactive though it is the active slot")
	}
	row, ok, _ := s.Row("openai-codex", id)
	if !ok {
		t.Fatal("row vanished")
	}
	var m map[string]any
	if err := json.Unmarshal(row.Cred, &m); err != nil {
		t.Fatal(err)
	}
	if m["access"] != "a2" || m["refresh"] != "r2" || m["expires"].(float64) != 20 {
		t.Fatalf("tokens = %v", m)
	}
	if m["accountId"] != "acct-9" {
		t.Fatalf("accountId lost: %v", m)
	}
	if row.Hint != "" {
		t.Fatalf("hint on an oauth row = %q, want none", row.Hint)
	}
}

func TestImportDoesNotTakeOverTheSlot(t *testing.T) {
	s := New(t.TempDir())
	row, err := s.Import("anthropic", keyCred("sk-imported"), "From Claude Code", "imported:claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if row.Origin != "imported:claude-code" || row.Label != "From Claude Code" {
		t.Fatalf("row = %+v", row)
	}
	if active, _ := s.ActiveID("anthropic"); active != "" {
		t.Fatalf("active = %q, want nothing — importing is not using", active)
	}
	// The same credential imported twice updates one row.
	again, err := s.Import("anthropic", keyCred("sk-imported"), "", "imported:claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != row.ID {
		t.Fatalf("ids = %q and %q, want the same row", again.ID, row.ID)
	}
	if rows, _ := s.Accounts("anthropic"); len(rows) != 1 {
		t.Fatalf("rows = %d, want one", len(rows))
	}
}

func TestSyncEntriesAndIdentity(t *testing.T) {
	s := New(t.TempDir())
	entries := map[string]json.RawMessage{
		"anthropic":  oauthCred("a", "r", 5, ""),
		"llama.cpp":  keyCred("local-key"),
		"openrouter": keyCred("sk-or-abcdefgh"),
	}
	if err := s.SyncEntries(entries, "vault"); err != nil {
		t.Fatal(err)
	}
	ids := New(t.TempDir())
	_ = ids
	for provider := range entries {
		active, err := s.ActiveID(provider)
		if err != nil || active == "" {
			t.Fatalf("%s active = %q (%v), want the synced row", provider, active, err)
		}
	}
	active, _ := s.ActiveID("anthropic")
	if err := s.SetIdentity("anthropic", active, "me@work.com", "Max 5x"); err != nil {
		t.Fatal(err)
	}
	// A blank pair leaves what a previous adapter learned alone.
	if err := s.SetIdentity("anthropic", active, "", ""); err != nil {
		t.Fatal(err)
	}
	row, _, _ := s.Row("anthropic", active)
	if row.Email != "me@work.com" || row.Plan != "Max 5x" {
		t.Fatalf("identity = %q %q", row.Email, row.Plan)
	}
	if err := s.SetHealth("anthropic", active, "ok", ""); err != nil {
		t.Fatal(err)
	}
	row, _, _ = s.Row("anthropic", active)
	if row.Health == nil || row.Health.State != "ok" || row.Health.At == "" {
		t.Fatalf("health = %+v", row.Health)
	}
}

func TestHintMasksBothEnds(t *testing.T) {
	for _, tc := range []struct {
		cred json.RawMessage
		want string
	}{
		{keyCred("sk-abcdefghijklmnop"), "sk-abcd…mnop"},
		{keyCred("short"), "••••"},
		{oauthCred("a", "r", 1, ""), ""},
	} {
		if got := Hint(tc.cred); got != tc.want {
			t.Errorf("Hint(%s) = %q, want %q", tc.cred, got, tc.want)
		}
	}
}
