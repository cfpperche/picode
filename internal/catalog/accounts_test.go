package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTwoAPIKeysSwitch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if err := PutAPIKey("xai", "sk-two"); err != nil {
		t.Fatal(err)
	}
	acc := AccountsFor("xai")
	if len(acc) != 2 {
		t.Fatalf("got %d accounts: %+v", len(acc), acc)
	}
	active := 0
	var firstID string
	for _, a := range acc {
		if a.Active {
			active++
			if a.Type != "api_key" {
				t.Fatalf("type %s", a.Type)
			}
		} else {
			firstID = a.ID
		}
	}
	if active != 1 {
		t.Fatalf("active %d", active)
	}
	if key := mustKey(t, home, "xai"); key != "sk-two" {
		t.Fatalf("auth.json %s", key)
	}
	if firstID == "" {
		t.Fatal("missing inactive")
	}
	if err := ActivateAccount("xai", firstID); err != nil {
		t.Fatal(err)
	}
	if key := mustKey(t, home, "xai"); key != "sk-one" {
		t.Fatalf("after switch %s", key)
	}
}

func TestRemoveActivePromotesOther(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if err := PutAPIKey("xai", "sk-two"); err != nil {
		t.Fatal(err)
	}
	var drop string
	for _, a := range AccountsFor("xai") {
		if a.Active {
			drop = a.ID
		}
	}
	if err := RemoveAccount("xai", drop); err != nil {
		t.Fatal(err)
	}
	acc := AccountsFor("xai")
	if len(acc) != 1 || !acc[0].Active {
		t.Fatalf("%+v", acc)
	}
	if key := mustKey(t, home, "xai"); key != "sk-one" {
		t.Fatalf("promoted %s", key)
	}
}

func TestRemoveLastClearsAuth(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	id := AccountsFor("xai")[0].ID
	if err := RemoveAccount("xai", id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".pi", "agent", "auth.json")); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".pi", "agent", "auth.json"))
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	if _, ok := m["xai"]; ok {
		t.Fatal("xai still in auth.json")
	}
	if n := len(AccountsFor("xai")); n != 0 {
		t.Fatalf("vault %d", n)
	}
}

func TestOAuthReloginUpdatesSameAccount(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutOAuth("anthropic", map[string]any{"type": "oauth", "access": "a1", "refresh": "r1", "expires": 1}); err != nil {
		t.Fatal(err)
	}
	if err := PutOAuth("anthropic", map[string]any{"type": "oauth", "access": "a2", "refresh": "r2", "expires": 2}); err != nil {
		t.Fatal(err)
	}
	acc := AccountsFor("anthropic")
	if len(acc) != 1 {
		t.Fatalf("got %d", len(acc))
	}
	raw, _ := os.ReadFile(filepath.Join(home, ".pi", "agent", "auth.json"))
	var m map[string]map[string]any
	_ = json.Unmarshal(raw, &m)
	if m["anthropic"]["refresh"] != "r2" {
		t.Fatalf("latest refresh not active: %s", raw)
	}
}

func TestRenameAccount(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	id := AccountsFor("xai")[0].ID
	if err := RenameAccount("xai", id, "work"); err != nil {
		t.Fatal(err)
	}
	if got := AccountsFor("xai")[0].Label; got != "work" {
		t.Fatalf("%s", got)
	}
}

func TestSameKeyDoesNotDuplicate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if err := PutAPIKey("xai", "sk-one"); err != nil {
		t.Fatal(err)
	}
	if n := len(AccountsFor("xai")); n != 1 {
		t.Fatalf("got %d", n)
	}
}

func TestVaultCredWithoutSwap(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := PutOAuth("anthropic", map[string]any{"type": "oauth", "access": "oa", "refresh": "or", "expires": 9}); err != nil {
		t.Fatal(err)
	}
	oauthID := AccountsFor("anthropic")[0].ID
	if err := PutAPIKey("anthropic", "sk-live"); err != nil {
		t.Fatal(err)
	}
	if ActiveAuthType("anthropic") != LoginAPIKey {
		t.Fatalf("active %s", ActiveAuthType("anthropic"))
	}
	cred, ok := VaultOAuth("anthropic", oauthID)
	if !ok || cred.Access != "oa" {
		t.Fatalf("vault oauth %+v ok=%v", cred, ok)
	}
	if err := UpdateOAuthTokensAccount("anthropic", oauthID, "oa2", "or2", 11); err != nil {
		t.Fatal(err)
	}
	if key := mustKey(t, home, "anthropic"); key != "sk-live" {
		t.Fatalf("auth.json swapped to %s", key)
	}
	cred, ok = VaultOAuth("anthropic", oauthID)
	if !ok || cred.Access != "oa2" || cred.Refresh != "or2" || cred.Expires != 11 {
		t.Fatalf("updated %+v ok=%v", cred, ok)
	}
	accs := AccountsFor("anthropic")
	var oauthKind string
	for _, a := range accs {
		if a.ID == oauthID {
			oauthKind = a.QuotaKind
		}
	}
	if oauthKind != LoginOAuth {
		t.Fatalf("quotaKind %q", oauthKind)
	}
}

func mustKey(t *testing.T, home, provider string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m[provider]["key"]
}
